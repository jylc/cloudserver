package auth

import (
	"net/url"
	"time"
)

const CrHeaderPrefix = "X-Cr-"

var General Auth

type Auth interface {
	Sign(body string, expires int64) string
	Check(body string, sign string) error
}

func SignURI(instance Auth, uri string, expires int64) (*url.URL, error) {
	if expires != 0 {
		expires += time.Now().Unix()
	}
	base, err := url.Parse(uri)
	if err != nil {
		return nil, err
	}
	sign := instance.Sign(base.Path, expires)
	queries := base.Query()
	queries.Set("sign", sign)
	base.RawQuery = queries.Encode()
	return base, nil
}
