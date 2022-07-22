package explorer

import (
	"context"
	"encoding/base64"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/jylc/cloudserver/models"
	"github.com/jylc/cloudserver/pkg/cache"
	"github.com/jylc/cloudserver/pkg/filesystem"
	"github.com/jylc/cloudserver/pkg/filesystem/fsctx"
	"github.com/jylc/cloudserver/pkg/serializer"
	"github.com/jylc/cloudserver/pkg/utils"
	"io/ioutil"
	"net/http"
	"net/url"
	"path"
	"strconv"
	"strings"
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

type SingleFileService struct {
	Path string `uri:"path" json:"path" binding:"required,min=1,max=65535"`
}

func (service *SingleFileService) Create(c *gin.Context) serializer.Response {
	fs, err := filesystem.NewFileSystemFromContext(c)
	if err != nil {
		return serializer.Err(serializer.CodePolicyNotAllowed, err.Error(), err)
	}
	defer fs.Recycle()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	fs.Use("BeforeUpload", filesystem.HookValidateFile)
	fs.Use("AfterUpload", filesystem.GenericAfterUpload)

	err = fs.Upload(ctx, &fsctx.FileStream{
		File:        ioutil.NopCloser(strings.NewReader("")),
		Size:        0,
		VirtualPath: path.Dir(service.Path),
		Name:        path.Base(service.Path),
	})
	if err != nil {
		return serializer.Err(serializer.CodeUploadFailed, err.Error(), err)
	}
	return serializer.Response{
		Code: 0,
	}
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

func (service *FileIDService) PutContent(ctx context.Context, c *gin.Context) serializer.Response {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	fileSize, err := strconv.ParseUint(c.Request.Header.Get("Content-Type"), 10, 64)
	if err != nil {
		return serializer.ParamErr("Unable to resolve file size", err)
	}
	fileData := fsctx.FileStream{
		MIMEType: c.Request.Header.Get("Content-Type"),
		File:     c.Request.Body,
		Size:     fileSize,
		Mode:     fsctx.Overwrite,
	}

	fs, err := filesystem.NewFileSystemFromContext(c)
	if err != nil {
		return serializer.Err(serializer.CodePolicyNotAllowed, err.Error(), err)
	}
	uploadCtx := context.WithValue(ctx, fsctx.GinCtx, c)

	fileID, _ := c.Get("object_id")
	originFile, _ := models.GetFilesByIDs([]uint{fileID.(uint)}, fs.User.ID)
	if len(originFile) == 0 {
		return serializer.Err(404, "file does not exist", nil)
	}
	fileData.Name = originFile[0].Name

	fileList, err := models.RemoveFilesWithSoftLinks([]models.File{originFile[0]})
	if err == nil && len(fileList) == 0 {
		originFile[0].SourceName = fs.GenerateSavePath(uploadCtx, &fileData)
		fileData.Mode &= ^fsctx.Overwrite
		fs.Use("AfterUpload", filesystem.HookUpdateSourceName)
		fs.Use("AfterUploadCanceled", filesystem.HookUpdateSourceName)
		fs.Use("AfterValidateFailed", filesystem.HookUpdateSourceName)
	}

	fs.Use("BeforeUpload", filesystem.HookResetPolicy)
	fs.Use("BeforeUpload", filesystem.HookValidateFile)
	fs.Use("BeforeUpload", filesystem.HookValidateCapacityDiff)
	fs.Use("AfterUploadCanceled", filesystem.HookCleanFileContent)
	fs.Use("AfterUploadCanceled", filesystem.HookClearFileSize)
	fs.Use("AfterUpload", filesystem.GenericAfterUpdate)
	fs.Use("AfterValidateFailed", filesystem.HookCleanFileContent)
	fs.Use("AfterValidateFailed", filesystem.HookClearFileSize)

	uploadCtx = context.WithValue(uploadCtx, fsctx.FileModelCtx, originFile[0])
	err = fs.Upload(uploadCtx, &fileData)
	if err != nil {
		return serializer.Err(serializer.CodeUploadFailed, err.Error(), err)
	}
	return serializer.Response{
		Code: 0,
	}
}

func (service *FileIDService) CreateDocPreviewSession(ctx context.Context, c *gin.Context) serializer.Response {
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
	downloadURL, err := fs.GetDownloadURL(ctx, objectID.(uint), "doc_preview_timeout")
	if err != nil {
		return serializer.Err(serializer.CodeNotSet, err.Error(), err)
	}

	srcB64 := base64.StdEncoding.EncodeToString([]byte(downloadURL))
	srcEncoded := url.QueryEscape(downloadURL)
	srcB64Encoded := url.QueryEscape(srcB64)
	return serializer.Response{
		Code: 0,
		Data: utils.Replace(models.GetSettingByName("office_preview_service"), map[string]string{
			"{$src}":    srcEncoded,
			"{$srcB64}": srcB64Encoded,
		}),
	}
}

func (service *ItemIDService) Sources(ctx context.Context, c *gin.Context) serializer.Response {
	fs, err := filesystem.NewFileSystemFromContext(c)
	if err != nil {
		return serializer.Err(serializer.CodePolicyNotAllowed, "Unable to initialize file system", err)
	}
	defer fs.Recycle()

	if len(service.Raw().Items) > fs.User.Group.OptionsSerialized.SourceBatchSize {
		return serializer.Err(serializer.CodeBatchSourceSize, "The maximum quantity limit for batch acquisition of external chains has been exceeded", nil)
	}
	res := make([]serializer.Sources, 0, len(service.Raw().Items))
	for _, id := range service.Raw().Items {
		fs.FileTarget = []models.File{}
		sourceURL, err := fs.GetSource(ctx, id)
		if len(fs.FileTarget) > 0 {
			current := serializer.Sources{
				URL:    sourceURL,
				Name:   fs.FileTarget[0].Name,
				Parent: fs.FileTarget[0].FolderID,
			}

			if err != nil {
				current.Error = err.Error()
			}
			res = append(res, current)
		}
	}
	return serializer.Response{
		Code: 0,
		Data: res,
	}
}
