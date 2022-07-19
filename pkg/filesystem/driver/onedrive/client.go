package onedrive

import (
	"errors"
	"github.com/jylc/cloudserver/models"
	"github.com/jylc/cloudserver/pkg/cluster"
	"github.com/jylc/cloudserver/pkg/request"
)

var (
	// ErrAuthEndpoint 无法解析授权端点地址
	ErrAuthEndpoint = errors.New("无法解析授权端点地址")
	// ErrInvalidRefreshToken 上传策略无有效的RefreshToken
	ErrInvalidRefreshToken = errors.New("上传策略无有效的RefreshToken")
	// ErrDeleteFile 无法删除文件
	ErrDeleteFile = errors.New("无法删除文件")
	// ErrClientCanceled 客户端取消操作
	ErrClientCanceled = errors.New("客户端取消操作")
)

type Client struct {
	Endpoints  *Endpoints
	Policy     *models.Policy
	Credential *Credential

	ClientID     string
	ClientSecret string
	Redirect     string

	Request           request.Client
	ClusterController cluster.Controller
}

type Endpoints struct {
	OAuthURL       string
	OAuthEndpoints *oauthEndpoint
	EndpointURL    string
	isInChina      bool
	DriverResource string
}

func NewClient(policy *models.Policy) (*Client, error) {
	client := &Client{
		Endpoints: &Endpoints{
			OAuthURL:       policy.BaseURL,
			EndpointURL:    policy.Server,
			DriverResource: policy.OptionsSerialized.OdDriver,
		},
		Credential: &Credential{
			RefreshToken: policy.AccessKey,
		},
		Policy:            policy,
		ClientID:          policy.BucketName,
		ClientSecret:      policy.SecretKey,
		Redirect:          policy.OptionsSerialized.OdRedirect,
		Request:           request.NewClient(),
		ClusterController: cluster.DefaultController,
	}
	if client.Endpoints.DriverResource == "" {
		client.Endpoints.DriverResource = "me/drive"
	}

	oauthBase := client.getOAuthEndpoint()
	if oauthBase == nil {
		return nil, ErrAuthEndpoint
	}
	client.Endpoints.OAuthEndpoints = oauthBase
	return client, nil
}
