package filesystem

import (
	"archive/zip"
	"context"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/jylc/cloudserver/models"
	"github.com/jylc/cloudserver/pkg/filesystem/fsctx"
	"github.com/jylc/cloudserver/pkg/utils"
	"github.com/mholt/archiver/v4"
	"github.com/sirupsen/logrus"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"time"
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

func (fs *FileSystem) Decompress(ctx context.Context, src, dst, encoding string) error {
	err := fs.ResetFileIfNotExist(ctx, src)
	if err != nil {
		return err
	}
	tempZipFilePath := ""
	defer func() {
		if tempZipFilePath != "" {
			if err := os.Remove(tempZipFilePath); err != nil {
				logrus.Warningf("Unable to delete temporary compressed file %s,%s", tempZipFilePath, err)
			}
		}
	}()
	fileStream, err := fs.Handler.Get(ctx, fs.FileTarget[0].SourceName)
	if err != nil {
		return err
	}

	defer fileStream.Close()
	tempZipFilePath = filepath.Join(
		utils.RelativePath(models.GetSettingByName("temp_path")),
		"decompress",
		fmt.Sprintf("archive_%d.zip", time.Now().UnixNano()),
	)
	zipFile, err := utils.CreateNestedFile(tempZipFilePath)
	if err != nil {
		logrus.Warningf("Unable to create temporary compressed file %s,%s", tempZipFilePath, err)
	}
	defer zipFile.Close()

	format, readStream, err := archiver.Identify(fs.FileTarget[0].SourceName, fileStream)
	if err != nil {
		logrus.Warningf("Unrecognized file format %s,%s", fs.FileTarget[0].SourceName, err)
		return err
	}

	extractor, ok := format.(archiver.Extractor)
	if !ok {
		return fmt.Errorf("file not an extractor %s", fs.FileTarget[0].SourceName)
	}
	var isZip bool
	switch extractor.(type) {
	case archiver.Zip:
		extractor = archiver.Zip{TextEncoding: encoding}
		isZip = true
	}

	reader := readStream
	if isZip {
		_, err = io.Copy(zipFile, readStream)
		if err != nil {
			logrus.Warningf("Unable to write to temporary compressed file %s,%s", tempZipFilePath, err)
			return err
		}

		fileStream.Close()

		zipFile.Seek(0, io.SeekStart)
		reader = zipFile
	}

	fs.Policy = &fs.User.Policy
	err = fs.DispatchHandler()
	if err != nil {
		return err
	}

	var wg sync.WaitGroup
	parallel := models.GetIntSetting("max_parallel_transfer", 4)
	worker := make(chan int, parallel)
	for i := 0; i < parallel; i++ {
		worker <- i
	}
	uploadFunc := func(fileStream io.ReadCloser, size int64, savePath, rawPath string) {
		defer func() {
			if isZip {
				worker <- 1
				wg.Done()
			}

			if err := recover(); err != nil {
				logrus.Warningf("Error uploading files in compressed package")
				fmt.Println(err)
			}
		}()

		err := fs.UploadFromStream(ctx, &fsctx.FileStream{
			File:        fileStream,
			Size:        uint64(size),
			Name:        path.Base(savePath),
			VirtualPath: path.Dir(savePath),
		}, true)
		fileStream.Close()
		if err != nil {
			logrus.Debugf("Unable to upload the file %s in the compressed package, %s, skipping", rawPath, err)
		}
	}

	err = extractor.Extract(ctx, reader, nil, func(ctx context.Context, f archiver.File) error {
		rawPath := utils.FormSlash(f.NameInArchive)
		savePath := path.Join(dst, rawPath)

		if !strings.HasPrefix(savePath, utils.FillSlash(path.Clean(dst))) {
			logrus.Warningf("%s: illegal file path", f.NameInArchive)
			return nil
		}
		if f.FileInfo.IsDir() {
			fs.CreateDirectory(ctx, savePath)
			return nil
		}

		fileStream, err := f.Open()
		if err != nil {
			logrus.Warningf("Unable to open the file %s in the compressed package, %s, skipping", rawPath, err)
			return nil
		}

		if !isZip {
			uploadFunc(fileStream, f.FileInfo.Size(), savePath, rawPath)
		} else {
			<-worker
			wg.Add(1)
			go uploadFunc(fileStream, f.FileInfo.Size(), savePath, rawPath)
		}
		return nil
	})
	wg.Wait()
	return err
}
