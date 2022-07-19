package models

import (
	"github.com/gofrs/uuid"
	"github.com/jylc/cloudserver/pkg/cache"
	"github.com/jylc/cloudserver/pkg/utils"
	"gorm.io/gorm"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type Policy struct {
	// 表字段
	gorm.Model
	Name               string
	Type               string
	Server             string
	BucketName         string
	IsPrivate          bool
	BaseURL            string
	AccessKey          string `gorm:"type:text"`
	SecretKey          string `gorm:"type:text"`
	MaxSize            uint64
	AutoRename         bool
	DirNameRule        string
	FileNameRule       string
	IsOriginLinkEnable bool
	Options            string `gorm:"type:text"`

	// 数据库忽略字段
	OptionsSerialized PolicyOption `gorm:"-"`
	MasterID          string       `gorm:"-"`
}

type PolicyOption struct {
	// Upyun访问Token
	Token string `json:"token"`
	// 允许的文件扩展名
	FileType []string `json:"file_type"`
	// MimeType
	MimeType string `json:"mimetype"`
	// OdRedirect Onedrive 重定向地址
	OdRedirect string `json:"od_redirect,omitempty"`
	// OdProxy Onedrive 反代地址
	OdProxy string `json:"od_proxy,omitempty"`
	// OdDriver OneDrive 驱动器定位符
	OdDriver string `json:"od_driver,omitempty"`
	// Region 区域代码
	Region string `json:"region,omitempty"`
	// ServerSideEndpoint 服务端请求使用的 Endpoint，为空时使用 Policy.Server 字段
	ServerSideEndpoint string `json:"server_side_endpoint,omitempty"`
	// 分片上传的分片大小
	ChunkSize uint64 `json:"chunk_size,omitempty"`
	// 分片上传时是否需要预留空间
	PlaceholderWithSize bool `json:"placeholder_with_size,omitempty"`
	// 每秒对存储端的 API 请求上限
	TPSLimit float64 `json:"tps_limit,omitempty"`
	// 每秒 API 请求爆发上限
	TPSLimitBurst int `json:"tps_limit_burst,omitempty"`
}

// thumbSuffix 支持缩略图处理的文件扩展名
var thumbSuffix = map[string][]string{
	"local":    {},
	"qiniu":    {".psd", ".jpg", ".jpeg", ".png", ".gif", ".webp", ".tiff", ".bmp"},
	"oss":      {".jpg", ".jpeg", ".png", ".gif", ".webp", ".tiff", ".bmp"},
	"cos":      {".jpg", ".jpeg", ".png", ".gif", ".webp", ".tiff", ".bmp"},
	"upyun":    {".svg", ".jpg", ".jpeg", ".png", ".gif", ".webp", ".tiff", ".bmp"},
	"s3":       {},
	"remote":   {},
	"onedrive": {"*"},
}

func GetPolicyByID(ID interface{}) (Policy, error) {
	cacheKey := "policy_" + strconv.Itoa(int(ID.(uint)))
	if policy, ok := cache.Get(cacheKey); ok {
		return policy.(Policy), nil
	}
	var policy Policy
	result := Db.First(&policy, ID)
	if result.Error == nil {
		_ = cache.Set(cacheKey, policy, -1)
	}
	return policy, result.Error
}

func (policy *Policy) IsThumbExist(name string) bool {
	if list, ok := thumbSuffix[policy.Type]; ok {
		if len(list) == 1 && list[0] == "*" {
			return true
		}
		return utils.ContainsString(list, strings.ToLower(filepath.Ext(name)))
	}
	return false
}

func (policy *Policy) IsThumbGenerateNeeded() bool {
	return policy.Type == "local"
}

// GeneratePath 生成存储文件的路径
func (policy *Policy) GeneratePath(uid uint, origin string) string {
	dirRule := policy.DirNameRule
	replaceTable := map[string]string{
		"{randomkey16}":    utils.RandStringRunes(16),
		"{randomkey8}":     utils.RandStringRunes(8),
		"{timestamp}":      strconv.FormatInt(time.Now().Unix(), 10),
		"{timestamp_nano}": strconv.FormatInt(time.Now().UnixNano(), 10),
		"{uid}":            strconv.Itoa(int(uid)),
		"{datetime}":       time.Now().Format("20060102150405"),
		"{date}":           time.Now().Format("20060102"),
		"{year}":           time.Now().Format("2006"),
		"{month}":          time.Now().Format("01"),
		"{day}":            time.Now().Format("02"),
		"{hour}":           time.Now().Format("15"),
		"{minute}":         time.Now().Format("04"),
		"{second}":         time.Now().Format("05"),
		"{path}":           origin + "/",
	}
	dirRule = utils.Replace(dirRule, replaceTable)
	return path.Clean(dirRule)
}

// GenerateFileName 生成存储文件名
func (policy *Policy) GenerateFileName(uid uint, origin string) string {
	// 未开启自动重命名时，直接返回原始文件名
	if !policy.AutoRename {
		return origin
	}

	fileRule := policy.FileNameRule

	replaceTable := map[string]string{
		"{randomkey16}":    utils.RandStringRunes(16),
		"{randomkey8}":     utils.RandStringRunes(8),
		"{timestamp}":      strconv.FormatInt(time.Now().Unix(), 10),
		"{timestamp_nano}": strconv.FormatInt(time.Now().UnixNano(), 10),
		"{uid}":            strconv.Itoa(int(uid)),
		"{datetime}":       time.Now().Format("20060102150405"),
		"{date}":           time.Now().Format("20060102"),
		"{year}":           time.Now().Format("2006"),
		"{month}":          time.Now().Format("01"),
		"{day}":            time.Now().Format("02"),
		"{hour}":           time.Now().Format("15"),
		"{minute}":         time.Now().Format("04"),
		"{second}":         time.Now().Format("05"),
		"{originname}":     origin,
		"{ext}":            filepath.Ext(origin),
		"{uuid}":           uuid.Must(uuid.NewV4()).String(),
	}

	fileRule = utils.Replace(fileRule, replaceTable)
	return fileRule
}

func (policy *Policy) UpdateAccessKeyAndClearCache(s string) error {
	err := Db.Model(policy).UpdateColumn("access_key", s).Error
	policy.CleanCache()
	return err
}

func (policy *Policy) CleanCache() {
	cache.Deletes([]string{strconv.FormatUint(uint64(policy.ID), 10)}, "policy_")
}

func (policy *Policy) CanStructureBeListed() bool {
	return policy.Type != "local" && policy.Type != "remote"
}

func (policy *Policy) IsDirectlyPreview() bool {
	return policy.Type == "local"
}
