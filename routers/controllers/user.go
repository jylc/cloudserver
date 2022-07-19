package controllers

import (
	"encoding/json"
	"fmt"
	"github.com/duo-labs/webauthn/webauthn"
	"github.com/gin-gonic/gin"
	"github.com/jylc/cloudserver/models"
	"github.com/jylc/cloudserver/pkg/authn"
	"github.com/jylc/cloudserver/pkg/request"
	"github.com/jylc/cloudserver/pkg/serializer"
	"github.com/jylc/cloudserver/pkg/utils"
	"github.com/jylc/cloudserver/service/user"
)

func UserLogin(c *gin.Context) {
	var service user.LoginService
	if err := c.ShouldBindJSON(&service); err == nil {
		res := service.Login(c)
		c.JSON(200, res)
	} else {
		c.JSON(200, ErrorResponse(err))
	}
}

func UserRegister(c *gin.Context) {
	var service user.RegisterService
	if err := c.ShouldBindJSON(&service); err == nil {
		res := service.Register(c)
		c.JSON(200, res)
	} else {
		c.JSON(200, ErrorResponse(err))
	}
}

func User2FALogin(c *gin.Context) {
	var service user.Enable2FA
	if err := c.ShouldBindJSON(&service); err == nil {
		res := service.Login(c)
		c.JSON(200, res)
	} else {
		c.JSON(200, ErrorResponse(err))
	}
}

func UserSendReset(c *gin.Context) {
	var service user.ResetEmailService
	if err := c.ShouldBindJSON(&service); err != nil {
		res := service.Reset(c)
		c.JSON(200, res)
	} else {
		c.JSON(200, ErrorResponse(err))
	}
}

func UserReset(c *gin.Context) {
	var service user.ResetService
	if err := c.ShouldBindJSON(&service); err != nil {
		res := service.Reset(c)
		c.JSON(200, res)
	} else {
		c.JSON(200, ErrorResponse(err))
	}
}

func UserActivate(c *gin.Context) {
	var service user.SettingService
	if err := c.ShouldBindUri(&service); err != nil {
		res := service.Activate(c)
		c.JSON(200, res)
	} else {
		c.JSON(200, ErrorResponse(err))
	}
}

func UserSignOut(c *gin.Context) {
	utils.DeleteSession(c, "user_id")
	c.JSON(200, serializer.Response{})
}

func StartLoginAuthn(c *gin.Context) {
	userName := c.Param("username")
	expectedUser, err := models.GetActivateUserByEmail(userName)
	if err != nil {
		c.JSON(200, serializer.Err(serializer.CodeUserNotFound, "User not exist", err))
		return
	}

	instance, err := authn.NewAuthnInstance()
	if err != nil {
		c.JSON(200, serializer.Err(serializer.CodeInitializeAuthn, "Cannot initialize authn", err))
		return
	}

	options, sessionData, err := instance.BeginLogin(expectedUser)
	if err != nil {
		c.JSON(200, ErrorResponse(err))
		return
	}
	val, err := json.Marshal(sessionData)
	if err != nil {
		c.JSON(200, ErrorResponse(err))
		return
	}
	utils.SetSession(c, map[string]interface{}{
		"registration-session": val,
	})
	c.JSON(200, serializer.Response{Code: 0, Data: options})
}

func StartRegAuthn(c *gin.Context) {
	currUser := CurrentUser(c)
	instance, err := authn.NewAuthnInstance()
	if err != nil {
		c.JSON(200, serializer.Err(serializer.CodeInternalSetting, "Unable to initialize authn", err))
		return
	}

	options, sessionData, err := instance.BeginLogin(currUser)
	if err != nil {
		c.JSON(200, ErrorResponse(err))
		return
	}
	val, err := json.Marshal(sessionData)
	if err != nil {
		c.JSON(200, ErrorResponse(err))
		return
	}

	utils.SetSession(c, map[string]interface{}{
		"registration-session": val,
	})
	c.JSON(200, serializer.Response{Code: 0, Data: options})
}

func FinishRegAuthn(c *gin.Context) {
	currUser := CurrentUser(c)
	sessionDataJSON := utils.GetSession(c, "registration-session").([]byte)
	var sessionData webauthn.SessionData
	err := json.Unmarshal(sessionDataJSON, &sessionData)

	instance, err := authn.NewAuthnInstance()
	if err != nil {
		c.JSON(200, serializer.Err(serializer.CodeInternalSetting, "Unable to initialize authn", err))
		return
	}

	credential, err := instance.FinishRegistration(currUser, sessionData, c.Request)
	if err != nil {
		c.JSON(200, ErrorResponse(err))
		return
	}

	err = currUser.RegisterAuthn(credential)
	if err != nil {
		c.JSON(200, ErrorResponse(err))
		return
	}

	c.JSON(2000, serializer.Response{
		Code: 0,
		Data: map[string]interface{}{
			"id":          credential.ID,
			"fingerprint": fmt.Sprintf("% X", credential.Authenticator.AAGUID),
		},
	})
}

