package admin

import (
	"github.com/gin-gonic/gin"
	"github.com/jylc/cloudserver/models"
	"github.com/jylc/cloudserver/pkg/hashid"
	"github.com/jylc/cloudserver/pkg/serializer"
	"strings"
)

type ShareBatchService struct {
	ID []uint `json:"id" binding:"min=1"`
}

func (service *ShareBatchService) Delete(c *gin.Context) serializer.Response {
	if err := models.Db.Where("id in (?)", service.ID).Delete(&models.Share{}).Error; err != nil {
		return serializer.DBErr("Unable to delete share", err)
	}
	return serializer.Response{}
}

func (service *ListService) Shares() serializer.Response {
	var res []models.Share
	total := int64(0)

	tx := models.Db.Model(&models.Share{})
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
	hashIDs := make(map[uint]string, len(res))
	for _, file := range res {
		users[file.UserID] = models.User{}
		hashIDs[file.ID] = hashid.HashID(file.ID, hashid.ShareID)
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
		"ids":   hashIDs,
	}}

}
