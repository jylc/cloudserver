package conf

import (
	"bytes"
	"github.com/go-playground/validator/v10"
	"github.com/jylc/cloudserver/pkg/utils"
	"github.com/sirupsen/logrus"
	"gopkg.in/ini.v1"
	"time"
)

type databaseConf struct {
	Type        string `ini:"type" `
	Host        string `ini:"host"`
	Port        string `ini:"port"`
	User        string `ini:"user"`
	Password    string `ini:"password"`
	Name        string `ini:"name"`
	TablePrefix string `ini:"tableprefix"`
	Charset     string `ini:"charset"`
}

type systemConf struct {
	AppMode       string `ini:"app_mode" validate:"eq=development|eq=release"`
	Port          string `ini:"port"`
	SessionSecret string `ini:"secret"`
	HashIDSalt    string `ini:"hashidsalt"`
}

type redisConf struct {
	Server   string `ini:"server"`
	Password string `ini:"password"`
	Db       string `ini:"db"`
}
type defaultConfig struct {
	Database *databaseConf
	Redis    *redisConf
	System   *systemConf
}

type corsConfig struct {
	AllowOrigins     []string
	AllowMethods     []string
	AllowHeaders     []string
	ExposeHeaders    []string
	AllowCredentials bool
	AllowOriginFunc  func(origin string) bool
	MaxAge           time.Duration
}

var cfg *ini.File

var dC = &defaultConfig{
	Database: Dbc,
	Redis:    Rc,
	System:   Sc,
}

func Init(path string) {
	var err error
	if path != "" && utils.Exist(path) {
		//如果存在配置文件，就将文件存在的配置覆盖默认配置
		cfg, err = ini.Load(path)
		if err != nil {
			logrus.Error(err)
			return
		}

		configMaps := map[string]interface{}{
			"Database": Dbc,
			"System":   Sc,
			"Redis":    Rc,
		}

		for name, entity := range configMaps {
			err = mapTo(name, entity)
			if err != nil {
				logrus.Panic(err)
			}
		}

		err = cfg.SaveTo(path)
		if err != nil {
			logrus.Error(err)
		}
	} else {
		file, err := utils.CreateNestedFile(path)
		if err != nil {
			logrus.Panic(err)
		}
		_ = file.Close()

		cfg = ini.Empty()
		err = ini.ReflectFrom(cfg, &dC)
		if err != nil {
			logrus.Panic(err)
		}
		err = cfg.SaveTo(path)
		if err != nil {
			logrus.Panic(err)
		}
	}
	buffer := new(bytes.Buffer)
	_, _ = cfg.WriteTo(buffer)
	logrus.Println(buffer)

}

func mapTo(name string, entity interface{}) error {
	err := cfg.Section(name).MapTo(entity)
	if err != nil {
		return err
	}
	validate := validator.New()
	err = validate.Struct(entity)
	if err != nil {
		return err
	}
	return nil
}
