package main

import (
	_ "embed"
	"flag"
	"github.com/jylc/cloudserver/bootstrap"
	"github.com/jylc/cloudserver/pkg/conf"
	"github.com/jylc/cloudserver/pkg/utils"
	"github.com/jylc/cloudserver/routers"
	"github.com/mholt/archiver/v4"
	"github.com/sirupsen/logrus"
	"io"
	"strings"
)

var (
	configFile string
	scriptName string
)

//go:embed frontend.zip
var assets string

//init 初始化参数
func init() {
	flag.StringVar(&configFile, "config", utils.RelativePath("config.ini"), "config file name")
	flag.StringVar(&scriptName, "database-script", "", "database script name")
	flag.Parse()
	staticFile := archiver.ArchiveFS{
		Stream: io.NewSectionReader(strings.NewReader(assets), 0, int64(len(assets))),
		Format: archiver.Zip{},
	}
	bootstrap.Init(configFile, staticFile)
}

func main() {
	r := routers.RouterInit()
	err := r.Run(conf.Sc.Port)
	if err != nil {
		logrus.Errorf("cannot listen port[%s],%s\n", conf.Sc.Port, err)
	}
}
