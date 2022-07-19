package serializer

import (
	"encoding/gob"
	"github.com/jylc/cloudserver/models"
	"time"
)

type UploadSession struct {
	Key            string
	UID            uint
	VirtualPath    string
	Name           string
	Size           uint64
	SavePath       string
	LastModified   *time.Time
	Policy         models.Policy
	Callback       string
	CallbackSecret string
	UploadURL      string
	UploadID       string
	Credential     string
}

type UploadCredential struct {
	SessionID   string   `json:"sessionID"`
	ChunkSize   uint64   `json:"chunkSize"`
	Expires     int64    `json:"expires"`
	UploadURLs  []string `json:"uploadURLs,omitempty"`
	Credential  string   `json:"credential,omitempty"`
	UploadID    string   `json:"uploadID,omitempty"`
	Callback    string   `json:"callback,omitempty"`
	Path        string   `json:"path,omitempty"`
	AccessKey   string   `json:"ak,omitempty"`
	KeyTime     string   `json:"keyTime,omitempty"`
	Policy      string   `json:"policy,omitempty"`
	CompleteURL string   `json:"completeURL,omitempty"`
}

type UploadCallback struct {
	PicInfo string `json:"pic_info"`
}

type GeneralUploadCallbackFailed struct {
	Error string `json:"error"`
}

func init() {
	gob.Register(UploadSession{})
}
