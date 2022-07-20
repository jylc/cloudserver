package slaveinmaster

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"github.com/jylc/cloudserver/models"
	"github.com/jylc/cloudserver/pkg/cluster"
	"github.com/jylc/cloudserver/pkg/filesystem/driver"
	"github.com/jylc/cloudserver/pkg/filesystem/fsctx"
	"github.com/jylc/cloudserver/pkg/filesystem/response"
	"github.com/jylc/cloudserver/pkg/mq"
	"github.com/jylc/cloudserver/pkg/request"
	"github.com/jylc/cloudserver/pkg/serializer"
	"net/url"
	"time"
)

type Driver struct {
	node    cluster.Node
	handler driver.Handler
	policy  *models.Policy
	client  request.Client
}

func (d *Driver) Put(ctx context.Context, file fsctx.FileHeader) error {
	defer file.Close()

	fileInfo := file.Info()
	req := serializer.SlaveTransferReq{
		Src:    fileInfo.Src,
		Dst:    fileInfo.SavePath,
		Policy: d.policy,
	}

	body, err := json.Marshal(req)
	if err != nil {
		return err
	}

	resChan := mq.GlobalMQ.Subscribe(req.Hash(models.GetSettingByName("siteID")), 0)
	defer mq.GlobalMQ.Unsubscribe(req.Hash(models.GetSettingByName("siteID")), resChan)

	res, err := d.client.Request("PUT", "task/transfer", bytes.NewReader(body)).CheckHTTPResponse(200).DecodeResponse()
	if err != nil {
		return err
	}

	if res.Code != 0 {
		return serializer.NewErrorFromResponse(res)
	}

	waitTimeout := models.GetIntSetting("slave_transfer_timeout", 172800)
	select {
	case <-time.After(time.Duration(waitTimeout) * time.Second):
		return ErrWaitResultTimeout
	case msg := <-resChan:
		if msg.Event != serializer.SlaveTransferSuccess {
			return errors.New(msg.Content.(serializer.SlaveTransferResult).Error)
		}
	}

	return nil
}

func (d *Driver) Delete(ctx context.Context, files []string) ([]string, error) {
	return d.handler.Delete(ctx, files)
}

func (d *Driver) Get(ctx context.Context, path string) (response.RSCloser, error) {
	return nil, ErrNotImplemented
}

func (d *Driver) Thumb(ctx context.Context, path string) (*response.ContentResponse, error) {
	return nil, ErrNotImplemented
}

func (d *Driver) Source(ctx context.Context, path string, url url.URL, ttl int64, isDownload bool, speed int) (string, error) {
	return "", ErrNotImplemented
}

func (d *Driver) Token(ctx context.Context, ttl int64, uploadSession *serializer.UploadSession, file fsctx.FileHeader) (*serializer.UploadCredential, error) {
	return nil, ErrNotImplemented
}

func (d *Driver) CancelToken(ctx context.Context, uploadSession *serializer.UploadSession) error {
	return nil
}

func (d *Driver) List(ctx context.Context, path string, recursive bool) ([]response.Object, error) {
	return nil, ErrNotImplemented
}

func NewDriver(node cluster.Node, handler driver.Handler, policy *models.Policy) driver.Handler {
	var endpoint *url.URL
	if serverURL, err := url.Parse(node.DBModel().Server); err == nil {
		var controller *url.URL
		controller, _ = url.Parse("/api/v3/slave/")
		endpoint = serverURL.ResolveReference(controller)
	}

	signTTL := models.GetIntSetting("slave_api_timeout", 60)
	return &Driver{
		node:    node,
		handler: handler,
		policy:  policy,
		client: request.NewClient(
			request.WithMasterMeta(),
			request.WithTimeout(time.Duration(signTTL)*time.Second),
			request.WithCredential(node.SlaveAuthInstance(), int64(signTTL)),
			request.WithEndpoint(endpoint.String()),
		),
	}
}
