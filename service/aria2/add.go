package aria2

import (
	"github.com/gin-gonic/gin"
	"github.com/jylc/cloudserver/models"
	"github.com/jylc/cloudserver/pkg/aria2"
	"github.com/jylc/cloudserver/pkg/aria2/common"
	"github.com/jylc/cloudserver/pkg/aria2/monitor"
	"github.com/jylc/cloudserver/pkg/cluster"
	"github.com/jylc/cloudserver/pkg/filesystem"
	"github.com/jylc/cloudserver/pkg/mq"
	"github.com/jylc/cloudserver/pkg/serializer"
)

type BatchAddURLService struct {
	URLs []string `json:"url" binding:"required"`
	Dst  string   `json:"dst" binding:"required,min=1"`
}

func (service *BatchAddURLService) Add(c *gin.Context, taskType int) serializer.Response {
	fs, err := filesystem.NewFileSystemFromContext(c)
	if err != nil {
		return serializer.Err(serializer.CodePolicyNotAllowed, err.Error(), err)
	}
	defer fs.Recycle()

	if !fs.User.Group.OptionsSerialized.Aria2 {
		return serializer.Err(serializer.CodeGroupNotAllowed, "The current user group cannot perform this operation", nil)
	}

	if exist, _ := fs.IsPathExist(service.Dst); !exist {
		return serializer.Err(serializer.CodeBatchAria2Size, "Storage path does not exist", nil)
	}

	limit := fs.User.Group.OptionsSerialized.Aria2BatchSize
	if limit > 0 && len(service.URLs) > limit {
		return serializer.Err(serializer.CodeBatchAria2Size, "Exceed aria2 batch size", nil)
	}

	res := make([]serializer.Response, 0, len(service.URLs))
	for _, target := range service.URLs {
		subService := &AddURLService{
			URL: target,
			Dst: service.Dst,
		}
		addRes := subService.Add(c, fs, taskType)
		res = append(res, addRes)
	}
	return serializer.Response{Data: res}
}

type AddURLService struct {
	URL string `json:"url" binding:"required"`
	Dst string `json:"dst" binding:"required,min=1"`
}

func (service *AddURLService) Add(c *gin.Context, fs *filesystem.FileSystem, taskType int) serializer.Response {
	if fs == nil {
		var err error
		fs, err := filesystem.NewFileSystemFromContext(c)
		if err != nil {
			return serializer.Err(serializer.CodePolicyNotAllowed, err.Error(), err)
		}
		defer fs.Recycle()

		if !fs.User.Group.OptionsSerialized.Aria2 {
			return serializer.Err(serializer.CodeGroupNotAllowed, "The current user group cannot perform this operation", nil)
		}

		if exist, _ := fs.IsPathExist(service.Dst); !exist {
			return serializer.Err(serializer.CodeNotFound, "Storage path does not exist", nil)
		}
	}

	downloads := models.GetDownloadsByStatusAndUser(0, fs.User.ID, common.Downloading, common.Paused, common.Ready)
	limit := fs.User.Group.OptionsSerialized.Aria2BatchSize
	if limit > 0 && len(downloads)+1 > limit {
		return serializer.Err(serializer.CodeBatchAria2Size, "Exceed aria2 batch size", nil)
	}

	task := &models.Download{
		Status: common.Ready,
		Type:   taskType,
		Dst:    service.Dst,
		UserID: fs.User.ID,
		Source: service.URL,
	}

	lb := aria2.GetLoadBalancer()

	err, node := cluster.Default.BalanceNodeByFeature("aria2", lb)

	if err != nil {
		return serializer.Err(serializer.CodeInternalSetting, "Aria2 instance acquisition failed", err)
	}
	gid, err := node.GetAria2Instance().CreateTask(task, fs.User.Group.OptionsSerialized.Aria2Options)
	if err != nil {
		return serializer.Err(serializer.CodeNotSet, "Task creation failed", err)
	}

	task.GID = gid
	task.NodeID = node.ID()
	_, err = task.Create()

	if err != nil {
		return serializer.DBErr("Task creation failed", err)
	}

	monitor.NewMonitor(task, cluster.Default, mq.GlobalMQ)
	return serializer.Response{}
}
