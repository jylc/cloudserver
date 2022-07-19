package admin

import (
	"github.com/gin-gonic/gin"
	"github.com/jylc/cloudserver/models"
	"github.com/jylc/cloudserver/pkg/serializer"
	"github.com/jylc/cloudserver/pkg/task"
	"strings"
)

type TaskBatchService struct {
	ID []uint `json:"id" binding:"min=1"`
}

type ImportTaskService struct {
	UID       uint   `json:"uid" binding:"required"`
	PolicyID  uint   `json:"policy_id" binding:"required"`
	Src       string `json:"src" binding:"required,min=1,max=65535"`
	Dst       string `json:"dst" binding:"required,min=1,max=655351"`
	Recursive bool   `json:"recursive"`
}

func (service *ListService) Downloads() serializer.Response {
	var res []models.Download
	total := int64(0)

	tx := models.Db.Model(&models.Download{})
	if service.OrderBy != "" {
		tx = tx.Order(service.OrderBy)
	}

	for k, v := range service.Conditions {
		tx = tx.Where(k+" = ?", v)
	}

	if len(service.Searches) > 0 {
		search := ""
		for k, v := range service.Searches {
			search += k + " like '%" + v + "%' OR "
		}
		search = strings.TrimSuffix(search, " OR ")
		tx = tx.Where(search)
	}

	tx.Count(&total)
	tx.Limit(service.PageSize).Offset((service.Page - 1) * service.PageSize).Find(&res)

	users := make(map[uint]models.User)
	for _, file := range res {
		users[file.UserID] = models.User{}
	}

	userIDs := make([]uint, 0, len(users))
	for k := range users {
		userIDs = append(userIDs, k)
	}

	var userList []models.User
	models.Db.Where("id in (?)", userIDs).Find(&userList)

	for _, v := range userList {
		users[v.ID] = v
	}

	return serializer.Response{Data: map[string]interface{}{
		"total": total,
		"items": res,
		"users": users,
	}}
}

func (service *TaskBatchService) Delete(c *gin.Context) serializer.Response {
	if err := models.Db.Where("id in (?)", service.ID).Delete(&models.Download{}).Error; err != nil {
		return serializer.DBErr("Cannot delete task", err)
	}
	return serializer.Response{}
}
func (service *ListService) Tasks() serializer.Response {
	var res []models.Task
	total := int64(0)

	tx := models.Db.Model(&models.Download{})
	if service.OrderBy != "" {
		tx = tx.Order(service.OrderBy)
	}

	for k, v := range service.Conditions {
		tx = tx.Where(k+" = ?", v)
	}

	if len(service.Searches) > 0 {
		search := ""
		for k, v := range service.Searches {
			search += k + " like '%" + v + "%' OR "
		}
		search = strings.TrimSuffix(search, " OR ")
		tx = tx.Where(search)
	}

	tx.Count(&total)
	tx.Limit(service.PageSize).Offset((service.Page - 1) * service.PageSize).Find(&res)

	users := make(map[uint]models.User)
	for _, file := range res {
		users[file.UserID] = models.User{}
	}

	userIDs := make([]uint, 0, len(users))
	for k := range users {
		userIDs = append(userIDs, k)
	}

	var userList []models.User
	models.Db.Where("id in (?)", userIDs).Find(&userList)

	for _, v := range userList {
		users[v.ID] = v
	}

	return serializer.Response{Data: map[string]interface{}{
		"total": total,
		"items": res,
		"users": users,
	}}
}

func (service *TaskBatchService) DeleteGeneral(c *gin.Context) serializer.Response {
	if err := models.Db.Where("id in (?)", service.ID).Delete(&models.Task{}).Error; err != nil {
		return serializer.DBErr("Cannot delete task", err)
	}
	return serializer.Response{}
}

func (service *ImportTaskService) Create(c *gin.Context, user *models.User) serializer.Response {
	job, err := task.NewImportTask(service.UID, service.PolicyID, service.Src, service.Dst, service.Recursive)
	if err != nil {
		return serializer.Err(serializer.CodeNotSet, "Task creation failed", err)
	}
	task.TaskPool.Submit(job)
	return serializer.Response{}
}
