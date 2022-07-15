package local

import (
	"context"
	"errors"
	"fmt"
	"github.com/jylc/cloudserver/models"
	"github.com/jylc/cloudserver/pkg/auth"
	"github.com/jylc/cloudserver/pkg/cache"
	"github.com/jylc/cloudserver/pkg/filesystem/fsctx"
	"github.com/jylc/cloudserver/pkg/filesystem/response"
	"github.com/jylc/cloudserver/pkg/serializer"
	"github.com/jylc/cloudserver/pkg/utils"
	"github.com/sirupsen/logrus"
	"io"
	"io/fs"
	"net/url"
	"os"
	"path/filepath"
)

const (
	Perm = 0744
)

type Driver struct {
	Policy *models.Policy
}

func (handler Driver) List(ctx context.Context, path string, recursive bool) ([]response.Object, error) {
	var res []response.Object

	root := utils.RelativePath(filepath.FromSlash(path))

	err := filepath.Walk(root, func(path string, info fs.FileInfo, err error) error {
		if path == root {
			return nil
		}
		if err != nil {
			logrus.Warnf("cannot ergodic catalogue %s, %s\n", path, err)
			return filepath.SkipDir
		}

		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}

		res = append(res, response.Object{
			Name:         info.Name(),
			RelativePath: filepath.ToSlash(rel),
			Source:       path,
			Size:         uint64(info.Size()),
			IsDir:        info.IsDir(),
			LastModify:   info.ModTime(),
		})

		if !recursive && info.IsDir() {
			return filepath.SkipDir
		}

		return nil
	})
	return res, err
}

func (handler Driver) Get(ctx context.Context, path string) (response.RSCloser, error) {
	file, err := os.Open(utils.RelativePath(path))
	if err != nil {
		logrus.Debugf("cannot open file:%s\n", err)
		return nil, err
	}
	return file, err
}

func (handler Driver) Put(ctx context.Context, file fsctx.FileHeader) error {
	defer file.Close()
	fileInfo := file.Info()
	dst := utils.RelativePath(filepath.FromSlash(fileInfo.SavePath))

	if fileInfo.Mode&fsctx.Overwrite != fsctx.Overwrite {
		if utils.Exist(dst) {
			logrus.Warningf("A file with the same physical name already exists or is unavailable: %s\n", dst)
			return errors.New("A file with the same physical name already exists or is unavailable")
		}
	}

	basePath := filepath.Dir(dst)
	if !utils.Exist(basePath) {
		err := os.MkdirAll(basePath, Perm)
		if err != nil {
			logrus.Warningf("cannot create path, %s\n", err)
			return err
		}
	}

	var (
		out *os.File
		err error
	)
	openMode := os.O_CREATE | os.O_RDWR
	if fileInfo.Mode&fsctx.Append == fsctx.Append {
		openMode |= os.O_APPEND
	} else {
		openMode |= os.O_TRUNC
	}

	out, err = os.OpenFile(dst, openMode, Perm)
	if err != nil {
		logrus.Warningf("cannot open or create file, %s\n", err)
		return err
	}
	defer out.Close()

	if fileInfo.Mode&fsctx.Append == fsctx.Append {
		stat, err := out.Stat()
		if err != nil {
			logrus.Warningf("cannot read file info, %s\n", err)
			return err
		}

		if uint64(stat.Size()) < fileInfo.AppendStart {
			return errors.New("the file fragment that has not been uploaded is inconsistent with the expected size")
		} else if uint64(stat.Size()) > fileInfo.AppendStart {
			out.Close()
			if err := handler.Truncate(ctx, dst, fileInfo.AppendStart); err != nil {
				return fmt.Errorf("an error occurred while overwriting the fragment: %w\n", err)
			}

			out, err := os.OpenFile(dst, openMode, Perm)
			defer out.Close()
			if err != nil {
				logrus.Warningf("cannot open or create file, %s\n", err)
				return err
			}
		}
	}

	_, err = io.Copy(out, file)
	return err
}

func (handler Driver) Truncate(ctx context.Context, src string, size uint64) error {
	logrus.Warningf("truncate file [%s] to [%d]\n", src, size)
	out, err := os.OpenFile(src, os.O_WRONLY, Perm)
	if err != nil {
		logrus.Warningf("cannot open file, %s\n", err)
		return err
	}

	defer out.Close()
	return out.Truncate(int64(size))
}

func (handler Driver) Delete(ctx context.Context, files []string) ([]string, error) {
	deletedFailed := make([]string, 0, len(files))
	var retErr error

	for _, value := range files {
		filePath := utils.RelativePath(filepath.FromSlash(value))
		if utils.Exist(filePath) {
			err := os.Remove(filePath)
			if err != nil {
				logrus.Warningf("cannot delete file, %s\n", err)
				retErr = err
				deletedFailed = append(deletedFailed, value)
			}
		}
		_ = os.Remove(utils.RelativePath(value + models.GetSettingByNameWithDefault("thumb_file_suffix", "._thumb")))
	}

	return deletedFailed, retErr
}

func (handler Driver) Thumb(ctx context.Context, path string) (*response.ContentResponse, error) {
	file, err := handler.Get(ctx, path+models.GetSettingByNameWithDefault("thumb_file_suffix", "._thumb"))
	if err != nil {
		return nil, err
	}
	return &response.ContentResponse{
		Redirect: false,
		Content:  file,
	}, nil
}

func (handler Driver) Source(ctx context.Context, path string, baseURL url.URL, ttl int64, isDownload bool, speed int) (string, error) {
	file, ok := ctx.Value(fsctx.FileModelCtx).(models.File)
	if !ok {
		return "", errors.New("unable to get file record context")
	}

	if handler.Policy.BaseURL != "" {
		cdnURL, err := url.Parse(handler.Policy.BaseURL)
		if err != nil {
			return "", err
		}
		baseURL = *cdnURL
	}

	var (
		signedURI *url.URL
		err       error
	)

	if isDownload {
		downloadSessionID := utils.RandStringRunes(16)
		err := cache.Set("download_"+downloadSessionID, file, int(ttl))
		if err != nil {
			return "", serializer.NewError(serializer.CodeCacheOperation, "failed to create download session", err)
		}

		signedURI, err = auth.SignURI(auth.General, fmt.Sprintf("/api/v3/file/download/%s", downloadSessionID), ttl)
	} else {
		signedURI, err = auth.SignURI(auth.General, fmt.Sprintf("/api/v3/file/get/%d/%s", file.ID, file.Name), ttl)
	}

	if err != nil {
		return "", serializer.NewError(serializer.CodeEncryptError, "unable to sign URL", err)
	}

	finalURL := baseURL.ResolveReference(signedURI).String()
	return finalURL, nil
}

func (handler Driver) Token(ctx context.Context, ttl int64, uploadSession *serializer.UploadSession, file fsctx.FileHeader) (*serializer.UploadCredential, error) {
	if utils.Exist(uploadSession.SavePath) {
		return nil, errors.New("placeholder file already exist")
	}

	return &serializer.UploadCredential{
		SessionID: uploadSession.Key,
		ChunkSize: handler.Policy.OptionsSerialized.ChunkSize,
	}, nil
}

func (handler Driver) CancelToken(ctx context.Context, uploadSession *serializer.UploadSession) error {
	return nil
}
