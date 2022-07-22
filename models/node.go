package models

import (
	"encoding/json"
	"gorm.io/gorm"
)

type Node struct {
	gorm.Model
	Status       NodeStatus
	Name         string
	Type         ModelType
	Server       string
	SlaveKey     string `gorm:"type:text"`
	MasterKey    string `gorm:"type:text"`
	Aria2Enabled bool
	Aria2Options string `gorm:"type:text"`
	Rank         int

	Aria2OptionsSerialized Aria2Option `gorm:"-"`
}

type Aria2Option struct {
	Server   string `json:"server,omitempty"`
	Token    string `json:"token,omitempty"`
	TempPath string `json:"temp_path,omitempty"`
	Options  string `json:"options,omitempty"`
	Interval int    `json:"interval,omitempty"`
	Timeout  int    `json:"timeout,omitempty"`
}

type NodeStatus int
type ModelType int

const (
	NodeActive NodeStatus = iota
	NodeSuspend
)

const (
	SlaveNodeType ModelType = iota
	MasterNodeType
)

func GetNodesByStatus(status ...NodeStatus) ([]Node, error) {
	var nodes []Node
	result := Db.Where("status in (?)", status).Find(&nodes)
	return nodes, result.Error
}

func GetNodeByID(ID interface{}) (Node, error) {
	var node Node
	result := Db.First(&node, ID)
	return node, result.Error
}

func (node *Node) SetStatus(status NodeStatus) error {
	node.Status = status
	return Db.Model(&node).Updates(map[string]interface{}{
		"status": status,
	}).Error
}

func (node *Node) AfterFind() (err error) {
	if node.Aria2Options != "" {
		err = json.Unmarshal([]byte(node.Aria2Options), &node.Aria2OptionsSerialized)
	}
	return err
}
