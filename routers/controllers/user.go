package controllers

import (
	"encoding/json"
	"github.com/duo-labs/webauthn/webauthn"
	"github.com/gin-gonic/gin"
	"github.com/jylc/cloudserver/models"
	"github.com/jylc/cloudserver/pkg/authn"
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
