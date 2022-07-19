package cluster

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/jylc/cloudserver/models"
	"github.com/jylc/cloudserver/pkg/aria2/common"
	"github.com/jylc/cloudserver/pkg/auth"
	"github.com/jylc/cloudserver/pkg/conf"
	"github.com/jylc/cloudserver/pkg/request"
	"github.com/jylc/cloudserver/pkg/serializer"
	"github.com/sirupsen/logrus"
	"net/url"
	"strings"
	"sync"
	"time"
)

type SlaveNode struct {
	Model  *models.Node
	Active bool

	caller   slaveCaller
	callback func(bool, uint)
	close    chan bool
	lock     sync.RWMutex
}

func (node *SlaveNode) IsFeatureEnabled(feature string) bool {
	node.lock.RLock()
	defer node.lock.RUnlock()
	switch feature {
	case "aria2":
		return node.Model.Aria2Enabled
	default:
		return false
	}
}

func (node *SlaveNode) SubscribeStatusChange(callback func(isActive bool, id uint)) {
	node.lock.Lock()
	node.callback = callback
	node.lock.Unlock()
}

func (node *SlaveNode) IsActive() bool {
	node.lock.RLock()
	defer node.lock.RUnlock()
	return node.Active
}

func (node *SlaveNode) GetAria2Instance() common.Aria2 {
	//TODO implement me
	panic("implement me")
}

func (node *SlaveNode) ID() uint {
	node.lock.RLock()
	defer node.lock.RUnlock()
	return node.Model.ID
}

func (node *SlaveNode) Kill() {
	node.lock.RLock()
	defer node.lock.RUnlock()
	if node.close != nil {
		close(node.close)
	}
}

func (node *SlaveNode) IsMaster() bool {
	return false
}

func (node *SlaveNode) MasterAuthInstance() auth.Auth {
	node.lock.RLock()
	defer node.lock.RUnlock()
	return auth.HMACAuth{SecretKey: []byte(node.Model.MasterKey)}
}

func (node *SlaveNode) SlaveAuthInstance() auth.Auth {
	node.lock.RLock()
	defer node.lock.RUnlock()
	return auth.HMACAuth{
		SecretKey: []byte(node.Model.SlaveKey),
	}
}

func (node *SlaveNode) DBModel() *models.Node {
	node.lock.RLock()
	defer node.lock.RUnlock()
	return node.Model
}

type slaveCaller struct {
	parent *SlaveNode
	Client request.Client
}

func (node *SlaveNode) Init(nodeModel *models.Node) {
	node.lock.Lock()
	node.Model = nodeModel

	var endpoint *url.URL
	if serverURL, err := url.Parse(node.Model.Server); err == nil {
		var controller *url.URL
		controller, _ = url.Parse("/api/v3/slave/")
		endpoint = serverURL.ResolveReference(controller)
	}

	signTTL := models.GetIntSetting("slave_api_timeout", 60)
	node.caller.Client = request.NewClient(
		request.WithMasterMeta(),
		request.WithTimeout(time.Duration(signTTL)*time.Second),
		request.WithCredential(auth.HMACAuth{SecretKey: []byte(nodeModel.SlaveKey)}, int64(signTTL)),
		request.WithEndpoint(endpoint.String()))

	node.caller.parent = node
	if node.close != nil {
		node.lock.Unlock()
		node.close <- true
		go node.StartPingLoop()
	} else {
		node.Active = true
		node.lock.Unlock()
		go node.StartPingLoop()
	}
}

