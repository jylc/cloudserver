package filesystem

import (
	"context"
	"github.com/jylc/cloudserver/models"
	"github.com/jylc/cloudserver/pkg/conf"
	"github.com/jylc/cloudserver/pkg/filesystem/fsctx"
	"github.com/jylc/cloudserver/pkg/filesystem/response"
)

// HandledExtension 可以生成缩略图的文件扩展名
var HandledExtension = []string{"jpg", "jpeg", "png", "gif"}

func (fs *FileSystem) GenerateThumbnail(ctx context.Context, file *models.File) {
	if !IsInExtensionList(HandledExtension, file.Name) {
		return
	}
}

func (fs *FileSystem) GetThumb(ctx context.Context, id uint) (*response.ContentResponse, error) {
	err := fs.resetFileIDIfNotExist(ctx, id)
	if err != nil || fs.FileTarget[0].PicInfo == "" {
		return &response.ContentResponse{
			Redirect: false,
		}, ErrObjectNotExist
	}
	w, h := fs.GenerateThumbnailSize(0, 0)
	ctx = context.WithValue(ctx, fsctx.ThumbSizeCtx, [2]uint{w, h})
	ctx = context.WithValue(ctx, fsctx.FileModelCtx, fs.FileTarget[0])
	res, err := fs.Handler.Thumb(ctx, fs.FileTarget[0].SourceName)

	if err != nil && fs.Policy.Type == "local" {
		fs.GenerateThumbnail(ctx, &fs.FileTarget[0])
		res, err = fs.Handler.Thumb(ctx, fs.FileTarget[0].SourceName)
	}

	if err == nil && conf.Sc.Role == "master" {
		res.MaxAge = models.GetIntSetting("preview_timeout", 60)
	}
	return res, nil
}

func (fs *FileSystem) GenerateThumbnailSize(w, h int) (uint, uint) {
	return uint(models.GetIntSetting("thumb_width", 400)), uint(models.GetIntSetting("thumb_width", 300))
}
