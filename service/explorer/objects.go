package explorer

import (
	"context"
	"encoding/gob"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/jylc/cloudserver/models"
	"github.com/jylc/cloudserver/pkg/auth"
	"github.com/jylc/cloudserver/pkg/cache"
	"github.com/jylc/cloudserver/pkg/filesystem"
	"github.com/jylc/cloudserver/pkg/hashid"
	"github.com/jylc/cloudserver/pkg/serializer"
	"github.com/jylc/cloudserver/pkg/task"
	"github.com/jylc/cloudserver/pkg/utils"
	"math"
	"path"
	"strings"
	"time"
)

type ItemMoveService struct {
	SrcDir string        `json:"src_dir" binding:"required,min=1,max=65535"`
	Src    ItemIDService `json:"src"`
	Dst    string        `json:"dst" binding:"required,min=1,max=65535"`
}

type ItemRenameService struct {
	Src     ItemIDService `json:"src"`
	NewName string        `json:"new_name" binding:"required,min=1,max=255"`
}

type ItemService struct {
	Items []uint `json:"items"`
	Dirs  []uint `json:"dirs"`
}

type ItemIDService struct {
	Items  []string `json:"items"`
	Dirs   []string `json:"dirs"`
	Source *ItemService
}

type ItemCompressService struct {
	Src  ItemIDService `json:"src"`
	Dst  string        `json:"dst" binding:"required,min=1,max=65535"`
	Name string        `json:"name" binding:"required,min=1,max=255"`
}

type ItemDecompressService struct {
	Src      string `json:"src"`
	Dst      string `json:"dst" binding:"required,min=1,max=65535"`
	Encoding string `json:"encoding"`
}

type ItemPropertyService struct {
	ID        string `binding:"required"`
	TraceRoot bool   `form:"trace_root"`
	IsFolder  bool   `form:"is_folder"`
}

func init() {
	gob.Register(ItemIDService{})
}

func (service *ItemIDService) Raw() *ItemService {
	if service.Source != nil {
		return service.Source
	}

	service.Source = &ItemService{
		Dirs:  make([]uint, 0, len(service.Dirs)),
		Items: make([]uint, 0, len(service.Items)),
	}

	for _, folder := range service.Dirs {
		id, err := hashid.DecodeHashID(folder, hashid.FolderID)
		if err == nil {
			service.Source.Dirs = append(service.Source.Dirs, id)
		}
	}

	for _, file := range service.Items {
		id, err := hashid.DecodeHashID(file, hashid.FileID)
		if err == nil {
			service.Source.Items = append(service.Source.Items, id)
		}
	}
	return service.Source
}

func (service *ItemIDService) Archive(ctx context.Context, c *gin.Context) serializer.Response {
	fs, err := filesystem.NewFileSystemFromContext(c)
	if err != nil {
		return serializer.Err(serializer.CodePolicyNotAllowed, err.Error(), err)
	}
	defer fs.Recycle()

	if !fs.User.Group.OptionsSerialized.ArchiveDownload {
		return serializer.Err(serializer.CodeGroupNotAllowed, "The current user group cannot perform this operation", nil)
	}

	ttl := models.GetIntSetting("archive_timeout", 30)
	downloadSessionID := utils.RandStringRunes(16)
	cache.Set("archive_"+downloadSessionID, *service, ttl)
	cache.Set("archive_user_"+downloadSessionID, *fs.User, ttl)
	signURL, err := auth.SignURI(
		auth.General,
		fmt.Sprintf("/api/v3/file/archive/%s/archive.zip", downloadSessionID),
		int64(ttl),
	)

	return serializer.Response{
		Code: 0,
		Data: signURL.String(),
	}

}

