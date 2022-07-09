package user

import (
	"github.com/gin-gonic/gin"
	"github.com/jylc/cloudserver/models"
	"github.com/jylc/cloudserver/pkg/serializer"
	"github.com/jylc/cloudserver/pkg/utils"
)

type LoginService struct {
	UserName string `form:"userName" json:"userName" binding:"required,email"`
	Password string `form:"Password" json:"Password" binding:"required,min=4,max=64"`
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
