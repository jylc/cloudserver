package middleware

import (
	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/memcached"
	"github.com/gin-contrib/sessions/redis"
	"github.com/gin-gonic/gin"
	"github.com/jylc/cloudserver/pkg/conf"
	"github.com/jylc/cloudserver/pkg/utils"
	"github.com/sirupsen/logrus"
)

var Store memcached.Store

func Session(secret string) gin.HandlerFunc {
	var err error

	Store, err = redis.NewStoreWithDB(10, "tcp", conf.Rc.Server, conf.Rc.Password, conf.Rc.Db, []byte(secret))
	if err != nil {
		logrus.Panicf("cannot connect redis[%s],%s\n", conf.Rc.Server, err)
	}
	return sessions.Sessions("cloudreve-session", Store)
}

func CSRFInit() gin.HandlerFunc {
	return func(c *gin.Context) {
		utils.SetSession(c, map[string]interface{}{"CSRF": true})
		c.Next()
	}
}
