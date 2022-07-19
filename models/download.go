package models

import (
	"encoding/json"
	"github.com/jylc/cloudserver/pkg/aria2/rpc"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type Download struct {
	gorm.Model
	Status         int
	Type           int
	Source         string `gorm:"type:text"`
	TotalSize      uint64
	DownloadedSize uint64
	GID            string `gorm:"size:32,index:gid"`
	Speed          int
	Parent         string `gorm:"type:text"`
	Attrs          string `gorm:"size:4294967295"`
	Error          string `gorm:"type:text"`
	Dst            string `gorm:"type:text"`
	UserID         uint
	TaskID         uint
	NodeID         uint

	User       *User          `gorm:"PRELOAD:false,association_autoupdate:false"`
	StatusInfo rpc.StatusInfo `gorm:"-"`
	Task       *Task          `gorm:"-"`
}

func (download *Download) AfterFind() error {
	var err error
	if download.Attrs != "" {
		err = json.Unmarshal([]byte(download.Attrs), &download.StatusInfo)
	}

	if download.TaskID != 0 {
		download.Task, _ = GetTasksByID(download.TaskID)
	}
	return err
}

func (download *Download) BeforeSave() error {
	var err error
	if download.Attrs != "" {
		err = json.Unmarshal([]byte(download.Attrs), &download.StatusInfo)
	}
	return err
}

func (download *Download) Create() (uint, error) {
	if err := Db.Create(download).Error; err != nil {
		logrus.Warningf("unable to insert offline download record, %s", err)
		return 0, err
	}
	return download.ID, nil
}

func (download *Download) Save() error {
	if err := Db.Save(download).Error; err != nil {
		logrus.Warningf("unable to update offline download record, %s", err)
		return err
	}
	return nil
}

func GetDownloadsByStatusAndUser(page, uid uint, status ...int) []Download {
	var tasks []Download
	dbChain := Db
	if page > 0 {
		dbChain = dbChain.Limit(10).Offset(int((page - 1) * 10)).Order("update_at DESC")
	}

	dbChain.Where("user_id = ? and status in (?)", uid, status).Find(&tasks)
	return tasks
}

func GetDownloadByID(gid string, uid int) (*Download, error) {
	download := &Download{}
	result := Db.Where("user_id = ? and g_id = ?", uid, gid).Find(download)
	return download, result.Error
}

func (download *Download) GetOwner() *User {
	if download.User == nil {
		if user, err := GetUserByID(download.UserID); err != nil {
			return &user
		}
	}
	return download.User
}

func (download *Download) Delete() error {
	return Db.Model(&download).Delete(download).Error
}

func (download *Download) GetNodeID() uint {
	if download.NodeID == 0 {
		return 1
	}
	return download.NodeID
}
