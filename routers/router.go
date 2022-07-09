package routers

import (
	"github.com/gin-contrib/cors"
	"github.com/gin-contrib/gzip"
	"github.com/gin-gonic/gin"
	"github.com/jylc/cloudserver/middleware"
	"github.com/jylc/cloudserver/pkg/conf"
	"github.com/jylc/cloudserver/routers/controllers"
)

func RouterInit() *gin.Engine {
	r := gin.Default()

	//静态资源；压缩数据减少网络传输
	r.Use(gzip.Gzip(gzip.DefaultCompression, gzip.WithExcludedPaths([]string{"/api/"})))
	r.Use(middleware.FrontendFileHandler())

	version := r.Group("/api/v3")
	version.Use(middleware.Session(conf.Sc.SessionSecret))
	//跨域
	CORSInit(r)
	version.Use(middleware.CurrentUser())
	version.Use(middleware.CacheControl())

	{
		site := version.Group("site")
		{
			//ping
			site.GET("ping", controllers.Ping)
			//验证码
			site.GET("captcha", controllers.Captcha)
			//站点全局配置
			site.GET("config", middleware.CSRFInit(), controllers.SiteConfig)
		}

		user := version.Group("user")
		{
			user.POST("session", middleware.CaptchaRequired("login_captcha"), controllers.UserLogin)
			user.POST("", middleware.IsFunctionEnabled("register_enabled"),
				middleware.CaptchaRequired("reg_captcha"))
		}
	}
	return r
}

func CORSInit(r *gin.Engine) {
	if conf.Cc.AllowOrigins[0] != "UNSET" {
		r.Use(cors.New(cors.Config{
			AllowOrigins:     conf.Cc.AllowOrigins,
			AllowMethods:     conf.Cc.AllowMethods,
			AllowHeaders:     conf.Cc.AllowHeaders,
			AllowCredentials: conf.Cc.AllowCredentials,
			AllowOriginFunc:  conf.Cc.AllowOriginFunc,
			MaxAge:           conf.Cc.MaxAge,
		}))
	}
}
