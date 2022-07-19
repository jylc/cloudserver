package filesystem

import (
	"context"
	"fmt"
	"github.com/jylc/cloudserver/models"
	"github.com/jylc/cloudserver/pkg/filesystem/fsctx"
	"github.com/jylc/cloudserver/pkg/hashid"
	"github.com/jylc/cloudserver/pkg/serializer"
	"github.com/jylc/cloudserver/pkg/utils"
	"path"
	"strings"
)

func (fs *FileSystem) Delete(ctx context.Context, dirs, files []uint, force bool) error {
	var deletedFiles = make([]*models.File, 0, len(fs.FileTarget))
	var allFiles = make([]*models.File, 0, len(fs.FileTarget))

	if len(dirs) > 0 {
		err := fs.ListDeleteDirs(ctx, dirs)
		if err != nil {
			return err
		}
	}

	if len(files) > 0 {
		err := fs.ListDeleteFiles(ctx, files)
		if err != nil {
			return err
		}
	}
	filesToBeDelete, err := models.RemoveFilesWithSoftLinks(fs.FileTarget)
	if err != nil {
		return ErrDBListObjects.WithError(err)
	}

	policyGroup := fs.GroupFileByPolicy(ctx, filesToBeDelete)
	failed := fs.deleteGroupedFile(ctx, policyGroup)
	for i := 0; i < len(fs.FileTarget); i++ {
		if !utils.ContainsString(failed[fs.FileTarget[i].PolicyID], fs.FileTarget[i].SourceName) {
			deletedFiles = append(deletedFiles, &fs.FileTarget[i])
		}
		allFiles = append(allFiles, &fs.FileTarget[i])
	}
	if force {
		deletedFiles = allFiles
	}

	err = models.DeleteFiles(deletedFiles, fs.User.ID)
	if err != nil {
		return ErrDBDeleteObjects.WithError(err)
	}

	deletedFileIDs := make([]uint, len(deletedFiles))
	for k, file := range deletedFiles {
		deletedFileIDs[k] = file.ID
	}

	models.DeleteShareBySourceIDs(deletedFileIDs, false)

	if len(deletedFiles) == len(allFiles) {
		var allFolderIDs = make([]uint, 0, len(fs.DirTarget))
		for _, value := range fs.DirTarget {
			allFolderIDs = append(allFolderIDs, value.ID)
		}
		err = models.DeleteFolderByIDs(allFolderIDs)
		if err != nil {
			return ErrDBDeleteObjects.WithError(err)
		}

		models.DeleteShareBySourceIDs(allFolderIDs, true)
	}

	if notDeleted := len(fs.FileTarget) - len(deletedFiles); notDeleted > 0 {
		return serializer.NewError(serializer.CodeNotFullySuccess,
			fmt.Sprintf("Failed to delete %d file(s).", notDeleted),
			nil)
	}
	return nil
}

func (fs *FileSystem) ListDeleteDirs(ctx context.Context, ids []uint) error {
	folders, err := models.GetRecursiveChildFolder(ids, fs.User.ID, true)
	if err != nil {
		return ErrDBListObjects.WithError(err)
	}

	for i := 0; i < len(folders); i++ {
		if folders[i].ParentID == nil {
			folders = append(folders[:i], folders[i+1:]...)
			break
		}
	}

	fs.SetTargetDir(&folders)

	files, err := models.GetChildFilesOfFolders(&folders)
	if err != nil {
		return ErrDBListObjects.WithError(err)
	}
	fs.SetTargetFile(&files)
	return nil
}

func (fs *FileSystem) ListDeleteFiles(ctx context.Context, ids []uint) error {
	files, err := models.GetFilesByIDs(ids, fs.User.ID)
	if err != nil {
		return ErrDBListObjects.WithError(err)
	}
	fs.SetTargetFile(&files)
	return nil
}

