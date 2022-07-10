package models

import "gorm.io/gorm"

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
