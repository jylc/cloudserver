package chunk

import "C"
import (
	"context"
	"fmt"
	"github.com/jylc/cloudserver/pkg/filesystem/chunk/backoff"
	"github.com/jylc/cloudserver/pkg/filesystem/fsctx"
	"github.com/sirupsen/logrus"
	"io"
	"os"
)

const bufferTempPattern = "cdChunk.*.tmp"

type ProcessFunc func(c *Group, chunk io.Reader) error

type Group struct {
	file              fsctx.FileHeader
	chunkSize         uint64
	backoff           backoff.Backoff
	enableRetryBuffer bool

	fileInfo     *fsctx.UploadTaskInfo
	currentIndex int
	chunkNum     uint64
	bufferTemp   *os.File
}

func NewGroup(file fsctx.FileHeader, chunkSize uint64, backoff backoff.Backoff, useBuffer bool) *Group {
	c := &Group{
		file:              file,
		chunkSize:         chunkSize,
		backoff:           backoff,
		fileInfo:          file.Info(),
		currentIndex:      -1,
		enableRetryBuffer: useBuffer,
	}
	if c.chunkSize == 0 {
		c.chunkSize = c.fileInfo.Size
	}

	if c.fileInfo.Size == 0 {
		c.chunkNum = 1
	} else {
		c.chunkNum = c.fileInfo.Size / c.chunkSize
		if c.fileInfo.Size%c.chunkSize != 0 {
			c.chunkNum++
		}
	}
	return c
}
func (c *Group) TempAvailable() bool {
	if c.bufferTemp != nil {
		state, _ := c.bufferTemp.Stat()
		return state != nil && state.Size() == c.Length()
	}
	return false
}

func (c *Group) Process(processor ProcessFunc) error {
	reader := io.LimitReader(c.file, int64(c.chunkSize))
	if c.enableRetryBuffer && c.bufferTemp == nil && !c.file.Seekable() {
		c.bufferTemp, _ = os.CreateTemp("", bufferTempPattern)
		reader = io.TeeReader(reader, c.bufferTemp)
	}

	if c.bufferTemp != nil {
		defer func() {
			if c.bufferTemp != nil {
				c.bufferTemp.Close()
				os.Remove(c.bufferTemp.Name())
				c.bufferTemp = nil
			}
		}()

		if c.TempAvailable() {
			if _, err := c.bufferTemp.Seek(0, io.SeekStart); err != nil {
				return fmt.Errorf("failed to seek temp file back to chunk start: %w", err)
			}

			logrus.Debugf("Chunk %d will be read from temp file %q.", c.Index(), c.bufferTemp.Name())
			reader = c.bufferTemp
		}
	}

	err := processor(c, reader)
	if err != nil {
		if err != context.Canceled && (c.file.Seekable() || c.TempAvailable()) && c.backoff.Next() {
			if c.file.Seekable() {
				if _, seekErr := c.file.Seek(c.Start(), io.SeekStart); seekErr != nil {
					return fmt.Errorf("failed to seek back to chunk start: %w, last error: %s", seekErr, err)
				}
			}
			logrus.Debugf("Retrying chunk %d, last err: %s", c.currentIndex, err)
			return c.Process(processor)
		}
		return err
	}

	logrus.Debugf("Chunk %d processed", c.currentIndex)
	return nil
}

func (c *Group) Start() int64 {
	return int64(uint64(c.Index()) * c.chunkSize)
}

func (c *Group) Total() int64 {
	return int64(c.fileInfo.Size)
}

func (c *Group) Num() int {
	return int(c.chunkNum)
}

func (c *Group) RangeHeader() string {
	return fmt.Sprintf("bytes %d-%d/%d", c.Start(), c.Start()+c.Length()-1, c.Total())
}

func (c *Group) Index() int {
	return c.currentIndex
}

func (c *Group) Next() bool {
	c.currentIndex++
	c.backoff.Reset()
	return c.currentIndex < int(c.chunkNum)
}

func (c *Group) Length() int64 {
	contentLength := c.chunkSize
	if c.Index() == int(c.chunkNum-1) {
		contentLength = c.fileInfo.Size - c.chunkSize*(c.chunkNum-1)
	}
	return int64(contentLength)
}

func (c *Group) IsLast() bool {
	return c.Index() == int(c.chunkNum-1)
}
