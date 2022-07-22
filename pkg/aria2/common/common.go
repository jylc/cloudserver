package common

import (
	"github.com/jylc/cloudserver/models"
	"github.com/jylc/cloudserver/pkg/aria2/rpc"
	"github.com/jylc/cloudserver/pkg/serializer"
)

const (
	// Ready 准备就绪
	Ready = iota
	// Downloading 下载中
	Downloading
	// Paused 暂停中
	Paused
	// Error 出错
	Error
	// Complete 完成
	Complete
	// Canceled 取消/停止
	Canceled
	// Unknown 未知状态
	Unknown
)

const (
	// URLTask 从URL添加的任务
	URLTask = iota
	// TorrentTask 种子任务
	TorrentTask
)

type Aria2 interface {
	Init() error

	CreateTask(task *models.Download, options map[string]interface{}) (string, error)

	Status(task *models.Download) (rpc.StatusInfo, error)

	Cancel(task *models.Download) error

	Select(task *models.Download, files []int) error

	GetConfig() models.Aria2Option

	DeleteTempFile(*models.Download) error
}

var (
	// ErrNotEnabled 功能未开启错误
	ErrNotEnabled = serializer.NewError(serializer.CodeNoPermissionErr, "离线下载功能未开启", nil)
	// ErrUserNotFound 未找到下载任务创建者
	ErrUserNotFound = serializer.NewError(serializer.CodeNotFound, "无法找到任务创建者", nil)
)

type DummyAria2 struct {
}

func (instance *DummyAria2) Init() error {
	return nil
}

func (instance *DummyAria2) CreateTask(task *models.Download, options map[string]interface{}) (string, error) {
	return "", ErrNotEnabled
}

func (instance *DummyAria2) Status(task *models.Download) (rpc.StatusInfo, error) {
	return rpc.StatusInfo{}, ErrNotEnabled
}

func (instance *DummyAria2) Cancel(task *models.Download) error {
	return ErrNotEnabled
}

func (instance *DummyAria2) Select(task *models.Download, files []int) error {
	return ErrNotEnabled
}

func (instance *DummyAria2) GetConfig() models.Aria2Option {
	return models.Aria2Option{}
}

func (instance *DummyAria2) DeleteTempFile(download *models.Download) error {
	return ErrNotEnabled
}
func GetStatus(status string) int {
	switch status {
	case "complete":
		return Complete
	case "active":
		return Downloading
	case "waiting":
		return Ready
	case "paused":
		return Paused
	case "error":
		return Error
	case "removed":
		return Canceled
	default:
		return Unknown
	}
}
