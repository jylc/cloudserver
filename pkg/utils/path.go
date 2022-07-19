package utils

import (
	"path/filepath"
	"strings"
)

func RelativePath(name string) string {
	path, err := filepath.Abs(name)
	if err != nil {
		return ""
	}
	return path
}

func SplitPath(path string) []string {
	if len(path) == 0 || path[0] != '/' {
		return []string{}
	}

	if path == "/" {
		return []string{"/"}
	}

	pathSplit := strings.Split(path, "/")
	pathSplit[0] = "/"
	return pathSplit
}