func FinishLoginAuthn(c *gin.Context) {
	userName := c.Param("username")
	expectedUser, err := models.GetActivateUserByEmail(userName)
	if err != nil {
		c.JSON(200, serializer.Err(serializer.CodeUserNotFound, "User not exist", err))
	}
	sessionDataJSON := utils.GetSession(c, "registration-session").([]byte)
	var sessionData webauthn.SessionData
	err = json.Unmarshal(sessionDataJSON, &sessionData)
	instance, err := authn.NewAuthnInstance()
	if err != nil {
		c.JSON(200, serializer.Err(serializer.CodeInitializeAuthn, "Cannot initialize authn", err))
		return
	}
	_, err = instance.FinishLogin(expectedUser, sessionData, c.Request)
	if err != nil {
		c.JSON(200, serializer.Err(serializer.CodeWebAuthnCredentialError, "Verification failed", err))
		return
	}
	utils.SetSession(c, map[string]interface{}{
		"user_id": expectedUser.ID,
	})
	c.JSON(200, serializer.BuildUserResponse(expectedUser))
}

func GetUserAvatar(c *gin.Context) {
	var service user.AvatarService
	if err := c.ShouldBindUri(&service); err == nil {
		res := service.Get(c)
		if res.Code == -301 {
			c.Redirect(301, res.Data.(string))
		}
	} else {
		c.JSON(200, ErrorResponse(err))
	}
}

func UserMe(c *gin.Context) {
	currUser := CurrentUser(c)
	res := serializer.BuildUserResponse(*currUser)
	c.JSON(200, res)
}

func UserStorage(c *gin.Context) {
	currUser := CurrentUser(c)
	res := serializer.BuildUserStorageResponse(*currUser)
	c.JSON(200, res)
}

func UserTasks(c *gin.Context) {
	var service user.SettingListService
	if err := c.ShouldBindQuery(&service); err == nil {
		res := service.ListTasks(c, CurrentUser(c))
		c.JSON(200, res)
	} else {
		c.JSON(200, ErrorResponse(err))
	}
}

func UserSetting(c *gin.Context) {
	var service user.SettingService
	if err := c.ShouldBindUri(&service); err == nil {
		res := service.Setting(c, CurrentUser(c))
		c.JSON(200, res)
	} else {
		c.JSON(200, ErrorResponse(err))
	}
}

func UseGravatar(c *gin.Context) {
	u := CurrentUser(c)
	if err := u.Update(map[string]interface{}{"avatar": "gravatar"}); err != nil {
		c.JSON(200, serializer.Err(serializer.CodeDBError, "unable to update Avatar", err))
		return
	}
	c.JSON(200, serializer.Response{})
}

func UploadAvatar(c *gin.Context) {
	maxSize := models.GetIntSetting("avatar_size", 2097152)
	if c.Request.ContentLength == -1 || c.Request.ContentLength > int64(maxSize) {
		request.BlackHole(c.Request.Body)
		c.JSON(200, serializer.Err(serializer.CodeUploadFailed, "avatar size is too large", nil))
		return
	}

	file, err := c.FormFile("avatar")
	if err != nil {
		c.JSON(200, serializer.Err(serializer.CodeIOFailed, "unable to read avatar data", err))
		return
	}

	r, err := file.Open()
	if err != nil {
		c.JSON(200, serializer.Err(serializer.CodeIOFailed, "unable to read avatar data", err))
		return
	}

	avatar, err := humb.NewThumbFromFile(r, file.Filename)
	if err != nil {
		c.JSON(200, serializer.Err(serializer.CodeIOFailed, "unable to parse image data", err))
		return
	}

	u := CurrentUser(c)
	err = avatar.CreateAvatar(u.ID)
	if err != nil {
		c.JSON(200, serializer.Err(serializer.CodeIOFailed, "unable to create Avatar", err))
		return
	}

	if err := u.Update(map[string]interface{}{
		"avatar": "file",
	}); err != nil {
		c.JSON(200, serializer.Err(serializer.CodeDBError, "unable to update Avatar", err))
		return
	}
	c.JSON(200, serializer.Response{})
}

func UpdateOption(c *gin.Context) {
	var service user.SettingUpdateService
	if err := c.ShouldBindUri(&service); err == nil {
		var (
			subService user.OptionsChangeHandler
			subErr     error
		)

		switch service.Option {
		case "nick":
			subService = &user.ChangerNick{}
		case "homepage":
			subService = &user.HomePage{}
		case "password":
			subService = &user.Password{}
		case "2fa":
			subService = &user.Enable2FA{}
		case "authn":
			subService = &user.DeleteWebAuthn{}
		case "theme":
			subService = &user.ThemeChose{}
		default:
			subService = &user.ChangerNick{}
		}

		subErr = c.ShouldBindJSON(subService)
		if subErr != nil {
			c.JSON(200, ErrorResponse(subErr))
			return
		}

		res := subService.Update(c, CurrentUser(c))
		c.JSON(200, res)
	} else {
		c.JSON(200, ErrorResponse(err))
	}
}

func UserInit2FA(c *gin.Context) {
	var service user.SettingService
	if err := c.ShouldBindUri(&service); err == nil {
		res := service.Init2FA(c, CurrentUser(c))
		c.JSON(200, res)
	} else {
		c.JSON(200, ErrorResponse(err))
	}
}
