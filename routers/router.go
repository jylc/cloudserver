package routers

import (
	"github.com/gin-contrib/cors"
	"github.com/gin-contrib/gzip"
	"github.com/gin-gonic/gin"
	"github.com/jylc/cloudserver/middleware"
	"github.com/jylc/cloudserver/pkg/auth"
	"github.com/jylc/cloudserver/pkg/conf"
	"github.com/jylc/cloudserver/pkg/hashid"
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
			user.POST("",
				middleware.IsFunctionEnabled("register_enabled"),
				middleware.CaptchaRequired("reg_captcha"),
				controllers.UserRegister)
			user.POST("2fa", controllers.User2FALogin)
			user.POST("reset", middleware.CaptchaRequired("forget_captcha"), controllers.UserSendReset)
			user.PATCH("reset", controllers.UserReset)

			user.GET("activate/:id",
				middleware.SignRequired(auth.General),
				middleware.HashID(hashid.UserID),
				controllers.UserActivate)

			user.GET("authn/:username",
				middleware.IsFunctionEnabled("authn_enabled"),
				controllers.StartLoginAuthn)

			user.POST("authn/finish/:username",
				middleware.IsFunctionEnabled("authn_enabled"),
				controllers.FinishLoginAuthn)

			user.GET("profile/:id",
				middleware.HashID(hashid.UserID),
				controllers.GetUserShare)

			user.GET("avatar/:id/:size",
				middleware.HashID(hashid.UserID),
				controllers.GetUserAvatar)
		}

		sign := version.Group("")
		sign.Use(middleware.SignRequired(auth.General))
		{
			file := version.Group("file")
			{
				file.GET("get/:id/:name", controllers.AnonymousGetContent)
			}
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
