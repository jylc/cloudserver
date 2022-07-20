package monitor

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/jylc/cloudserver/models"
	"github.com/jylc/cloudserver/pkg/aria2/common"
	"github.com/jylc/cloudserver/pkg/aria2/rpc"
	"github.com/jylc/cloudserver/pkg/cluster"
	"github.com/jylc/cloudserver/pkg/filesystem"
	"github.com/jylc/cloudserver/pkg/filesystem/fsctx"
	"github.com/jylc/cloudserver/pkg/mq"
	"github.com/jylc/cloudserver/pkg/task"
	"github.com/sirupsen/logrus"
	"path/filepath"
	"strconv"
	"time"
)

type Monitor struct {
	Task     *models.Download
	Interval time.Duration

	notifier <-chan mq.Message
	node     cluster.Node
	retried  int
}

var MAX_RETRY = 10

func NewMonitor(task *models.Download, pool cluster.Pool, myClient mq.MQ) {
	monitor := &Monitor{
		Task:     task,
		notifier: make(chan mq.Message),
		node:     pool.GetNodeByID(task.GetNodeID()),
	}

	if monitor.node != nil {
		monitor.Interval = time.Duration(monitor.node.GetAria2Instance().GetConfig().Interval) * time.Second
		go monitor.Loop(myClient)

		monitor.notifier = myClient.Subscribe(monitor.Task.GID, 0)
	} else {
		monitor.setErrorStatus(errors.New("node unavailable"))
	}
}

func (monitor *Monitor) Loop(mqClient mq.MQ) {
	defer mqClient.Unsubscribe(monitor.Task.GID, monitor.notifier)

	interval := 50 * time.Millisecond
	for {
		select {
		case <-monitor.notifier:
			if monitor.Update() {
				return
			}
		case <-time.After(interval):
			interval = monitor.Interval
			if monitor.Update() {
				return
			}
		}
	}
}

func (monitor *Monitor) Update() bool {
	status, err := monitor.node.GetAria2Instance().Status(monitor.Task)
	if err != nil {
		monitor.retried++
		logrus.Warningf("Unable to get the status of download task [%s],%s", monitor.Task.GID, err)

		if monitor.retried > MAX_RETRY {
			logrus.Warningf("Unable to get the status of download task [%s], exceeding the maximum retry limit,%s", monitor.Task.GID, err)
			monitor.setErrorStatus(err)
			monitor.RemoveTempFolder()
			return true
		}
		return false
	}
	monitor.retried = 0

	if len(status.FollowedBy) > 0 {
		logrus.Debugf("Offline download [%s] redirect to [%s]", monitor.Task.GID, status.FollowedBy[0])
		monitor.Task.GID = status.FollowedBy[0]
		monitor.Task.Save()
		return false
	}

	if err := monitor.UpdateTaskInfo(status); err != nil {
		logrus.Warningf("Unable to update task information [%s] of download task [%s].", monitor.Task.GID, err)
		monitor.setErrorStatus(err)
		monitor.RemoveTempFolder()
		return true
	}

	logrus.Debugf("Offline download [%s] update status [%s]", status.Gid, status.Status)

	switch status.Status {
	case "complete":
		return monitor.Complete(task.TaskPool)
	case "err":
		return monitor.Error(status)
	case "active", "waiting", "paused":
		return false
	case "removed":
		monitor.Task.Status = common.Canceled
		monitor.Task.Save()
		monitor.RemoveTempFolder()
		return true
	default:
		logrus.Warning("Download task [%s] returned unknown status information [%s],", monitor.Task.GID, status.Status)
		return true
	}
}

func (monitor *Monitor) UpdateTaskInfo(status rpc.StatusInfo) error {
	originSize := monitor.Task.TotalSize

	monitor.Task.GID = status.Gid
	monitor.Task.Status = common.GetStatus(status.Status)

	total, err := strconv.ParseUint(status.TotalLength, 10, 64)
	if err != nil {
		total = 0
	}
	downloaded, err := strconv.ParseUint(status.CompletedLength, 10, 64)
	if err != nil {
		downloaded = 0
	}
	monitor.Task.TotalSize = total
	monitor.Task.DownloadedSize = downloaded
	monitor.Task.GID = status.Gid
	monitor.Task.Parent = status.Dir

	speed, err := strconv.Atoi(status.DownloadSpeed)
	if err != nil {
		speed = 0
	}

	monitor.Task.Speed = speed
	attrs, _ := json.Marshal(status)
	monitor.Task.Attrs = string(attrs)

	if err := monitor.Task.Save(); err != nil {
		return err
	}

	if originSize != monitor.Task.TotalSize {
		if err := monitor.ValidateFile(); err != nil {
			monitor.node.GetAria2Instance().Cancel(monitor.Task)
			return err
		}
	}
	return nil
}

func (monitor *Monitor) ValidateFile() error {
	user := monitor.Task.GetOwner()
	if user == nil {
		return common.ErrUserNotFound
	}

	fs, err := filesystem.NewFileSystem(user)
	if err != nil {
		return err
	}
	defer fs.Recycle()

	file := &fsctx.FileStream{
		Size: monitor.Task.TotalSize,
	}

	if err := filesystem.HookValidateCapacity(context.Background(), fs, file); err != nil {
		return err
	}

	for _, fileInfo := range monitor.Task.StatusInfo.Files {
		if fileInfo.Selected == "true" {
			fileSize, _ := strconv.ParseUint(fileInfo.Length, 10, 64)
			file := &fsctx.FileStream{
				Size: fileSize,
				Name: filepath.Base(fileInfo.Path),
			}
			if err := filesystem.HookValidateFile(context.Background(), fs, file); err != nil {
				return err
			}
		}
	}
	return nil
}

func (monitor *Monitor) Error(status rpc.StatusInfo) bool {
	monitor.setErrorStatus(errors.New(status.ErrorMessage))

	monitor.RemoveTempFolder()
	return true
}

func (monitor *Monitor) RemoveTempFolder() {
	monitor.node.GetAria2Instance().DeleteTempFile(monitor.Task)
}

func (monitor *Monitor) Complete(pool task.Pool) bool {
	file := make([]string, 0, len(monitor.Task.StatusInfo.Files))
	sizes := make(map[string]uint64, len(monitor.Task.StatusInfo.Files))

	for i := 0; i < len(monitor.Task.StatusInfo.Files); i++ {
		fileInfo := monitor.Task.StatusInfo.Files[i]
		if fileInfo.Selected == "true" {
			file = append(file, fileInfo.Path)
			size, _ := strconv.ParseUint(fileInfo.Length, 10, 64)
			sizes[fileInfo.Path] = size
		}
	}

	job, err := task.NewTransferTask(
		monitor.Task.UserID,
		file,
		monitor.Task.Dst,
		monitor.Task.Parent,
		true,
		monitor.node.ID(),
		sizes,
	)
	if err != nil {
		monitor.setErrorStatus(err)
		monitor.RemoveTempFolder()
		return true
	}

	pool.Submit(job)

	monitor.Task.TaskID = job.Model().ID
	monitor.Task.Save()

	return true
}

func (monitor *Monitor) setErrorStatus(err error) {
	monitor.Task.Status = common.Error
	monitor.Task.Error = err.Error()
	monitor.Task.Save()
}
