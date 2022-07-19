package admin

import (
	"github.com/jylc/cloudserver/models"
	"github.com/jylc/cloudserver/pkg/cluster"
	"github.com/jylc/cloudserver/pkg/serializer"
	"strings"
)

type AddNodeService struct {
	Node models.Node `json:"node" binding:"required"`
}

type ToggleNodeService struct {
	ID      uint              `uri:"id"`
	Desired models.NodeStatus `uir:"desired"`
}

type NodeService struct {
	ID uint `uri:"id" json:"id" binding:"required"`
}

func (service *ListService) Nodes() serializer.Response {
	var res []models.Node
	total := int64(0)

	tx := models.Db.Model(&models.Node{})
	if service.OrderBy != "" {
		tx = tx.Order(service.OrderBy)
	}

	for k, v := range service.Conditions {
		tx = tx.Where(k+" = ?", v)
	}

	if len(service.Searches) > 0 {
		search := ""
		for k, v := range service.Searches {
			search += k + " like '%" + v + "%' OR "
		}
		search = strings.TrimSuffix(search, " OR ")
		tx = tx.Where(search)
	}

	tx.Count(&total)
	tx.Limit(service.PageSize).Offset((service.Page - 1) * service.PageSize).Find(&res)

	isActive := make(map[uint]bool)
	for i := 0; i < len(res); i++ {
		if node := cluster.Default.GetNodeByID(res[i].ID); node != nil {
			isActive[res[i].ID] = node.IsActive()
		}
	}

	return serializer.Response{Data: map[string]interface{}{
		"total":  total,
		"items":  res,
		"active": isActive,
	}}
}

func (service *AddNodeService) Add() serializer.Response {
	if service.Node.ID > 0 {
		if err := models.Db.Save(&service.Node).Error; err != nil {
			return serializer.ParamErr("Node save failed", err)
		}
	} else {
		if err := models.Db.Create(&service.Node).Error; err != nil {
			return serializer.ParamErr("Node addition failed", err)
		}
	}

	if service.Node.Status == models.NodeActive {
		cluster.Default.Add(&service.Node)
	}

	return serializer.Response{Data: service.Node.ID}
}

func (service *ToggleNodeService) Toggle() serializer.Response {
	node, err := models.GetNodeByID(service.ID)
	if err != nil {
		return serializer.DBErr("Node not found", err)
	}
	if node.ID <= 1 {
		return serializer.Err(serializer.CodeNoPermissionErr, "System node cannot be changed", err)
	}
	if err = node.SetStatus(service.Desired); err != nil {
		return serializer.DBErr("Unable to change node state", err)
	}

	if service.Desired == models.NodeActive {
		cluster.Default.Add(&node)
	} else {
		cluster.Default.Delete(node.ID)
	}

	return serializer.Response{}
}

func (service *NodeService) Delete() serializer.Response {
	node, err := models.GetNodeByID(service.ID)
	if err != nil {
		return serializer.DBErr("Node not found", err)
	}
	if node.ID <= 1 {
		return serializer.Err(serializer.CodeNoPermissionErr, "System node cannot be changed", err)
	}

	cluster.Default.Delete(node.ID)
	if err := models.Db.Delete(&node).Error; err != nil {
		return serializer.DBErr("Unable to delete node state", err)
	}
	return serializer.Response{}
}

func (service *NodeService) Get() serializer.Response {
	node, err := models.GetNodeByID(service.ID)
	if err != nil {
		return serializer.DBErr("Node not found", err)
	}
	return serializer.Response{Data: node}
}
