package user

import (
	"github.com/gin-gonic/gin"
	"github.com/jylc/cloudserver/models"
	"github.com/jylc/cloudserver/pkg/email"
	"github.com/jylc/cloudserver/pkg/hashid"
	"github.com/jylc/cloudserver/pkg/serializer"
	"github.com/jylc/cloudserver/pkg/utils"
	"github.com/pquerna/otp/totp"
	"net/url"
)

type LoginService struct {
	UserName string `form:"userName" json:"userName" binding:"required,email"`
	Password string `form:"Password" json:"Password" binding:"required,min=4,max=64"`
}

type ResetEmailService struct {
	UserName string `form:"userName" json:"userName" binding:"required,email"`
}

type ResetService struct {
	Password string `form:"Password" json:"Password" binding:"required,min=4,max=64"`
	ID       string `json:"id" binding:"required"`
	Secret   string `json:"secret" binding:"required"`
}

func (service *ResetService) Reset(c *gin.Context) serializer.Response {
	uid, err := hashid.DecodeHashID(service.ID, hashid.UserID)
	if err != nil {
		return serializer.Err(serializer.CodeInvalidTempLink, "Invalid link", err)
	}

	user, err := models.GetActivateUserByID(uid)
	if err != nil {
		return serializer.Err(serializer.CodeUserNotFound, "User not found", nil)
	}
	user.SetPassword(service.Password)
	if err := user.Update(map[string]interface{}{"password": user.Password}); err != nil {
		return serializer.DBErr("Failed to reset password", err)
	}
	return serializer.Response{}
}

func (service *LoginService) Login(c *gin.Context) serializer.Response {
	expectedUser, err := models.GetUserByEmail(service.UserName)
	if err != nil {
		return serializer.Err(serializer.CodeCredentialInvalid, "Wrong password or email address", err)
	}
	if authOK, _ := expectedUser.CheckPassword(service.Password); !authOK {
		return serializer.Err(serializer.CodeCredentialInvalid, "Wrong password or email address", nil)
	}
	if expectedUser.Status == models.Baned || expectedUser.Status == models.OveruseBaned {
		return serializer.Err(serializer.CodeUserBaned, "This account has been blocked", nil)
	}
	if expectedUser.Status == models.NotActivate {
		return serializer.Err(serializer.CodeUserNotActivated, "This account is not activated", nil)
	}
	if expectedUser.TwoFactor != "" {
		utils.SetSession(c, map[string]interface{}{
			"2fa_user_id": expectedUser.ID,
		})
		return serializer.Response{Code: 203}
	}

	utils.SetSession(c, map[string]interface{}{
		"user_id": expectedUser.ID,
	})
	return serializer.BuildUserResponse(expectedUser)
}

func (service *Enable2FA) Login(c *gin.Context) serializer.Response {
	if uid, ok := utils.GetSession(c, "2fa_user_id").(uint); ok {
		expectedUser, err := models.GetActivateUserByID(uid)
		if err != nil {
			return serializer.Err(serializer.CodeUserNotFound, "User not found", nil)
		}
		if !totp.Validate(service.Code, expectedUser.TwoFactor) {
			return serializer.Err(serializer.Code2FACodeErr, "2FA code not correct", nil)
		}
		utils.DeleteSession(c, "2fa_user_id")
		utils.SetSession(c, map[string]interface{}{
			"user_id": expectedUser.ID,
		})
		return serializer.BuildUserResponse(expectedUser)
	}
	return serializer.Err(serializer.CodeLoginSessionNotExist, "Login session not exist", nil)
}
func (service *ResetEmailService) Reset(c *gin.Context) serializer.Response {
	if user, err := models.GetUserByEmail(service.UserName); err != nil {
		if user.Status == models.Baned || user.Status == models.OveruseBaned {
			return serializer.Err(serializer.CodeUserBaned, "This user is banned", nil)
		}
		if user.Status == models.NotActivate {
			return serializer.Err(serializer.CodeUserNotActivated, "This user is not activated", nil)
		}

		secret := utils.RandStringRunes(32)

		controller, _ := url.Parse("/reset")
		finalURL := models.GetSiteURL().ResolveReference(controller)
		queries := finalURL.Query()
		queries.Add("id", hashid.HashID(user.ID, hashid.UserID))
		queries.Add("sign", secret)
		finalURL.RawQuery = queries.Encode()

		title, body := email.NewResetEmail(service.UserName, finalURL.String())
		if err := email.Send(user.Email, title, body); err != nil {
			return serializer.Err(serializer.CodeFailedSendEmail, "Failed to send email", err)
		}
	}
	return serializer.Response{}
}
