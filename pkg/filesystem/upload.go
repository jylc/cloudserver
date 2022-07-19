package filesystem

import (
	"context"
	"github.com/gin-gonic/gin"
	"github.com/jylc/cloudserver/models"
	"github.com/jylc/cloudserver/pkg/filesystem/fsctx"
	"github.com/jylc/cloudserver/pkg/request"
	"github.com/sirupsen/logrus"
	"path"
)

const (
	UploadSessionMetaKey     = "upload_session"
	UploadSessionCtx         = "uploadSession"
	UserCtx                  = "user"
	UploadSessionCachePrefix = "callback_"
)

func (fs *FileSystem) Upload(ctx context.Context, file *fsctx.FileStream) (err error) {
	err = fs.Trigger(ctx, "BeforeUpload", file)
	if err != nil {
		request.BlackHole(file)
		return err
	}

	var savePath string
	if file.SavePath == "" {
		if originFile, ok := ctx.Value(fsctx.FileModelCtx).(models.File); ok {
			savePath = originFile.SourceName
		} else {
			savePath = fs.GenerateSavePath(ctx, file)
		}
		file.SavePath = savePath
	}

	if file.Mode&fsctx.Nop != fsctx.Nop {
		go fs.CancelUpload(ctx, savePath, file)

		err = fs.Handler.Put(ctx, file)
		if err != nil {
			fs.Trigger(ctx, "AfterUploadFailed", file)
			return err
		}
	}

	err = fs.Trigger(ctx, "AfterUpload", file)

	if err != nil {
		followUpErr := fs.Trigger(ctx, "AfterValidateFailed", file)
		if followUpErr != nil {
			logrus.Debugf("AfterValidateFiled Hook execution failed, %s", followUpErr)
		}
		return err
	}
	return nil
}

func (fs *FileSystem) GenerateSavePath(ctx context.Context, file fsctx.FileHeader) string {
	fileInfo := file.Info()
	return path.Join(
		fs.Policy.GeneratePath(
			fs.User.Model.ID,
			fileInfo.VirtualPath,
		),
		fs.Policy.GenerateFileName(
			fs.User.Model.ID,
			fileInfo.FileName,
		))
}

func (fs *FileSystem) CancelUpload(ctx context.Context, path string, file fsctx.FileHeader) {
	var reqContext context.Context
	if ginCtx, ok := ctx.Value(fsctx.GinCtx).(*gin.Context); ok {
		reqContext = ginCtx.Request.Context()
	} else if reqCtx, ok := ctx.Value(fsctx.HTTPCtx).(context.Context); ok {
		reqContext = reqCtx
	} else {
		return
	}

	select {
	case <-reqContext.Done():
		select {
		case <-ctx.Done():
		default:
			logrus.Debugf("client cancels uploading")
			if fs.Hooks["AfterUploadCanceled"] == nil {
				return
			}
			err := fs.Trigger(ctx, "AfterUploadCanceled", file)
			if err != nil {
				logrus.Debugf("execute AfterUploadCanceled failed, %s", err)
			}
		}
	}
}
