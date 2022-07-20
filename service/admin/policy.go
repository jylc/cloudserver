package admin

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/jylc/cloudserver/models"
	"github.com/jylc/cloudserver/pkg/auth"
	"github.com/jylc/cloudserver/pkg/cache"
	"github.com/jylc/cloudserver/pkg/filesystem/driver/cos"
	"github.com/jylc/cloudserver/pkg/filesystem/driver/onedrive"
	"github.com/jylc/cloudserver/pkg/request"
	"github.com/jylc/cloudserver/pkg/serializer"
	"github.com/jylc/cloudserver/pkg/utils"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type PathTestService struct {
	Path string `json:"path" binding:"required"`
}

type SlaveTestService struct {
	Secret string `json:"secret" binding:"required"`
	Server string `json:"server" binding:"required"`
}

type AddPolicyService struct {
	Policy models.Policy `json:"policy" binding:"required"`
}

type PolicyService struct {
	ID     int    `uri:"id" json:"id" binding:"required"`
	Region string `json:"region"`
}

func (service *AddPolicyService) Add() serializer.Response {
	if service.Policy.Type != "local" && service.Policy.Type != "remote" {
		service.Policy.DirNameRule = strings.TrimPrefix(service.Policy.DirNameRule, "/")
	}
	if service.Policy.ID > 0 {
		if err := models.Db.Save(&service.Policy).Error; err != nil {
			return serializer.ParamErr("Storage policy save failed", err)
		}
	} else {
		if err := models.Db.Create(&service.Policy).Error; err != nil {
			return serializer.ParamErr("Storage policy addition failed", err)
		}
	}

	service.Policy.CleanCache()
	return serializer.Response{Data: service.Policy.ID}
}

func (service *ListService) Policies() serializer.Response {
	var res []models.Policy
	total := int64(0)

	tx := models.Db.Model(&models.Policy{})
	if service.OrderBy != "" {
		tx = tx.Order(service.OrderBy)
	}

	for k, v := range service.Conditions {
		tx = tx.Where(k+" = ?", v)
	}

	tx.Count(&total)

	tx.Limit(service.PageSize).Offset((service.Page - 1) * service.PageSize).Find(&res)

	statics := make(map[uint][2]int, len(res))
	for i := 0; i < len(res); i++ {
		total := [2]int{}
		row := models.Db.Model(&models.File{}).Where("policy_id = ?", res[i].ID).Select("count(id),sum(size)").Row()
		row.Scan(&total[0], &total[1])
		statics[res[i].ID] = total
	}

	return serializer.Response{
		Data: map[string]interface{}{
			"total":   total,
			"items":   res,
			"statics": statics,
		},
	}
}

func (service *PathTestService) Test() serializer.Response {
	policy := models.Policy{DirNameRule: service.Path}
	path := policy.GeneratePath(1, "/My File")
	path = filepath.Join(path, "test.txt")
	file, err := utils.CreateNestedFile(utils.RelativePath(path))
	if err != nil {
		return serializer.ParamErr(fmt.Sprintf("Unable to create path %s, %s", path, err.Error()), nil)
	}

	file.Close()
	os.Remove(path)

	return serializer.Response{}
}

func (service *SlaveTestService) Test() serializer.Response {
	slave, err := url.Parse(service.Server)
	if err != nil {
		return serializer.ParamErr("Unable to resolve the slave address, "+err.Error(), nil)
	}

	controller, _ := url.Parse("/api/v2/slave/ping")

	body := map[string]string{
		"callback": models.GetSiteURL().String(),
	}
	bodyByte, _ := json.Marshal(body)
	r := request.NewClient()
	res, err := r.Request(
		"POST",
		slave.ResolveReference(controller).String(),
		bytes.NewReader(bodyByte),
		request.WithTimeout(time.Duration(10)*time.Second),
		request.WithCredential(
			auth.HMACAuth{SecretKey: []byte(service.Secret)},
			int64(models.GetIntSetting("slave_api_timeout", 60)),
		),
	).DecodeResponse()
	if err != nil {
		return serializer.ParamErr("No connection to slave, "+err.Error(), nil)
	}
	if res.Code != 0 {
		return serializer.ParamErr("Successfully received from the slave, but the slave returned: "+res.Msg, nil)
	}

	return serializer.Response{}
}

func (service *PolicyService) AddCORS() serializer.Response {
	policy, err := models.GetPolicyByID(service.ID)
	if err != nil {
		return serializer.Err(serializer.CodeNotFound, "Storage policy does not exist", nil)
	}
	switch policy.Type {
	case "oss":
	case "cos":
	case "c3":
	default:
		return serializer.ParamErr("This policy is not supported", nil)
	}
	return serializer.Response{}
}

func (service *PolicyService) AddSCF() serializer.Response {
	policy, err := models.GetPolicyByID(service.ID)
	if err != nil {
		return serializer.Err(serializer.CodeNotFound, "Storage policy does not exist", nil)
	}
	if err := cos.CreateSCF(&policy, service.Region); err != nil {
		return serializer.Err(serializer.CodeInternalSetting, "云函数创建失败", err)
	}

	return serializer.Response{}
}

func (service *PolicyService) GetOAuth(c *gin.Context) serializer.Response {
	policy, err := models.GetPolicyByID(service.ID)
	if err != nil || policy.Type != "onedrive" {
		return serializer.Err(serializer.CodeNotFound, "Storage policy does not exist", nil)
	}

	client, err := onedrive.NewClient(&policy)
	if err != nil {
		return serializer.Err(serializer.CodeInternalSetting, "Unable to initialize onedrive client", err)
	}
	utils.SetSession(c, map[string]interface{}{
		"onedrive_oauth_policy": policy.ID,
	})

	cache.Deletes([]string{policy.BucketName}, "onedrive_")

	return serializer.Response{Data: client.OAuthURL(context.Background(), []string{
		"offline_access",
		"file.readwrite.all",
	})}
}

func (service *PolicyService) Get() serializer.Response {
	policy, err := models.GetPolicyByID(service.ID)
	if err != nil {
		return serializer.Err(serializer.CodeNotFound, "Storage policy does not exist", nil)
	}
	return serializer.Response{Data: policy}
}

func (service *PolicyService) Delete() serializer.Response {
	if service.ID == 1 {
		return serializer.Err(serializer.CodeNoPermissionErr, "The default storage policy cannot be deleted", nil)
	}

	policy, err := models.GetPolicyByID(service.ID)
	if err != nil {
		return serializer.Err(serializer.CodeNotFound, "Storage policy does not exist", nil)
	}

	total := 0
	row := models.Db.Model(&models.File{}).Where("policy_id = ?", service.ID).Select("count(id)").Row()
	row.Scan(&total)
	if total > 0 {
		return serializer.ParamErr(fmt.Sprintf("There are %d files still using this storage policy. Please delete these files first", total), nil)
	}

	var groups []models.Group
	models.Db.Model(&models.Group{}).Where(
		"policies like ?",
		fmt.Sprintf("%%[%d]%%", service.ID),
	).Find(&groups)

	if len(groups) > 0 {
		return serializer.ParamErr(fmt.Sprintf("There are %d user groups bound to this storage policy, please unbind first", len(groups)), nil)
	}

	models.Db.Delete(&policy)
	policy.CleanCache()
	return serializer.Response{}

}
