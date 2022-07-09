package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/jylc/cloudserver/models"
	"github.com/jylc/cloudserver/pkg/serializer"
)

func CacheControl() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Cache-Control", "private, no-cache")
	}
}

func IsFunctionEnabled(key string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !models.IsTrueVal(models.GetSettingByName(key)) {
			c.JSON(200, serializer.Err(serializer.CodeFeatureNotEnabled, "This feature is not enabled", nil))
			c.Abort()
			return
		}
		c.Next()
	}
}
