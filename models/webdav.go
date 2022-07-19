package models

import "gorm.io/gorm"

type Webdav struct {
	gorm.Model
	Name     string
	Password string `gorm:"unique_index:password_only_on"`
	UserID   uint   `gorm:"unique_index:password_only_on"`
	Root     string `gorm:"type:text"`
}

func ListWebDAVAccounts(uid uint) []Webdav {
	var accounts []Webdav
	Db.Where("user_id = ?", uid).Order("create_at desc").Find(&accounts)
	return accounts
}

func DeleteWebDAVAccountByID(id, uid uint) {
	Db.Where("user_id = ? and id = ?", uid, id).Delete(&Webdav{})
}

func GetWebdavByPassword(password string, uid uint) (*Webdav, error) {
	webdav := &Webdav{}
	res := Db.Where("user_id = ? and password = ?", uid, password).First(webdav)
	return webdav, res.Error
}

func (webdav *Webdav) Create() (uint, error) {
	if err := Db.Create(webdav).Error; err != nil {
		return 0, err
	}
	return webdav.ID, nil
}
