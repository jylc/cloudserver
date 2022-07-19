package admin

import (
	"fmt"
	"github.com/jylc/cloudserver/models"
	"github.com/jylc/cloudserver/pkg/serializer"
)

type AddGroupService struct {
	Group models.Group `json:"group" binding:"required"`
}

type GroupService struct {
	ID uint `json:"id" uri:"id" binding:"required"`
}

func (service *ListService) Groups() serializer.Response {
	var res []models.Group
	total := int64(0)

	tx := models.Db.Model(&models.Group{})
	if service.OrderBy != "" {
		tx = tx.Order(service.OrderBy)
	}

	for k, v := range service.Conditions {
		tx = tx.Where(k+" = ?", v)
	}

	tx.Count(&total)

	tx.Limit(service.PageSize).Offset((service.Page - 1) * service.PageSize).Find(&res)

	statics := make(map[uint]int, len(res))
	for i := 0; i < len(res); i++ {
		total := 0
		row := models.Db.Model(&models.User{}).Where("group_id = ?", res[i].ID).Select("count(id)").Row()
		row.Scan(&total)
		statics[res[i].ID] = total
	}

	policies := make(map[uint]models.Policy)
	for i := 0; i < len(res); i++ {
		for _, p := range res[i].PolicyList {
			if _, ok := policies[p]; !ok {
				policies[p], _ = models.GetPolicyByID(p)
			}
		}
	}
	return serializer.Response{Data: map[string]interface{}{
		"total":    total,
		"item":     res,
		"statics":  statics,
		"policies": policies,
	}}
}

func (service *AddGroupService) Add() serializer.Response {
	if service.Group.ID > 0 {
		if err := models.Db.Save(&service.Group).Error; err != nil {
			return serializer.ParamErr("User group save failed", err)
		}
	} else {
		if err := models.Db.Create(&service.Group).Error; err != nil {
			return serializer.ParamErr("User group addition failed", err)
		}
	}

	return serializer.Response{Data: service.Group.ID}
}

func (service *GroupService) Delete() serializer.Response {
	group, err := models.GetGroupByID(service.ID)
	if err != nil {
		return serializer.Err(serializer.CodeNotFound, "User group does not exist", err)
	}

	if group.ID <= 3 {
		return serializer.Err(serializer.CodeNoPermissionErr, "System user group cannot be deleted", err)
	}

	total := 0
	row := models.Db.Model(&models.User{}).Where("group_id = ?", service.ID).Select("count(id)").Row()
	row.Scan(&total)
	if total > 0 {
		return serializer.ParamErr(fmt.Sprintf("There are %d users who still belong to this user group. Please delete these users or change the user group first", total), nil)
	}

	models.Db.Delete(&group)
	return serializer.Response{}
}

func (service *GroupService) Get() serializer.Response {
	group, err := models.GetGroupByID(service.ID)
	if err != nil {
		return serializer.Err(serializer.CodeNotFound, "User group does not exist", err)
	}
	return serializer.Response{Data: group}
}
