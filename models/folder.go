package models

import (
	"gorm.io/gorm"
	"path"
)

type Folder struct {
	gorm.Model
	Name     string `gorm:"unique_index:idx_only_on_name"`
	ParentID *uint  `gorm:"index:parent_id;unique_index:idx_only_one_name"`
	OwnerID  uint   `gorm:"index:owner_id"`

	Position string `gorm:"-"`
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
