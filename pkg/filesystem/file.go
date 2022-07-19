package filesystem

import (
	"context"
	"github.com/juju/ratelimit"
	"github.com/jylc/cloudserver/models"
	"github.com/jylc/cloudserver/pkg/cache"
	"github.com/jylc/cloudserver/pkg/filesystem/fsctx"
	"github.com/jylc/cloudserver/pkg/filesystem/response"
	"github.com/jylc/cloudserver/pkg/serializer"
	"github.com/sirupsen/logrus"
	"io"
)

type lrs struct {
	response.RSCloser
	r io.Reader
}

func (fs *FileSystem) GetDownloadContent(ctx context.Context, id uint) (response.RSCloser, error) {
	rs, err := fs.GetContent(ctx, id)
	if err != nil {
		return nil, err
	}
	return fs.withSpeedLimit(rs), nil
}

func (fs *FileSystem) GetContent(ctx context.Context, id uint) (response.RSCloser, error) {
	err := fs.resetFileIDIfNotExist(ctx, id)
	if err != nil {
		return nil, nil
	}
	ctx = context.WithValue(ctx, fsctx.FileModelCtx, fs.FileTarget[0])

	rs, err := fs.Handler.Get(ctx, fs.FileTarget[0].SourceName)
	if err != nil {
		return nil, ErrIO.WithError(ErrIO)
	}
	return rs, nil
}

func (fs *FileSystem) GroupFileByPolicy(ctx context.Context, files []models.File) map[uint][]*models.File {
	var policyGroup = make(map[uint][]*models.File)
	for key := range files {
		if file, ok := policyGroup[files[key].PolicyID]; ok {
			policyGroup[files[key].PolicyID] = append(file, &files[key])
		} else {
			policyGroup[files[key].PolicyID] = make([]*models.File, 0)
			policyGroup[files[key].PolicyID] = append(policyGroup[files[key].PolicyID], &files[key])
		}
	}
	return policyGroup
}

func (fs *FileSystem) resetFileIDIfNotExist(ctx context.Context, id uint) error {
	if len(fs.FileTarget) == 0 {
		file, err := models.GetFilesByIDs([]uint{id}, fs.User.ID)
		if err != nil || len(file) == 0 {
			return ErrObjectNotExist
		}
		fs.FileTarget = []models.File{file[0]}
	}

	if parent, ok := ctx.Value(fsctx.LimitParentCtx).(*models.Folder); ok {
		if parent.ID != fs.FileTarget[0].FolderID {
			return ErrObjectNotExist
		}
	}
	return fs.resetPolicyToFirstFile(ctx)
}

func (fs *FileSystem) withSpeedLimit(rs response.RSCloser) response.RSCloser {
	if fs.User.Group.SpeedLimit != 0 {
		speed := fs.User.Group.SpeedLimit
		bucket := ratelimit.NewBucketWithRate(float64(speed), int64(speed))
		lrs := lrs{rs, ratelimit.Reader(rs, bucket)}
		return lrs
	}
	return rs
}

func (fs *FileSystem) deleteGroupedFile(ctx context.Context, files map[uint][]*models.File) map[uint][]string {
	failed := make(map[uint][]string, len(files))
	for policyID, toBeDeletedFiles := range files {
		sourceNamesAll := make([]string, 0, len(toBeDeletedFiles))
		uploadSessions := make([]*serializer.UploadSession, 0, len(toBeDeletedFiles))

		for i := 0; i < len(toBeDeletedFiles); i++ {
			sourceNamesAll = append(sourceNamesAll, toBeDeletedFiles[i].SourceName)

			if toBeDeletedFiles[i].UploadSessionID != nil {
				if session, ok := cache.Get(UploadSessionCachePrefix + *toBeDeletedFiles[i].UploadSessionID); ok {
					uploadSession := session.(serializer.UploadSession)
					uploadSessions = append(uploadSessions, &uploadSession)
				}
			}
		}

		fs.Policy = toBeDeletedFiles[0].GetPolicy()
		err := fs.DispatchHandler()
		if err != nil {
			failed[policyID] = sourceNamesAll
			continue
		}

		for _, upSession := range uploadSessions {
			if err := fs.Handler.CancelToken(ctx, upSession); err != nil {
				logrus.Warningf("cannot cancel [%s]'s upload session: %s", upSession.Name, err)
			}

			cache.Deletes([]string{upSession.Key}, UploadSessionCachePrefix)
		}
		failedFile, _ := fs.Handler.Delete(ctx, sourceNamesAll)
		failed[policyID] = failedFile
	}
	return failed
}

func (fs *FileSystem) AddFile(ctx context.Context, parent *models.Folder, file fsctx.FileHeader) (*models.File, error) {
	err := fs.Trigger(ctx, "BeforeAddFile", file)
	if err != nil {
		return nil, err
	}

	uploadInfo := file.Info()
	newFile := models.File{
		Name:               uploadInfo.FileName,
		SourceName:         uploadInfo.SavePath,
		UserID:             fs.User.ID,
		Size:               uploadInfo.Size,
		FolderID:           parent.ID,
		PolicyID:           fs.Policy.ID,
		MetadataSerialized: uploadInfo.Metadata,
		UploadSessionID:    uploadInfo.UploadSessionID,
	}

	if fs.Policy.IsThumbExist(uploadInfo.FileName) {
		newFile.PicInfo = "1,1"
	}

	err = newFile.Create()
	if err != nil {
		if err := fs.Trigger(ctx, "AfterValidateFailed", file); err != nil {
			logrus.Debug("AfterValidateFailed Hook execution failed,%s", err)
		}
		return nil, ErrFileExisted.WithError(err)
	}

	fs.User.Storage += newFile.Size
	return &newFile, err
}

func (fs *FileSystem) ResetFileIfNotExist(ctx context.Context, path string) error {
	if len(fs.FileTarget) == 0 {
		exist, file := fs.IsFileExist(path)
		if !exist {
			return ErrObjectNotExist
		}
		fs.FileTarget = []models.File{*file}
	}
	return fs.resetPolicyToFirstFile(ctx)
}

func (fs *FileSystem) Preview(ctx context.Context, id uint, isText bool) (*response.ContentResponse, error) {
	err := fs.resetFileIDIfNotExist(ctx, id)
	if err != nil {
		return nil, err
	}
	sizeLimit := models.GetIntSetting("maxEditSize", 2<<20)
	if isText && fs.FileTarget[0].Size > uint64(sizeLimit) {
		return nil, ErrFileSizeTooBig
	}

	if isText || fs.Policy.IsDirectlyPreview() {
		resp, err := fs.GetDownloadContent(ctx, id)
		if err != nil {
			return nil, err
		}
		return &response.ContentResponse{
			Redirect: false,
			Content:  resp,
		}, nil
	}

	ttl := models.GetIntSetting("preview_timeout", 60)
	previewURL, err := fs.SignURL(ctx, &fs.FileTarget[0], int64(ttl), false)
	if err != nil {
		return nil, err
	}
	return &response.ContentResponse{
		Redirect: true,
		URL:      previewURL,
		MaxAge:   ttl,
	}, nil
}
