package models

import "gorm.io/gorm"

type Tag struct {
	gorm.Model
	Name       string //标签名
	Icon       string //图标标识
	Color      string //图标颜色
	Type       int    //标签类型
	Expression string `gorm:"type:text"` //搜索表达式
	UserID     uint   //创建者ID
}

func GetTagsByUID(uid uint) ([]Tag, error) {
	var tag []Tag
	result := Db.Where("user_id = ?", uid).Find(&tag)
	return tag, result.Error
}
