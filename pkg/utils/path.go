package utils

import "path/filepath"

func RelativePath(name string) string {
	path, err := filepath.Abs(name)
	if err != nil {
		return ""
	}
	return path
}
