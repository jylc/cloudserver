package utils

import (
	"path"
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

func RemoveSlash(path string) string {
	if len(path) > 1 {
		return strings.TrimSuffix(path, "/")
	}
	return path
}

func FormSlash(old string) string {
	return path.Clean(strings.ReplaceAll(old, "\\", "/"))
}

func FillSlash(path string) string {
	if path == "/" {
		return path
	}
	return path + "/"
}
