package user

import (
	"crypto/md5"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/jylc/cloudserver/models"
	"github.com/jylc/cloudserver/pkg/serializer"
	"github.com/jylc/cloudserver/pkg/utils"
	"github.com/pquerna/otp/totp"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

type Enable2FA struct {
	Code string `json:"code" binding:"required"`
}

func (service *Enable2FA) Update(c *gin.Context, user *models.User) serializer.Response {
	if user.TwoFactor == "" {
		secret, ok := utils.GetSession(c, "2fa_init").(string)
		if !ok {
			return serializer.Err(serializer.CodeParamErr, "Two step verification is not initialized", nil)
		}

		if !totp.Validate(service.Code, secret) {
			return serializer.ParamErr("Incorrect verification code", nil)
		}

		if err := user.Update(map[string]interface{}{"two_factor": secret}); err != nil {
			return serializer.DBErr("Unable to update the two-step verification settings", err)
		}

	} else {
		// 关闭2FA
		if !totp.Validate(service.Code, user.TwoFactor) {
			return serializer.ParamErr("Incorrect verification code", nil)
		}

		if err := user.Update(map[string]interface{}{"two_factor": ""}); err != nil {
			return serializer.DBErr("Unable to update the two-step verification settings", err)
		}
	}

	return serializer.Response{}
}

type SettingService struct {
}

type AvatarService struct {
	Size string `uri:"size" binding:"required,eq=l|eq=m|eq=s"`
}

type SettingListService struct {
	Page int `form:"page" binding:"required,min=1"`
}

type SettingUpdateService struct {
	Option string `uri:"option" binding:"required,eq=nick|eq=theme|eq=homepage|eq=vip|eq=qq|eq=policy|eq=password|eq=2fa|eq=authn"`
}

type OptionsChangeHandler interface {
	Update(*gin.Context, *models.User) serializer.Response
}

type ChangerNick struct {
	Nick string `json:"nick" binding:"required,min=1,max=255"`
}

func (service *ChangerNick) Update(c *gin.Context, user *models.User) serializer.Response {
	if err := user.Update(map[string]interface{}{"nick": service.Nick}); err != nil {
		return serializer.DBErr("Unable to update nickname", err)
	}
	return serializer.Response{}
}

type PolicyChange struct {
	ID string `json:"id" binding:"required"`
}

type HomePage struct {
	Enabled bool `json:"status"`
}

func (service *HomePage) Update(c *gin.Context, user *models.User) serializer.Response {
	user.OptionsSerialized.ProfileOff = !service.Enabled
	if err := user.UpdateOptions(); err != nil {
		return serializer.DBErr("Storage policy switching failed", err)
	}

	return serializer.Response{}
}

type PasswordChange struct {
	Old string `json:"old" binding:"required,min=4,max=64"`
	New string `json:"new" binding:"required,min=4,max=64"`
}

func (service *PasswordChange) Update(c *gin.Context, user *models.User) serializer.Response {
	if ok, _ := user.CheckPassword(service.Old); !ok {
		return serializer.Err(serializer.CodeParamErr, "The original password is incorrect", nil)
	}

	user.SetPassword(service.New)
	if err := user.Update(map[string]interface{}{"password": user.Password}); err != nil {
		return serializer.DBErr("Password change failed", err)
	}

	return serializer.Response{}
}

type DeleteWebAuthn struct {
	ID string `json:"id" binding:"required"`
}

func (service *DeleteWebAuthn) Update(c *gin.Context, user *models.User) serializer.Response {
	user.RemoveAuthn(service.ID)
	return serializer.Response{}
}

type ThemeChose struct {
	Theme string `json:"theme" binding:"required,hexcolor|rgb|rgba|hsl"`
}

func (service *ThemeChose) Update(c *gin.Context, user *models.User) serializer.Response {
	user.OptionsSerialized.PreferredTheme = service.Theme
	if err := user.UpdateOptions(); err != nil {
		return serializer.DBErr("Topic switching failed", err)
	}

	return serializer.Response{}
}

func (service *AvatarService) Get(c *gin.Context) serializer.Response {
	uid, _ := c.Get("object_id")
	user, err := models.GetActivateUserByID(uid.(uint))
	if err != nil {
		return serializer.Err(serializer.CodeNotFound, "user is not exist", err)
	}
	if user.Avatar == "" {
		c.Status(404)
		return serializer.Response{}
	}

	sizes := map[string]string{
		"s": models.GetSettingByName("avatar_size_s"),
		"m": models.GetSettingByName("avatar_size_m"),
		"l": models.GetSettingByName("avatar_size_l"),
	}
	if user.Avatar == "gravatar" {
		server := models.GetSettingByName("gravatar_server")
		gravatarRoot, err := url.Parse(server)
		if err != nil {
			return serializer.Err(serializer.CodeInternalSetting, "cannot parse Gravatar server address", err)
		}
		emailLowered := strings.ToLower(user.Email)
		has := md5.Sum([]byte(emailLowered))
		avatar, _ := url.Parse(fmt.Sprintf("/avatar/%x?d=mm&s=%s", has, sizes[service.Size]))
		return serializer.Response{
			Code: -301,
			Data: gravatarRoot.ResolveReference(avatar).String(),
		}
	}

	if user.Avatar == "file" {
		avatarRoot := utils.RelativePath(models.GetSettingByName("avatar_path"))
		sizeToInt := map[string]string{
			"s": "0",
			"m": "1",
			"l": "2",
		}

		avatar, err := os.Open(filepath.Join(avatarRoot, fmt.Sprintf("avatar_%d_%s.png", user.ID, sizeToInt[service.Size])))
		if err != nil {
			c.Status(404)
			return serializer.Response{}
		}
		defer avatar.Close()
		http.ServeContent(c.Writer, c.Request, "avatar.png", user.UpdatedAt, avatar)
		return serializer.Response{}
	}
	c.Status(404)
	return serializer.Response{}
}

func (service *SettingListService) ListTasks(c *gin.Context, user *models.User) serializer.Response {
	tasks, total := models.ListTasks(user.ID, service.Page, 10, "updated_at desc")
	return serializer.BuildTaskList(tasks, total)
}

func (service *SettingService) Settings(c *gin.Context, user *models.User) serializer.Response {
	return serializer.Response{
		Data: map[string]interface{}{
			"uid":          user.ID,
			"homepage":     !user.OptionsSerialized.ProfileOff,
			"two_factor":   user.TwoFactor != "",
			"prefer_theme": user.OptionsSerialized.PreferredTheme,
			"themes":       models.GetSettingByName("themes"),
			"authn":        serializer.BuildWebAuthnList(user.WebAuthnCredentials()),
		},
	}
}

func (service *SettingService) Init2FA(c *gin.Context, user *models.User) serializer.Response {
	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      "Cloudreve",
		AccountName: user.Email,
	})
	if err != nil {
		return serializer.Err(serializer.CodeInternalSetting, "Failed to generate verification key", err)
	}

	utils.SetSession(c, map[string]interface{}{"2fa_init": key.Secret()})
	return serializer.Response{Data: key.Secret()}
}
