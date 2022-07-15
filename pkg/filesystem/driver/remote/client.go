package remote

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/gofrs/uuid"
	"github.com/jylc/cloudserver/models"
	"github.com/jylc/cloudserver/pkg/auth"
	"github.com/jylc/cloudserver/pkg/filesystem/chunk"
	"github.com/jylc/cloudserver/pkg/filesystem/chunk/backoff"
	"github.com/jylc/cloudserver/pkg/filesystem/fsctx"
	"github.com/jylc/cloudserver/pkg/request"
	"github.com/jylc/cloudserver/pkg/serializer"
	"github.com/sirupsen/logrus"
	"io"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"
)

const (
	basePath        = "/api/v3/slave/"
	OverwriteHeader = auth.CrHeaderPrefix + "Overwrite"
	chunkRetrySleep = time.Duration(5) * time.Second
)

type Client interface {
	CreateUploadSession(ctx context.Context, session *serializer.UploadSession, ttl int64, overwrite bool) error

	GetUploadURL(ttl int64, sessionID string) (string, string, error)

	Upload(ctx context.Context, file fsctx.FileHeader) error

	DeleteUploadSession(ctx context.Context, sessionID string) error
}

func NewClient(policy *models.Policy) (Client, error) {
	authInstance := auth.HMACAuth{SecretKey: []byte(policy.SecretKey)}
	serverURL, err := url.Parse(policy.Server)
	if err != nil {
		return nil, err
	}
	base, err := url.Parse(basePath)
	signTTL := models.GetIntSetting("slave_api_timeout", 60)

	return &remoteClient{
		policy:       policy,
		authInstance: authInstance,
		httpClient: request.NewClient(
			request.WithEndpoint(serverURL.ResolveReference(base).String()),
			request.WithCredential(authInstance, int64(signTTL)),
			request.WithMasterMeta(),
			request.WithSlaveMeta(policy.AccessKey),
		),
	}, nil
}

type remoteClient struct {
	policy       *models.Policy
	authInstance auth.Auth
	httpClient   request.Client
}

func (c *remoteClient) Upload(ctx context.Context, file fsctx.FileHeader) error {
	ttl := models.GetIntSetting("upload_session_timeout", 86400)
	fileInfo := file.Info()
	session := &serializer.UploadSession{
		Key:          uuid.Must(uuid.NewV4()).String(),
		VirtualPath:  fileInfo.VirtualPath,
		Name:         fileInfo.FileName,
		Size:         fileInfo.Size,
		SavePath:     fileInfo.SavePath,
		LastModified: fileInfo.LastModified,
		Policy:       *c.policy,
	}

	overwrite := fileInfo.Mode&fsctx.Overwrite == fsctx.Overwrite
	if err := c.CreateUploadSession(ctx, session, int64(ttl), overwrite); err != nil {
		return fmt.Errorf("failed to crate upload session %w", err)
	}
	chunks := chunk.NewGroup(file, c.policy.OptionsSerialized.ChunkSize, &backoff.ConstantBackoff{
		Max:   models.GetIntSetting("chunk_retries", 5),
		Sleep: chunkRetrySleep,
	}, models.IsTrueVal(models.GetSettingByName("use_temp_chunk_buffer")))

	uploadFunc := func(current *chunk.Group, content io.Reader) error {
		return c.uploadChunk(ctx, session.Key, current.Index(), content, overwrite, current.Length())
	}

	for chunks.Next() {
		if err := chunks.Process(uploadFunc); err != nil {
			if err := c.DeleteUploadSession(ctx, session.Key); err != nil {
				logrus.Warningf("failed to delete upload session: %s\n", err)
			}
			return fmt.Errorf("failed to upload chunk #%d: %w", chunks.Index(), err)
		}
	}
	return nil
}

func (c *remoteClient) DeleteUploadSession(ctx context.Context, sessionID string) error {
	resp, err := c.httpClient.Request(
		"DELETE",
		"upload/"+sessionID,
		nil,
		request.WithContext(ctx),
	).CheckHTTPResponse(200).DecodeResponse()
	if err != nil {
		return nil
	}

	if resp.Code != 0 {
		return serializer.NewErrorFromResponse(resp)
	}
	return nil
}

func (c *remoteClient) CreateUploadSession(ctx context.Context, session *serializer.UploadSession, ttl int64, overwrite bool) error {
	reqBodyEncoded, err := json.Marshal(map[string]interface{}{
		"session":   session,
		"ttl":       ttl,
		"overwrite": overwrite,
	})
	if err != nil {
		return err
	}

	bodyReader := strings.NewReader(string(reqBodyEncoded))
	resp, err := c.httpClient.Request(
		"PUT",
		"upload",
		bodyReader,
		request.WithContext(ctx)).CheckHTTPResponse(200).DecodeResponse()
	if err != nil {
		return err
	}

	if resp.Code != 0 {
		return serializer.NewErrorFromResponse(resp)
	}

	return nil
}

func (c *remoteClient) GetUploadURL(ttl int64, sessionID string) (string, string, error) {
	base, err := url.Parse(c.policy.Server)
	if err != nil {
		return "", "", err
	}

	base.Path = path.Join(base.Path, basePath, "upload", sessionID)
	req, err := http.NewRequest("POST", base.String(), nil)
	if err != nil {
		return "", "", err
	}
	req = auth.SignRequest(c.authInstance, req, ttl)
	return req.URL.String(), req.Header["Authorization"][0], nil
}

func (c *remoteClient) uploadChunk(ctx context.Context, sessionID string, index int, chunk io.Reader, overwrite bool, size int64) error {
	resp, err := c.httpClient.Request(
		"POST",
		fmt.Sprintf("upload/%s?chunk=%d", sessionID, index),
		chunk,
		request.WithContext(ctx),
		request.WithTimeout(time.Duration(0)),
		request.WithContentLength(size),
		request.WithHeader(map[string][]string{
			OverwriteHeader: {
				fmt.Sprintf("%t", overwrite),
			},
		})).CheckHTTPResponse(200).DecodeResponse()
	if err != nil {
		return err
	}

	if resp.Code != 0 {
		return serializer.NewErrorFromResponse(resp)
	}
	return nil
}
