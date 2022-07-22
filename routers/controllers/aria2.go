package controllers

import (
	"context"
	"github.com/gin-gonic/gin"
	"github.com/jylc/cloudserver/pkg/aria2/common"
	"github.com/jylc/cloudserver/service/aria2"
	"github.com/jylc/cloudserver/service/explorer"
)

func AddAria2URL(c *gin.Context) {
	var addService aria2.BatchAddURLService
	if err := c.ShouldBindJSON(&addService); err == nil {
		res := addService.Add(c, common.URLTask)
		c.JSON(200, res)
	} else {
		c.JSON(200, ErrorResponse(err))
	}
}

func SelectAria2File(c *gin.Context) {
	var selectService aria2.SelectFileService
	if err := c.ShouldBindJSON(&selectService); err == nil {
		res := selectService.Select(c)
		c.JSON(200, res)
	} else {
		c.JSON(200, ErrorResponse(err))
	}
}

func AddAria2Torrent(c *gin.Context) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var service explorer.FileIDService
	if err := c.ShouldBindUri(&service); err == nil {
		res := service.CreateDownloadSession(ctx, c)
		if res.Code != 0 {
			c.JSON(200, res)
			return
		}
		var addService aria2.AddURLService
		addService.URL = res.Data.(string)

		if err := c.ShouldBindJSON(&addService); err == nil {
			addService.URL = res.Data.(string)
			res := addService.Add(c, nil, common.URLTask)
			c.JSON(200, res)
		} else {
			c.JSON(200, ErrorResponse(err))
		}
	} else {
		c.JSON(200, ErrorResponse(err))
	}
}

func CancelAria2Download(c *gin.Context) {
	var selectService aria2.DownloadTaskService
	if err := c.ShouldBindUri(&selectService); err == nil {
		res := selectService.Delete(c)
		c.JSON(200, res)
	} else {
		c.JSON(200, ErrorResponse(err))
	}
}

func ListDownloading(c *gin.Context) {
	var service aria2.DownloadListService
	if err := c.ShouldBindQuery(&service); err == nil {
		res := service.Downloading(c, CurrentUser(c))
		c.JSON(200, res)
	} else {
		c.JSON(200, ErrorResponse(err))
	}
}

func ListFinished(c *gin.Context) {
	var service aria2.DownloadListService
	if err := c.ShouldBindQuery(&service); err == nil {
		res := service.Finished(c, CurrentUser(c))
		c.JSON(200, res)
	} else {
		c.JSON(200, ErrorResponse(err))
	}
}
