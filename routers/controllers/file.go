package controllers

import (
	"context"
	"github.com/gin-gonic/gin"
	"github.com/jylc/cloudserver/service/explorer"
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
