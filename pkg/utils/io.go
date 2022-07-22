package utils

import (
	"github.com/sirupsen/logrus"
	"os"
	"path/filepath"
)

func CreatNestedFile(path string) (*os.File, error) {
	basePath := filepath.Dir(path)
	if !Exists(basePath) {
		err := os.MkdirAll(basePath, 0700)
		if err != nil {
			logrus.Warning("无法创建目录，%s", err)
			return nil, err
		}
	}

	return os.Create(path)
}

func Exists(name string) bool {
	if _, err := os.Stat(name); err != nil {
		if os.IsNotExist(err) {
			return false
		}
	}
	return true
}
