package filesystem

import (
	"context"
	"github.com/jylc/cloudserver/models"
	"github.com/jylc/cloudserver/pkg/cache"
	"github.com/jylc/cloudserver/pkg/cluster"
	"github.com/jylc/cloudserver/pkg/filesystem/driver/local"
	"github.com/jylc/cloudserver/pkg/filesystem/fsctx"
	"github.com/jylc/cloudserver/pkg/serializer"
	"github.com/sirupsen/logrus"
	"io/ioutil"
	"strings"
)

type Hook func(ctx context.Context, fs *FileSystem, file fsctx.FileHeader) error

func (fs *FileSystem) Use(name string, hook Hook) {
	if fs.Hooks == nil {
		fs.Hooks = make(map[string][]Hook)
	}

	if _, ok := fs.Hooks[name]; ok {
		fs.Hooks[name] = append(fs.Hooks[name], hook)
		return
	}
	fs.Hooks[name] = []Hook{hook}
}

func (fs *FileSystem) CleanHooks(name string) {
	if name == "" {
		fs.Hooks = nil
	} else {
		delete(fs.Hooks, name)
	}
}

func HookTruncateFileTo(size uint64) Hook {
	return func(ctx context.Context, fs *FileSystem, fileHeader fsctx.FileHeader) error {
		if handler, ok := fs.Handler.(local.Driver); ok {
			return handler.Truncate(ctx, fileHeader.Info().SavePath, size)
		}
		return nil
	}
}

func HookChunkUploaded(ctx context.Context, fs *FileSystem, fileHeader fsctx.FileHeader) error {
	fileInfo := fileHeader.Info()
	return fileInfo.Model.(*models.File).UpdateSize(fileInfo.AppendStart + fileInfo.Size)
}

func HookValidateCapacity(ctx context.Context, fs *FileSystem, file fsctx.FileHeader) error {
	if fs.User.GetRemainingCapacity() < file.Info().Size {
		return ErrInsufficientCapacity
	}
	return nil
}

func HookChunkUpload(ctx context.Context, fs *FileSystem, fileHeader fsctx.FileHeader) error {
	fileInfo := fileHeader.Info()
	return fileInfo.Model.(*models.File).UpdateSize(fileInfo.AppendStart + fileInfo.Size)
}

func HookChunkUploadFailed(ctx context.Context, fs *FileSystem, fileHeader fsctx.FileHeader) error {
	fileInfo := fileHeader.Info()
	return fileInfo.Model.(*models.File).UpdateSize(fileInfo.AppendStart)
}

func HookPopPlaceholderToFile(picInfo string) Hook {
	return func(ctx context.Context, fs *FileSystem, fileHeader fsctx.FileHeader) error {
		fileInfo := fileHeader.Info()
		fileModel := fileInfo.Model.(*models.File)
		if picInfo == "" && fs.Policy.IsThumbExist(fileInfo.FileName) {
			picInfo = "1.1"
		}
		return fileModel.PopChunkToFile(fileInfo.LastModified, picInfo)
	}
}
func HookGenerateThumb(ctx context.Context, fs *FileSystem, fileHeader fsctx.FileHeader) error {
	fileMode := fileHeader.Info().Model.(*models.File)
	if fs.Policy.IsThumbGenerateNeeded() {
		fs.recycleLock.Lock()
		go func() {
			defer fs.recycleLock.Unlock()
			_, _ = fs.Handler.Delete(ctx, []string{fileMode.SourceName + models.GetSettingByNameWithDefault("thumb_file_suffix", "._thumb")})
			fs.GenerateThumbnail(ctx, fileMode)
		}()
	}
	return nil
}

func HookDeleteUploadSession(id string) Hook {
	return func(ctx context.Context, fs *FileSystem, file fsctx.FileHeader) error {
		cache.Deletes([]string{id}, UploadSessionCachePrefix)
		return nil
	}
}

func SlaveAfterUpload(session *serializer.UploadSession) Hook {
	return func(ctx context.Context, fs *FileSystem, fileHeader fsctx.FileHeader) error {
		fileInfo := fileHeader.Info()

		file := models.File{
			Name:       fileInfo.FileName,
			SourceName: fileInfo.SavePath,
		}

		fs.GenerateThumbnail(ctx, &file)

		if session.Callback == "" {
			return nil
		}

		callbackBody := serializer.UploadCallback{
			PicInfo: file.PicInfo,
		}
		return cluster.RemoteCallback(session.Callback, callbackBody)
	}
}

