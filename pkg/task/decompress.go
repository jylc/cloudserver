package task

import (
	"context"
	"encoding/json"
	"github.com/jylc/cloudserver/models"
	"github.com/jylc/cloudserver/pkg/filesystem"
)

type DecompressTask struct {
	User      *models.User
	TaskModel *models.Task
	TaskProps DecompressProps
	Err       *JobError

	zipPath string
}

func (job *DecompressTask) Type() int {
	return DecompressTaskType
}

func (job *DecompressTask) Creator() uint {
	return job.User.ID
}

func (job *DecompressTask) Props() string {
	res, _ := json.Marshal(job.TaskProps)
	return string(res)
}

func (job *DecompressTask) Model() *models.Task {
	return job.TaskModel
}

func (job *DecompressTask) SetStatus(status int) {
	job.TaskModel.SetStatus(status)
}

func (job *DecompressTask) SetErrorMsg(msg string, err error) {
	jobErr := &JobError{Msg: msg}
	if err != nil {
		jobErr.Error = err.Error()
	}
	job.SetError(jobErr)
}

func (job *DecompressTask) Do() {
	fs, err := filesystem.NewFileSystem(job.User)
	if err != nil {
		job.SetErrorMsg("Unable to create file system", err)
		return
	}
	job.TaskModel.SetProgress(DecompressingProgress)

	err = fs.Decompress(context.Background(), job.TaskProps.Src, job.TaskProps.Dst, job.TaskProps.Encoding)
	if err != nil {
		job.SetErrorMsg("Decompression failed", err)
		return
	}
}

func (job *DecompressTask) SetError(jobError *JobError) {
	//TODO implement me
	panic("implement me")
}

func (job *DecompressTask) GetError() *JobError {
	//TODO implement me
	panic("implement me")
}

type DecompressProps struct {
	Src      string `json:"src"`
	Dst      string `json:"dst"`
	Encoding string `json:"encoding"`
}

func NewDecompressTaskFromModel(task *models.Task) (Job, error) {
	user, err := models.GetActivateUserByID(task.UserID)
	if err != nil {
		return nil, err
	}

	newTask := &DecompressTask{
		User:      &user,
		TaskModel: task,
	}

	err = json.Unmarshal([]byte(task.Props), &newTask.TaskProps)
	if err != nil {
		return nil, err
	}
	return newTask, nil
}

func NewDecompressTask(user *models.User, src, dst, encoding string) (Job, error) {
	newTask := &DecompressTask{
		User: user,
		TaskProps: DecompressProps{
			Src:      src,
			Dst:      dst,
			Encoding: encoding,
		},
	}

	record, err := Record(newTask)
	if err != nil {
		return nil, err
	}
	newTask.TaskModel = record

	return newTask, nil
}
