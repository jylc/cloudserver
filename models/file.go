package models

import (
	"errors"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
	"path"
	"time"
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

func (file *File) GetSize() uint64 {
	return file.Size
}

func (file *File) GetName() string {
	return file.Name
}

func (file *File) ModTime() time.Time {
	return file.UpdatedAt
}

func (file *File) IsDir() bool {
	return false
}

func (file *File) GetPosition() string {
	return file.Position
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

func (file *File) UpdateSize(value uint64) error {
	tx := Db.Begin()
	var sizeDelta uint64
	operator := "+"
	user := User{}
	user.ID = file.UserID
	if value > file.Size {
		sizeDelta = value - file.Size
	} else {
		operator = "-"
		sizeDelta = file.Size - value
	}

	if res := tx.Model(&file).Where("size = ?", file.Size).Set("gorm:association_autoupdate", false).Update("size", value); res.Error != nil {
		tx.Rollback()
		return res.Error
	}
	if err := user.ChangeStorage(tx, operator, sizeDelta); err != nil {
		tx.Rollback()
		return err
	}

	file.Size = value
	return tx.Commit().Error
}

func (file *File) PopChunkToFile(lastModified *time.Time, picInfo string) error {
	file.UploadSessionID = nil
	if lastModified != nil {
		file.UpdatedAt = *lastModified
	}
	return Db.Model(file).UpdateColumns(map[string]interface{}{
		"upload_session_id": file.UploadSessionID,
		"updated_at":        file.UpdatedAt,
		"pic_info":          picInfo,
	}).Error
}

func GetFilesByUploadSession(sessionID string, uid uint) (*File, error) {
	file := File{}
	result := Db.Where("user_id = ? and upload_session_id = ?", uid, sessionID).Find(&file)
	return &file, result.Error
}

func RemoveFilesWithSoftLinks(files []File) ([]File, error) {
	filteredFiles := make([]File, 0)

	var filesWithSoftLinks []File
	tx := Db
	for _, value := range files {
		tx = tx.Or("source_name = ? and policy_id = ? and id != ?", value.SourceName, value.PolicyID, value.ID)
	}
	result := tx.Find(&filesWithSoftLinks)
	if result.Error != nil {
		return nil, result.Error
	}

	if len(filesWithSoftLinks) == 0 {
		filteredFiles = files
	} else {
		for i := 0; i < len(files); i++ {
			finder := false
			for _, value := range filesWithSoftLinks {
				if value.PolicyID == files[i].PolicyID && value.SourceName == files[i].SourceName {
					finder = true
					break
				}
			}

			if !finder {
				filteredFiles = append(filteredFiles, files[i])
			}
		}
	}

	return filteredFiles, nil
}

func DeleteFiles(files []*File, uid uint) error {
	tx := Db.Begin()
	user := &User{}
	user.ID = uid
	var size uint64
	for _, file := range files {
		if file.UserID != uid {
			tx.Rollback()
			return errors.New("user id not consistent")
		}

		result := tx.Unscoped().Where("size = ?", file.Size).Delete(file)
		if result.Error != nil {
			tx.Rollback()
			return result.Error
		}

		if result.RowsAffected == 0 {
			tx.Rollback()
			return errors.New("file size is dirty")
		}

		size += file.Size
	}
	if err := user.ChangeStorage(tx, "-", size); err != nil {
		tx.Rollback()
		return err
	}
	return tx.Commit().Error
}

func GetChildFilesOfFolders(folders *[]Folder) ([]File, error) {
	folderIDs := make([]uint, 0, len(*folders))
	for _, value := range *folders {
		folderIDs = append(folderIDs, value.ID)
	}

	var files []File
	result := Db.Where("folder_id in (?)", folderIDs).Find(&files)
	return files, result.Error
}

func (file *File) Create() error {
	tx := Db.Begin()
	if err := tx.Create(file).Error; err != nil {
		logrus.Warningf("Unable to insert file record, %s", err)
		tx.Rollback()
		return err
	}

	user := &User{}
	user.ID = file.UserID
	if err := user.ChangeStorage(tx, "+", file.Size); err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit().Error
}

func (folder *Folder) GetChildFile(name string) (*File, error) {
	var file File
	result := Db.Where("folder_id = ? AND name = ?", folder.ID, name).Find(&file)

	if result.Error == nil {
		file.Position = path.Join(folder.Position, folder.Name)
	}
	return &file, result.Error
}

func (file *File) UpdateSourceName(value string) error {
	return Db.Model(&file).Set("gorm:association_autoupdate", false).Update("source_name", value).Error
}

func (folder *Folder) Rename(new string) error {
	return Db.Model(&folder).UpdateColumn("name", new).Error
}

func (file *File) Rename(new string) error {
	return Db.Model(&file).UpdateColumn("name", new).Error
}

func (file *File) CanCopy() bool {
	return file.UploadSessionID == nil
}

func GetUploadPlaceholderFiles(uid uint) []*File {
	query := Db
	if uid != 0 {
		query = query.Where("user_id = ?", uid)
	}

	var files []*File
	query.Where("upload_session_id is not NULL").Find(&files)
	return files
}
