package admin

import (
	"github.com/jylc/cloudserver/models"
	"github.com/jylc/cloudserver/pkg/serializer"
)

type ListService struct {
	Page       int               `json:"page" binding:"min=1,required"`
	PageSize   int               `json:"page_size" binding:"min=1,required"`
	OrderBy    string            `json:"order_by"`
	Conditions map[string]string `form:"conditions"`
	Searches   map[string]string `form:"searches"`
}

func (service *NoParamService) GroupList() serializer.Response {
	var res []models.Group
	models.Db.Model(&models.Group{}).Find(&res)
	return serializer.Response{Data: res}
}
