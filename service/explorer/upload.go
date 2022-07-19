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
	"github.com/jylc/cloudserver/pkg/serializer"
	"strconv"
)

type UploadService struct {
	ID    string `uri:"sessionId" binding:"required"`
	Index int    `uri:"index" form:"index" binding:"min=0"`
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
