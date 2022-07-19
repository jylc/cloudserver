package cluster

import (
	"github.com/jylc/cloudserver/models"
	"github.com/jylc/cloudserver/pkg/aria2/common"
	"github.com/jylc/cloudserver/pkg/auth"
	"github.com/jylc/cloudserver/pkg/serializer"
)

type Node interface {
	Init(node *models.Node)

	IsFeatureEnabled(feature string) bool

	SubscribeStatusChange(callback func(isActive bool, id uint))

	Ping(req *serializer.NodePingReq) (*serializer.NodePingResp, error)

	IsActive() bool

	GetAria2Instance() common.Aria2

	ID() uint

	Kill()

	IsMaster() bool

	MasterAuthInstance() auth.Auth

	SlaveAuthInstance() auth.Auth

	DBModel() *models.Node
}

func NewNodeFromDBModel(node *models.Node) Node {
	switch node.Type {
	case models.SlaveNodeType:
		slave := &SlaveNode{}
		slave.Init(node)
		return slave
	default:
		master := &MasterNode{}
		master.Init(node)
		return master
	}
}
