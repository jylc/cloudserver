package models

import (
	"github.com/jylc/cloudserver/pkg/conf"
	"github.com/sirupsen/logrus"
)

func needMigration() bool {
	var setting Setting
	return Db.Where("name = ?", "db_version_"+conf.RequiredDBVersion).First(&setting).Error != nil
}

func migration() {
	if !needMigration() {
		logrus.Infof("current databse version match required\n")
		return
	}

	logrus.Infof("migrate the database")

}