func (fs *FileSystem) Trigger(ctx context.Context, name string, file fsctx.FileHeader) error {
	if hooks, ok := fs.Hooks[name]; ok {
		for _, hook := range hooks {
			err := hook(ctx, fs, file)
			if err != nil {
				logrus.Warningf("Hook execution failed, %s", err)
				return err
			}
		}
	}
	return nil
}

func HookValidateFile(ctx context.Context, fs *FileSystem, file fsctx.FileHeader) error {
	fileInfo := file.Info()

	if !fs.ValidateFileSize(ctx, fileInfo.Size) {
		return ErrFileSizeTooBig
	}

	if !fs.ValidateLegalName(ctx, fileInfo.FileName) {
		return ErrIllegalObjectName
	}

	if !fs.ValidateExtension(ctx, fileInfo.FileName) {
		return ErrFileExtensionNotAllowed
	}
	return nil
}

func HookUpdateSourceName(ctx context.Context, fs *FileSystem, file fsctx.FileHeader) error {
	originFile, ok := ctx.Value(fsctx.FileModelCtx).(models.File)
	if !ok {
		return ErrObjectNotExist
	}
	return originFile.UpdateSourceName(originFile.SourceName)
}

func HookResetPolicy(ctx context.Context, fs *FileSystem, file fsctx.FileHeader) error {
	originFile, ok := ctx.Value(fsctx.FileModelCtx).(models.File)
	if !ok {
		return ErrObjectNotExist
	}
	fs.Policy = originFile.GetPolicy()
	return fs.DispatchHandler()
}

func HookValidateCapacityDiff(ctx context.Context, fs *FileSystem, newFile fsctx.FileHeader) error {
	originFile := ctx.Value(fsctx.FileModelCtx).(models.File)
	newFileSize := newFile.Info().Size
	if newFileSize > originFile.Size {
		return HookValidateCapacity(ctx, fs, newFile)
	}
	return nil
}

func HookCleanFileContent(ctx context.Context, fs *FileSystem, file fsctx.FileHeader) error {
	return fs.Handler.Put(ctx, &fsctx.FileStream{
		File:     ioutil.NopCloser(strings.NewReader("")),
		SavePath: file.Info().SavePath,
		Size:     0,
		Mode:     fsctx.Overwrite,
	})
}

func HookClearFileSize(ctx context.Context, fs *FileSystem, file fsctx.FileHeader) error {
	originFile, ok := ctx.Value(fsctx.FileModelCtx).(models.File)
	if !ok {
		return ErrObjectNotExist
	}
	return originFile.UpdateSize(0)
}

func HookCancelContext(ctx context.Context, fs *FileSystem, file fsctx.FileHeader) error {
	cancelFunc, ok := ctx.Value(fsctx.FileModelCtx).(context.CancelFunc)
	if ok {
		cancelFunc()
	}
	return nil
}

func GenericAfterUpdate(ctx context.Context, fs *FileSystem, newFile fsctx.FileHeader) error {
	originFile, ok := ctx.Value(fsctx.FileModelCtx).(models.File)
	if !ok {
		return ErrObjectNotExist
	}

	newFile.SetModel(&originFile)

	err := originFile.UpdateSize(newFile.Info().Size)
	if err != nil {
		return err
	}
	return nil
}

func HookDeleteTempFile(ctx context.Context, fs *FileSystem, file fsctx.FileHeader) error {
	_, err := fs.Handler.Delete(ctx, []string{file.Info().SavePath})
	if err != nil {
		logrus.Warningf("Unable to clean up uploaded temporary files, %s", err)
	}
	return nil
}

func GenericAfterUpload(ctx context.Context, fs *FileSystem, fileHeader fsctx.FileHeader) error {
	fileInfo := fileHeader.Info()
	folder, err := fs.CreateDirectory(ctx, fileInfo.VirtualPath)
	if err != nil {
		return err
	}
	if ok, file := fs.IsChildFileExist(folder, fileInfo.FileName); ok {
		if file.UploadSessionID != nil {
			return ErrFileUploadSessionExisted
		}
		return ErrFileExisted
	}

	file, err := fs.AddFile(ctx, folder, fileHeader)
	if err != nil {
		return ErrInsertFileRecord
	}
	fileHeader.SetModel(file)
	return nil
}
