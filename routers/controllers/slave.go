package controllers

import (
	"context"
	"github.com/gin-gonic/gin"
	"github.com/jylc/cloudserver/pkg/request"
	"github.com/jylc/cloudserver/service/explorer"
	"github.com/jylc/cloudserver/service/node"
)

func SlaveNotificationPush(c *gin.Context) {
	var service node.SlaveNotificationService

	if err := c.ShouldBindUri(&service); err == nil {
		res := service.HandleSlaveNotificationPush(c)
		c.JSON(200, res)
	} else {
		c.JSON(200, ErrorResponse(err))
	}
}

func SlaveUpload(c *gin.Context) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var service explorer.UploadService
	if err := c.ShouldBindUri(&service); err == nil {
		res := service.SlaveUpload(ctx, c)
		c.JSON(200, res)
		request.BlackHole(c.Request.Body)
	} else {
		c.JSON(200, ErrorResponse(err))
	}
}

func SlaveGetUploadSession(c *gin.Context) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var service explorer.SlaveCreateUploadSessionService
	if err := c.ShouldBindJSON(&service); err == nil {
		res := service.Create(ctx, c)
		c.JSON(200, res)
	} else {
		c.JSON(200, ErrorResponse(err))
	}
}

func SlaveDeleteUploadSession(c *gin.Context) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var service explorer.UploadSessionService
	if err := c.ShouldBindUri(&service); err == nil {
		res := service.SlaveDelete(ctx, c)
		c.JSON(200, res)
	} else {
		c.JSON(200, ErrorResponse(err))
	}
}

func SlaveGetOneDriveCredential(c *gin.Context) {
	var service node.OneDriveCredentialService
	if err := c.ShouldBindUri(&service); err == nil {
		res := service.Get(c)
		c.JSON(200, res)
	} else {
		c.JSON(200, ErrorResponse(err))
	}
}
