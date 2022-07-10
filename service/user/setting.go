package user

import (
	"crypto/md5"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/jylc/cloudserver/models"
	"github.com/jylc/cloudserver/pkg/serializer"
	"github.com/jylc/cloudserver/pkg/utils"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

type Enable2FA struct {
	Code string `json:"code" binding:"required"`
}

type SettingService struct {
}

type AvatarService struct {
	Size string `uri:"size" binding:"required,eq=l|eq=m|eq=s"`
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
