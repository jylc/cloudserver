package models

import (
	"gorm.io/gorm"
	"path"
)

type File struct {
	gorm.Model
	Name            string `gorm:"unique_index:idx_only_one"`
	SourceName      string `gorm:"type:text"`
	UserID          uint   `gorm:"index:user_id;unique_index:idx_only_one"`
	Size            uint64
	PicInfo         string
	FolderID        uint `gorm:"index:folder_id;unique_index:idx_only_one"`
	PolicyID        uint
	UploadSessionID *string `gorm:"index:session_id;unique_index:session_only_one"`
	Metadata        string  `gorm:"type:text"`

	Policy             Policy            `gorm:"PRELOAD:false,association_autoupdate:false"`
	Position           string            `gorm:"-"`
	MetadataSerialized map[string]string `gorm:"-"`
}

func GetFilesByIDs(ids []uint, uid uint) ([]File, error) {
	return GetFilesByIDsFromTX(Db, ids, uid)
}

func GetFilesByIDsFromTX(tx *gorm.DB, ids []uint, uid uint) ([]File, error) {
	var files []File
	var result *gorm.DB
	if uid == 0 {
		result = tx.Where("id in (?)", ids).Find(&files)
	} else {
		result = tx.Where("id in (?) AND user_id = ?", ids, uid).Find(&files)
	}
	return files, result.Error
}

func (file *File) GetPolicy() *Policy {
	if file.Policy.Model.ID == 0 {
		file.Policy, _ = GetPolicyByID(file.PolicyID)
	}
	return &file.Policy
}

func (folder *Folder) GetChildFiles() ([]File, error) {
	var files []File
	result := Db.Where("folder_id = ?", folder.ID).Find(&files)

	if result.Error == nil {
		for i := 0; i < len(files); i++ {
			files[i].Position = path.Join(folder.Position, folder.Name)
		}
	}

	return files, result.Error
}
