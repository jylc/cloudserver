package models

import (
	"crypto/md5"
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"github.com/jylc/cloudserver/pkg/utils"
	"gorm.io/gorm"
	"strings"
)

const (
	Activate = iota
	NotActivate
	Baned
	OveruseBaned
)

type User struct {
	gorm.Model
	Email     string `gorm:"type:varchar(100);unique_index"`
	Nick      string `gorm:"size:50"`
	Password  string `json:"-"`
	Status    int
	GroupID   uint
	Storage   uint64
	TwoFactor string
	Avatar    string
	Options   string `json:"-" gorm:"size:4294967295"` //option 使用gorm存入数据库且不被json序列化
	Authn     string `gorm:"4294967295"`

	Group  Group  `gorm:"save_associations:false:false"`
	Policy Policy `gorm:"PRELOAD:false,association_autoupdate:false"`

	OptionsSerialized UserOption `gorm:"-"` //将option序列化且不被gorm存入数据库
}

type UserOption struct {
	ProfileOff     bool   `json:"profile_off,omitempty"`
	PreferredTheme string `json:"preferred_theme,omitempty"`
}

func GetActivateUserByID(uid interface{}) (User, error) {
	var user User
	result := Db.Set("gorm:auto_preload", true).Where("status = ?", Activate).Find(&user, uid)
	return user, result.Error
}

func GetActivateUserByEmail(email string) (User, error) {
	var user User
	result := Db.Set("gorm:auto_preload", true).Where("status = ? and email = ?", Activate, email).First(&user)
	return user, result.Error
}

func (user *User) IsAnonymous() bool {
	return user.ID == 0
}
func (user *User) CheckPassword(password string) (bool, error) {
	passwordStore := strings.Split(user.Password, ":")
	if len(passwordStore) != 2 && len(passwordStore) != 3 {
		return false, errors.New("unknown password type")
	}

	if len(passwordStore) == 3 {
		if passwordStore[0] != "md5" {
			return false, errors.New("unknown password type")
		}
		hash := md5.New()
		_, err := hash.Write([]byte(passwordStore[2] + password))
		bs := hex.EncodeToString(hash.Sum(nil))
		if err != nil {
			return false, err
		}
		return bs == passwordStore[1], nil
	}
	hash := sha1.New()
	_, err := hash.Write([]byte(password + passwordStore[0]))
	bs := hex.EncodeToString(hash.Sum(nil))
	if err != nil {
		return false, err
	}
	return bs == passwordStore[1], nil
}

func (user *User) SetPassword(password string) error {
	salt := utils.RandStringRunes(16)
	hash := sha1.New()
	_, err := hash.Write([]byte(password + salt))
	bs := hex.EncodeToString(hash.Sum(nil))
	if err != nil {
		return err
	}
	user.Password = salt + ":" + bs
	return nil
}

func NewAnonymousUser() *User {
	user := User{}
	user.Policy.Type = "anonymous"
	user.Group, _ = GetGroupByID(3)
	return &user
}

func NewUser() User {
	option := UserOption{}
	return User{
		OptionsSerialized: option,
	}
}

func GetUserByEmail(email string) (User, error) {
	var user User
	result := Db.Set("gorm:auto_preload", true).Where("email = ?", email).First(&user)
	return user, result.Error
}

func (user *User) Update(val map[string]interface{}) error {
	return Db.Model(user).Updates(val).Error
}

func GetUserByID(ID interface{}) (User, error) {
	var user User
	result := Db.Set("gorm:auto_preload", true).First(&user, ID)
	return user, result.Error
}

func (user *User) SetStatus(status int) {
	Db.Model(&user).Update("status", status)
}

func (user *User) ChangeStorage(tx *gorm.DB, operator string, size uint64) error {
	return tx.Model(user).Update("storage", gorm.Expr("storage"+operator+" ?", size)).Error
}
func (user *User) GetRemainingCapacity() uint64 {
	total := user.Group.MaxStorage
	if total <= user.Storage {
		return 0
	}
	return total - user.Storage
}

func (user *User) Root() (*Folder, error) {
	var folder Folder
	err := Db.Where("parent_id is NULL AND owner_id = ?", user.ID).First(&folder).Error
	return &folder, err
}

func (user *User) IncreaseStorageWithoutCheck(size uint64) {
	if size == 0 {
		return
	}
	user.Storage += size
	Db.Model(user).Update("storage", gorm.Expr("storage + ?", size))

}
