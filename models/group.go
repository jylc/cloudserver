package models

import "gorm.io/gorm"

type Group struct {
	gorm.Model
	Name          string
	Policies      string
	MaxStorage    uint64
	ShareEnabled  bool
	WebDAVEnabled bool
	SpeedLimit    int
	Options       string `json:"-" gorm:"size:4294967295"`

	PolicyList        []uint      `gorm:"-"`
	OptionsSerialized GroupOption `gorm:"-"`
}

type GroupOption struct {
	ArchiveDownload bool                   `json:"archive_download,omitempty"` // 打包下载
	ArchiveTask     bool                   `json:"archive_task,omitempty"`     // 在线压缩
	CompressSize    uint64                 `json:"compress_size,omitempty"`    // 可压缩大小
	DecompressSize  uint64                 `json:"decompress_size,omitempty"`
	OneTimeDownload bool                   `json:"one_time_download,omitempty"`
	ShareDownload   bool                   `json:"share_download,omitempty"`
	Aria2           bool                   `json:"aria2,omitempty"`         // 离线下载
	Aria2Options    map[string]interface{} `json:"aria2_options,omitempty"` // 离线下载用户组配置
	SourceBatchSize int                    `json:"source_batch,omitempty"`
	Aria2BatchSize  int                    `json:"aria2_batch,omitempty"`
}

func GetGroupByID(ID interface{}) (Group, error) {
	var group Group
	result := Db.First(&group, ID)
	return group, result.Error
}
