package filesystem

import (
	"github.com/jylc/cloudserver/models"
	"github.com/jylc/cloudserver/pkg/utils"
	"path"
)

func (fs *FileSystem) IsPathExist(path string) (bool, *models.Folder) {
	pathList := utils.SplitPath(path)
	if len(pathList) == 0 {
		return false, nil
	}

	var currentFolder *models.Folder

	if fs.Root != nil {
		currentFolder = fs.Root
	}

	for _, folderName := range pathList {
		var err error
		if folderName == "/" {
			if currentFolder != nil {
				continue
			}
			currentFolder, err = fs.User.Root()
			if err != nil {
				return false, nil
			}
		} else {
			currentFolder, err = currentFolder.GetChild(folderName)
			if err != nil {
				return false, nil
			}
		}
	}
	return true, currentFolder
}

func (fs *FileSystem) IsChildFileExist(folder *models.Folder, name string) (bool, *models.File) {
	file, err := folder.GetChildFile(name)
	return err == nil, file
}

func (fs *FileSystem) IsFileExist(fullPath string) (bool, *models.File) {
	basePath := path.Dir(fullPath)
	fileName := path.Base(fullPath)

	exist, parent := fs.IsPathExist(basePath)
	if !exist {
		return false, nil
	}

	file, err := parent.GetChildFile(fileName)
	return err == nil, file
}
