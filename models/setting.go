package models

import (
	"gorm.io/gorm"
	"net/url"
	"strconv"
)

type Setting struct {
	gorm.Model
	Type  string `gorm:"not null"`
	Name  string `gorm:"unique;not null;index:setting_key"`
	Value string `gorm:"size:‎65535"`
}

func GetSettingByNames(names ...string) map[string]string {
	var queryRes []Setting
	ans := make(map[string]string, 0)
	Db.Where("name IN (?)", names).Find(&queryRes)
	for _, setting := range queryRes {
		ans[setting.Name] = setting.Value
	}
	return ans
}

func GetSettingByType(types []string) map[string]string {
	var queryRes []Setting
	ans := make(map[string]string, 0)
	Db.Where("type IN (?)", types).Find(&queryRes)
	for _, setting := range queryRes {
		ans[setting.Name] = setting.Value
	}
	return ans
}

func IsTrueVal(val string) bool {
	return val == "1" || val == "true"
}

func GetIntSetting(key string, defaultVal int) int {
	res, err := strconv.Atoi(GetSettingByName(key))
	if err != nil {
		return defaultVal
	}
	return res
}

func GetSettingByName(name string) string {
	return GetSettingByNameFromTx(Db, name)
}

func GetSettingByNameFromTx(tx *gorm.DB, name string) string {
	var setting Setting
	if tx == nil {
		tx = Db
		if tx == nil {
			return ""
		}
	}
	result := tx.Where("name = ?", name).First(&setting)
	if result.Error == nil {
		return setting.Value
	}
	return ""
}

func GetSiteURL() *url.URL {
	base, err := url.Parse(GetSettingByName("siteURL"))
	if err != nil {
		base, _ = url.Parse("http://localhost")
	}
	return base
}

func GetSettingByNameWithDefault(name, fallback string) string {
	res := GetSettingByName(name)
	if res == "" {
		return fallback
	}
	return res
}
