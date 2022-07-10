package models

import (
	"gorm.io/gorm"
	"time"
)

type Share struct {
	gorm.Model
	Password        string
	IsDir           bool
	UserID          uint
	SourceID        uint
	Views           int
	Downloads       int
	RemainDownloads int
	Expires         *time.Time
	PreviewEnabled  bool
	SourceName      string `gorm:"index:source"`

	User   User   `gorm:"PRELOAD:false,association_autoupdate:false"`
	File   File   `gorm:"PRELOAD:false,association_autoupdate:false"`
	Folder Folder `gorm:"PRELOAD:false,association_autoupdate:false"`
}

func ListShares(uid uint, page, pageSize int, order string, publicOnly bool) ([]Share, int) {
	var (
		shares []Share
		total  int64
	)
	dbChain := Db
	dbChain = dbChain.Where("user_id = ?", uid)
	if publicOnly {
		dbChain = dbChain.Where("password = ?", "")
	}
	dbChain.Model(&Share{}).Count(&total)

	dbChain.Limit(pageSize).Offset((page - 1) * pageSize).Order(order).Find(&shares)
	return shares, int(total)
}

func (share *Share) Source() interface{} {
	if share.IsDir {
		return share.SourceFolder()
	}
	return share.SourceFile()
}

func (share *Share) SourceFolder() *Folder {
	if share.Folder.ID == 0 {
		folders, _ := GetFolderByIDs([]uint{share.SourceID}, share.UserID)
		if len(folders) > 0 {
			share.Folder = folders[0]
		}
	}
	return &share.Folder
}

func (share *Share) SourceFile() *File {
	if share.File.ID == 0 {
		files, _ := GetFilesByIDs([]uint{share.SourceID}, share.UserID)
		if len(files) > 0 {
			share.File = files[0]
		}
	}
	return &share.File
}
