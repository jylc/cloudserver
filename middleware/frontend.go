package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/jylc/cloudserver/bootstrap"
	"github.com/jylc/cloudserver/models"
	"github.com/jylc/cloudserver/pkg/utils"
	"github.com/sirupsen/logrus"
	"io"
	"net/http"
	"strings"
)

func FrontendFileHandler() gin.HandlerFunc {
	ignore := func(c *gin.Context) {
		c.Next()
	}

	if bootstrap.StaticFS == nil {
		return ignore
	}

	index, err := bootstrap.StaticFS.Open("/index.html")
	if err != nil {
		logrus.Errorf("cannot open file [%s], %s\n", "index.html", err)
		return ignore
	}
	indexBytes, err := io.ReadAll(index)
	if err != nil {
		logrus.Errorf("cannot read file [%s], %s\n", "index.html", err)
		return ignore
	}
	indexString := string(indexBytes)
	return func(c *gin.Context) {
		path := c.Request.URL.Path
		if strings.HasPrefix(path, "/api") || strings.HasPrefix(path, "/custom") {
			c.Next()
			return
		}

		//如果是"index.html"，"/"或者访问的路径不存在则返回index.html；index.html页面需要设置
		if (path == "/index.html") || (path == "/") || !bootstrap.StaticFS.Exists("/", path) {
			options := models.GetSettingByNames(
				"siteName", "siteKeywords", "siteScript", "pwa_small_icon",
			)
			finalIndex := utils.Replace(indexString, map[string]string{
				"{siteName}":       options["siteName"],
				"{siteKeywords}":   options["siteKeywords"],
				"{siteScript}":     options["siteScript"],
				"{pwa_small_icon}": options["pwa_small_icon"],
			})
			c.Header("Content-Type", "text/html")
			c.String(200, finalIndex)
			c.Abort()
			return
		}
		//如果是其他页面则渲染
		server := http.FileServer(bootstrap.StaticFS)
		server.ServeHTTP(c.Writer, c.Request)
		c.Abort()
	}
}
