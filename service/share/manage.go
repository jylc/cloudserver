package share

import (
	"github.com/gin-gonic/gin"
	"github.com/jylc/cloudserver/models"
	"github.com/jylc/cloudserver/pkg/hashid"
	"github.com/jylc/cloudserver/pkg/serializer"
	"net/url"
	"time"
)

type ShareCreateService struct {
	SourceID        string `json:"id" binding:"required"`
	IsDir           bool   `json:"is_dir"`
	Password        string `json:"password" binding:"max=255"`
	RemainDownloads int    `json:"downloads"`
	Expire          int    `json:"expire"`
	Preview         bool   `json:"preview"`
}

type ShareUpdateService struct {
	Prop  string `json:"prop"  binding:"required,eq=password|eq=preview_enabled"`
	Value string `json:"value" binding:"max=255"`
}

func (service *Service) Delete(c *gin.Context, user *models.User) serializer.Response {
	share := models.GetShareByHashID(c.Param("id"))
	if share == nil || share.Creator().ID != user.ID {
		return serializer.Err(serializer.CodeNotFound, "Sharing does not exist", nil)
	}
	if err := share.Delete(); err != nil {
		return serializer.Err(serializer.CodeDBError, "Share deletion failed", nil)
	}
	return serializer.Response{}
}

func (service *ShareUpdateService) Update(c *gin.Context) serializer.Response {
	shareCtx, _ := c.Get("share")
	share := shareCtx.(*models.Share)

	switch service.Prop {
	case "password":
		err := share.Update(map[string]interface{}{"password": service.Value})
		if err != nil {
			return serializer.Err(serializer.CodeDBError, "Unable to update sharing password", err)
		}
	case "preview_enabled":
		value := service.Value == "true"
		err := share.Update(map[string]interface{}{"preview_enabled": value})
		if err != nil {
			return serializer.Err(serializer.CodeDBError, "Unable to update sharing properties", err)
		}
		return serializer.Response{
			Data: value,
		}
	}
	return serializer.Response{
		Data: service.Value,
	}
}

func (service *ShareCreateService) Create(c *gin.Context) serializer.Response {
	userCtx, _ := c.Get("user")
	user := userCtx.(*models.User)

	if !user.Group.ShareEnabled {
		return serializer.Err(serializer.CodeNoPermissionErr, "You are not authorized to create sharing links", nil)
	}

	var (
		sourceID   uint
		sourceName string
		err        error
	)
	if service.IsDir {
		sourceID, err = hashid.DecodeHashID(service.SourceID, hashid.FolderID)
	} else {
		sourceID, err = hashid.DecodeHashID(service.SourceID, hashid.FileID)
	}
	if err != nil {
		return serializer.Err(serializer.CodeNotFound, "Original resource does not exist", nil)
	}

	exist := true
	if service.IsDir {
		folder, err := models.GetFoldersByIDs([]uint{sourceID}, user.ID)
		if err != nil || len(folder) == 0 {
			exist = false
		} else {
			sourceName = folder[0].Name
		}
	} else {
		file, err := models.GetFilesByIDs([]uint{sourceID}, user.ID)
		if err != nil || len(file) == 0 {
			exist = false
		} else {
			sourceName = file[0].Name
		}
	}
	if !exist {
		return serializer.Err(serializer.CodeNotFound, "Original resource does not exist", nil)
	}

	newShare := models.Share{
		Password:        service.Password,
		IsDir:           service.IsDir,
		UserID:          user.ID,
		SourceID:        sourceID,
		RemainDownloads: -1,
		PreviewEnabled:  service.Preview,
		SourceName:      sourceName,
	}
	if service.RemainDownloads > 0 {
		expires := time.Now().Add(time.Duration(service.Expire) * time.Second)
		newShare.RemainDownloads = service.RemainDownloads
		newShare.Expires = &expires
	}
	id, err := newShare.Create()
	if err != nil {
		return serializer.Err(serializer.CodeDBError, "Sharing link creation failed", err)
	}
	uid := hashid.HashID(id, hashid.ShareID)
	siteURL := models.GetSiteURL()
	sharePath, _ := url.Parse("/s/" + uid)
	shareURL := siteURL.ResolveReference(sharePath)

	return serializer.Response{
		Code: 0,
		Data: shareURL.String(),
	}
}
