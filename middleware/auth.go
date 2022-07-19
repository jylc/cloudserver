package middleware

import (
	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"github.com/jylc/cloudserver/models"
	"github.com/jylc/cloudserver/pkg/auth"
	"github.com/jylc/cloudserver/pkg/cache"
	"github.com/jylc/cloudserver/pkg/filesystem"
	"github.com/jylc/cloudserver/pkg/serializer"
	"net/http"
)

const (
	CallbackFailedStatusCode = http.StatusUnauthorized
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

func SignRequired(authInstance auth.Auth) gin.HandlerFunc {
	return func(c *gin.Context) {
		var err error
		switch c.Request.Method {
		case "PUT", "POST", "PATCH":
			err = auth.CheckRequest(authInstance, c.Request)
		default:
			err = auth.CheckURI(authInstance, c.Request.URL)
		}

		if err != nil {
			c.JSON(200, serializer.Err(serializer.CodeCredentialInvalid, err.Error(), err))
			c.Abort()
			return
		}
		c.Next()
	}
}

func UseUploadSession(policyType string) gin.HandlerFunc {
	return func(c *gin.Context) {
		resp := uploadCallbackCheck(c, policyType)
		if resp.Code != 0 {
			c.JSON(CallbackFailedStatusCode, resp)
			c.Abort()
			return
		}
		c.Next()
	}
}

func uploadCallbackCheck(c *gin.Context, policyType string) serializer.Response {
	sessionID := c.Param("sessionID")
	if sessionID == "" {
		return serializer.ParamErr("SessionID cannot be empty", nil)
	}

	callbackSessionRaw, exist := cache.Get(filesystem.UploadSessionCachePrefix + sessionID)
	if !exist {
		serializer.ParamErr("upload session does not exist or has expired", nil)
	}

	callbackSession := callbackSessionRaw.(serializer.UploadSession)
	c.Set(filesystem.UploadSessionCtx, &callbackSession)
	if callbackSession.Policy.Type != policyType {
		return serializer.Err(serializer.CodePolicyNotAllowed, "Policy not supported", nil)
	}
	_ = cache.Deletes([]string{sessionID}, filesystem.UploadSessionCachePrefix)

	user, err := models.GetActivateUserByID(callbackSession.UID)
	if err != nil {
		return serializer.Err(serializer.CodeCheckLogin, "cannot find user", err)
	}
	c.Set(filesystem.UserCtx, &user)
	return serializer.Response{}
}

func RemoteCallbackAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		session := c.MustGet(filesystem.UploadSessionCtx).(*serializer.UploadSession)
		authInstance := auth.HMACAuth{SecretKey: []byte(session.Policy.SecretKey)}
		if err := auth.CheckRequest(authInstance, c.Request); err != nil {
			c.JSON(CallbackFailedStatusCode, serializer.Err(serializer.CodeCredentialInvalid, err.Error(), err))
			c.Abort()
			return
		}
		c.Next()
	}
}

func AuthRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		if user, _ := c.Get("user"); user != nil {
			if _, ok := user.(*models.User); ok {
				c.Next()
				return
			}
		}
		c.JSON(200, serializer.CheckLogin())
		c.Abort()
	}
}

func IsAdmin() gin.HandlerFunc {
	return func(c *gin.Context) {
		user, _ := c.Get("user")
		if user.(*models.User).Group.ID != 1 && user.(*models.User).ID != 1 {
			c.JSON(200, serializer.Err(serializer.CodeAdminRequired, "you are not a member of the management group", nil))
			c.Abort()
			return
		}
		c.Next()
	}
}

func WebDAVAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.Method == "OPTIONS" {
			c.Next()
			return
		}

		username, password, ok := c.Request.BasicAuth()
		if !ok {
			c.Writer.Header()["WWW-Authenticate"] = []string{`Basic realm="cloudreve"`}
			c.Status(http.StatusUnauthorized)
			c.Abort()
			return
		}

		expectedUser, err := models.GetActivateUserByEmail(username)
		if err != nil {
			c.Status(http.StatusUnauthorized)
			c.Abort()
			return
		}

		webdav, err := models.GetWebdavByPassword(password, expectedUser.ID)
		if err != nil {
			c.Status(http.StatusUnauthorized)
			c.Abort()
			return
		}

		if !expectedUser.Group.WebDAVEnabled {
			c.Status(http.StatusForbidden)
			c.Abort()
			return
		}

		c.Set("user", &expectedUser)
		c.Set("webdav", webdav)
		c.Next()
	}
}
