package conf

import (
	"github.com/jylc/cloudserver/pkg/utils"
	"github.com/sirupsen/logrus"
	"gopkg.in/ini.v1"
)

type databaseConf struct {
	Type        string
	Host        string
	Port        string
	User        string
	Password    string
	TablePrefix string
	Charset     string
}

type systemConf struct {
	AppMode string
	Port    string
}

type redisConf struct {
	Server   string
	Password string
	Db       uint
}

var cfg *ini.File
var (
	Dbc *databaseConf
	Sc  *systemConf
	Rc  *redisConf
)

func Init(path string) {
	var err error
	if path != "" && utils.Exist(path) {
		cfg, err = ini.Load(path)
		if err != nil {
			logrus.Error(err)
			return
		}
		Dbc = new(databaseConf)
		err = cfg.Section("database").MapTo(Dbc)
		if err != nil {
			logrus.Error(err)
			return
		}
		Sc = new(systemConf)
		err = cfg.Section("system").MapTo(Sc)
		if err != nil {
			logrus.Error(err)
			return
		}
		Rc = new(redisConf)
		err = cfg.Section("redis").MapTo(Rc)
		if err != nil {
			logrus.Error(err)
			return
		}
	}
}
