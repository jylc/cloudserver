package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"io"
	"strconv"
	"strings"
	"time"
)

type HMACAuth struct {
	SecretKey []byte
}

func (auth HMACAuth) Sign(body string, expires int64) string {
	h := hmac.New(sha256.New, auth.SecretKey)
	expiresTimeStamp := strconv.FormatInt(expires, 10)
	_, err := io.WriteString(h, body+":"+expiresTimeStamp)
	if err != nil {
		return ""
	}

	return base64.URLEncoding.EncodeToString(h.Sum(nil)) + ":" + expiresTimeStamp
}

func (auth HMACAuth) Check(body string, sign string) error {
	signSlice := strings.Split(sign, ":")
	if signSlice[len(signSlice)-1] == "" {
		return ErrExpiresMissing
	}

	expires, err := strconv.ParseInt(signSlice[len(signSlice)-1], 10, 64)
	if err != nil {
		return ErrAuthFailed.WithError(err)
	}

	if expires < time.Now().Unix() && expires != 0 {
		return ErrExpired
	}

	if auth.Sign(body, expires) != sign {
		return ErrAuthFailed
	}
	return nil
}
