package controllers

import (
	"context"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/jylc/cloudserver/models"
	"github.com/jylc/cloudserver/pkg/filesystem"
	"github.com/jylc/cloudserver/pkg/request"
	"github.com/jylc/cloudserver/pkg/serializer"
	"github.com/jylc/cloudserver/service/explorer"
	"net/http"
)

func AnonymousGetContent(c *gin.Context) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var service explorer.FileAnonymousGetService
	if err := c.ShouldBindUri(&service); err == nil {
		res := service.Download(ctx, c)
		if res.Code != 0 {
			c.JSON(200, res)
		}
	} else {
		c.JSON(200, ErrorResponse(err))
	}
}

func AnonymousPermLink(c *gin.Context) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var service explorer.FileAnonymousGetService
	if err := c.ShouldBindUri(&service); err == nil {
		res := service.Source(ctx, c)
		if res.Code == -302 {
			c.Redirect(302, res.Data.(string))
			return
		}

		if res.Code != 0 {
			c.JSON(200, ErrorResponse(err))
		}
	} else {
		c.JSON(200, ErrorResponse(err))
	}
}

func Download(c *gin.Context) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var service explorer.DownloadService
	if err := c.ShouldBindUri(&service); err == nil {
		res := service.Download(ctx, c)
		if res.Code != 0 {
			c.JSON(200, res)
		}
	} else {
		c.JSON(200, ErrorResponse(err))
	}
}

func DownloadArchive(c *gin.Context) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var service explorer.ArchiveService
	if err := c.ShouldBindUri(&service); err != nil {
		service.DownloadArchived(ctx, c)
	} else {
		c.JSON(200, ErrorResponse(err))
	}
}

func FileUpload(c *gin.Context) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var service explorer.UploadService
	if err := c.ShouldBindUri(&service); err != nil {
		res := service.LocalUpload(ctx, c)
		c.JSON(200, res)
		request.BlackHole(c.Request.Body)
	} else {
		c.JSON(200, ErrorResponse(err))
	}
}

func GetUploadSession(c *gin.Context) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var service explorer.CreateUploadSessionService
	if err := c.ShouldBindJSON(&service); err == nil {
		res := service.Create(ctx, c)
		c.JSON(200, res)
	} else {
		c.JSON(200, ErrorResponse(err))
	}
}

func DeleteUploadSession(c *gin.Context) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var service explorer.UploadSessionService
	if err := c.ShouldBindUri(&service); err == nil {
		res := service.Delete(ctx, c)
		c.JSON(200, res)
	} else {
		c.JSON(200, ErrorResponse(err))
	}
}

func DeleteAllUploadSession(c *gin.Context) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	res := explorer.DeleteAllUploadSession(ctx, c)
	c.JSON(200, res)
}

func PutContent(c *gin.Context) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var service explorer.FileIDService
	if err := c.ShouldBindUri(&service); err == nil {
		res := service.PutContent(ctx, c)
		c.JSON(200, res)
	} else {
		c.JSON(200, ErrorResponse(err))
	}
}

func CreateFile(c *gin.Context) {
	var service explorer.SingleFileService
	if err := c.ShouldBindJSON(&service); err == nil {
		res := service.Create(c)
		c.JSON(200, res)
	} else {
		c.JSON(200, ErrorResponse(err))
	}
}

func CreateDownloadSession(c *gin.Context) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var service explorer.FileIDService
	if err := c.ShouldBindUri(&service); err == nil {
		res := service.CreateDownloadSession(ctx, c)
		c.JSON(200, res)
	} else {
		c.JSON(200, ErrorResponse(err))
	}
}

func Preview(c *gin.Context) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var service explorer.FileIDService
	if err := c.ShouldBindUri(&service); err == nil {
		res := service.PreviewContent(ctx, c, false)
		if res.Code == -301 {
			c.Redirect(301, res.Data.(string))
			return
		}
		if res.Code != 0 {
			c.JSON(200, res)
		}
	} else {
		c.JSON(200, ErrorResponse(err))
	}
}

func PreviewText(c *gin.Context) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var service explorer.FileIDService
	if err := c.ShouldBindUri(&service); err == nil {
		res := service.PreviewContent(ctx, c, false)
		if res.Code != 0 {
			c.JSON(200, res)
		}
	} else {
		c.JSON(200, ErrorResponse(err))
	}
}
func GetDocPreview(c *gin.Context) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var service explorer.FileIDService
	if err := c.ShouldBindUri(&service); err == nil {
		res := service.CreateDocPreviewSession(ctx, c)
		c.JSON(200, res)
	} else {
		c.JSON(200, ErrorResponse(err))
	}
}

func Thumb(c *gin.Context) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	fs, err := filesystem.NewFileSystemFromContext(c)
	if err != nil {
		c.JSON(200, serializer.Err(serializer.CodePolicyNotAllowed, err.Error(), err))
		return
	}
	defer fs.Recycle()

	fileID, ok := c.Get("object_id")
	if !ok {
		c.JSON(200, serializer.ParamErr("file does not exist", err))
		return
	}
	resp, err := fs.GetThumb(ctx, fileID.(uint))
	if err != nil {
		c.JSON(200, serializer.Err(serializer.CodeNotSet, "Unable to get thumbnail", err))
		return
	}
	if resp.Redirect {
		c.Header("Cache-Control", fmt.Sprintf("max-age=%d", resp.MaxAge))
		c.Redirect(http.StatusMovedPermanently, resp.URL)
		return
	}

	defer resp.Content.Close()

	http.ServeContent(c.Writer, c.Request, "thumb."+models.GetSettingByNameWithDefault("thumb_encode_method", "jpg"), fs.FileTarget[0].UpdatedAt, resp.Content)
}

func GetSource(c *gin.Context) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var service explorer.ItemIDService
	if err := c.ShouldBindJSON(&service); err == nil {
		res := service.Sources(ctx, c)
		c.JSON(200, res)
	} else {
		c.JSON(200, ErrorResponse(err))
	}
}

func Archive(c *gin.Context) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var service explorer.ItemIDService
	if err := c.ShouldBindJSON(&service); err == nil {
		res := service.Archive(ctx, c)
		c.JSON(200, res)
	} else {
		c.JSON(200, ErrorResponse(err))
	}
}

func Compress(c *gin.Context) {
	var service explorer.ItemCompressService
	if err := c.ShouldBindJSON(&service); err == nil {
		res := service.CreateCompressTask(c)
		c.JSON(200, res)
	} else {
		c.JSON(200, ErrorResponse(err))
	}
}

func Decompress(c *gin.Context) {
	var service explorer.ItemDecompressService
	if err := c.ShouldBindJSON(&service); err == nil {
		res := service.CreateDecompressTask(c)
		c.JSON(200, res)
	} else {
		c.JSON(200, ErrorResponse(err))
	}
}

func SearchFile(c *gin.Context) {
	var service explorer.ItemSearchService
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
