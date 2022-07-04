package utils

import "os"

func Exist(path string) bool {
	_, err := os.Stat(path)
	if err != nil {
		return false
	}
	return true
}
