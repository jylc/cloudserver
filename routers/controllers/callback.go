package controllers

import "github.com/gin-gonic/gin"

func RemoteCallback(c *gin.Context) {
	var callbackBody callback.RemoteUploadCallbackService
	if err := c.ShouldBindJSON(&callbackBody); err == nil {
		res := callbackBody.ProcssCallback(callbackBody, c)
		c.JSON(200, res)
	} else {
		c.JSON(200, ErrorResponse(err))
	}
}
