package explorer

import (
	"context"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/jylc/cloudserver/models"
	"github.com/jylc/cloudserver/pkg/auth"
	"github.com/jylc/cloudserver/pkg/cache"
	"github.com/jylc/cloudserver/pkg/filesystem"
	"github.com/jylc/cloudserver/pkg/filesystem/driver/local"
	"github.com/jylc/cloudserver/pkg/filesystem/fsctx"
	"github.com/jylc/cloudserver/pkg/hashid"
	"github.com/jylc/cloudserver/pkg/serializer"
	"github.com/sirupsen/logrus"
	"io/ioutil"
	"strconv"
	"strings"
	"time"
)

type UploadService struct {
	ID    string `uri:"sessionId" binding:"required"`
	Index int    `uri:"index" form:"index" binding:"min=0"`
}

type CreateUploadSessionService struct {
	Path         string `json:"path" binding:"required"`
	Size         uint64 `json:"size" binding:"min=0"`
	Name         string `json:"name" binding:"required"`
	PolicyID     string `json:"policy_id" binding:"required"`
	LastModified int64  `json:"last_modified"`
}

func (service *CreateUploadSessionService) Create(ctx context.Context, c *gin.Context) serializer.Response {
	fs, err := filesystem.NewFileSystemFromContext(c)
	if err != nil {
		return serializer.Err(serializer.CodePolicyNotAllowed, err.Error(), err)
	}

	rawID, err := hashid.DecodeHashID(service.PolicyID, hashid.PolicyID)
	if err != nil {
		return serializer.Err(serializer.CodeNotFound, "Storage policy does not exist", err)
	}

	if fs.Policy.ID != rawID {
		return serializer.Err(serializer.CodePolicyNotAllowed, "The storage policy has changed, please refresh the file list and add this task again", nil)
	}

	file := &fsctx.FileStream{
		Size:        service.Size,
		Name:        service.Name,
		VirtualPath: service.Path,
		File:        ioutil.NopCloser(strings.NewReader("")),
	}

	if service.LastModified > 0 {
		lastModified := time.UnixMilli(service.LastModified)
		file.LastModified = &lastModified
	}

	credential, err := fs.CreateUploadSession(ctx, file)
	if err != nil {
		return serializer.Err(serializer.CodeNotSet, err.Error(), err)
	}
	return serializer.Response{
		Code: 0,
		Data: credential,
	}
}

func (service *UploadService) SlaveUpload(ctx context.Context, c *gin.Context) serializer.Response {
	uploadSessionRaw, ok := cache.Get(filesystem.UploadSessionCachePrefix + service.ID)
	if !ok {
		serializer.Err(serializer.CodeUploadSessionExpired, "slave upload session expired or not exist", nil)
	}
	uploadSession := uploadSessionRaw.(serializer.UploadSession)

	fs, err := filesystem.NewAnonymousFileSystem()
	if err != nil {
		return serializer.Err(serializer.CodePolicyNotAllowed, err.Error(), err)
	}
	fs.Handler = local.Driver{}

	service.Index, _ = strconv.Atoi(c.Query("chunk"))
	mode := fsctx.Append
	if c.GetHeader(auth.CrHeaderPrefix+"Overwrite") == "true" {
		mode |= fsctx.Overwrite
	}
	return processChunkUpload(ctx, c, fs, &uploadSession, service.Index, nil, mode)
}

func processChunkUpload(ctx context.Context, c *gin.Context, fs *filesystem.FileSystem, session *serializer.UploadSession, index int, file *models.File, mode fsctx.WriteMode) serializer.Response {
	chunkSize := session.Policy.OptionsSerialized.ChunkSize
	isLastChunk := session.Policy.OptionsSerialized.ChunkSize == 0 || uint64(index+1)*chunkSize >= session.Size
	expectedLength := chunkSize
	if isLastChunk {
		expectedLength = session.Size - uint64(index)*chunkSize
	}

	fileSize, err := strconv.ParseUint(c.Request.Header.Get("Content-Length"), 10, 64)
	if err != nil || (expectedLength != fileSize) {
		return serializer.Err(
			serializer.CodeInvalidContentLength,
			fmt.Sprintf("Invalid Content-Length (expected: %d)", expectedLength),
			err)
	}

	if index > 0 {
		mode |= fsctx.Overwrite
	}

	fileData := fsctx.FileStream{
		MIMEType:     c.Request.Header.Get("Content-Type"),
		File:         c.Request.Body,
		Size:         fileSize,
		Name:         session.Name,
		VirtualPath:  session.VirtualPath,
		SavePath:     session.SavePath,
		Mode:         mode,
		AppendStart:  chunkSize * uint64(index),
		Model:        file,
		LastModified: session.LastModified,
	}

	fs.Use("AfterUploadCanceled", filesystem.HookTruncateFileTo(fileData.AppendStart))
	fs.Use("AfterValidateFailed", filesystem.HookTruncateFileTo(fileData.AppendStart))

	if file != nil {
		fs.Use("BeforeUpload", filesystem.HookValidateCapacity)
		fs.Use("AfterUpload", filesystem.HookChunkUploaded)
		fs.Use("AfterValidateFailed", filesystem.HookChunkUploadFailed)
		if isLastChunk {
			fs.Use("AfterUpload", filesystem.HookPopPlaceholderToFile(""))
			fs.Use("AfterUpload", filesystem.HookGenerateThumb)
			fs.Use("AfterUpload", filesystem.HookDeleteUploadSession(session.Key))
		}
	} else {
		if isLastChunk {
			fs.Use("AfterUpload", filesystem.SlaveAfterUpload(session))
			fs.Use("AfterUpload", filesystem.HookDeleteUploadSession(session.Key))
		}
	}

	uploadCtx := context.WithValue(ctx, fsctx.GinCtx, c)
	err = fs.Upload(uploadCtx, &fileData)
	if err != nil {
		return serializer.Err(serializer.CodeUploadFailed, err.Error(), err)
	}
	return serializer.Response{}
}

