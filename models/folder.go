package models

import (
	"errors"
	"github.com/jylc/cloudserver/pkg/utils"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
	"path"
	"time"
)

type Folder struct {
	gorm.Model
	Name     string `gorm:"unique_index:idx_only_on_name"`
	ParentID *uint  `gorm:"index:parent_id;unique_index:idx_only_one_name"`
	OwnerID  uint   `gorm:"index:owner_id"`

	Position string `gorm:"-"`
}

func (folder *Folder) GetSize() uint64 {
	return 0
}

func (folder *Folder) GetName() string {
	return folder.Name
}

func (folder *Folder) ModTime() time.Time {
	return folder.UpdatedAt
}

func (folder *Folder) IsDir() bool {
	return true
}

func (folder *Folder) GetPosition() string {
	return folder.Position
}

func (folder *Folder) Create() (uint, error) {
	if err := Db.FirstOrCreate(folder, *folder).Error; err != nil {
		folder.Model = gorm.Model{}
		err2 := Db.First(folder, *folder).Error
		return folder.ID, err2
	}
	return folder.ID, nil
}

func GetFolderByIDs(ids []uint, uid uint) ([]Folder, error) {
	var folders []Folder
	result := Db.Where("id in (?) AND owner_id = ?", ids, uid).Find(&folders)
	return folders, result.Error
}

func (folder *Folder) GetChildFolder() ([]Folder, error) {
	var folders []Folder
	result := Db.Where("parent_id = ?", folder.ID).Find(&folders)

	if result.Error == nil {
		for i := 0; i < len(folders); i++ {
			folders[i].Position = path.Join(folder.Position, folder.Name)
		}
	}
	return folders, result.Error
}

func GetRecursiveChildFolder(dirs []uint, uid uint, includeSelf bool) ([]Folder, error) {
	folders := make([]Folder, 0, len(dirs))
	var parFolders []Folder
	result := Db.Where("owner_id =? and id in (?)", uid, dirs).Find(&parFolders)
	if result.Error != nil {
		return folders, result.Error
	}

	var parentIDs = make([]uint, 0, len(parFolders))
	for _, folder := range parFolders {
		parentIDs = append(parentIDs, folder.ID)
	}

	if includeSelf {
		folders = append(folders, parFolders...)
	}

	parFolders = []Folder{}

	for i := 0; i < 65535; i++ {
		result = Db.Where("owner_id = ? and parent_id in (?)", uid, parentIDs).Find(&parFolders)
		if len(parFolders) == 0 {
			break
		}

		parentIDs = make([]uint, 0, len(parFolders))
		for _, folder := range parFolders {
			parentIDs = append(parentIDs, folder.ID)
		}

		folders = append(folders, parFolders...)
		parFolders = []Folder{}
	}
	return folders, result.Error
}

func DeleteFolderByIDs(ids []uint) error {
	result := Db.Where("id in (?)", ids).Unscoped().Delete(&Folder{})
	return result.Error
}

func (folder *Folder) GetChild(name string) (*Folder, error) {
	var resFolder Folder
	err := Db.Where("parent_id = ? AND owner_id = ? AND name = ?", folder.ID, folder.OwnerID, name).First(&resFolder).Error
	if err == nil {
		resFolder.Position = path.Join(folder.Position, folder.Name)
	}
	return &resFolder, err
}

func (folder *Folder) CopyFolderTo(folderID uint, dstFolder *Folder) (size uint64, err error) {
	subFolders, err := GetRecursiveChildFolder([]uint{folderID}, folder.OwnerID, true)
	if err != nil {
		return 0, nil
	}

	var subFolderIDs = make([]uint, len(subFolders))
	for key, value := range subFolders {
		subFolderIDs[key] = value.ID
	}

	var newIDCache = make(map[uint]uint)
	for _, folder := range subFolders {
		var newID uint
		if folder.ID == folderID {
			newID = dstFolder.ID
		} else if IDCache, ok := newIDCache[*folder.ParentID]; ok {
			newID = IDCache
		} else {
			logrus.Warningf("Unable to get new parent directory: %d", folder.ParentID)
			return size, errors.New("unable to get new parent directory")
		}

		oldID := folder.ID
		folder.Model = gorm.Model{}
		folder.ParentID = &newID
		folder.OwnerID = dstFolder.OwnerID
		if err = Db.Create(&folder).Error; err != nil {
			return size, err
		}

		newIDCache[oldID] = folder.ID
	}

	var originFiles = make([]File, 0, len(subFolderIDs))
	if err := Db.Where(
		"user_id = ? and folder_id in (?)",
		folder.OwnerID,
		subFolderIDs,
	).Find(&originFiles).Error; err != nil {
		return 0, err
	}

	for _, oldFile := range originFiles {
		if !oldFile.CanCopy() {
			logrus.Warningf("Unable to copy the file being uploaded [%s], skipping", oldFile.Name)
			continue
		}

		oldFile.Model = gorm.Model{}
		oldFile.FolderID = newIDCache[oldFile.FolderID]
		oldFile.UserID = dstFolder.OwnerID
		if err := Db.Create(&oldFile).Error; err != nil {
			return size, err
		}

		size += oldFile.Size
	}
	return size, nil
}

func (folder *Folder) MoveOrCopyFileTo(files []uint, dstFolder *Folder, isCopy bool) (uint64, error) {
	var copiedSize uint64
	if isCopy {
		var originFiles = make([]File, 0, len(files))
		if err := Db.Where(
			"id in (?) and user_id = ? and folder_id = ?",
			files,
			folder.OwnerID,
			folder.ID,
		).Find(&originFiles).Error; err != nil {
			return 0, err
		}

		for _, oldFile := range originFiles {
			if !oldFile.CanCopy() {
				logrus.Warningf("Unable to copy the file being uploaded [%s], skipping", oldFile.Name)
				continue
			}

			oldFile.Model = gorm.Model{}
			oldFile.FolderID = dstFolder.ID
			oldFile.UserID = dstFolder.OwnerID

			if err := Db.Create(&oldFile).Error; err != nil {
				return copiedSize, err
			}
			copiedSize += oldFile.Size
		}
	} else {
		err := Db.Model(File{}).Where(
			"id in (?) and user_id = ? and folder_id = ?",
			files,
			folder.OwnerID,
			folder.ID,
		).Updates(map[string]interface{}{
			"folder_id": dstFolder.ID,
		}).Error
		if err != nil {
			return 0, err
		}
	}
	return copiedSize, nil
}

func (folder *Folder) MoveFolderTo(dirs []uint, dstFolder *Folder) error {
	if folder.OwnerID == dstFolder.OwnerID && utils.ContainsUint(dirs, dstFolder.ID) {
		return errors.New("cannot move a folder into itself")
	}

	err := Db.Model(Folder{}).Where(
		"id in (?) and owner_id = ? and parent_id = ?",
		dirs,
		folder.OwnerID,
		folder.ID).Updates(map[string]interface{}{
		"parent_id": dstFolder.ID,
	}).Error

	return err
}
