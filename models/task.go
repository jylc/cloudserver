package models

import (
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type Task struct {
	gorm.Model
	Status   int
	Type     int
	UserID   uint
	Progress int
	Error    string `gorm:"type:text"`
	Props    string `gorm:"type:text"`
}

func (task *Task) Create() (uint, error) {
	if err := Db.Create(task).Error; err != nil {
		logrus.Warningf("unable to insert task record, %s", err)
		return 0, err
	}
	return task.ID, nil
}

func (task *Task) SetStatus(status int) error {
	return Db.Model(task).Select("status").Updates(map[string]interface{}{"status": status}).Error
}

func (task *Task) SetProgress(progress int) error {
	return Db.Model(task).Select("progress").Updates(map[string]interface{}{"progress": progress}).Error
}

func (task *Task) SetError(err string) error {
	return Db.Model(task).Select("error").Updates(map[string]interface{}{"error": err}).Error
}

func GetTasksByID(id interface{}) (*Task, error) {
	task := &Task{}
	result := Db.Where("id = ?", id).First(task)
	return task, result.Error
}

func GetTasksByStatus(status ...int) []Task {
	var tasks []Task
	Db.Where("status in (?)", status).Find(&tasks)
	return tasks
}

func ListTasks(uid uint, page, pageSize int, order string) ([]Task, int) {
	var (
		tasks []Task
		total int64
	)

	dbChain := Db
	dbChain = dbChain.Where("user_id = ?", uid)

	dbChain.Model(&Task{}).Count(&total)
	dbChain.Limit(pageSize).Offset((page - 1) * pageSize).Order(order).Find(&tasks)

	return tasks, int(total)
}
