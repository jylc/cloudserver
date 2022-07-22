package cluster

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"github.com/jylc/cloudserver/models"
	"github.com/jylc/cloudserver/pkg/aria2/common"
	"github.com/jylc/cloudserver/pkg/aria2/rpc"
	"github.com/jylc/cloudserver/pkg/auth"
	"github.com/jylc/cloudserver/pkg/mq"
	"github.com/jylc/cloudserver/pkg/request"
	"github.com/jylc/cloudserver/pkg/serializer"
	"gorm.io/gorm"
	"net/url"
	"sync"
)

var DefaultController Controller

type Controller interface {
	HandleHeartBeat(req *serializer.NodePingReq) (serializer.NodePingResp, error)

	GetAria2Instance(string) (common.Aria2, error)

	SendNotification(string, string, mq.Message) error

	SubmitTask(string, interface{}, string, func(interface{})) error

	GetMasterInfo(string) (*MasterInfo, error)

	GetOneDriveToken(string, uint) (string, error)
}

type slaveController struct {
	masters map[string]MasterInfo
	lock    sync.RWMutex
}

func (c *slaveController) HandleHeartBeat(req *serializer.NodePingReq) (serializer.NodePingResp, error) {
	c.lock.Lock()
	defer c.lock.Unlock()

	req.Node.AfterFind()

	origin, ok := c.masters[req.SiteID]
	if (ok && req.IsUpdate) || !ok {
		if ok {
			origin.Instance.Kill()
		}

		masterUrl, err := url.Parse(req.SiteID)
		if err != nil {
			return serializer.NodePingResp{}, err
		}

		c.masters[req.SiteID] = MasterInfo{
			ID:  req.SiteID,
			URL: masterUrl,
			TTL: req.CredentialTTL,
			Client: request.NewClient(
				request.WithEndpoint(masterUrl.String()),
				request.WithSlaveMeta(fmt.Sprintf("%d", req.Node.ID)),
				request.WithCredential(auth.HMACAuth{
					SecretKey: []byte(req.Node.MasterKey),
				}, int64(req.CredentialTTL)),
			),
			jobTracker: make(map[string]bool),
			Instance: NewNodeFromDBModel(&models.Node{
				Model:                  gorm.Model{ID: req.Node.ID},
				MasterKey:              req.Node.MasterKey,
				Type:                   models.MasterNodeType,
				Aria2Enabled:           req.Node.Aria2Enabled,
				Aria2OptionsSerialized: req.Node.Aria2OptionsSerialized,
			}),
		}
	}
	return serializer.NodePingResp{}, nil
}

func (c *slaveController) GetAria2Instance(id string) (common.Aria2, error) {
	c.lock.Lock()
	defer c.lock.Unlock()

	if node, ok := c.masters[id]; ok {
		return node.Instance.GetAria2Instance(), nil
	}
	return nil, ErrMasterNotFound
}

func (c *slaveController) SendNotification(id string, subject string, msg mq.Message) error {
	c.lock.Lock()

	if node, ok := c.masters[id]; ok {
		c.lock.RUnlock()

		body := bytes.Buffer{}
		enc := gob.NewEncoder(&body)
		if err := enc.Encode(&msg); err != nil {
			return err
		}

		res, err := node.Client.Request(
			"PUT",
			fmt.Sprintf("/api/v3/slave/notification/%s", subject),
			&body,
		).CheckHTTPResponse(200).DecodeResponse()
		if err != nil {
			return err
		}

		if res.Code != 0 {
			return serializer.NewErrorFromResponse(res)
		}

		return nil
	}

	c.lock.RUnlock()
	return ErrMasterNotFound
}

func (c *slaveController) SubmitTask(id string, job interface{}, hash string, submitter func(interface{})) error {
	c.lock.Lock()
	defer c.lock.Unlock()

	if node, ok := c.masters[id]; ok {
		if _, ok := node.jobTracker[hash]; ok {
			return nil
		}

		node.jobTracker[hash] = true
		submitter(job)
		return nil
	}
	return ErrMasterNotFound
}

func (c *slaveController) GetMasterInfo(id string) (*MasterInfo, error) {
	c.lock.RLock()
	defer c.lock.RUnlock()

	if node, ok := c.masters[id]; ok {
		return &node, nil
	}
	return nil, ErrMasterNotFound
}

func (c *slaveController) GetOneDriveToken(id string, policyID uint) (string, error) {
	c.lock.RLock()

	if node, ok := c.masters[id]; ok {
		c.lock.RUnlock()

		res, err := node.Client.Request(
			"GET",
			fmt.Sprintf("/api/v3/slave/credential/onedrive/%d", policyID),
			nil,
		).CheckHTTPResponse(200).DecodeResponse()
		if err != nil {
			return "", err
		}

		if res.Code != 0 {
			return "", serializer.NewErrorFromResponse(res)
		}

		return res.Data.(string), nil
	}

	c.lock.RUnlock()
	return "", ErrMasterNotFound
}

type MasterInfo struct {
	ID       string
	TTL      int
	URL      *url.URL
	Instance Node
	Client   request.Client

	jobTracker map[string]bool
}

func InitController() {
	DefaultController = &slaveController{
		masters: make(map[string]MasterInfo),
	}
	gob.Register(rpc.StatusInfo{})
}
