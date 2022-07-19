package node

import (
	"encoding/gob"
	"github.com/gin-gonic/gin"
	"github.com/jylc/cloudserver/models"
	"github.com/jylc/cloudserver/pkg/conf"
	"github.com/jylc/cloudserver/pkg/filesystem/driver/onedrive"
	"github.com/jylc/cloudserver/pkg/mq"
	"github.com/jylc/cloudserver/pkg/serializer"
)

type SlaveNotificationService struct {
	Subject string `uri:"subject" binding:"required"`
}

type OneDriveCredentialService struct {
	PolicyID int `uri:"id" binding:"required"`
}

func (s *SlaveNotificationService) HandleSlaveNotificationPush(c *gin.Context) serializer.Response {
	var msg mq.Message
	dec := gob.NewDecoder(c.Request.Body)
	if err := dec.Decode(&msg); err != nil {
		return serializer.ParamErr("cannot parse notification message", err)
	}
	mq.GlobalMQ.Publish(s.Subject, msg)
	return serializer.Response{}
}

func (s *OneDriveCredentialService) Get(c *gin.Context) serializer.Response {
	policy, err := models.GetPolicyByID(s.PolicyID)
	if err != nil {
		return serializer.Err(serializer.CodeNotFound, "Cannot found storage policy", err)
	}

	client, err := onedrive.NewClient(&policy)
	if err != nil {
		return serializer.Err(serializer.CodeInternalSetting, "Cannot initialize OneDrive client", err)
	}

	if err := client.UpdateCredential(c, conf.Sc.Role == "slave"); err != nil {
		return serializer.Err(serializer.CodeInternalSetting, "Cannot refresh OneDrive credential", err)
	}

	return serializer.Response{Data: client.Credential.AccessToken}
}
