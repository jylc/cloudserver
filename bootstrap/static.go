package bootstrap

import (
	"encoding/json"
	"github.com/jylc/cloudserver/pkg/conf"
	"github.com/sirupsen/logrus"
	"io"
	"io/fs"
	"net/http"
)

type staticVersion struct {
	Version string `json:"version"`
	Name    string `json:"name"`
}

type GinFS struct {
	FS http.FileSystem
}

func (g *GinFS) Open(name string) (http.File, error) {
	return g.FS.Open(name)
}

func (g *GinFS) Exists(prefix, filepath string) bool {
	_, err := g.FS.Open(filepath)
	if err != nil {
		logrus.Errorf("file does not exist [%s], %s\n", filepath, err)
		return false
	}
	return true
}

var StaticFS *GinFS

func staticInit(staticFile fs.FS) {
	StaticFS = &GinFS{FS: http.FS(staticFile)}
	file, err := StaticFS.Open("version.json")
	if err != nil {
		logrus.Error(err)
		return
	}
	all, err := io.ReadAll(file)
	if err != nil {
		logrus.Panicf("read file failed:%s\n", err)
		return
	}

	var version staticVersion
	err = json.Unmarshal(all, &version)
	if err != nil {
		logrus.Panicf("cannot parse json file:%s\n", err)
		return
	}
	if version.Name != conf.StaticName {
		logrus.Panicf("need matched frontend sources\n")
		return
	}

	if version.Version != conf.RequiredFrontendVersion {
		logrus.Panicf("need matched frontend version\n")
		return
	}

}
