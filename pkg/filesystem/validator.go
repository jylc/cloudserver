package filesystem

import (
	"context"
	"github.com/jylc/cloudserver/pkg/utils"
	"path/filepath"
	"strings"
)

var reservedCharacter = []string{"\\", "?", "*", "<", "\"", ":", ">", "/", "|"}

func IsInExtensionList(extList []string, fileName string) bool {
	ext := strings.ToLower(filepath.Ext(fileName))
	if len(ext) == 0 {
		return false
	}

	if utils.ContainsString(extList, ext[1:]) {
		return true
	}
	return false
}

func (fs *FileSystem) ValidateLegalName(ctx context.Context, name string) bool {
	for _, value := range reservedCharacter {
		if strings.Contains(name, value) {
			return false
		}
	}
	if len(name) >= 256 {
		return false
	}

	if len(name) == 0 {
		return false
	}

	if strings.HasSuffix(name, " ") {
		return false
	}

	return true
}

func (fs *FileSystem) ValidateExtension(ctx context.Context, fileName string) bool {
	if len(fs.Policy.OptionsSerialized.FileType) == 0 {
		return true
	}

	return IsInExtensionList(fs.Policy.OptionsSerialized.FileType, fileName)
}

func (fs *FileSystem) ValidateFileSize(ctx context.Context, size uint64) bool {
	if fs.Policy.MaxSize == 0 {
		return true
	}
	return size <= fs.Policy.MaxSize
}
