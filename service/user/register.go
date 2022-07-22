package user

import (
	"github.com/gin-gonic/gin"
	"github.com/jylc/cloudserver/models"
	"github.com/jylc/cloudserver/pkg/auth"
	"github.com/jylc/cloudserver/pkg/email"
	"github.com/jylc/cloudserver/pkg/hashid"
	"github.com/jylc/cloudserver/pkg/serializer"
	"net/url"
	"strings"
)

type RegisterService struct {
	UserName string `form:"userName" json:"userName" binding:"required,email"`
	Password string `form:"Password" json:"Password" binding:"required,min=4,max=64"`
}

func (service *RegisterService) Register(c *gin.Context) serializer.Response {
	options := models.GetSettingByNames("email_active")

	isEmailRequired := models.IsTrueVal(options["email_active"])
	defaultGroup := models.GetIntSetting("default_group", 2)

	user := models.NewUser()
	user.Email = service.UserName
	user.Nick = strings.Split(service.UserName, "@")[0]
	user.SetPassword(service.Password)
	user.Status = models.Active
	if isEmailRequired {
		user.Status = models.NotActivate
	}
	user.GroupID = uint(defaultGroup)
	userNotActivated := false
	if err := models.Db.Create(&user).Error; err != nil {
		expectedUser, err := models.GetUserByEmail(service.UserName)
		if expectedUser.Status == models.NotActivate {
			userNotActivated = true
			user = expectedUser
		} else {
			serializer.Err(serializer.CodeEmailExisted, "Email already in use", err)
		}
	}

	if isEmailRequired {
		base := models.GetSiteURL()
		userID := hashid.HashID(user.ID, hashid.UserID)
		controller, _ := url.Parse("/api/v3/user/activate/" + userID)
		activateURL, err := auth.SignURI(auth.General, base.ResolveReference(controller).String(), 86400)
		if err != nil {
			return serializer.Err(serializer.CodeEncryptError, "Failed to sign the activation link", err)
		}
		credential := activateURL.Query().Get("sign")
		controller, _ = url.Parse("/activate")
		finalURL := base.ResolveReference(controller)
		queries := finalURL.Query()
		queries.Add("id", userID)
		queries.Add("sign", credential)
		finalURL.RawQuery = queries.Encode()

		title, body := email.NewActivationEmail(user.Email, finalURL.String())
		if err := email.Send(user.Email, title, body); err != nil {
			return serializer.Err(serializer.CodeFailedSendEmail, "Failed to send activation email", err)
		}

		if userNotActivated == true {
			return serializer.Err(serializer.CodeEmailSent, "User is not activated, activation email has been resent", nil)
		} else {
			return serializer.Response{Code: 203}
		}
	}
	return serializer.Response{}
}

func (service *SettingService) Activate(c *gin.Context) serializer.Response {
	uid, _ := c.Get("object_id")
	user, err := models.GetUserByID(uid.(int))
	if err != nil {
		return serializer.Err(serializer.CodeUserNotFound, "User not found", err)
	}

	if user.Status != models.NotActivate {
		return serializer.Err(serializer.CodeUserCannotActivate, "This user cannot be activated", nil)
	}
	user.SetStatus(models.Active)
	return serializer.Response{Data: user.Email}
}
