package admin

import (
	"context"
	"github.com/gin-gonic/gin"
	"github.com/jylc/cloudserver/models"
	"github.com/jylc/cloudserver/pkg/filesystem"
	"github.com/jylc/cloudserver/pkg/filesystem/fsctx"
	"github.com/jylc/cloudserver/pkg/serializer"
	"github.com/jylc/cloudserver/service/explorer"
	"strings"
)

type FileService struct {
	ID uint `uri:"id" json:"id" binding:"required"`
}

type FileBatchService struct {
	ID    []uint `json:"id" binding:"min=1"`
	Force bool   `json:"force"`
}

type ListFolderService struct {
	Path string `uri:"path" binding:"required,max=65535"`
	ID   uint   `uri:"id" binding:"required"`
	Type string `uri:"type" binding:"eq=policy|eq=user"`
}

func (service *ListService) Files() serializer.Response {
	var res []models.File
	total := int64(0)

	tx := models.Db.Model(&models.File{})
	if service.OrderBy != "" {
		tx = tx.Order(service.OrderBy)
	}

	for k, v := range service.Conditions {
		tx = tx.Where(k+" = ?", v)
	}

	if len(service.Searches) > 0 {
		search := ""
		for k, v := range service.Searches {
			search += k + " like '%" + v + "%' OR "
		}
		search = strings.TrimPrefix(search, " OR ")
		tx = tx.Where(search)
	}

	tx.Count(&total)
	tx.Limit(service.PageSize).Offset((service.Page - 1) * service.PageSize).Find(&res)

	users := make(map[uint]models.User)
	for _, file := range res {
		users[file.UserID] = models.User{}
	}

	userIDs := make([]uint, 0, len(users))
	for k := range users {
		userIDs = append(userIDs, k)
	}

	var userList []models.User
	models.Db.Where("id in (?)", userIDs).Find(&userList)

	for _, v := range userList {
		users[v.ID] = v
	}

	return serializer.Response{Data: map[string]interface{}{
		"total": total,
		"items": res,
		"users": users,
	}}
}

func (service *FileService) Get(c *gin.Context) serializer.Response {
	file, err := models.GetFilesByIDs([]uint{service.ID}, 0)
	if err != nil {
		return serializer.Err(serializer.CodeNotFound, "file does not exist", err)
	}

	ctx := context.WithValue(context.Background(), fsctx.FileModelCtx, &file[0])
	var subService explorer.FileIDService
	res := subService.PreviewContent(ctx, c, false)
	return res
}

func (service *FileBatchService) Delete(c *gin.Context) serializer.Response {
	files, err := models.GetFilesByIDs(service.ID, 0)
	if err != nil {
		return serializer.DBErr("Unable to list files to be deleted", err)
	}
	userFile := make(map[uint][]models.File)
	for i := 0; i < len(files); i++ {
		if _, ok := userFile[files[i].UserID]; !ok {
			userFile[files[i].UserID] = []models.File{}
		}
		userFile[files[i].UserID] = append(userFile[files[i].UserID], files[i])
	}

	go func(files map[uint][]models.File) {
		for uid, file := range files {
			user, err := models.GetUserByID(uid)
			if err != nil {
				continue
			}

			fs, err := filesystem.NewFileSystem(&user)
			if err != nil {
				fs.Recycle()
				continue
			}
			ids := make([]uint, 0, len(file))
			for i := 0; i < len(file); i++ {
				ids = append(ids, file[i].ID)
			}
			fs.Delete(context.Background(), []uint{}, ids, service.Force)
			fs.Recycle()
		}
	}(userFile)
	return serializer.Response{}
}

func (service *ListFolderService) List(c *gin.Context) serializer.Response {
	if service.Type == "policy" {
		policy, err := models.GetPolicyByID(service.ID)
		if err != nil {
			return serializer.Err(serializer.CodeNotFound, "Storage policy does not exist", err)
		}

		fs, err := filesystem.NewAnonymousFileSystem()
		if err != nil {
			return serializer.Err(serializer.CodeInternalSetting, "Unable to create file system", err)
		}
		defer fs.Recycle()

		fs.Policy = &policy
		res, err := fs.ListPhysical(c.Request.Context(), service.Path)
		if err != nil {
			return serializer.Err(serializer.CodeIOFailed, "Unable to list directories", err)
		}
		return serializer.Response{
			Data: serializer.BuildObjectList(0, res, nil),
		}
	}
	user, err := models.GetUserByID(service.ID)
	if err != nil {
		return serializer.Err(serializer.CodeNotFound, "user does not exist", err)
	}

	fs, err := filesystem.NewFileSystem(&user)
	if err != nil {
		return serializer.Err(serializer.CodeInternalSetting, "Unable to create file system", err)
	}
	defer fs.Recycle()

	res, err := fs.List(c.Request.Context(), service.Path, nil)
	if err != nil {
		return serializer.Err(serializer.CodeIOFailed, "Unable to list directories", err)
	}

	return serializer.Response{Data: serializer.BuildObjectList(0, res, nil)}
}
