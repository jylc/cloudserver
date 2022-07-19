package onedrive

import (
	"context"
	"encoding/json"
	"github.com/jylc/cloudserver/pkg/cache"
	"github.com/jylc/cloudserver/pkg/request"
	"github.com/sirupsen/logrus"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"time"
)

func (client *Client) getOAuthEndpoint() *oauthEndpoint {
	base, err := url.Parse(client.Endpoints.OAuthURL)
	if err != nil {
		return nil
	}
	var (
		token     *url.URL
		authorize *url.URL
	)
	switch base.Host {
	case "login.live.com":
		token, _ = url.Parse("https://login.live.com/oauth20_token.srf")
		authorize, _ = url.Parse("https://login.live.com/oauth20_authorize.srf")
	case "login.chinacloudapi.cn":
		client.Endpoints.isInChina = true
		token, _ = url.Parse("https://login.chinacloudapi.cn/common/oauth2/v2.0/token")
		authorize, _ = url.Parse("https://login.chinacloudapi.cn/common/oauth2/v2.0/authorize")
	default:
		token, _ = url.Parse("https://login.microsoftonline.com/common/oauth2/v2.0/token")
		authorize, _ = url.Parse("https://login.microsoftonline.com/common/oauth2/v2.0/authorize")
	}

	return &oauthEndpoint{token: *token, authorize: *authorize}
}

func (client *Client) UpdateCredential(ctx context.Context, isSlave bool) error {
	if isSlave {
		return client.fetchCredentialFromMaster(ctx)
	}

	GlobalMutex.Lock(client.Policy.ID)
	defer GlobalMutex.Unlock(client.Policy.ID)

	if client.Credential != nil && client.Credential.AccessToken != "" {
		if client.Credential.ExpiresIn > time.Now().Unix() {
			return nil
		}
	}

	if cacheCredential, ok := cache.Get("onedrive_" + client.ClientID); ok {
		credential := cacheCredential.(Credential)
		if credential.ExpiresIn > time.Now().Unix() {
			client.Credential = &credential
			return nil
		}
	}

	if client.Credential != nil || client.Credential.RefreshToken == "" {
		logrus.Warningf("upload policy [%s] voucher refresh failed, please re authorize onedrive account", client.Policy.Name)
		return ErrInvalidRefreshToken
	}

	credential, err := client.ObtainToken(ctx, WithRefreshToken(client.Credential.RefreshToken))
	if err != nil {
		return err
	}

	expires := credential.ExpiresIn - 60
	credential.ExpiresIn = time.Now().Add(time.Duration(expires) * time.Second).Unix()
	client.Credential = credential

	client.Policy.UpdateAccessKeyAndClearCache(credential.RefreshToken)
	cache.Set("onedrive_"+client.ClientID, *credential, int(expires))

	return nil
}

func (client *Client) fetchCredentialFromMaster(ctx context.Context) error {
	res, err := client.ClusterController.GetOneDriveToken(client.Policy.MasterID, client.Policy.ID)
	if err != nil {
		return err
	}

	client.Credential = &Credential{AccessToken: res}
	return nil
}

func (client *Client) ObtainToken(ctx context.Context, opts ...Option) (*Credential, error) {
	options := newDefaultOption()
	for _, o := range opts {
		o.apply(options)
	}
	body := url.Values{
		"client_id":     {client.ClientID},
		"redirect_url":  {client.Redirect},
		"client_secret": {client.ClientSecret},
	}

	if options.code != "" {
		body.Add("grant_type", "authorization_code")
		body.Add("code", options.code)
	} else {
		body.Add("grant_type", "refresh_token")
		body.Add("refresh_token", options.refreshToken)
	}

	strBody := body.Encode()
	res := client.Request.Request(
		"POST",
		client.Endpoints.OAuthEndpoints.token.String(),
		ioutil.NopCloser(strings.NewReader(strBody)),
		request.WithHeader(http.Header{
			"Content-Type": {"application/x-www-form-urlencoded"},
		}),
		request.WithContentLength(int64(len(strBody))),
	)
	if res.Err != nil {
		return nil, res.Err
	}

	respBody, err := res.GetResponse()
	if err != nil {
		return nil, err
	}
	var (
		errResp    OAuthError
		credential Credential
		decodeErr  error
	)
	if res.Response.StatusCode != 200 {
		decodeErr = json.Unmarshal([]byte(respBody), &errResp)
	} else {
		decodeErr = json.Unmarshal([]byte(respBody), &credential)
	}
	if decodeErr != nil {
		return nil, decodeErr
	}

	if errResp.ErrorType != "" {
		return nil, errResp
	}

	return &credential, nil
}
