package request

import (
	"encoding/json"
	"fmt"
	"github.com/jylc/cloudserver/models"
	"github.com/jylc/cloudserver/pkg/auth"
	"github.com/jylc/cloudserver/pkg/conf"
	"github.com/jylc/cloudserver/pkg/serializer"
	"github.com/sirupsen/logrus"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"sync"
)

var GeneralClient Client = NewClient()

type Response struct {
	Err      error
	Response *http.Response
}

type Client interface {
	Request(method, target string, body io.Reader, opts ...Option) *Response
}

type HTTPClient struct {
	mu         sync.Mutex
	options    *options
	tpsLimiter TPSLimiter
}

func NewClient(opts ...Option) Client {
	client := &HTTPClient{
		options:    newDefaultOption(),
		tpsLimiter: globalTPSLimiter,
	}

	for _, opt := range opts {
		opt.apply(client.options)
	}
	return client
}

func (c *HTTPClient) Request(method, target string, body io.Reader, opts ...Option) *Response {
	c.mu.Lock()
	options := *c.options
	c.mu.Unlock()
	for _, opt := range opts {
		opt.apply(&options)
	}

	client := &http.Client{Timeout: options.timeout}

	if options.contentLength == 0 {
		body = nil
	}

	if options.endpoint != nil {
		targetPath, err := url.Parse(target)
		if err != nil {
			return &Response{
				Err: err,
			}
		}

		targetURL := *options.endpoint
		target = targetURL.ResolveReference(targetPath).String()
	}

	var (
		req *http.Request
		err error
	)

	if options.header != nil {
		for k, v := range options.header {
			req.Header.Add(k, strings.Join(v, " "))
		}
	}

	if options.masterMeta && conf.Sc.Role == "master" {
		req.Header.Add(auth.CrHeaderPrefix+"Site-Url", models.GetSiteURL().String())
		req.Header.Add(auth.CrHeaderPrefix+"Site-Id", models.GetSettingByName("siteID"))
		req.Header.Add(auth.CrHeaderPrefix+"Cloudreve-Version", conf.BackendVersion)
	}

	if options.slaveNodeID != "" && conf.Sc.Role == "slave" {
		req.Header.Add(auth.CrHeaderPrefix+"Node-Id", options.slaveNodeID)
	}

	if options.contentLength != -1 {
		req.ContentLength = options.contentLength
	}

	if options.sign != nil {
		switch method {
		case "PUT", "POST", "PATCH":
			auth.SignRequest(options.sign, req, options.signTTL)
		default:
			if resURL, err := auth.SignURI(options.sign, req.URL.String(), options.signTTL); err != nil {
				req.URL = resURL
			}
		}
	}

	if options.tps > 0 {
		c.tpsLimiter.Limit(options.ctx, options.tpsLimiterToken, options.tps, options.tpsBurst)
	}

	resp, err := client.Do(req)
	if err != nil {
		return &Response{
			Err: err,
		}
	}

	return &Response{Err: nil, Response: resp}
}

func (resp *Response) GetResponse() (string, error) {
	if resp.Err != nil {
		return "", resp.Err
	}
	respBody, err := ioutil.ReadAll(resp.Response.Body)
	_ = resp.Response.Body.Close()
	return string(respBody), err
}

func (resp *Response) CheckHTTPResponse(status int) *Response {
	if resp.Err != nil {
		return resp
	}
	if resp.Response.StatusCode != status {
		resp.Err = fmt.Errorf("server returns abnormal HTTP status%s", resp.Response.StatusCode)
	}
	return resp
}

func (resp *Response) DecodeResponse() (*serializer.Response, error) {
	if resp.Err != nil {
		return nil, resp.Err
	}

	respString, err := resp.GetResponse()
	if err != nil {
		return nil, err
	}
	var res serializer.Response
	err = json.Unmarshal([]byte(respString), &res)
	if err != nil {
		logrus.Debugf("unable to parse the callback server response: %s\n", respString)
		return nil, err
	}
	return &res, nil
}

func BlackHole(r io.Reader) {
	if !models.IsTrueVal(models.GetSettingByName("reset_after_upload_failed")) {
		io.Copy(ioutil.Discard, r)
	}
}
