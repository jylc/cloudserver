package explorer

import (
	"context"
	"github.com/gin-gonic/gin"
	"github.com/jylc/cloudserver/pkg/cache"
	"github.com/jylc/cloudserver/pkg/filesystem"
	"github.com/jylc/cloudserver/pkg/serializer"
	"github.com/jylc/cloudserver/pkg/utils"
)

type SlaveCreateUploadSessionService struct {
	Session   serializer.UploadSession `json:"session" binding:"required"`
	TTL       int64                    `json:"ttl"`
	Overwrite bool                     `json:"overwrite"`
}

func (service *SlaveCreateUploadSessionService) Create(ctx context.Context, c *gin.Context) serializer.Response {
	if !service.Overwrite && utils.Exist(service.Session.SavePath) {
		return serializer.Err(serializer.CodeConflict, "placeholder file already exist", nil)
	}

	err := cache.Set(
		filesystem.UploadSessionCachePrefix+service.Session.Key,
		service.Session,
		int(service.TTL),
	)

	if err != nil {
		return serializer.Err(serializer.CodeCacheOperation, "Failed to create upload session in slave node", err)
	}
	return serializer.Response{}
}
