package onedrive

import "net/url"

type Credential struct {
	TokenType    string `json:"token_type"`
	ExpiresIn    int64  `json:"expires_in"`
	Scope        string `json:"scope"`
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	UserID       string `json:"user_id"`
}

type oauthEndpoint struct {
	token     url.URL
	authorize url.URL
}

type OAuthError struct {
	ErrorType        string `json:"error-type"`
	ErrorDescription string `json:"error_description"`
	CorrelationID    string `json:"correlation_id"`
}

func (err OAuthError) Error() string {
	return err.ErrorDescription
}
