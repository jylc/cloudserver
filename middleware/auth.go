package middleware

import (
	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"github.com/jylc/cloudserver/models"
)

func CurrentUser() gin.HandlerFunc {
	return func(c *gin.Context) {
		session := sessions.Default(c)
		uid := session.Get("user_id")
		if uid != nil {
			user, err := models.GetActivateUserByID(uid)
			if err == nil {
				c.Set("user", &user)
			}
		}
		c.Next()
	}
}
