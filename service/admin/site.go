package admin

import (
	"encoding/gob"
	"github.com/jylc/cloudserver/models"
	"github.com/jylc/cloudserver/pkg/cache"
	"github.com/jylc/cloudserver/pkg/conf"
	"github.com/jylc/cloudserver/pkg/email"
	"github.com/jylc/cloudserver/pkg/serializer"
	"time"
)

func init() {
	gob.Register(map[string]interface{}{})
	gob.Register(map[string]string{})
}

type NoParamService struct {
}

type BatchSettingChangeService struct {
	Options []SettingChangeService `json:"options"`
}

type SettingChangeService struct {
	Key   string `json:"key" binding:"required"`
	Value string `json:"value"`
}

type BatchSettingGet struct {
	Keys []string `json:"keys"`
}

type MailTestService struct {
	Email string `json:"to" binding:"email"`
}

func (service *MailTestService) Send() serializer.Response {
	if err := email.Send(service.Email, "send test", "this is a test email"); err != nil {
		return serializer.Err(serializer.CodeInternalSetting, "send failed"+err.Error(), nil)
	}
	return serializer.Response{}
}

func (service *NoParamService) Summary() serializer.Response {
	versions := map[string]string{
		"backend": conf.BackendVersion,
		"db":      conf.RequiredDBVersion,
		"commit":  conf.LastCommit,
		"is_pro":  conf.IsPro,
	}

	if res, ok := cache.Get("admin_summary"); ok {
		resMap := res.(map[string]interface{})
		resMap["version"] = versions
		resMap["siteURL"] = models.GetSettingByName("siteURL")
		return serializer.Response{Data: resMap}
	}

	total := 12

	files := make([]int64, total)
	users := make([]int64, total)
	shares := make([]int64, total)
	date := make([]string, total)

	toRound := time.Now()
	timeBase := time.Date(toRound.Year(), toRound.Month(), toRound.Day()+1, 0, 0, 0, 0, toRound.Location())
	for day := range files {
		start := timeBase.Add(-time.Duration(total-day) * time.Hour * 24)
		end := timeBase.Add(-time.Duration(total-day-1) * time.Hour * 24)
		date[day] = start.Format("1月2日")
		models.Db.Model(&models.User{}).Where("create-at BETWEEN ? AND ? ", start, end).Count(&users[day])
		models.Db.Model(&models.File{}).Where("create-at BETWEEN ? AND ? ", start, end).Count(&files[day])
		models.Db.Model(&models.Share{}).Where("create-at BETWEEN ? AND ? ", start, end).Count(&shares[day])
	}

	fileTotal := int64(0)
	userTotal := int64(0)
	publicShareTotal := int64(0)
	secretShareTotal := int64(0)
	models.Db.Model(&models.User{}).Count(&userTotal)
	models.Db.Model(&models.File{}).Count(&fileTotal)
	models.Db.Model(&models.Share{}).Where("password = ?", "").Count(&publicShareTotal)
	models.Db.Model(&models.Share{}).Where("password <> ?", "").Count(&secretShareTotal)

	resp := map[string]interface{}{
		"date":             date,
		"files":            files,
		"users":            users,
		"shares":           shares,
		"version":          versions,
		"siteURL":          models.GetSettingByName("siteURL"),
		"fileTotal":        fileTotal,
		"userTotal":        userTotal,
		"publicShareTotal": publicShareTotal,
		"secretShareTotal": secretShareTotal,
	}
	cache.Set("admin_summary", resp, 86400)
	return serializer.Response{Data: resp}
}

func (service *BatchSettingChangeService) Change() serializer.Response {
	cacheClean := make([]string, 0, len(service.Options))
	tx := models.Db.Begin()
	for _, setting := range service.Options {
		if err := tx.Model(&models.Setting{}).Where("name = ?", setting.Key).Update("value", setting.Value).Error(); err != nil {
			cache.Deletes(cacheClean, "setting_")
			tx.Rollback()
			return serializer.DBErr("setting "+setting.Key+" update failed", err)
		}
		cacheClean = append(cacheClean, setting.Key)
	}
	if err := tx.Commit().Error; err != nil {
		return serializer.DBErr("Setting update failed", err)
	}
	cache.Deletes(cacheClean, "setting_")
	return serializer.Response{}
}

func (service *BatchSettingGet) Get() serializer.Response {
	options := models.GetSettingByNames(service.Keys...)
	return serializer.Response{Data: options}
}
