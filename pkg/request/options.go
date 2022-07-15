package request

import (
	"context"
	"github.com/jylc/cloudserver/pkg/auth"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type Option interface {
	apply(*options)
}

type options struct {
	timeout         time.Duration
	header          http.Header
	sign            auth.Auth
	signTTL         int64
	ctx             context.Context
	contentLength   int64
	masterMeta      bool
	endpoint        *url.URL
	slaveNodeID     string
	tpsLimiterToken string
	tps             float64
	tpsBurst        int
}

type optionFunc func(*options)

func (f optionFunc) apply(o *options) {
	f(o)
}

func newDefaultOption() *options {
	return &options{
		header:        http.Header{},
		timeout:       time.Duration(30) * time.Second,
		contentLength: -1,
		ctx:           context.Background(),
	}
}

func WithEndpoint(endpoint string) Option {
	if !strings.HasPrefix(endpoint, "/") {
		endpoint += "/"
	}
	endpointURL, _ := url.Parse(endpoint)
	return optionFunc(func(o *options) {
		o.endpoint = endpointURL
	})
}

func WithCredential(instance auth.Auth, ttl int64) Option {
	return optionFunc(func(o *options) {
		o.sign = instance
		o.signTTL = ttl
	})
}

func WithMasterMeta() Option {
	return optionFunc(func(o *options) {
		o.masterMeta = true
	})
}

func WithSlaveMeta(s string) Option {
	return optionFunc(func(o *options) {
		o.slaveNodeID = s
	})
}
func WithContext(c context.Context) Option {
	return optionFunc(func(o *options) {
		o.ctx = c
	})
}

func WithTimeout(t time.Duration) Option {
	return optionFunc(func(o *options) {
		o.timeout = t
	})
}

func WithContentLength(s int64) Option {
	return optionFunc(func(o *options) {
		o.contentLength = s
	})
}

func WithHeader(header http.Header) Option {
	return optionFunc(func(o *options) {
		for k, v := range header {
			o.header[k] = v
		}
	})
}

func WithoutHeader(header []string) Option {
	return optionFunc(func(o *options) {
		for _, v := range header {
			delete(o.header, v)
		}
	})
}
