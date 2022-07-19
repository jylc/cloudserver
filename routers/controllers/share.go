package controllers

import (
	"context"
	"github.com/gin-gonic/gin"
	"github.com/jylc/cloudserver/models"
	"github.com/jylc/cloudserver/pkg/serializer"
	"github.com/jylc/cloudserver/pkg/utils"
	"github.com/jylc/cloudserver/service/share"
	"path"
	"strings"
)

func GetUserShare(c *gin.Context) {
	var service share.UserGetService
	if err := c.ShouldBindQuery(&service); err == nil {
		res := service.Get(c)
		c.JSON(200, res)
	} else {
		c.JSON(200, ErrorResponse(err))
	}
}

func GetShare(c *gin.Context) {
	var service share.ShareGetService
	if err := c.ShouldBindQuery(&service); err == nil {
		res := service.Get(c)
		c.JSON(200, res)
	} else {
		c.JSON(200, ErrorResponse(err))
	}
}

func GetShareDownload(c *gin.Context) {
	var service share.Service
	if err := c.ShouldBindQuery(&service); err == nil {
		res := service.CreateDownloadSession(c)
		c.JSON(200, res)
	} else {
		c.JSON(200, ErrorResponse(err))
	}
}
func PreviewShare(c *gin.Context) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var service share.Service
	if err := c.ShouldBindQuery(&service); err == nil {
		res := service.PreviewContent(ctx, c, false)
		if res.Code == -301 {
			c.Redirect(302, res.Data.(string))
			return
		}

		if res.Code != 0 {
			c.JSON(200, res)
		}
	} else {
		c.JSON(200, ErrorResponse(err))
	}
}

func GetShareDocPreview(c *gin.Context) {
	var service share.Service
	if err := c.ShouldBindQuery(&service); err == nil {
		res := service.CreateDocPreviewSession(c)
		c.JSON(200, res)
	} else {
		c.JSON(200, ErrorResponse(err))
	}
}

func PreviewShareText(c *gin.Context) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var service share.Service
	if err := c.ShouldBindQuery(&service); err == nil {
		res := service.PreviewContent(ctx, c, nil)
		if res.Code != 0 {
			c.JSON(200, res)
		}
	} else {
		c.JSON(200, ErrorResponse(err))
	}
}

func ListSharedFolder(c *gin.Context) {
	var service share.Service
	if err := c.ShouldBindUri(&service); err == nil {
		res := service.List(c)
		c.JSON(200, res)
	} else {
		c.JSON(200, ErrorResponse(err))
	}
}

func SearchSharedFolder(c *gin.Context) {
	var service share.SearchService
	if err := c.ShouldBindUri(&service); err != nil {
		c.JSON(200, ErrorResponse(err))
		return
	}

	if err := c.ShouldBindQuery(&service); err != nil {
		c.JSON(200, ErrorResponse(err))
		return
	}

	res := service.Search(c)
	c.JSON(200, res)
}

func ArchiveShare(c *gin.Context) {
	var service share.ArchiveService
	if err := c.ShouldBindJSON(&service); err == nil {
		res := service.Archive(c)
		c.JSON(200, res)
	} else {
		c.JSON(200, ErrorResponse(err))
	}
}

func PreviewShareReadme(c *gin.Context) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var service share.Service
	if err := c.ShouldBindQuery(&service); err == nil {
		allowFileName := []string{"readme.txt", "readme.md"}
		fileName := strings.ToLower(path.Base(service.Path))
		if !utils.ContainsString(allowFileName, fileName) {
			c.JSON(200, serializer.ParamErr("not readme file", nil))
		}

		if shareCtx, ok := c.Get("share"); ok {
			if !shareCtx.(*models.Share).IsDir {
				c.JSON(200, serializer.ParamErr("There is no readme for this share", nil))
			}
		}

		res := service.PreviewContent(ctx, c, true)
		if res.Code != 0 {
			c.JSON(200, res)
		}
	} else {
		c.JSON(200, ErrorResponse(err))
	}
}

func ShareThumb(c *gin.Context) {
	var service share.Service
	if err := c.ShouldBindQuery(&service); err == nil {
		res := service.Thumb(c)
		if res.Code >= 0 {
			c.JSON(200, res)
		}
	} else {
		c.JSON(200, ErrorResponse(err))
	}
}

func SearchShare(c *gin.Context) {
	var service share.ShareListService
	if err := c.ShouldBindQuery(&service); err == nil {
		res := service.Search(c)
		c.JSON(200, res)
	} else {
		c.JSON(200, ErrorResponse(err))
	}
}

func CreateShare(c *gin.Context) {
	var service share.ShareCreateService
	if err := c.ShouldBindJSON(&service); err == nil {
		res := service.Create(c)
		c.JSON(200, res)
	} else {
		c.JSON(200, ErrorResponse(err))
	}
}

func ListShare(c *gin.Context) {
	var service share.ShareListService
	if err := c.ShouldBindQuery(&service); err == nil {
		res := service.List(c, CurrentUser(c))
		c.JSON(200, res)
	} else {
		c.JSON(200, ErrorResponse(err))
	}
}

func UpdateShare(c *gin.Context) {
	var service share.ShareUpdateService
	if err := c.ShouldBindJSON(&service); err == nil {
		res := service.Update(c)
		c.JSON(200, res)
	} else {
		c.JSON(200, ErrorResponse(err))
	}
}
func DeleteShare(c *gin.Context) {
	var service share.Service
	if err := c.ShouldBindUri(&service); err == nil {
		res := service.Delete(c, CurrentUser(c))
		c.JSON(200, res)
	} else {
		c.JSON(200, ErrorResponse(err))
	}
}
