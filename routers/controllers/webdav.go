package controllers

import (
	"github.com/gin-gonic/gin"
	"github.com/jylc/cloudserver/pkg/filesystem"
	"github.com/sirupsen/logrus"
	"golang.org/x/net/webdav"
	"sync"
)

var handler *webdav.Handler

func init() {
	handler = &webdav.Handler{
		Prefix:     "/dav",
		LockSystem: make(map[uint]webdav.LockSystem),
		Logger:     &sync.Mutex{},
	}
}

func GetWebDAVAccounts(c *gin.Context) {
	var service setting.WebDAVListService
	if err := c.ShouldBindUri(&service); err == nil {
		res := service.Accounts(c, CurrentUser(c))
		c.JSON(200, res)
	} else {
		c.JSON(200, ErrorResponse(err))
	}
}

func CreateWebDAVAccounts(c *gin.Context) {
	var service setting.WebDAVAccountCreateService
	if err := c.ShouldBindUri(&service); err == nil {
		res := service.Create(c, CurrentUser(c))
		c.JSON(200, res)
	} else {
		c.JSON(200, ErrorResponse(err))
	}
}

func DeleteWebDAVAccounts(c *gin.Context) {
	var service setting.WebDAVAccountService
	if err := c.ShouldBindJSON(&service); err == nil {
		res := service.Delete(c, CurrentUser(c))
		c.JSON(200, res)
	} else {
		c.JSON(200, ErrorResponse(err))
	}
}

func ServeWebDAV(c *gin.Context) {
	fs, err := filesystem.NewFileSystemFromContext(c)
	if err != nil {
		logrus.Warningf("Unable to initialize file system for WebDAV， %s", err)
		return
	}

	if webdavCtx, ok := c.Get("webdav"); ok {
		application := webdavCtx.(*models.Webdav)

		if application.Root != "/" {
			if exist, root := fs.IsPathExist(application.Root); exist {
				root.Position = ""
				root.Name = "/"
				fs.Root = root
			}
		}
	}
	handler.ServeHTTP(c.Writer, c.Request, fs)
}
