package middleware

import (
	"bytes"
	"encoding/json"
	"github.com/gin-gonic/gin"
	"github.com/jylc/cloudserver/models"
	"github.com/jylc/cloudserver/pkg/serializer"
	"github.com/jylc/cloudserver/pkg/utils"
	"github.com/mojocn/base64Captcha"
	"io"
	"io/ioutil"
)

const (
	captchaNotMatch = "CAPTCHA not match."
	captchaRefresh  = "Verification failed, please refresh the page and retry."
)

type req struct {
	CaptchaCode string `json:"captchaCode"`
	Ticket      string `json:"ticket"`
	Randstr     string `json:"randstr"`
}

func CaptchaRequired(configName string) gin.HandlerFunc {
	return func(c *gin.Context) {
		options := models.GetSettingByNames(configName,
			"captcha_type",
			"captcha_ReCaptchaSecret",
			"captcha_TCaptcha_SecretId",
			"captcha_TCaptcha_SecretKey",
			"captcha_TCaptcha_CaptchaAppId",
			"captcha_TCaptcha_AppSecretKey")
		isCaptchaRequired := models.IsTrueVal(options[configName])
		if isCaptchaRequired {
			var service req
			bodyCopy := new(bytes.Buffer)
			_, err := io.Copy(bodyCopy, c.Request.Body)
			if err != nil {
				c.JSON(200, serializer.Err(serializer.CodeCaptchaError, captchaNotMatch, err))
				c.Abort()
				return
			}

			bodyData := bodyCopy.Bytes()
			err = json.Unmarshal(bodyData, &service)
			if err != nil {
				c.JSON(200, serializer.Err(serializer.CodeCaptchaError, captchaNotMatch, err))
				c.Abort()
				return
			}
			c.Request.Body = ioutil.NopCloser(bytes.NewReader(bodyData))
			switch options["captcha_type"] {
			case "normal":
				captchaID := utils.GetSession(c, "captchaID")
				utils.DeleteSession(c, "captchaID")
				if captchaID == nil || !base64Captcha.VerifyCaptcha(captchaID.(string), service.CaptchaCode) {
					c.JSON(200, serializer.Err(serializer.CodeCaptchaError, captchaNotMatch, err))
					c.Abort()
					return
				}
				break
			}
		}
		c.Next()
	}
}
