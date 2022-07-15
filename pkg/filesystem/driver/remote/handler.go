package remote

import (
	"context"
	"github.com/jylc/cloudserver/models"
	"github.com/jylc/cloudserver/pkg/auth"
	"github.com/jylc/cloudserver/pkg/filesystem/fsctx"
	"github.com/jylc/cloudserver/pkg/filesystem/response"
	"github.com/jylc/cloudserver/pkg/request"
	"github.com/jylc/cloudserver/pkg/serializer"
	"net/url"
)

type Driver struct {
	Client       request.Client
	Policy       *models.Policy
	AuthInstance auth.Auth
	uploadClient Client
}

func (d Driver) Put(ctx context.Context, file fsctx.FileHeader) error {
	//TODO implement me
	panic("implement me")
}

func (d Driver) Delete(ctx context.Context, files []string) ([]string, error) {
	//TODO implement me
	panic("implement me")
}

func (d Driver) Get(ctx context.Context, path string) (response.RSCloser, error) {
	//TODO implement me
	panic("implement me")
}

func (d Driver) Thumb(ctx context.Context, path string) (*response.ContentResponse, error) {
	//TODO implement me
	panic("implement me")
}

func (d Driver) Source(ctx context.Context, path string, url url.URL, ttl int64, isDownload bool, speed int) (string, error) {
	//TODO implement me
	panic("implement me")
}

func (d Driver) Token(ctx context.Context, ttl int64, uploadSession *serializer.UploadSession, file fsctx.FileHeader) (*serializer.UploadCredential, error) {
	//TODO implement me
	panic("implement me")
}

func (d Driver) CancelToken(ctx context.Context, uploadSession *serializer.UploadSession) error {
	//TODO implement me
	panic("implement me")
}

func (d Driver) List(ctx context.Context, path string, recursive bool) ([]response.Object, error) {
	//TODO implement me
	panic("implement me")
}

func NewDriver(policy *models.Policy) (*Driver, error) {
	client, err := NewClient(policy)
	if err != nil {
		return nil, err
	}
	return &Driver{
		Policy:       policy,
		Client:       request.NewClient(),
		AuthInstance: auth.HMACAuth{SecretKey: []byte(policy.SecretKey)},
		uploadClient: client,
	}, nil
}
