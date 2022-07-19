package task

import (
	"context"
	"encoding/json"
	"github.com/jylc/cloudserver/models"
	"github.com/jylc/cloudserver/pkg/filesystem"
	"github.com/jylc/cloudserver/pkg/filesystem/fsctx"
	"github.com/sirupsen/logrus"
	"path"
)

type ImportTask struct {
	User      *models.User
	TaskModel *models.Task
	TaskProps ImportProps
	Err       *JobError
}

func (job *ImportTask) Type() int {
	return ImportTaskType
}

func (job *ImportTask) Creator() uint {
	return job.User.ID
}

func (job *ImportTask) Props() string {
	res, _ := json.Marshal(job.TaskProps)
	return string(res)
}

func (job *ImportTask) Model() *models.Task {
	return job.TaskModel
}

func (job *ImportTask) SetStatus(status int) {
	job.TaskModel.SetStatus(status)
}

func (job *ImportTask) Do() {
	ctx := context.Background()

	policy, err := models.GetPolicyByID(job.TaskProps.PolicyID)
	if err != nil {
		job.SetErrorMsg("Storage policy not found", err)
		return
	}

	job.User.Policy = policy
	fs, err := filesystem.NewFileSystem(job.User)
	if err != nil {
		job.SetErrorMsg(err.Error(), nil)
		return
	}
	defer fs.Recycle()

	fs.Policy = &policy
	if err := fs.DispatchHandler(); err != nil {
		job.SetErrorMsg("Unable to distribute storage policy", err)
		return
	}

	fs.Use("BeforeAddFile", filesystem.HookValidateFile)
	fs.Use("BeforeAddFile", filesystem.HookValidateCapacity)

	job.TaskModel.SetProgress(ListingProgress)
	coxIgnoreConflict := context.WithValue(context.Background(), fsctx.IgnoreDirectoryConflictCtx, true)
	objects, err := fs.Handler.List(ctx, job.TaskProps.Src, job.TaskProps.Recursive)
	if err != nil {
		job.SetErrorMsg("Unable to list files", err)
		return
	}

	job.TaskModel.SetProgress(InsertingProgress)

	pathCache := make(map[string]*models.Folder, len(objects))
	for _, object := range objects {
		if object.IsDir {
			virtualPath := path.Join(job.TaskProps.Dst, object.RelativePath)
			folder, err := fs.CreateDirectory(coxIgnoreConflict, virtualPath)
			if err != nil {
				logrus.Warningf("Import task failed to create user directory [%s],%s", virtualPath, err)
			} else if folder.ID > 0 {
				pathCache[virtualPath] = folder
			}
		}
	}

	for _, object := range objects {
		if !object.IsDir {
			virtualPath := path.Dir(path.Join(job.TaskProps.Dst, object.RelativePath))
			fileHeader := fsctx.FileStream{
				Size:        object.Size,
				VirtualPath: virtualPath,
				Name:        object.Name,
				SavePath:    object.Source,
			}

			parentFolder := &models.Folder{}
			if parent, ok := pathCache[virtualPath]; ok {
				parentFolder = parent
			} else {
				folder, err := fs.CreateDirectory(context.Background(), virtualPath)
				if err != nil {
					logrus.Warningf("Import task failed to create user directory [%s],%s", object.RelativePath, err)
					continue
				}
				parentFolder = folder
			}
			_, err := fs.AddFile(context.Background(), parentFolder, &fileHeader)
			if err != nil {
				logrus.Warningf("Import task failed to create insert file [%s],%s", object.RelativePath, err)
				if err == filesystem.ErrInsufficientCapacity {
					job.SetErrorMsg("Insufficient capacity", err)
					return
				}
			}
		}
	}
}

func (job *ImportTask) SetError(err *JobError) {
	job.Err = err
	res, _ := json.Marshal(job.Err)
	job.TaskModel.SetError(string(res))
}

func (job *ImportTask) GetError() *JobError {
	return job.Err
}

type ImportProps struct {
	PolicyID  uint   `json:"policy_id"`
	Src       string `json:"src"`
	Recursive bool   `json:"is_recursive"`
	Dst       string `json:"dst"`
}

func NewImportTask(user, policy uint, src, dst string, recursive bool) (Job, error) {
	creator, err := models.GetActivateUserByID(user)
	if err != nil {
		return nil, err
	}

	newTask := &ImportTask{
		User: &creator,
		TaskProps: ImportProps{
			PolicyID:  policy,
			Recursive: recursive,
			Src:       src,
			Dst:       dst,
		},
	}

	record, err := Record(newTask)
	if err != nil {
		return nil, err
	}
	newTask.TaskModel = record
	return newTask, nil
}

func (job *ImportTask) SetErrorMsg(msg string, err error) {
	jobErr := &JobError{Msg: msg}
	if err != nil {
		jobErr.Error = err.Error()
	}
	job.SetError(jobErr)
}
