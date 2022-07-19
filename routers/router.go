package routers

import (
	"github.com/gin-contrib/cors"
	"github.com/gin-contrib/gzip"
	"github.com/gin-gonic/gin"
	"github.com/jylc/cloudserver/middleware"
	"github.com/jylc/cloudserver/pkg/auth"
	"github.com/jylc/cloudserver/pkg/cluster"
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
				file.GET("source/:id/:name", controllers.AnonymousPermLink)
				file.GET("download/:id", controllers.Download)
				file.GET("archive/:sessionID/archive.zip", controllers.DownloadArchive)
			}
		}

		slave := version.Group("slave")
		slave.Use(middleware.SlaveRPCSignRequired(cluster.Default))
		{
			slave.PUT("notification/:subject", controllers.SlaveNotificationPush)
			upload := slave.Group("upload")
			{
				upload.POST(":sessionId", controllers.SlaveUpload)
				upload.PUT("", controllers.SlaveGetUploadSession)
				upload.DELETE(":sessionId", controllers.SlaveDeleteUploadSession)
			}
			slave.GET("credential/onedrive/:id", controllers.SlaveGetOneDriveCredential)
		}

		callback := version.Group("callback")
		{
			callback.POST("remote/:sessionID/:key",
				middleware.UseUploadSession("remote"),
				middleware.RemoteCallbackAuth(),
				controllers.RemoteCallback)
		}

		share := version.Group("share", middleware.ShareAvailable())
		{
			share.GET("info/:id", controllers.GetShare)

			share.PUT("download/:id",
				middleware.CheckShareUnlocked(),
				middleware.BeforeShareDownload(),
				controllers.GetShareDownload,
			)

			share.GET("preview/:id",
				middleware.CSRFCheck(),
				middleware.CheckShareUnlocked(),
				middleware.ShareCanPreview(),
				middleware.BeforeShareDownload(),
				controllers.PreviewShare,
			)

			share.GET("doc/:id",
				middleware.CheckShareUnlocked(),
				middleware.ShareCanPreview(),
				middleware.BeforeShareDownload(),
				controllers.GetShareDocPreview,
			)

			share.GET("content/:id",
				middleware.CheckShareUnlocked(),
				middleware.BeforeShareDownload(),
				controllers.PreviewShareText,
			)

			share.GET("list/:id/*path",
				middleware.CheckShareUnlocked(),
				controllers.ListSharedFolder,
			)

			share.GET("search/:id/:type/:keywords",
				middleware.CheckShareUnlocked(),
				controllers.SearchSharedFolder,
			)

			share.POST("archive/:id",
				middleware.CheckShareUnlocked(),
				middleware.BeforeShareDownload(),
				controllers.ArchiveShare,
			)

			share.GET("readme/:id",
				middleware.CheckShareUnlocked(),
				controllers.PreviewShareReadme,
			)

			share.GET("thumb/:id/:file",
				middleware.CheckShareUnlocked(),
				middleware.ShareCanPreview(),
				controllers.ShareThumb,
			)

			version.Group("share").GET("search", controllers.SearchShare)
		}

		auth := version.Group("")
		auth.Use(middleware.AuthRequired())
		{
			admin := auth.Group("admin", middleware.IsAdmin())
			{
				admin.GET("summary", controllers.AdminSummary)
				admin.GET("news", controllers.AdminNews)
				admin.PATCH("setting", controllers.AdminChangeSetting)
				admin.POST("setting", controllers.AdminGetSetting)
				admin.GET("groups", controllers.AdminGetGroups)
				admin.GET("reload/:service", controllers.AdminReloadService)
				admin.POST("mailTest", controllers.AdminSendTestMail)

				aria2 := admin.Group("aria2")
				{
					aria2.POST("test", controllers.AdminTestAria2)
				}

				policy := admin.Group("policy")
				{
					policy.POST("list", controllers.AdminListPolicy)
					policy.POST("test/path", controllers.AdminTestPath)
					policy.POST("test/slave", controllers.AdminTestSlave)
					policy.POST("", controllers.AdminAddPolicy)
					policy.POST("cors", controllers.AdminAddCORS)
					policy.POST("scf", controllers.AdminAddSCF)
					policy.GET(":id/oauth", controllers.AdminOneDriveOAuth)
					policy.GET(":id", controllers.AdminGetPolicy)
					policy.DELETE(":id", controllers.AdminDeletePolicy)
				}

				group := admin.Group("group")
				{
					group.POST("list", controllers.AdminListGroup)
					group.GET(":id", controllers.AdminGetGroup)
					group.POST("", controllers.AdminAddGroup)
					group.DELETE(":id", controllers.AdminDeleteGroup)
				}

				user := admin.Group("user")
				{
					user.POST("list", controllers.AdminListUser)
					user.GET(":id", controllers.AdminGetUser)
					user.POST("", controllers.AdminAddUser)
					user.POST("delete", controllers.AdminDeleteUser)
					user.PATCH("ban/:id", controllers.AdminBanUser)
				}

				file := admin.Group("file")
				{
					file.POST("list", controllers.AdminListFile)
					file.GET("preview/:id", controllers.AdminGetFile)
					file.POST("delete", controllers.AdminDeleteFile)
					file.GET("folders/:type/:id/*path", controllers.AdminListFolders)
				}

				share := admin.Group("share")
				{
					share.POST("list", controllers.AdminListShare)
					share.POST("delete", controllers.AdminDeleteShare)
				}

				download := admin.Group("share")
				{
					download.POST("list", controllers.AdminListDownload)
					download.POST("delete", controllers.AdminDeleteDownload)
				}

				task := admin.Group("task")
				{
					task.POST("list", controllers.AdminListTask)
					task.POST("delete", controllers.AdminDeleteTask)
					task.POST("import", controllers.AdminCreateImportTask)
				}

				node := admin.Group("node")
				{
					node.POST("list", controllers.AdminListNodes)
					node.POST("aria2/test", controllers.AdminTestAria2)
					node.POST("", controllers.AdminAddNode)
					node.PATCH("enable/:id/:desired", controllers.AdminToggleNode)
					node.DELETE(":id", controllers.AdminDeleteNode)
					node.GET(":id", controllers.AdminGetNode)
				}
			}

			user := auth.Group("user")
			{
				user.GET("me", controllers.UserMe)
				user.GET("storage", controllers.UserStorage)
				user.GET("session", controllers.UserSignOut)

				authn := user.Group("authn", middleware.IsFunctionEnabled("authn_enabled"))
				{
					authn.PUT("", controllers.StartRegAuthn)
					authn.PUT("finish", controllers.FinishRegAuthn)
				}

				setting := user.Group("setting")
				{
					setting.GET("tasks", controllers.UserTasks)
					setting.GET("", controllers.UserSetting)
					setting.POST("avatar", controllers.UploadAvatar)
					setting.PUT("avatar", controllers.UseGravatar)
					setting.PATCH(":option", controllers.UpdateOption)
					setting.GET("2fa", controllers.UserInit2FA)
				}
			}

			file := auth.Group("file", middleware.HashID(hashid.FileID))
			{
				upload := file.Group("upload")
				{
					upload.POST(":sessionId/:index", controllers.FileUpload)
					upload.PUT("", controllers.GetUploadSession)
					upload.DELETE(":sessionId", controllers.DeleteUploadSession)
					upload.DELETE("", controllers.DeleteAllUploadSession)
				}

				file.PUT("update/:id", controllers.PutContent)
				file.POST("create", controllers.CreateFile)
				file.PUT("download/:id", controllers.CreateDownloadSession)
				file.GET("preview/:id", controllers.Preview)
				file.GET("content/:id", controllers.PreviewText)
				file.GET("doc/:id", controllers.GetDocPreview)
				file.GET("thumb/:id", controllers.Thumb)
				file.POST("source", controllers.GetSource)
				file.POST("archive", controllers.Archive)
				file.POST("compress", controllers.Compress)
				file.POST("decompress", controllers.Decompress)
				file.GET("search/:type/:keywords", controllers.SearchFile)
			}

			aria2 := auth.Group("aria2")
			{
				aria2.POST("url", controllers.AddAria2URL)
				aria2.POST("torrent/:id", middleware.HashID(hashid.FileID), controllers.AddAria2Torrent)
				aria2.PUT("select/:gid", controllers.SelectAria2File)
				aria2.DELETE("task/:gid", controllers.CancelAria2Download)
				aria2.GET("downloading", controllers.ListDownloading)
				aria2.GET("finished", controllers.ListFinished)
			}

			directory := auth.Group("directory")
			{
				directory.PUT("", controllers.CreateDirectory)
				directory.GET("*path", controllers.ListDirectory)
			}

			object := auth.Group("object")
			{
				object.DELETE("", controllers.Delete)
				object.PATCH("", controllers.Move)
				object.POST("copy", controllers.Copy)
				object.POST("rename", controllers.Rename)
				object.GET("property/:id", controllers.GetProperty)
			}

			share := auth.Group("share")
			{
				share.POST("", controllers.CreateShare)
				share.GET("", controllers.ListShare)
				share.PATCH(":id",
					middleware.ShareAvailable(),
					middleware.ShareOwner(),
					controllers.UpdateShare,
				)
				share.DELETE(":id", controllers.DeleteShare)
			}

			tag := auth.Group("tag")
			{
				tag.POST("filter", controllers.CreateFilterTag)
				tag.POST("link", controllers.CreateLinkTag)
				tag.DELETE(":id", middleware.HashID(hashid.TagID), controllers.DeleteTag)
			}

			webdav := auth.Group("webdav")
			{
				webdav.GET("accounts", controllers.GetWebDAVAccounts)
				webdav.POST("accounts", controllers.CreateWebDAVAccounts)
				webdav.DELETE("accounts/:id", controllers.DeleteWebDAVAccounts)
			}
		}
	}
	initWebDAV(r.Group("dav"))
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

func initWebDAV(group *gin.RouterGroup) {
	{
		group.Use(middleware.WebDAVAuth())

		group.Any("/*path", controllers.ServeWebDAV)
		group.Any("", controllers.ServeWebDAV)
		group.Handle("PROPFIND", "/*path", controllers.ServeWebDAV)
		group.Handle("PROPFIND", "", controllers.ServeWebDAV)
		group.Handle("MKCOL", "/*path", controllers.ServeWebDAV)
		group.Handle("LOCK", "/*path", controllers.ServeWebDAV)
		group.Handle("UNLOCK", "/*path", controllers.ServeWebDAV)
		group.Handle("PROPPATCH", "/*path", controllers.ServeWebDAV)
		group.Handle("COPY", "/*path", controllers.ServeWebDAV)
		group.Handle("MOVE", "/*path", controllers.ServeWebDAV)
	}
}
