package explorer

import (
	"context"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/jylc/cloudserver/models"
	"github.com/jylc/cloudserver/pkg/cache"
	"github.com/jylc/cloudserver/pkg/filesystem"
	"github.com/jylc/cloudserver/pkg/filesystem/fsctx"
	"github.com/jylc/cloudserver/pkg/serializer"
	"net/http"
	"net/url"
)

type FileAnonymousGetService struct {
	ID   uint   `uri:"id" binding:"required,min=1"`
	Name string `uri:"name" binding:"required"`
}

type DownloadService struct {
	ID string `uri:"id" binding:"required"`
}

type ArchiveService struct {
	ID string `uri:"sessionID" binding:"required"`
}

type FileIDService struct {
}

func (service *FileAnonymousGetService) Download(ctx context.Context, c *gin.Context) serializer.Response {
	fs, err := filesystem.NewAnonymousFileSystem()
	if err != nil {
		return serializer.Err(serializer.CodeGroupNotAllowed, err.Error(), err)
	}
	defer fs.Recycle()
	err = fs.SetTargetFileByIDs([]uint{service.ID})
	if err != nil {
		return serializer.Err(serializer.CodeNotSet, err.Error(), err)
	}
	rs, err := fs.GetDownloadContent(ctx, 0)
	defer rs.Close()
	if err != nil {
		return serializer.Err(serializer.CodeNotSet, err.Error(), err)
	}

	http.ServeContent(c.Writer, c.Request, service.Name, fs.FileTarget[0].UpdatedAt, rs)

	return serializer.Response{
		Code: 0,
	}
}

func (service *FileAnonymousGetService) Source(ctx context.Context, c *gin.Context) serializer.Response {
	fs, err := filesystem.NewAnonymousFileSystem()
	if err != nil {
		serializer.Err(serializer.CodeGroupNotAllowed, err.Error(), err)
	}
	defer fs.Recycle()

	err = fs.SetTargetFileByIDs([]uint{service.ID})
	if err != nil {
		serializer.Err(serializer.CodeNotSet, err.Error(), err)
	}

	res, err := fs.SignURL(ctx, &fs.FileTarget[0], int64(models.GetIntSetting("preview_timeout", 60)), false)
	if err != nil {
		return serializer.Err(serializer.CodeNotSet, err.Error(), err)
	}
	return serializer.Response{
		Code: -302,
		Data: res,
	}
}

func (service *DownloadService) Download(ctx context.Context, c *gin.Context) serializer.Response {
	fs, err := filesystem.NewFileSystemFromContext(c)
	if err != nil {
		return serializer.Err(serializer.CodePolicyNotAllowed, err.Error(), err)
	}
	defer fs.Recycle()

	file, exist := cache.Get("download_" + service.ID)
	if !exist {
		return serializer.Err(404, "file download session does not exist", nil)
	}
	fs.FileTarget = []models.File{file.(models.File)}

	ctx = context.WithValue(ctx, fsctx.GinCtx, c)
	rs, err := fs.GetDownloadContent(ctx, 0)
	if err != nil {
		return serializer.Err(serializer.CodeNotSet, err.Error(), err)
	}
	defer rs.Close()

	c.Header("Content-Disposition", "attachment; filename=\""+url.PathEscape(fs.FileTarget[0].Name)+"\"")

	if fs.User.Group.OptionsSerialized.OneTimeDownload {
		_ = cache.Deletes([]string{service.ID}, "download_")
	}

	http.ServeContent(c.Writer, c.Request, fs.FileTarget[0].Name, fs.FileTarget[0].UpdatedAt, rs)
	return serializer.Response{
		Code: 0,
	}
}

func (service *ArchiveService) DownloadArchived(ctx context.Context, c *gin.Context) serializer.Response {
	userRaw, exist := cache.Get("archive_user_" + service.ID)
	if !exist {
		serializer.Err(404, "archived session does not exist", nil)
	}
	user := userRaw.(models.User)

	fs, err := filesystem.NewFileSystem(&user)
	if err != nil {
		return serializer.Err(serializer.CodePolicyNotAllowed, err.Error(), err)
	}
	defer fs.Recycle()

	archiveSession, exist := cache.Get("archive_" + service.ID)
	if !exist {
		return serializer.Err(404, "archived session does not exist", nil)
	}

	c.Header("Content-Disposition", "attachment")
	c.Header("Content-Type", "application/zip")
	itemService := archiveSession.(ItemIDService)
	items := itemService.Raw()
	ctx = context.WithValue(ctx, fsctx.GinCtx, c)
	err = fs.Compress(ctx, c.Writer, items.Dirs, items.Items, true)
	if err != nil {
		return serializer.Err(serializer.CodeNotSet, "unable to create compressed file", err)
	}
	return serializer.Response{
		Code: 0,
	}
}

func (service *FileIDService) PreviewContent(ctx context.Context, c *gin.Context, isText bool) serializer.Response {
	fs, err := filesystem.NewFileSystemFromContext(c)
	if err != nil {
		return serializer.Err(serializer.CodePolicyNotAllowed, err.Error(), err)
	}
	defer fs.Recycle()

	objectID, _ := c.Get("object_id")

	if file, ok := ctx.Value(fsctx.FileModelCtx).(*models.File); ok {
		fs.SetTargetFile(&[]models.File{*file})
		objectID = uint(0)
	}

	if folder, ok := ctx.Value(fsctx.FolderModelCtx).(*models.Folder); ok {
		fs.Root = folder
		path := ctx.Value(fsctx.PathCtx).(string)
		err := fs.ResetFileIfNotExist(ctx, path)
		if err != nil {
			return serializer.Err(serializer.CodeNotFound, err.Error(), err)
		}
		objectID = uint(0)
	}

	resp, err := fs.Preview(ctx, objectID.(uint), isText)
	if err != nil {
		return serializer.Err(serializer.CodeNotSet, err.Error(), err)
	}

	if resp.Redirect {
		c.Header("Cache-Control", fmt.Sprintf("max-age=%d", resp.MaxAge))
		return serializer.Response{
			Code: -301,
			Data: resp.URL,
		}
	}
	defer resp.Content.Close()

	if isText {
		c.Header("Cache-Control", "no-cache")
	}

	http.ServeContent(c.Writer, c.Request, fs.FileTarget[0].Name, fs.FileTarget[0].UpdatedAt, resp.Content)
	return serializer.Response{
		Code: 0,
	}
}

func (service *FileIDService) CreateDownloadSession(ctx context.Context, c *gin.Context) serializer.Response {
	fs, err := filesystem.NewFileSystemFromContext(c)
	if err != nil {
		return serializer.Err(serializer.CodePolicyNotAllowed, err.Error(), err)
	}
	defer fs.Recycle()

	objectID, _ := c.Get("object_id")

	downloadURL, err := fs.GetDownloadURL(ctx, objectID.(uint), "download_timeout")
	if err != nil {
		return serializer.Err(serializer.CodeNotSet, err.Error(), err)
	}
	return serializer.Response{
		Code: 0,
		Data: downloadURL,
	}
}
