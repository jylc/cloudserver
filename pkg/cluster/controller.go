package cluster

import (
	"github.com/jylc/cloudserver/pkg/aria2/common"
	"github.com/jylc/cloudserver/pkg/mq"
	"github.com/jylc/cloudserver/pkg/request"
	"github.com/jylc/cloudserver/pkg/serializer"
	"net/url"
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
}

type MasterInfo struct {
	ID       string
	TTL      int
	URL      *url.URL
	Instance Node
	Client   request.Client

	jobTracker map[string]bool
}
