package setting

import (
	"github.com/gin-gonic/gin"
	"github.com/jylc/cloudserver/models"
	"github.com/jylc/cloudserver/pkg/serializer"
	"github.com/jylc/cloudserver/pkg/utils"
)

type WebDAVListService struct {
}

type WebDAVAccountService struct {
	ID uint `uri:"id" binding:"required,min=1"`
}

type WebDAVAccountCreateService struct {
	Path string `json:"path" binding:"required,min=1,max=65535"`
	Name string `json:"name" binding:"required,min=1,max=255"`
}

type WebDAVMountCreateService struct {
	Path   string `json:"path" binding:"required,min=1,max=65535"`
	Policy string `json:"policy" binding:"required,min=1"`
}

func (service *WebDAVListService) Accounts(c *gin.Context, user *models.User) serializer.Response {
	accounts := models.ListWebDAVAccounts(user.ID)
	return serializer.Response{
		Data: map[string]interface{}{
			"accounts": accounts,
		},
	}
}

func (service *WebDAVAccountService) Delete(c *gin.Context, user *models.User) serializer.Response {
	models.DeleteWebDAVAccountByID(service.ID, user.ID)
	return serializer.Response{}
}

func (service *WebDAVAccountCreateService) Create(c *gin.Context, user *models.User) serializer.Response {
	account := models.Webdav{
		Name:     service.Name,
		Password: utils.RandStringRunes(32),
		UserID:   user.ID,
		Root:     service.Path,
	}
	if _, err := account.Create(); err != nil {
		return serializer.Err(serializer.CodeDBError, "create failed", err)
	}

	return serializer.Response{
		Data: map[string]interface{}{
			"id":         account.ID,
			"password":   account.Password,
			"created_at": account.CreatedAt,
		},
	}
}
