package models

import (
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/jylc/cloudserver/pkg/cache"
	"github.com/jylc/cloudserver/pkg/hashid"
	"github.com/jylc/cloudserver/pkg/utils"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
	"strings"
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

func DeleteShareBySourceIDs(sources []uint, isDir bool) error {
	return Db.Where("source_id in (?) and is_dir = ?", sources, isDir).Delete(&Share{}).Error
}

func (share *Share) Viewed() {
	share.Views++
	Db.Model(share).UpdateColumn("views", gorm.Expr("views + ?", 1))
}

func (share *Share) Creator() *User {
	if share.User.ID == 0 {
		share.User, _ = GetUserByID(share.UserID)
	}
	return &share.User
}
func SearchShares(page, pageSize int, order, keywords string) ([]Share, int) {
	var (
		shares []Share
		total  int64
	)

	keywordList := strings.Split(keywords, " ")
	availableList := make([]string, 0, len(keywordList))
	for i := 0; i < len(keywordList); i++ {
		if len(keywordList[i]) > 0 {
			availableList = append(availableList, keywordList[i])
		}
	}

	if len(availableList) == 0 {
		return shares, 0
	}
	dbChain := Db
	dbChain = dbChain.Where("password = ? and remain_downloads <> 0 and (expires is NULL or expires > ?) and source_name like ?", "", time.Now(), "%"+strings.Join(availableList, "%")+"%")

	dbChain.Model(&Share{}).Count(&total)
	dbChain.Limit(pageSize).Offset((page - 1) * pageSize).Order(order).Find(&shares)
	return shares, int(total)
}

func GetShareByHashID(hashID string) *Share {
	id, err := hashid.DecodeHashID(hashID, hashid.ShareID)
	if err != nil {
		return nil
	}
	var share Share
	result := Db.First(&share, id)
	if result.Error != nil {
		return nil
	}
	return &share
}

func (share *Share) Delete() error {
	return Db.Model(share).Delete(share).Error
}

func (share *Share) Update(props map[string]interface{}) error {
	return Db.Model(share).Updates(props).Error
}

func (share *Share) Create() (uint, error) {
	if err := Db.Create(share).Error; err != nil {
		logrus.Warningf("Unable to insert database record,%s", err)
		return 0, err
	}
	return share.ID, nil
}

func (share *Share) IsAvailable() bool {
	if share.RemainDownloads == 0 {
		return false
	}
	if share.Expires != nil && time.Now().After(*share.Expires) {
		return false
	}

	if share.Creator().Status != Active {
		return false
	}

	var sourceID uint
	if share.IsDir {
		folder := share.SourceFolder()
		sourceID = folder.ID
	} else {
		file := share.SourceFile()
		sourceID = file.ID
	}
	if sourceID == 0 {
		// TODO 是否要在这里删除这个无效分享？
		return false
	}

	return true
}

func (share *Share) CanBeDownloadBy(user *User) error {
	if !user.Group.OptionsSerialized.ShareDownload {
		if user.IsAnonymous() {
			return errors.New("未登录用户无法下载")
		}
		return errors.New("您当前的用户组无权下载")
	}
	return nil
}

func (share *Share) DownloadBy(user *User, c *gin.Context) error {
	if !share.WasDownloadedBy(user, c) {
		share.Downloaded()
		if !user.IsAnonymous() {
			cache.Set(fmt.Sprintf("share_%d_%d", share.ID, user.ID), true,
				GetIntSetting("share_download_session_timeout", 2073600))
		} else {
			utils.SetSession(c, map[string]interface{}{fmt.Sprintf("share_%d_%d", share.ID, user.ID): true})
		}
	}
	return nil
}

func (share *Share) WasDownloadedBy(user *User, c *gin.Context) (exist bool) {
	if user.IsAnonymous() {
		exist = utils.GetSession(c, fmt.Sprintf("share_%d_%d", share.ID, user.ID)) != nil
	} else {
		_, exist = cache.Get(fmt.Sprintf("share_%d_%d", share.ID, user.ID))
	}

	return exist
}

func (share *Share) Downloaded() {
	share.Downloads++
	if share.RemainDownloads > 0 {
		share.RemainDownloads--
	}
	Db.Model(share).Updates(map[string]interface{}{
		"downloads":        share.Downloads,
		"remain_downloads": share.RemainDownloads,
	})
}
