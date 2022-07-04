package main

import (
	_ "embed"
	"flag"
	"github.com/gin-gonic/gin"
	"github.com/jylc/cloudserver/bootstrap"
	"github.com/jylc/cloudserver/pkg/utils"
	"net/http"
)

var (
	configFile string
)

//go:embed frontend.zip
var frontend string

//init 初始化参数
func init() {
	flag.StringVar(&configFile, "config", utils.RelativePath("config.ini"), "config file name")
	flag.Parse()
	bootstrap.Init(configFile)
}

func main() {
	r := gin.Default()

	r.GET("/ping", func(context *gin.Context) {
		context.JSON(http.StatusOK, gin.H{
			"message": "pong",
		})
	})
	r.Run(":8081")
}
