package filesystem

import (
	"context"
	"github.com/jylc/cloudserver/models"
)

// HandledExtension 可以生成缩略图的文件扩展名
var HandledExtension = []string{"jpg", "jpeg", "png", "gif"}

func (fs *FileSystem) GenerateThumbnail(ctx context.Context, file *models.File) {
	if !IsInExtensionList(HandledExtension, file.Name) {
		return
	}

}
