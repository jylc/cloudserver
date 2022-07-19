package controllers

import (
	"context"
	"github.com/gin-gonic/gin"
	"github.com/jylc/cloudserver/service/explorer"
)

func Delete(c *gin.Context) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var service explorer.ItemIDService
	if err := c.ShouldBindJSON(&service); err == nil {
		res := service.Delete(ctx, c)
		c.JSON(200, res)
	} else {
		c.JSON(200, ErrorResponse(err))
	}
}

func Move(c *gin.Context) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var service explorer.ItemMoveService
	if err := c.ShouldBindJSON(&service); err == nil {
		res := service.Move(ctx, c)
		c.JSON(200, res)
	} else {
		c.JSON(200, ErrorResponse(err))
	}
}

func Copy(c *gin.Context) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var service explorer.ItemMoveService
	if err := c.ShouldBindJSON(&service); err == nil {
		res := service.Copy(ctx, c)
		c.JSON(200, res)
	} else {
		c.JSON(200, ErrorResponse(err))
	}
}

func Rename(c *gin.Context) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var service explorer.ItemRenameService
	if err := c.ShouldBindJSON(&service); err == nil {
		res := service.Rename(ctx, c)
		c.JSON(200, res)
	} else {
		c.JSON(200, ErrorResponse(err))
	}
}

func GetProperty(c *gin.Context) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var service explorer.ItemPropertyService
	service.ID = c.Param("id")
	if err := c.ShouldBindQuery(&service); err == nil {
		res := service.GetProperty(ctx, c)
		c.JSON(200, res)
	} else {
		c.JSON(200, ErrorResponse(err))
	}
}
