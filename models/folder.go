package models

import "gorm.io/gorm"

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
