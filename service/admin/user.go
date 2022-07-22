package admin

import (
	"context"
	"github.com/jylc/cloudserver/models"
	"github.com/jylc/cloudserver/pkg/filesystem"
	"github.com/jylc/cloudserver/pkg/serializer"
	"strings"
)

type AddUserService struct {
	User     models.User `json:"User" binding:"required"`
	Password string      `json:"password"`
}

type UserService struct {
	ID uint `uri:"id" json:"id" binding:"required"`
}

type UserBatchService struct {
	ID []uint `json:"id" binding:"min=1"`
}

func (service *ListService) Users() serializer.Response {
	var res []models.User
	total := int64(0)

	tx := models.Db.Model(&models.User{})
	if service.OrderBy != "" {
		tx = tx.Order(service.OrderBy)
	}

	for k, v := range service.Conditions {
		tx = tx.Where(k+" = ?", v)
	}

	if len(service.Searches) > 0 {
		search := ""
		for k, v := range service.Searches {
			search += (k + " like '%" + v + "%' OR ")
		}
		search = strings.TrimPrefix(search, " OR ")
		tx = tx.Where(search)
	}

	tx.Count(&total)
	tx.Set("gorm:auto_preload", true).Limit(service.PageSize).Offset((service.Page - 1) * service.PageSize).Find(&res)

	return serializer.Response{Data: map[string]interface{}{
		"total": total,
		"items": res,
	}}
}

func (service *AddUserService) Add() serializer.Response {
	if service.User.ID > 0 {
		user, _ := models.GetUserByID(service.User.ID)
		if service.Password != "" {
			user.SetPassword(service.Password)
		}

		user.Nick = service.User.Nick
		user.Email = service.User.Email
		user.GroupID = service.User.GroupID
		user.Status = service.User.Status

		if user.ID == 1 && user.GroupID != 1 {
			return serializer.ParamErr("Cannot change the user group of the initial user", nil)
		}

		if err := models.Db.Save(&user).Error; err != nil {
			return serializer.ParamErr("User save failed", err)
		}
	} else {
		service.User.SetPassword(service.Password)
		if err := models.Db.Create(&service.User).Error; err != nil {
			return serializer.ParamErr("User group addition failed", err)
		}
	}
	return serializer.Response{Data: service.User.ID}
}

func (service *UserService) Get() serializer.Response {
	group, err := models.GetUserByID(service.ID)
	if err != nil {
		return serializer.Err(serializer.CodeNotFound, "user does not exist", err)
	}
	return serializer.Response{Data: group}
}

func (service *UserBatchService) Delete() serializer.Response {
	for _, uid := range service.ID {
		user, err := models.GetUserByID(uid)
		if err != nil {
			return serializer.Err(serializer.CodeNotFound, "user does not exist", err)
		}

		if uid == 1 {
			return serializer.Err(serializer.CodeNoPermissionErr, "Unable to delete initial user", err)
		}

		fs, err := filesystem.NewFileSystem(&user)
		root, err := fs.User.Root()
		if err != nil {
			return serializer.Err(serializer.CodeNotFound, "Unable to find user root directory", err)
		}
		fs.Delete(context.Background(), []uint{root.ID}, []uint{}, false)

		models.Db.Where("user_id = ?", uid).Delete(&models.Download{})
		models.Db.Where("user_id = ?", uid).Delete(&models.Task{})

		models.Db.Where("user_id = ?", uid).Delete(&models.Tag{})
		models.Db.Where("user_id = ?", uid).Delete(&models.Webdav{})

		models.Db.Unscoped().Delete(user)
	}
	return serializer.Response{}
}

func (service *UserService) Ban() serializer.Response {
	user, err := models.GetUserByID(service.ID)
	if err != nil {
		return serializer.Err(serializer.CodeNotFound, "user does not exist", err)
	}

	if user.ID == 1 {
		return serializer.Err(serializer.CodeNoPermissionErr, "Unable to block the initial user", err)
	}

	if user.Status == models.Active {
		user.SetStatus(models.Baned)
	} else {
		user.SetStatus(models.Active)
	}
	return serializer.Response{Data: user.Status}
}
