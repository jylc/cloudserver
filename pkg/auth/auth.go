package auth

import (
	"bytes"
	"fmt"
	"github.com/jylc/cloudserver/models"
	"github.com/jylc/cloudserver/pkg/serializer"
	"io/ioutil"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"
)

var (
	ErrAuthFailed        = serializer.NewError(serializer.CodeNoPermissionErr, "authorization failed", nil)
	ErrAuthHeaderMissing = serializer.NewError(serializer.CodeNoPermissionErr, "authorization header is missing", nil)
	ErrExpiresMissing    = serializer.NewError(serializer.CodeNoPermissionErr, "expire timestamp is missing", nil)
	ErrExpired           = serializer.NewError(serializer.CodeSignExpired, "sign expired", nil)
)

const CrHeaderPrefix = "X-Cr-"

var General Auth

type Auth interface {
	Sign(body string, expires int64) string
	Check(body string, sign string) error
}

func SignRequest(instance Auth, r *http.Request, expires int64) *http.Request {
	if expires > 0 {
		expires += time.Now().Unix()
	}

	sign := instance.Sign(getSignContent(r), expires)
	r.Header["Authorization"] = []string{"Bearer " + sign}
	return r
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

func CheckRequest(instance Auth, r *http.Request) error {
	var (
		sign []string
		ok   bool
	)
	if sign, ok = r.Header["Authorization"]; !ok || len(sign) == 0 {
		return ErrAuthHeaderMissing
	}
	sign[0] = strings.TrimPrefix(sign[0], "Bearer ")
	return instance.Check(getSignContent(r), sign[0])
}

func getSignContent(r *http.Request) (rawSignString string) {
	var body []byte
	if !strings.Contains(r.URL.Path, "/api/v2/slave/upload/") {
		if r.Body != nil {
			body, _ = ioutil.ReadAll(r.Body)
			_ = r.Body.Close()
			r.Body = ioutil.NopCloser(bytes.NewReader(body))
		}
	}

	var signedHeader []string
	for k, _ := range r.Header {
		if strings.HasPrefix(k, CrHeaderPrefix) && k != CrHeaderPrefix+"Filename" {
			signedHeader = append(signedHeader, fmt.Sprintf("%s=%s", k, r.Header.Get(k)))
		}
	}
	sort.Strings(signedHeader)
	rawSignString = serializer.NewRequestSignString(r.URL.Path, strings.Join(signedHeader, "&"), string(body))
	return
}

func CheckURI(instance Auth, url *url.URL) error {
	queries := url.Query()
	sign := queries.Get("sign")
	queries.Del("sign")
	url.RawQuery = queries.Encode()
	return instance.Check(url.Path, sign)
}

func Init() {
	var secretKey string
	secretKey = models.GetSettingByName("secret_key")
	General = HMACAuth{
		SecretKey: []byte(secretKey),
	}
}
