package driver

import (
	"context"
	"github.com/jylc/cloudserver/pkg/filesystem/fsctx"
	"github.com/jylc/cloudserver/pkg/filesystem/response"
	"github.com/jylc/cloudserver/pkg/serializer"
	"net/url"
)

type Handler interface {
	Put(ctx context.Context, file fsctx.FileHeader) error

	Delete(ctx context.Context, files []string) ([]string, error)

	Get(ctx context.Context, path string) (response.RSCloser, error)

	Thumb(ctx context.Context, path string) (*response.ContentResponse, error)

	Source(ctx context.Context, path string, url url.URL, ttl int64, isDownload bool, speed int) (string, error)

	Token(ctx context.Context, ttl int64, uploadSession *serializer.UploadSession, file fsctx.FileHeader) (*serializer.UploadCredential, error)

	CancelToken(ctx context.Context, uploadSession *serializer.UploadSession) error

	List(ctx context.Context, path string, recursive bool) ([]response.Object, error)
}
