package response

import (
	"io"
	"time"
)

type ContentResponse struct {
	Redirect bool
	Content  RSCloser
	URL      string
	MaxAge   int
}

type RSCloser interface {
	io.ReadSeeker
	io.Closer
}

type Object struct {
	Name         string    `json:"name"`
	RelativePath string    `json:"relative_path"`
	Source       string    `json:"source"`
	Size         uint64    `json:"size"`
	IsDir        bool      `json:"is_dir"`
	LastModify   time.Time `json:"last_modify"`
}