func (service *ItemCompressService) CreateCompressTask(c *gin.Context) serializer.Response {
	fs, err := filesystem.NewFileSystemFromContext(c)
	if err != nil {
		return serializer.Err(serializer.CodePolicyNotAllowed, err.Error(), err)
	}
	defer fs.Recycle()

	if !fs.User.Group.OptionsSerialized.ArchiveTask {
		return serializer.Err(serializer.CodeGroupNotAllowed, "The current user group cannot perform this operation", nil)
	}

	if !strings.HasSuffix(service.Name, ".zip") {
		service.Name += ".zip"
	}

	if exist, _ := fs.IsPathExist(service.Dst); !exist {
		return serializer.Err(serializer.CodeNotFound, "Storage path does not exist", nil)
	}
	if exist, _ := fs.IsPathExist(path.Join(service.Dst, service.Name)); exist {
		return serializer.ParamErr("A file named "+service.Name+" already exists", nil)
	}

	if !fs.ValidateLegalName(context.Background(), service.Name) {
		return serializer.ParamErr("Illegal file name", nil)
	}

	if !fs.ValidateExtension(context.Background(), service.Name) {
		return serializer.ParamErr("Files with this extension are not allowed to be stored", nil)
	}

	folders, err := models.GetRecursiveChildFolder(service.Src.Raw().Dirs, fs.User.ID, true)
	if err != nil {
		return serializer.Err(serializer.CodeDBError, "Unable to list subdirectories", err)
	}

	files, err := models.GetChildFilesOfFolders(&folders)
	if err != nil {
		return serializer.Err(serializer.CodeDBError, "Unable to list sub files", err)
	}

	var totalSize uint64
	for i := 0; i < len(files); i++ {
		totalSize += files[i].Size
	}

	if fs.User.Group.OptionsSerialized.CompressSize != 0 && totalSize > fs.User.Group.OptionsSerialized.CompressSize {
		return serializer.Err(serializer.CodeParamErr, "文件太大", nil)
	}

	compressRatio := 0.4
	spaceNeeded := uint64(math.Round(float64(totalSize) * compressRatio))
	if fs.User.GetRemainingCapacity() < spaceNeeded {
		return serializer.Err(serializer.CodeParamErr, "剩余空间不足", err)
	}

	job, err := task.NewCompressTask(fs.User, path.Join(service.Dst, service.Name), service.Src.Raw().Dirs, service.Src.Raw().Items)
	if err != nil {
		return serializer.Err(serializer.CodeNotSet, "任务创建失败", err)
	}

	task.TaskPool.Submit(job)
	return serializer.Response{}
}

func (service *ItemDecompressService) CreateDecompressTask(c *gin.Context) serializer.Response {
	fs, err := filesystem.NewFileSystemFromContext(c)
	if err != nil {
		return serializer.Err(serializer.CodePolicyNotAllowed, err.Error(), err)
	}
	defer fs.Recycle()

	if !fs.User.Group.OptionsSerialized.ArchiveTask {
		return serializer.Err(serializer.CodeGroupNotAllowed, "The current user group cannot perform this operation", nil)
	}

	if exist, _ := fs.IsPathExist(service.Dst); !exist {
		return serializer.Err(serializer.CodeNotFound, "Storage path does not exist", nil)
	}

	exist, file := fs.IsFileExist(service.Src)
	if !exist {
		return serializer.Err(serializer.CodeNotFound, "file does not exist", nil)
	}

	if fs.User.Group.OptionsSerialized.DecompressSize != 0 && file.Size > fs.User.Group.
		OptionsSerialized.DecompressSize {
		return serializer.Err(serializer.CodeParamErr, "File too large", nil)
	}

	var (
		suffixes = []string{".zip", ".gz", ".xz", ".tar", ".rar"}
		matched  bool
	)
	for _, suffix := range suffixes {
		if strings.HasSuffix(file.Name, suffix) {
			matched = true
			break
		}
	}
	if !matched {
		return serializer.Err(serializer.CodeParamErr, "Compressed files in this format are not supported", nil)
	}

	job, err := task.NewDecompressTask(fs.User, service.Src, service.Dst, service.Encoding)
	if err != nil {
		return serializer.Err(serializer.CodeNotSet, "Task creation failed", err)
	}
	task.TaskPool.Submit(job)

	return serializer.Response{}
}

func (service *ItemIDService) Delete(ctx context.Context, c *gin.Context) serializer.Response {
	fs, err := filesystem.NewFileSystemFromContext(c)
	if err != nil {
		return serializer.Err(serializer.CodePolicyNotAllowed, err.Error(), err)
	}
	defer fs.Recycle()

	items := service.Raw()
	err = fs.Delete(ctx, items.Dirs, items.Items, false)
	if err != nil {
		return serializer.Err(serializer.CodeNotSet, err.Error(), err)
	}

	return serializer.Response{
		Code: 0,
	}
}

func (service *ItemMoveService) Move(ctx context.Context, c *gin.Context) serializer.Response {
	fs, err := filesystem.NewFileSystemFromContext(c)
	if err != nil {
		return serializer.Err(serializer.CodePolicyNotAllowed, err.Error(), err)
	}
	defer fs.Recycle()

	items := service.Src.Raw()
	err = fs.Move(ctx, items.Dirs, items.Items, service.SrcDir, service.Dst)
	if err != nil {
		return serializer.Err(serializer.CodeNotSet, err.Error(), err)
	}
	return serializer.Response{
		Code: 0,
	}
}

