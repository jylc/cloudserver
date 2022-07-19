package controllers

import "github.com/gin-gonic/gin"

func CreateDirectory(c *gin.Context) {
	var service explorer.DirectoryService
	if err := c.ShouldBindJSON(&service); err == nil {
		res := service.CreateDirectory(c)
		c.JSON(200, res)
	} else {
		c.JSON(200, ErrorResponse(err))
	}
}

func ListDirectory(c *gin.Context) {
	var service explorer.DirectoryService
	if err := c.ShouldBindUri(&service); err == nil {
		res := service.ListDirectory(c)
		c.JSON(200, res)
	} else {
		c.JSON(200, ErrorResponse(err))
	}
}
