package routers

import (
	"github.com/gin-contrib/gzip"
	"github.com/gin-gonic/gin"
	"github.com/jylc/cloudserver/middleware"
	"github.com/jylc/cloudserver/pkg/conf"
)

func RouterInit() *gin.Engine {
	r := gin.Default()

	r.Use(gzip.Gzip(gzip.DefaultCompression, gzip.WithExcludedPaths([]string{"/api/"})))
	r.Use(middleware.FrontendFileHandler())
	r.Use(middleware.Session(conf.Sc.SessionSecret))
	return r
}
