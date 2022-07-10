package fsctx

import (
	"errors"
	"io"
	"time"
)

type WriteMode int

const (
	Overwrite WriteMode = 0x00001
	Append    WriteMode = 0x00002
	Nop       WriteMode = 0x00004
)

type UploadTaskInfo struct {
	Size            uint64
	MIMEType        string
	FileName        string
	VirtualPath     string
	Mode            WriteMode
	Metadata        map[string]string
	LastModified    *time.Time
	SavePath        string
	UploadSessionID *string
	AppendStart     uint64
	Model           interface{}
	Src             string
}

type FileHeader interface {
	io.Reader
	io.Closer
	io.Seeker
	Info() *UploadTaskInfo
	SetSize(uint642 uint64)
	SetMode(fileModel interface{})
	Seekable() bool
}

type FileStream struct {
	Mode            WriteMode
	LastModified    *time.Time
	Metadata        map[string]string
	File            io.ReadCloser
	Seeker          io.Seeker
	Size            uint64
	VirtualPath     string
	Name            string
	MIMEType        string
	SavePath        string
	UploadSessionID *string
	AppendStart     uint64
	Model           interface{}
	Src             string
}

func (file *FileStream) Read(p []byte) (n int, err error) {
	if file.File != nil {
		return file.File.Read(p)
	}
	return 0, io.EOF
}

func (file *FileStream) Seek(offset int64, whence int) (int64, error) {
	if file.Seekable() {
		return file.Seeker.Seek(offset, whence)
	}
	return 0, errors.New("no seeker")
}

func (file *FileStream) Seekable() bool {
	return file.Seeker != nil
}

func (file *FileStream) Info() *UploadTaskInfo {
	return &UploadTaskInfo{
		Size:            file.Size,
		MIMEType:        file.MIMEType,
		FileName:        file.Name,
		VirtualPath:     file.VirtualPath,
		Mode:            file.Mode,
		Metadata:        file.Metadata,
		LastModified:    file.LastModified,
		SavePath:        file.SavePath,
		UploadSessionID: file.UploadSessionID,
		AppendStart:     file.AppendStart,
		Model:           file.Model,
		Src:             file.Src,
	}
}

func (file *FileStream) SetSize(size uint64) {
	file.Size = size
}
func (file *FileStream) SetModel(fileModel interface{}) {
	file.Model = fileModel
}