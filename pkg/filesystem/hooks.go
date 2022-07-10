package filesystem

import (
	"context"
	"github.com/jylc/cloudserver/pkg/filesystem/fsctx"
)

type Hook func(ctx context.Context, fs *FileSystem, file fsctx.FileHeader) error

func (fs *FileSystem) Use(name string, hook Hook) {
}
