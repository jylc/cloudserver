package callback

import (
	"context"
	"github.com/gin-gonic/gin"
	"github.com/jylc/cloudserver/models"
	"github.com/jylc/cloudserver/pkg/filesystem"
	"github.com/jylc/cloudserver/pkg/filesystem/fsctx"
	"github.com/jylc/cloudserver/pkg/serializer"
)

type RemoteUploadCallbackService struct {
	Data serializer.UploadCallback `json:"data" binding:"required"`
}

func (service RemoteUploadCallbackService) GetBody() serializer.UploadCallback {
	return service.Data
}

type CallbackProcessService interface {
	GetBody() serializer.UploadCallback
}

func ProcessCallback(service CallbackProcessService, c *gin.Context) serializer.Response {
	callbackBody := service.GetBody()
	fs, err := filesystem.NewFileSystemFromCallback(c)
	if err != nil {
		return serializer.Err(serializer.CodePolicyNotAllowed, err.Error(), err)
	}

	defer fs.Recycle()

	uploadSession := c.MustGet(filesystem.UploadSessionCtx).(*serializer.UploadSession)

	file, err := models.GetFilesByUploadSession(uploadSession.Key, fs.User.ID)
	if err != nil {
		return serializer.Err(serializer.CodeUploadSessionExpired, "LocalUpload session file placeholder not exist", err)
	}

	fileData := fsctx.FileStream{
		Size:         uploadSession.Size,
		Name:         uploadSession.Name,
		VirtualPath:  uploadSession.VirtualPath,
		SavePath:     uploadSession.SavePath,
		Mode:         fsctx.Nop,
		Model:        file,
		LastModified: uploadSession.LastModified,
	}

	if !fs.Policy.IsUploadPlaceholderWithSize() {
		fs.Use("AfterUpload", filesystem.HookValidateCapacity)
		fs.Use("AfterUpload", filesystem.HookChunkUploaded)
	}
	fs.Use("AfterUpload", filesystem.HookPopPlaceholderToFile(callbackBody.PicInfo))
	fs.Use("AfterValidateFailed", filesystem.HookDeleteTempFile)

	err = fs.Upload(context.Background(), &fileData)
	if err != nil {
		return serializer.Err(serializer.CodeUploadFailed, err.Error(), err)
	}
	return serializer.Response{}
}
