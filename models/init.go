package models

import (
	"fmt"
	"github.com/jylc/cloudserver/pkg/conf"
	"github.com/sirupsen/logrus"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"gorm.io/gorm/schema"
	"strings"
	"time"
)

var Db *gorm.DB

func Init() {
	var (
		db  *gorm.DB
		err error
	)
	dbType := strings.ToLower(conf.Dbc.Type)
	switch dbType {
	case "mysql":
		dsn := fmt.Sprintf("%s:%s@(%s:%s)/%s?charset=%s&parseTime=True&loc=Local",
			conf.Dbc.User,
			conf.Dbc.Password,
			conf.Dbc.Host,
			conf.Dbc.Port,
			conf.Dbc.Name,
			conf.Dbc.Charset)
		db, err = gorm.Open(mysql.New(mysql.Config{
			DSN:                       dsn,   // DSN data source name
			DefaultStringSize:         256,   // string 类型字段的默认长度
			DisableDatetimePrecision:  true,  // 禁用 datetime 精度，MySQL 5.6 之前的数据库不支持
			DontSupportRenameIndex:    true,  // 重命名索引时采用删除并新建的方式，MySQL 5.7 之前的数据库和 MariaDB 不支持重命名索引
			DontSupportRenameColumn:   true,  // 用 `change` 重命名列，MySQL 8 之前的数据库和 MariaDB 不支持重命名列
			SkipInitializeWithVersion: false, // 根据当前 MySQL 版本自动配置
		}), &gorm.Config{
			NamingStrategy: schema.NamingStrategy{
				TablePrefix: conf.Dbc.TablePrefix,
			},
		})

	default:
		logrus.Panicf("cannot recognize database [%s]\n", dbType)
	}
	if err != nil {
		logrus.Panicf("cann not open database [%s], %s\n", dbType, err)
	}

	if strings.Compare(conf.Sc.AppMode, "development") == 0 {
		db.Logger = logger.Default.LogMode(logger.Info)
	} else {
		db.Logger = logger.Default.LogMode(logger.Warn)
	}
	sqlDb, err := db.DB()
	if err != nil {
		logrus.Panicf("get database failed, %s\n", err)
	}
	sqlDb.SetMaxIdleConns(50)
	sqlDb.SetMaxOpenConns(50)
	sqlDb.SetConnMaxLifetime(time.Second * 30)
	Db = db
}
