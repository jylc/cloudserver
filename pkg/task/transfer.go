package task

import (
	"context"
	"encoding/json"
	"github.com/jylc/cloudserver/models"
	"github.com/jylc/cloudserver/pkg/cluster"
	"github.com/jylc/cloudserver/pkg/filesystem"
	"github.com/jylc/cloudserver/pkg/filesystem/fsctx"
	"github.com/jylc/cloudserver/pkg/utils"
	"github.com/sirupsen/logrus"
	"os"
	"path"
	"path/filepath"
	"strings"
)

type TransferTask struct {
	User      *models.User
	TaskModel *models.Task
	TaskProps TransferProps
	Err       *JobError

	zipPath string
}

func (job *TransferTask) Type() int {
	return TransferTaskType
}

func (job *TransferTask) Creator() uint {
	return job.User.ID
}

func (job *TransferTask) Props() string {
	res, _ := json.Marshal(job.TaskProps)
	return string(res)
}

func (job *TransferTask) Model() *models.Task {
	return job.TaskModel
}

func (job *TransferTask) SetStatus(status int) {
	job.TaskModel.SetStatus(status)
}

func (job *TransferTask) Do() {
	defer job.Recycle()

	fs, err := filesystem.NewFileSystem(job.User)
	if err != nil {
		job.SetErrorMsg(err.Error(), nil)
		return
	}

	for index, file := range job.TaskProps.Src {
		job.TaskModel.SetProgress(index)

		dst := path.Join(job.TaskProps.Dst, filepath.Base(file))
		if job.TaskProps.TrimPath {
			trim := utils.FormSlash(job.TaskProps.Parent)
			src := utils.FormSlash(file)
			dst = path.Join(job.TaskProps.Dst, strings.TrimPrefix(src, trim))
		}

		if job.TaskProps.NodeID > 1 {
			node := cluster.Default.GetNodeByID(job.TaskProps.NodeID)
			if node == nil {
				job.SetErrorMsg("Slave node unavailable", nil)
			}

			fs.SwitchToSlaveHandler(node)
			err = fs.UploadFromStream(context.Background(), &fsctx.FileStream{
				File:        nil,
				Size:        job.TaskProps.SrcSizes[file],
				Name:        path.Base(dst),
				VirtualPath: path.Dir(dst),
				Src:         file,
			}, false)
		} else {
			err = fs.UploadFromPath(context.Background(), file, dst, 0)
		}

		if err != nil {
			job.SetErrorMsg("File transfer failed", err)
		}
	}
}

func (job *TransferTask) SetError(err *JobError) {
	job.Err = err
	res, _ := json.Marshal(job.Err)
	job.TaskModel.SetError(string(res))
}

func (job *TransferTask) GetError() *JobError {
	return job.Err
}

type TransferProps struct {
	Src      []string          `json:"src"`
	SrcSizes map[string]uint64 `json:"src_size"`
	Parent   string            `json:"parent"`
	Dst      string            `json:"dst"`
	TrimPath bool              `json:"trim_path"`
	NodeID   uint              `json:"node_id"`
}

func NewTransferTask(user uint, src []string, dst, parent string, trim bool, node uint, sizes map[string]uint64) (Job, error) {
	creator, err := models.GetActivateUserByID(user)
	if err != nil {
		return nil, err
	}

	newTask := &TransferTask{
		User: &creator,
		TaskProps: TransferProps{
			Src:      src,
			SrcSizes: sizes,
			Parent:   parent,
			Dst:      dst,
			TrimPath: trim,
			NodeID:   node,
		},
	}

	record, err := Record(newTask)
	if err != nil {
		return nil, err
	}

	newTask.TaskModel = record
	return newTask, nil
}

func (job *TransferTask) Recycle() {
	if job.TaskProps.NodeID == 1 {
		err := os.RemoveAll(job.TaskProps.Parent)
		if err != nil {
			logrus.Warningf("Unable to delete transit temporary directory [%s],%s", job.TaskProps.Parent, err)
		}
	}
}

func (job *TransferTask) SetErrorMsg(msg string, err error) {
	jobErr := &JobError{Msg: msg}
	if err != nil {
		jobErr.Error = err.Error()
	}
	job.SetError(jobErr)
}

func NewTransferTaskFromModel(task *models.Task) (Job, error) {
	user, err := models.GetActivateUserByID(task.UserID)
	if err != nil {
		return nil, err
	}

	newTask := &TransferTask{
		User:      &user,
		TaskModel: task,
	}
	err = json.Unmarshal([]byte(task.Props), &newTask.TaskProps)
	if err != nil {
		return nil, err
	}
	return newTask, nil
}
