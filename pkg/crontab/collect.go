package crontab

import (
	"context"
	"github.com/jylc/cloudserver/models"
	"github.com/jylc/cloudserver/pkg/cache"
	"github.com/jylc/cloudserver/pkg/filesystem"
	"github.com/jylc/cloudserver/pkg/utils"
	"github.com/sirupsen/logrus"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func garbageCollect() {
	collectArchiveFile()

	if store, ok := cache.Store.(*cache.MemoStore); ok {
		collectCache(store)
	}
	logrus.Info("The scheduled task [cron_garbage_collect] is completed")
}

func collectArchiveFile() {
	tempPath := utils.RelativePath(models.GetSettingByName("temp_path"))
	expires := models.GetIntSetting("download_timeout", 30)

	root := filepath.Join(tempPath, "archive")
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err == nil && !info.IsDir() && strings.HasPrefix(filepath.Base(path), "archive_") && time.Now().Sub(info.ModTime()).Seconds() > float64(expires) {
			logrus.Debugf("Delete the expired package download temporary file [%s]", path)
			if err := os.Remove(path); err != nil {
				logrus.Debugf("Failed to delete temporary file [%s],%s", path, err)
			}
		}
		return nil
	})

	if err != nil {
		logrus.Debugf("[scheduled task] cannot list temporary packaging directory")
	}
}

func collectCache(store *cache.MemoStore) {
	logrus.Debugf("Clean up memory cache")
	store.GarbageCollect()
}

func uploadSessionCollect() {
	placeholders := models.GetUploadPlaceholderFiles(0)

	userToFiles := make(map[uint][]uint)
	for _, file := range placeholders {
		_, sessionExist := cache.Get(filesystem.UploadSessionCachePrefix + *file.UploadSessionID)
		if sessionExist {
			continue
		}

		if _, ok := userToFiles[file.UserID]; !ok {
			userToFiles[file.UserID] = make([]uint, 0)
		}

		userToFiles[file.UserID] = append(userToFiles[file.UserID], file.ID)
	}

	for uid, filesIDs := range userToFiles {
		user, err := models.GetUserByID(uid)
		if err != nil {
			logrus.Warningf("The user of the upload session does not exist,%s", err)
			continue
		}

		fs, err := filesystem.NewFileSystem(&user)
		if err != nil {
			logrus.Warningf("Unable to initialize file system,%s", err)
			continue
		}

		if err = fs.Delete(context.Background(), []uint{}, filesIDs, false); err != nil {
			logrus.Warningf("Unable to delete upload session,%s", err)
		}
		fs.Recycle()
	}
	logrus.Info("The scheduled task [cron_recycle_upload_session] is completed")
}
