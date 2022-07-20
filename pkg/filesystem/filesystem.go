package filesystem

import (
	"context"
	"errors"
	"github.com/gin-gonic/gin"
	"github.com/jylc/cloudserver/models"
	"github.com/jylc/cloudserver/pkg/cluster"
	"github.com/jylc/cloudserver/pkg/filesystem/driver"
	"github.com/jylc/cloudserver/pkg/filesystem/driver/local"
	"github.com/jylc/cloudserver/pkg/filesystem/driver/remote"
	"github.com/jylc/cloudserver/pkg/filesystem/driver/shadow/slaveinmaster"
	"github.com/jylc/cloudserver/pkg/filesystem/fsctx"
	"github.com/jylc/cloudserver/pkg/serializer"
	"sync"
)

var FSPool = sync.Pool{
	New: func() interface{} {
		return &FileSystem{}
	},
}

type FileSystem struct {
	User       *models.User
	Policy     *models.Policy
	FileTarget []models.File
	DirTarget  []models.Folder
	Root       *models.Folder
	Lock       sync.Mutex

	Hooks       map[string][]Hook
	Handler     driver.Handler
	recycleLock sync.Mutex
}

func getEmptyFS() *FileSystem {
	fs := FSPool.Get().(*FileSystem)
	return fs
}

func (fs *FileSystem) Recycle() {
	fs.recycleLock.Lock()
}

func (fs *FileSystem) reset() {
	fs.User = nil
	fs.CleanTargets()
	fs.Policy = nil
	fs.Hooks = nil
	fs.Handler = nil
	fs.Root = nil
	fs.Lock = sync.Mutex{}
	fs.recycleLock = sync.Mutex{}
}

func (fs *FileSystem) CleanTargets() {
	fs.FileTarget = fs.FileTarget[:0]
	fs.DirTarget = fs.DirTarget[:0]
}

func NewAnonymousFileSystem() (*FileSystem, error) {
	fs := getEmptyFS()
	fs.User = &models.User{}

	anonymousGroup, err := models.GetGroupByID(3)
	if err != nil {
		fs.User.Group = anonymousGroup
	}
	return fs, nil
}

func (fs *FileSystem) SetTargetFileByIDs(ids []uint) error {
	files, err := models.GetFilesByIDs(ids, 0)
	if err != nil || len(files) == 0 {
		return ErrFileExisted.WithError(err)
	}
	fs.SetTargetFile(&files)
	return nil
}

func (fs *FileSystem) SetTargetFile(files *[]models.File) {
	if len(fs.FileTarget) == 0 {
		fs.FileTarget = *files
	} else {
		fs.FileTarget = append(fs.FileTarget, *files...)
	}
}

func (fs *FileSystem) SignURL(ctx context.Context, file *models.File, ttl int64, isDownload bool) (string, error) {
	fs.FileTarget = []models.File{*file}
	ctx = context.WithValue(ctx, fsctx.FileModelCtx, *file)

	err := fs.resetPolicyToFirstFile(ctx)
	if err != nil {
		return "", err
	}

	siteURL := models.GetSiteURL()
	source, err := fs.Handler.Source(ctx, fs.FileTarget[0].SourceName, *siteURL, ttl, isDownload, fs.User.Group.SpeedLimit)
	if err != nil {
		return "", serializer.NewError(serializer.CodeNotSet, "cannot get external link", err)
	}
	return source, nil
}

func (fs *FileSystem) resetPolicyToFirstFile(ctx context.Context) error {
	if len(fs.FileTarget) == 0 {
		return ErrObjectNotExist
	}

	fs.Policy = fs.FileTarget[0].GetPolicy()
	err := fs.DispatchHandler()
	if err != nil {
		return err
	}
	return nil
}

func (fs *FileSystem) DispatchHandler() error {
	if fs.Policy == nil {
		return errors.New("have not set policy")
	}
	policyType := fs.Policy.Type
	currentType := fs.Policy

	switch policyType {
	case "mock", "anonymous":
		return nil
	case "local":
		fs.Handler = local.Driver{
			Policy: currentType,
		}
		return nil
	case "remote":
		handler, err := remote.NewDriver(currentType)
		if err != nil {
			return err
		}
		fs.Handler = handler
	default:
		return ErrUnknownPolicyType
	}
	return nil
}

func NewFileSystem(user *models.User) (*FileSystem, error) {
	fs := getEmptyFS()
	fs.User = user
	fs.Policy = &fs.User.Policy

	err := fs.DispatchHandler()
	return fs, err
}

func NewFileSystemFromContext(c *gin.Context) (*FileSystem, error) {
	user, exists := c.Get("user")
	if !exists {
		return NewAnonymousFileSystem()
	}
	fs, err := NewFileSystem(user.(*models.User))
	return fs, err
}

func (fs *FileSystem) SetTargetDir(dirs *[]models.Folder) {
	if len(fs.DirTarget) == 0 {
		fs.DirTarget = *dirs
	} else {
		fs.DirTarget = append(fs.DirTarget, *dirs...)
	}
}

func (fs *FileSystem) SwitchToSlaveHandler(node cluster.Node) {
	fs.Handler = slaveinmaster.NewDriver(node, fs.Handler, fs.Policy)
}
