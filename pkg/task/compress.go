package task

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/jylc/cloudserver/models"
	"github.com/jylc/cloudserver/pkg/filesystem"
	"github.com/jylc/cloudserver/pkg/utils"
	"github.com/sirupsen/logrus"
	"os"
	"path/filepath"
	"time"
)

type CompressTask struct {
	User      *models.User
	TaskModel *models.Task
	TaskProps CompressProps
	Err       *JobError

	zipPath string
}

func (job *CompressTask) Type() int {
	return CompressTaskType
}

func (job *CompressTask) Creator() uint {
	return job.User.ID
}

func (job *CompressTask) Props() string {
	res, _ := json.Marshal(job.TaskProps)
	return string(res)
}

func (job *CompressTask) Model() *models.Task {
	return job.TaskModel
}

func (job *CompressTask) SetStatus(status int) {
	job.TaskModel.SetStatus(status)
}

func (job *CompressTask) Do() {
	fs, err := filesystem.NewFileSystem(job.User)
	if err != nil {
		job.SetErrorMsg(err.Error())
		return
	}

	logrus.Debugf("Start compressing files")
	job.TaskModel.SetProgress(CompressingProgress)

	saveFolder := "compress"
	zipFilePath := filepath.Join(
		utils.RelativePath(models.GetSettingByName("temp_path")),
		saveFolder,
		fmt.Sprintf("archive_%d.zip", time.Now().UnixNano()),
	)

	zipFile, err := utils.CreateNestedFile(zipFilePath)
	if err != nil {
		logrus.Warningf("%s", err)
		job.SetErrorMsg(err.Error())
		return
	}

	defer zipFile.Close()

	ctx := context.Background()
	err = fs.Compress(ctx, zipFile, job.TaskProps.Dirs, job.TaskProps.Files, false)
	if err != nil {
		job.SetErrorMsg(err.Error())
		return
	}

	job.zipPath = zipFilePath
	zipFile.Close()
	logrus.Debugf("Save the compressed file to%s and start uploading", zipFilePath)
	job.TaskModel.SetProgress(TransferringProgress)

	err = fs.UploadFromPath(ctx, zipFilePath, job.TaskProps.Dst, 0)
	if err != nil {
		job.SetErrorMsg(err.Error())
		return
	}
	job.removeZipFile()
}

func (job *CompressTask) SetError(err *JobError) {
	job.Err = err
	res, _ := json.Marshal(job.Err)
	job.TaskModel.SetError(string(res))

	job.removeZipFile()
}

func (job *CompressTask) GetError() *JobError {
	return job.Err
}

func (job *CompressTask) SetErrorMsg(msg string) {
	job.SetError(&JobError{Msg: msg})
}

func (job *CompressTask) removeZipFile() {
	if job.zipPath != "" {
		if err := os.Remove(job.zipPath); err != nil {
			logrus.Warningf("Unable to delete temporary compressed file %s,%s", job.zipPath, err)
		}
	}
}

type CompressProps struct {
	Dirs  []uint `json:"dirs"`
	Files []uint `json:"files"`
	Dst   string `json:"dst"`
}

func NewCompressTaskFromModel(task *models.Task) (Job, error) {
	user, err := models.GetActivateUserByID(task.UserID)
	if err != nil {
		return nil, err
	}
	newTask := &CompressTask{
		User:      &user,
		TaskModel: task,
	}
	err = json.Unmarshal([]byte(task.Props), &newTask.TaskProps)
	if err != nil {
		return nil, err
	}
	return newTask, nil
}

func NewCompressTask(user *models.User, dst string, dirs, files []uint) (Job, error) {
	newTask := &CompressTask{
		User: user,
		TaskProps: CompressProps{
			Dirs:  dirs,
			Files: files,
			Dst:   dst,
		},
	}

	record, err := Record(newTask)
	if err != nil {
		return nil, err
	}
	newTask.TaskModel = record

	return newTask, nil
}
