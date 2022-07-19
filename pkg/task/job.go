package task

import (
	"github.com/jylc/cloudserver/models"
)

// 任务类型
const (
	// CompressTaskType 压缩任务
	CompressTaskType = iota
	// DecompressTaskType 解压缩任务
	DecompressTaskType
	// TransferTaskType 中转任务
	TransferTaskType
	// ImportTaskType 导入任务
	ImportTaskType
)

// 任务状态
const (
	// Queued 排队中
	Queued = iota
	// Processing 处理中
	Processing
	// Error 失败
	Error
	// Canceled 取消
	Canceled
	// Complete 完成
	Complete
)

// 任务进度
const (
	// PendingProgress 等待中
	PendingProgress = iota
	// CompressingProgress  压缩中
	CompressingProgress
	// DecompressingProgress  解压缩中
	DecompressingProgress
	// DownloadingProgress  下载中
	DownloadingProgress
	// TransferringProgress  转存中
	TransferringProgress
	// ListingProgress 索引中
	ListingProgress
	// InsertingProgress 插入中
	InsertingProgress
)

type Job interface {
	Type() int
	Creator() uint
	Props() string
	Model() *models.Task
	SetStatus(int)
	Do()
	SetError(*JobError)
	GetError() *JobError
}

type JobError struct {
	Msg   string `json:"msg,omitempty"`
	Error string `json:"error,omitempty"`
}

func Record(job Job) (*models.Task, error) {
	record := models.Task{
		Status:   Queued,
		Type:     job.Type(),
		UserID:   job.Creator(),
		Progress: 0,
		Error:    "",
		Props:    job.Props(),
	}

	_, err := record.Create()
	return &record, err
}
