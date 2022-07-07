package bootstrap

import (
	"github.com/gin-gonic/gin"
	"github.com/jylc/cloudserver/pkg/conf"
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
	staticInit(staticFile)
}
