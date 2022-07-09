package controllers

import (
	"github.com/gin-gonic/gin"
	"github.com/jylc/cloudserver/service/user"
)

func UserLogin(c *gin.Context) {
	var service user.LoginService
	if err := c.ShouldBindJSON(&service); err != nil {
		res := service.Login(c)
		c.JSON(200, res)
	} else {
		c.JSON(200, ErrorResponse(err))
	}
}

func UserRegister(c *gin.Context) {

}
