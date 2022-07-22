package bootstrap

import (
	"github.com/gin-gonic/gin"
	"github.com/jylc/cloudserver/models"
	"github.com/jylc/cloudserver/pkg/aria2"
	"github.com/jylc/cloudserver/pkg/auth"
	"github.com/jylc/cloudserver/pkg/cache"
	"github.com/jylc/cloudserver/pkg/cluster"
	"github.com/jylc/cloudserver/pkg/conf"
	"github.com/jylc/cloudserver/pkg/crontab"
	"github.com/jylc/cloudserver/pkg/email"
	"github.com/jylc/cloudserver/pkg/mq"
	"github.com/jylc/cloudserver/pkg/task"
	"io/fs"
	"strings"
)

func Init(path string, staticFile fs.FS) {
	loggerInit()
	appInit()
	conf.Init(path)
	if strings.Compare(conf.Sc.AppMode, "development") != 0 {
		gin.SetMode(gin.ReleaseMode)
	}
	startUp := []struct {
		model   string
		factory func()
	}{
		{
			"both",
			func() {
				cache.Init()
			},
		},
		{
			"both",
			func() {
				auth.Init()
			},
		},
		{
			"master",
			func() {
				email.Init()
			},
		}, {
			"both",
			func() {
				staticInit(staticFile)
			},
		}, {
			"master",
			func() {
				cluster.Init()
			},
		}, {
			"master",
			func() {
				models.Init()
			},
		}, {
			"both",
			func() {
				task.Init()
			},
		}, {
			"master",
			func() {
				aria2.Init(false, cluster.Default, mq.GlobalMQ)
			},
		}, {
			"master",
			func() {
				crontab.Init()
			},
		}, {
			"slave",
			func() {
				cluster.InitController()
			},
		},
	}

	for _, s := range startUp {
		switch s.model {
		case "master":
			if conf.Sc.Role == "master" {
				s.factory()
			}
		case "slave":
			if conf.Sc.Role == "slave" {
				s.factory()
			}
		default:
			s.factory()
		}
	}
}
