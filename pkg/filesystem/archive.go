package filesystem

import (
	"archive/zip"
	"context"
	"github.com/gin-gonic/gin"
	"github.com/jylc/cloudserver/models"
	"github.com/jylc/cloudserver/pkg/filesystem/fsctx"
	"github.com/sirupsen/logrus"
	"io"
	"path"
	"path/filepath"
)

func (fs *FileSystem) Compress(ctx context.Context, writer io.Writer, folderIDs, fileIDs []uint, isArchive bool) error {
	folders, err := models.GetFolderByIDs(folderIDs, fs.User.ID)
	if err != nil && len(folders) != 0 {
		return ErrDBListObjects
	}

	files, err := models.GetFilesByIDs(fileIDs, fs.User.ID)
	if err != nil && len(fileIDs) != 0 {
		return ErrDBListObjects
	}

	if parent, ok := ctx.Value(fsctx.LimitParentCtx).(*models.Folder); ok {
		for _, folder := range folders {
			if *folder.ParentID != parent.ID {
				return ErrObjectNotExist
			}
		}

		for _, file := range files {
			if file.FolderID != parent.ID {
				return ErrObjectNotExist
			}
		}
	}

	reqContext := ctx
	ginCtx, ok := ctx.Value(fsctx.GinCtx).(*gin.Context)
	if ok {
		reqContext = ginCtx.Request.Context()
	}

	for i := 0; i < len(folders); i++ {
		folders[i].Position = ""
	}

	for i := 0; i < len(files); i++ {
		files[i].Position = ""
	}

	zipWriter := zip.NewWriter(writer)
	defer zipWriter.Close()

	ctx = reqContext
	for i := 0; i < len(folders); i++ {
		select {
		case <-reqContext.Done():
			return ErrClientCanceled
		default:
			fs.doCompress(reqContext, nil, &folders[i], zipWriter, isArchive)
		}
	}

	for i := 0; i < len(files); i++ {
		select {
		case <-reqContext.Done():
			return ErrClientCanceled
		default:
			fs.doCompress(reqContext, &files[i], nil, zipWriter, isArchive)
		}
	}

	return nil
}

func (fs *FileSystem) doCompress(ctx context.Context, file *models.File, folder *models.Folder, zipWriter *zip.Writer, isArchive bool) {
	if file != nil {
		fs.Policy = file.GetPolicy()
		err := fs.DispatchHandler()
		if err != nil {
			logrus.Warningf("cannot compress file %s, %s", file.Name, err)
			return
		}

		fileToZip, err := fs.Handler.Get(
			context.WithValue(ctx, fsctx.FileModelCtx, *file),
			file.SourceName)
		if err != nil {
			logrus.Debugf("Open%s, %s", file.Name, err)
			return
		}

		if closer, ok := fileToZip.(io.Closer); ok {
			defer closer.Close()
		}

		header := &zip.FileHeader{
			Name:               filepath.FromSlash(path.Join(file.Position, file.Name)),
			Modified:           file.UpdatedAt,
			UncompressedSize64: file.Size,
		}

		if isArchive {
			header.Method = zip.Store
		} else {
			header.Method = zip.Deflate
		}

		writer, err := zipWriter.CreateHeader(header)
		if err != nil {
			return
		}

		_, err = io.Copy(writer, fileToZip)
	} else if folder != nil {
		subFiles, err := folder.GetChildFiles()
		if err == nil && len(subFiles) > 0 {
			for i := 0; i < len(subFiles); i++ {
				fs.doCompress(ctx, &subFiles[i], nil, zipWriter, isArchive)
			}
		}

		subFolders, err := folder.GetChildFolder()
		if err == nil && len(subFolders) > 0 {
			for i := 0; i < len(subFolders); i++ {
				fs.doCompress(ctx, nil, &subFolders[i], zipWriter, isArchive)
			}
		}
	}
}
