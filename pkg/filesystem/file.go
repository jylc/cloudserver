package filesystem

import (
	"context"
	"github.com/jylc/cloudserver/pkg/filesystem/fsctx"
	"github.com/jylc/cloudserver/pkg/filesystem/response"
)

func (fs *FileSystem) GetDownloadContent(ctx context.Context, id uint) (response.RSCloser, error) {
}

func (fs *FileSystem) GetContent(ctx context.Context, id uint) (response.RSCloser, error) {
	err := fs.resetFileIDIfNotExist(ctx, id)
	if err != nil {
		return nil, nil
	}
	ctx = context.WithValue(ctx, fsctx.FileModelCtx, fs.FileTarget[0])

	rs, err := fs.Handler.Get(ctx, fs.FileTarget[0].SourceName)
	if err != nil {
		return nil, ErrIO.WithError(ErrIO)
	}
	return rs, nil
}
