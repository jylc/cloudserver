package filesystem

import (
	"github.com/jylc/cloudserver/models"
	"github.com/jylc/cloudserver/pkg/filesystem/driver"
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
