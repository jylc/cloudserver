package utils

import (
	"os"
	"path/filepath"
)

func Exist(path string) bool {
	_, err := os.Stat(path)
	if err != nil {
		return false
	}
	return true
}

func CreateNestedFile(path string) (*os.File, error) {
	dir := filepath.Dir(path)
	if !Exist(dir) {
		err := os.MkdirAll(dir, 0777)
		if err != nil {
			return nil, err
		}
	}
	return os.Create(path)
}