type UploadSessionService struct {
	ID string `uri:"sessionId" binding:"required"`
}

func (service *UploadSessionService) Delete(ctx context.Context, c *gin.Context) serializer.Response {
	fs, err := filesystem.NewFileSystemFromContext(c)
	if err != nil {
		return serializer.Err(serializer.CodePolicyNotAllowed, err.Error(), err)
	}
	defer fs.Recycle()
	file, err := models.GetFilesByUploadSession(service.ID, fs.User.ID)
	if err != nil {
		return serializer.Err(serializer.CodeUploadSessionExpired, "Local Upload session file placeholder not exist", err)
	}
	if err := fs.Delete(ctx, []uint{}, []uint{file.ID}, false); err != nil {
		return serializer.Err(serializer.CodeInternalSetting, "Failed to delete upload session", err)
	}
	return serializer.Response{}
}

func (service *UploadSessionService) SlaveDelete(ctx context.Context, c *gin.Context) serializer.Response {
	fs, err := filesystem.NewAnonymousFileSystem()
	if err != nil {
		return serializer.Err(serializer.CodePolicyNotAllowed, err.Error(), err)
	}
	defer fs.Recycle()
	session, ok := cache.Get(filesystem.UploadSessionCachePrefix + service.ID)
	if !ok {
		return serializer.Err(serializer.CodeUploadSessionExpired, "Slave Upload session file placeholder not exist", nil)
	}

	if _, err := fs.Handler.Delete(ctx, []string{session.(serializer.UploadSession).SavePath}); err != nil {
		return serializer.Err(serializer.CodeInternalSetting, "Failed to delete temp file", err)
	}
	cache.Deletes([]string{service.ID}, filesystem.UploadSessionCachePrefix)
	return serializer.Response{}
}

func (service *UploadService) LocalUpload(ctx context.Context, c *gin.Context) serializer.Response {
	uploadSessionRaw, ok := cache.Get(filesystem.UploadSessionCachePrefix + service.ID)
	if !ok {
		return serializer.Err(serializer.CodeUploadSessionExpired, "LocalUpload session expired or not exist", nil)
	}

	uploadSession := uploadSessionRaw.(serializer.UploadSession)

	fs, err := filesystem.NewFileSystemFromContext(c)
	if err != nil {
		return serializer.Err(serializer.CodePolicyNotAllowed, err.Error(), err)
	}

	if uploadSession.UID != fs.User.ID {
		return serializer.Err(serializer.CodeUploadSessionExpired, "Local upload session expired or not exist", nil)
	}

	file, err := models.GetFilesByUploadSession(service.ID, fs.User.ID)
	if err != nil {
		return serializer.Err(serializer.CodeUploadSessionExpired, "Local upload session file placeholder not exist", err)
	}

	if !uploadSession.Policy.IsTransitUpload(uploadSession.Size) {
		return serializer.Err(serializer.CodePolicyNotAllowed, "Storage policy not supported", err)
	}

	fs.Policy = &uploadSession.Policy
	if err := fs.DispatchHandler(); err != nil {
		return serializer.Err(serializer.CodePolicyNotAllowed, "Unknown storage policy", err)
	}

	expextedSizeStart := file.Size
	actualSizeStart := uint64(service.Index) * uploadSession.Policy.OptionsSerialized.ChunkSize
	if uploadSession.Policy.OptionsSerialized.ChunkSize == 0 && service.Index > 0 {
		return serializer.Err(serializer.CodeInvalidChunkIndex, "Chunk index cannot be greater than 0", nil)
	}

	if expextedSizeStart < actualSizeStart {
		return serializer.Err(serializer.CodeInvalidChunkIndex, "Chunk must be uploaded in order", nil)
	}

	if expextedSizeStart > actualSizeStart {
		logrus.Info("Attempt to upload overlay fragment [%d] start=%d", service.Index, actualSizeStart)
	}
	return processChunkUpload(ctx, c, fs, &uploadSession, service.Index, file, fsctx.Append)
}

func DeleteAllUploadSession(ctx context.Context, c *gin.Context) serializer.Response {
	fs, err := filesystem.NewFileSystemFromContext(c)
	if err != nil {
		return serializer.Err(serializer.CodePolicyNotAllowed, err.Error(), err)
	}
	defer fs.Recycle()

	files := models.GetUploadPlaceholderFiles(fs.User.ID)
	fileIDs := make([]uint, len(files))
	for i, file := range files {
		fileIDs[i] = file.ID
	}

	if err := fs.Delete(ctx, []uint{}, fileIDs, false); err != nil {
		return serializer.Err(serializer.CodeInternalSetting, "Failed to cleanup upload session", err)
	}
	return serializer.Response{}
}
