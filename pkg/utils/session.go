package utils

import (
	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

func SetSession(c *gin.Context, list map[string]interface{}) {
	s := sessions.Default(c)
	for key, value := range list {
		s.Set(key, value)
	}
	err := s.Save()
	if err != nil {
		logrus.Warnf("cannot set session value, %s\n", err)
	}
}

func GetSession(c *gin.Context, key string) interface{} {
	s := sessions.Default(c)
	return s.Get(key)
}

func DeleteSession(c *gin.Context, key string) {
	s := sessions.Default(c)
	s.Delete(key)
	s.Save()
}

func ClearSession(c *gin.Context) {
	s := sessions.Default(c)
	s.Clear()
	s.Save()
}
