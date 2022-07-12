package bootstrap

import (
	"github.com/gin-gonic/gin"
	"github.com/jylc/cloudserver/models"
	"github.com/jylc/cloudserver/pkg/auth"
	"github.com/jylc/cloudserver/pkg/conf"
	"github.com/jylc/cloudserver/pkg/email"
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
				models.Init()
			},
		},
		{
			"both",
			func() {
				auth.Init()
			},
		},
		{
			"both",
			func() {
				email.Init()
			},
		}, {
			"both",
			func() {
				staticInit(staticFile)
			},
		},
	}

	for _, s := range startUp {
		if s.model == "both" {
			s.factory()
		}
	}
}
