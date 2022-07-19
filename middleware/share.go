package middleware

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/jylc/cloudserver/models"
	"github.com/jylc/cloudserver/pkg/serializer"
	"github.com/jylc/cloudserver/pkg/utils"
)

func ShareAvailable() gin.HandlerFunc {
	return func(c *gin.Context) {
		var user *models.User
		if userCtx, ok := c.Get("user"); ok {
			user = userCtx.(*models.User)
		} else {
			user = models.NewAnonymousUser()
		}

		share := models.GetShareByHashID(c.Param("id"))
		if share == nil || !share.IsAvailable() {
			c.JSON(200, serializer.Err(serializer.CodeNotFound, "share does not exist or has expired", nil))
			c.Abort()
			return
		}

		c.Set("user", user)
		c.Set("share", share)
		c.Next()
	}
}

func CheckShareUnlocked() gin.HandlerFunc {
	return func(c *gin.Context) {
		if shareCtx, ok := c.Get("share"); ok {
			share := shareCtx.(*models.Share)
			if share.Password != "" {
				sessionKey := fmt.Sprintf("share_unlock_%d", share.ID)
				unlocked := utils.GetSession(c, sessionKey) != nil
				if !unlocked {
					c.JSON(200, serializer.Err(serializer.CodeNoPermissionErr, "no access to this share", nil))
					c.Abort()
					return
				}
			}
			c.Next()
			return
		}
		c.Abort()
	}
}

func BeforeShareDownload() gin.HandlerFunc {
	return func(c *gin.Context) {
		if shareCtx, ok := c.Get("share"); ok {
			if userCtx, ok := c.Get("user"); ok {
				share := shareCtx.(*models.Share)
				user := userCtx.(*models.User)

				err := share.CanBeDownloadBy(user)
				if err != nil {
					c.JSON(200, serializer.Err(serializer.CodeNoPermissionErr, err.Error(), nil))
					c.Abort()
					return
				}
				err = share.DownloadBy(user, c)
				if err != nil {
					c.JSON(200, serializer.Err(serializer.CodeNoPermissionErr, err.Error(), nil))
					c.Abort()
					return
				}
				c.Next()
				return
			}
		}
		c.Abort()
	}
}

func ShareCanPreview() gin.HandlerFunc {
	return func(c *gin.Context) {
		if share, ok := c.Get("share"); ok {
			if share.(*models.Share).PreviewEnabled {
				c.Next()
				return
			}
			c.JSON(200, serializer.Err(serializer.CodeNoPermissionErr, "this share cannot be previewed", nil))
			c.Abort()
			return
		}
		c.Abort()
	}
}

func ShareOwner() gin.HandlerFunc {
	return func(c *gin.Context) {
		var user *models.User
		if userCtx, ok := c.Get("user"); ok {
			user = userCtx.(*models.User)
		} else {
			c.JSON(200, serializer.Err(serializer.CodeCheckLogin, "Please login first", nil))
			c.Abort()
			return
		}

		if share, ok := c.Get("share"); ok {
			if share.(*models.Share).Creator().ID != user.ID {
				c.JSON(200, serializer.Err(serializer.CodeNotFound, "Sharing does not exist", nil))
				c.Abort()
				return
			}
		}
		c.Next()
	}
}