func (fs *FileSystem) CreateDirectory(ctx context.Context, fullPath string) (*models.Folder, error) {
	if fullPath == "." || fullPath == "" {
		return nil, ErrRootProtected
	}

	if fullPath == "/" {
		if fs.Root != nil {
			return fs.Root, nil
		}
		return fs.User.Root()
	}

	fullPath = path.Clean(fullPath)
	base := path.Dir(fullPath)
	dir := path.Base(fullPath)

	dir = strings.TrimRight(dir, " ")

	if !fs.ValidateLegalName(ctx, dir) {
		return nil, ErrIllegalObjectName
	}

	isExist, parent := fs.IsPathExist(base)
	if !isExist {
		newParent, err := fs.CreateDirectory(ctx, base)
		if err != nil {
			return nil, err
		}
		parent = newParent
	}

	if ok, _ := fs.IsChildFileExist(parent, dir); ok {
		return nil, ErrFileExisted
	}

	newFolder := models.Folder{
		Name:     dir,
		ParentID: &parent.ID,
		OwnerID:  fs.User.ID,
	}
	_, err := newFolder.Create()
	if err != nil {
		return nil, fmt.Errorf("failed to create folder: %w", err)
	}

	return &newFolder, nil
}

func (fs *FileSystem) ListPhysical(ctx context.Context, dirPath string) ([]serializer.Object, error) {
	if err := fs.DispatchHandler(); fs.Policy == nil || err != nil {
		return nil, ErrUnknownPolicyType
	}

	if !fs.Policy.CanStructureBeListed() {
		return nil, nil
	}

	objects, err := fs.Handler.List(ctx, dirPath, false)
	if err != nil {
		return nil, err
	}

	var (
		folders []models.Folder
	)

	for _, object := range objects {
		if object.IsDir {
			folders = append(folders, models.Folder{Name: object.Name})
		}
	}
	return fs.listObjects(ctx, dirPath, nil, folders, nil), nil
}

func (fs *FileSystem) List(ctx context.Context, dirPath string, pathProcessor func(string) string) ([]serializer.Object, error) {
	isExist, folder := fs.IsPathExist(dirPath)
	if !isExist {
		return nil, ErrPathNotExist
	}
	fs.SetTargetDir(&[]models.Folder{*folder})

	var parentPath = path.Join(folder.Position, folder.Name)
	var childFolders []models.Folder
	var childFiles []models.File

	childFolders, _ = folder.GetChildFolder()

	childFiles, _ = folder.GetChildFiles()
	return fs.listObjects(ctx, parentPath, childFiles, childFolders, pathProcessor), nil
}

func (fs *FileSystem) listObjects(ctx context.Context, parent string, files []models.File, folders []models.Folder, pathProcessor func(string) string) []serializer.Object {
	shareKey := ""
	if key, ok := ctx.Value(fsctx.ShareKeyCtx).(string); ok {
		shareKey = key
	}

	objects := make([]serializer.Object, 0, len(files)+len(folders))

	var processedPath string

	for _, subFolder := range folders {
		if processedPath == "" {
			if pathProcessor != nil {
				processedPath = pathProcessor(parent)
			} else {
				processedPath = parent
			}
		}

		objects = append(objects, serializer.Object{
			ID:         hashid.HashID(subFolder.ID, hashid.FolderID),
			Name:       subFolder.Name,
			Path:       processedPath,
			Pic:        "",
			Size:       0,
			Type:       "dir",
			Date:       subFolder.UpdatedAt,
			CreateDate: subFolder.CreatedAt,
		})
	}

	for _, file := range files {
		if processedPath == "" {
			if pathProcessor != nil {
				processedPath = pathProcessor(parent)
			} else {
				processedPath = parent
			}
		}
		if file.UploadSessionID == nil {
			newFile := serializer.Object{
				ID:            hashid.HashID(file.ID, hashid.FileID),
				Name:          file.Name,
				Path:          processedPath,
				Pic:           file.PicInfo,
				Size:          file.Size,
				Type:          "file",
				Date:          file.UpdatedAt,
				CreateDate:    file.CreatedAt,
				SourceEnabled: file.GetPolicy().IsOriginLinkEnable,
			}
			if shareKey != "" {
				newFile.Key = shareKey
			}
			objects = append(objects, newFile)
		}
	}
	return objects
}
