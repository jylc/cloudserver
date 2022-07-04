package bootstrap

import "github.com/jylc/cloudserver/pkg/conf"

func Init(path string) {
	appInit()
	conf.Init(path)
}