func (node *SlaveNode) StartPingLoop() {
	node.lock.Lock()
	node.close = make(chan bool)
	node.lock.Unlock()

	tickDuration := time.Duration(models.GetIntSetting("slave_ping_interval", 300)) * time.Second
	recoverDuration := time.Duration(models.GetIntSetting("slave_recover_interval", 600)) * time.Second
	pingTicker := time.Duration(0)

	logrus.Debugf("slave [%s] start heart breaking", node.Model.Name)
	retry := 0
	recoverMode := false
	isFirstLoop := true

loop:
	for {
		select {
		case <-time.After(pingTicker):
			if pingTicker == 0 {
				pingTicker = tickDuration
			}

			logrus.Debugf("slave [%s] send Ping", node.Model.Name)
			res, err := node.Ping(node.getHeartbeatContent(isFirstLoop))
			if err != nil {
				logrus.Debugf("Ping slave [%s] error: %s", node.Model.Name, err)
				retry++
				if retry >= models.GetIntSetting("slave_node_retry", 3) {
					logrus.Debugf("slave [%s] Ping retry has reached the maximum limit, marking the slave node as unavailable", node.Model.Name)
					node.changeStatus(false)

					if !recoverMode {
						logrus.Debugf("slave [%s] enter recovery mode", node.Model.Name)
						pingTicker = recoverDuration
						recoverMode = true
					}
				}
			} else {
				if recoverMode {
					logrus.Debugf("slave [%s] recovered", node.Model.Name)
					pingTicker = tickDuration
					recoverMode = false
					isFirstLoop = true
				}

				logrus.Debugf("slave [%s] status: %s", node.Model.Name, res)
				node.changeStatus(true)
				retry = 0
			}
		case <-node.close:
			logrus.Debugf("slave [%s] accept close signal", node.Model.Name)
			break loop
		}
	}
}

func (node *SlaveNode) changeStatus(isActive bool) {
	node.lock.RLock()
	id := node.Model.ID
	if isActive != node.Active {
		node.lock.RUnlock()
		node.lock.Lock()
		node.Active = isActive
		node.lock.Unlock()
		node.callback(isActive, id)
	} else {
		node.lock.RUnlock()
	}
}

func (node *SlaveNode) Ping(req *serializer.NodePingReq) (*serializer.NodePingResp, error) {
	node.lock.RLock()
	defer node.lock.RUnlock()

	reqBodyEncoded, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	bodyReader := strings.NewReader(string(reqBodyEncoded))

	resp, err := node.caller.Client.Request(
		"POST",
		"heartbeat",
		bodyReader).CheckHTTPResponse(200).DecodeResponse()
	if err != nil {
		return nil, err
	}

	if resp.Code != 0 {
		return nil, serializer.NewErrorFromResponse(resp)
	}

	var res serializer.NodePingResp

	if resStr, ok := resp.Data.(string); ok {
		err = json.Unmarshal([]byte(resStr), &res)
		if err != nil {
			return nil, err
		}
	}
	return &res, nil
}

func (node *SlaveNode) getHeartbeatContent(isUpdate bool) *serializer.NodePingReq {
	return &serializer.NodePingReq{
		SiteURL:       models.GetSiteURL().String(),
		IsUpdate:      isUpdate,
		SiteID:        models.GetSettingByName("siteID"),
		Node:          node.Model,
		CredentialTTL: models.GetIntSetting("slave_api_timeout", 60),
	}
}

func RemoteCallback(url string, body serializer.UploadCallback) error {
	callbackBody, err := json.Marshal(struct {
		Data serializer.UploadCallback `json:"data"`
	}{
		Data: body,
	})

	if err != nil {
		return serializer.NewError(serializer.CodeCallbackError, "cannot encode callback body", err)
	}

	resp := request.GeneralClient.Request(
		"POST",
		url,
		bytes.NewReader(callbackBody),
		request.WithTimeout(time.Duration(conf.Slavec.CallbackTimeout)*time.Second),
		request.WithCredential(auth.General, int64(conf.Slavec.SignatureTTL)))
	if resp.Err != nil {
		return serializer.NewError(serializer.CodeCallbackError, "the slave cannot initiate a callback request", resp.Err)
	}

	response, err := resp.DecodeResponse()
	if err != nil {
		msg := fmt.Sprintf("the slave cannot parse the response returned by the host (StatusCode=%d)", resp.Response.StatusCode)
		return serializer.NewError(serializer.CodeCallbackError, msg, err)
	}

	if response.Code != 0 {
		return serializer.NewError(response.Code, response.Msg, errors.New(response.Error))
	}
	return nil
}
