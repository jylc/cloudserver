package controllers

import (
	"github.com/gin-gonic/gin"
	"github.com/jylc/cloudserver/service/share"
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
