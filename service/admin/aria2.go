package admin

import (
	"bytes"
	"encoding/json"
	"github.com/jylc/cloudserver/models"
	"github.com/jylc/cloudserver/pkg/auth"
	"github.com/jylc/cloudserver/pkg/request"
	"github.com/jylc/cloudserver/pkg/serializer"
	"net/url"
	"time"
)

type Aria2TestService struct {
	Server string           `json:"server"`
	RPC    string           `json:"rpc" binding:"required"`
	Secret string           `json:"secret"`
	Token  string           `json:"token"`
	Type   models.ModelType `json:"type"`
}

func (service *Aria2TestService) TestMaster() serializer.Response {
	res, err := aria2.TestRPCConnection(service.RPC, service.Token, 5)
	if err != nil {
		return serializer.ParamErr(err.Error(), err)
	}

	if res.Version == "" {
		return serializer.ParamErr("The RPC service returned an unexpected response", nil)
	}

	return serializer.Response{Data: res.Version}
}

func (service *Aria2TestService) TestSlave() serializer.Response {
	salve, err := url.Parse(service.Server)
	if err != nil {
		return serializer.ParamErr("Unable to resolve the slave address. "+err.Error(), nil)
	}
	controller, _ := url.Parse("/api/v3/slave/ping/aria2")
	service.Type = models.MasterNodeType
	bodyByte, _ := json.Marshal(service)

	r := request.NewClient()
	res, err := r.Request(
		"POST",
		slave.ResloveReference(controller).String(),
		bytes.NewReader(bodyByte),
		request.WithTimeout(time.Duration(10)*time.Second),
		request.WithCredential(
			auth.HMACAuth{SecretKey: []byte(service.Secret)},
			int64(models.GetIntSetting("slave_api_timeout", 60)),
		),
	).DecodeResponse()

	if err != nil {
		return serializer.ParamErr("No connection to slave, "+err.Error(), nil)
	}
	if res.Code != 0 {
		return serializer.ParamErr("Successfully received from the slave, but the slave returned: "+res.Msg, nil)
	}
	return serializer.Response{Data: res.Data.(string)}
}