func (service *ItemMoveService) Copy(ctx context.Context, c *gin.Context) serializer.Response {
	if len(service.Src.Items)+len(service.Src.Dirs) > 1 {
		return serializer.ParamErr("Only one object can be copied", nil)
	}
	fs, err := filesystem.NewFileSystemFromContext(c)
	if err != nil {
		return serializer.Err(serializer.CodePolicyNotAllowed, err.Error(), err)
	}
	defer fs.Recycle()
	err = fs.Copy(ctx, service.Src.Raw().Dirs, service.Src.Raw().Items, service.SrcDir, service.Dst)
	if err != nil {
		return serializer.Err(serializer.CodeNotSet, err.Error(), err)
	}

	return serializer.Response{
		Code: 0,
	}
}

func (service *ItemRenameService) Rename(ctx context.Context, c *gin.Context) serializer.Response {
	if len(service.Src.Items)+len(service.Src.Dirs) > 1 {
		return serializer.ParamErr("Only one object can be manipulated", nil)
	}
	fs, err := filesystem.NewFileSystemFromContext(c)
	if err != nil {
		return serializer.Err(serializer.CodePolicyNotAllowed, err.Error(), err)
	}
	defer fs.Recycle()
	err = fs.Rename(ctx, service.Src.Raw().Dirs, service.Src.Raw().Items, service.NewName)
	if err != nil {
		return serializer.Err(serializer.CodeNotSet, err.Error(), err)
	}

	return serializer.Response{
		Code: 0,
	}
}

func (service *ItemPropertyService) GetProperty(ctx context.Context, c *gin.Context) serializer.Response {
	userCtx, _ := c.Get("user")
	user := userCtx.(*models.User)

	var props serializer.ObjectProps
	props.QueryDate = time.Now()

	if !service.IsFolder {
		res, err := hashid.DecodeHashID(service.ID, hashid.FileID)
		if err != nil {
			return serializer.Err(serializer.CodeNotFound, "对象不存在", err)
		}

		file, err := models.GetFilesByIDs([]uint{res}, user.ID)
		if err != nil {
			return serializer.DBErr("找不到文件", err)
		}

		props.CreateAt = file[0].CreatedAt
		props.UpdateAt = file[0].UpdatedAt
		props.Policy = file[0].GetPolicy().Name
		props.Size = file[0].Size

		if service.TraceRoot {
			parent, err := models.GetFolderByIDs([]uint{file[0].FolderID}, user.ID)
			if err != nil {
				return serializer.DBErr("找不到父目录", err)
			}

			if err := parent[0].TraceRoot(); err != nil {
				return serializer.DBErr("无法溯源父目录", err)
			}

			props.Path = path.Join(parent[0].Position, parent[0].Name)
		}
	} else {
		res, err := hashid.DecodeHashID(service.ID, hashid.FolderID)
		if err != nil {
			return serializer.Err(serializer.CodeNotFound, "对象不存在", err)
		}
		folder, err := models.GetFolderByIDs([]uint{res}, user.ID)
		if err != nil {
			return serializer.DBErr("找不到目录", err)
		}

		props.CreateAt = folder[0].CreatedAt
		props.UpdateAt = folder[0].UpdatedAt

		if cacheRes, ok := cache.Get(fmt.Sprintf("folder_props_%d", res)); ok {
			res := cacheRes.(serializer.ObjectProps)
			res.CreateAt = props.CreateAt
			res.UpdateAt = props.UpdateAt
			return serializer.Response{Data: res}
		}

		childFolders, err := models.GetRecursiveChildFolder([]uint{folder[0].ID}, user.ID, true)
		if err != nil {
			return serializer.DBErr("无法列取子目录", err)
		}

		props.ChildFolderNum = len(childFolders) - 1

		files, err := models.GetChildFilesOfFolders(&childFolders)
		if err != nil {
			return serializer.DBErr("无法列取子文件", err)
		}

		props.ChildFileNum = len(files)
		for i := 0; i < len(files); i++ {
			props.Size += files[i].Size
		}

		if service.TraceRoot {
			if err := folder[0].TraceRoot(); err != nil {
				return serializer.DBErr("无法溯源父目录", err)
			}
			props.Path = folder[0].Position
		}

		cache.Set(fmt.Sprintf("folder_props_%d", res), props,
			models.GetIntSetting("folder_props_timeout", 300))
	}

	return serializer.Response{
		Code: 0,
		Data: props,
	}
}
