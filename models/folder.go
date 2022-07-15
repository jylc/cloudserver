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
