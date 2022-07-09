package controllers

import (
	"github.com/gin-gonic/gin"
	"github.com/jylc/cloudserver/models"
	"github.com/jylc/cloudserver/pkg/conf"
	"github.com/jylc/cloudserver/pkg/serializer"
	"github.com/jylc/cloudserver/pkg/utils"
	"github.com/mojocn/base64Captcha"
)

func Ping(c *gin.Context) {
	version := conf.BackendVersion
	c.JSON(200, serializer.Response{Code: 200, Data: version})
}

func Captcha(c *gin.Context) {
	options := models.GetSettingByNames(
		"captcha_IsShowHollowLine",
		"captcha_IsShowNoiseDot",
		"captcha_IsShowNoiseText",
		"captcha_IsShowSlimeLine",
		"captcha_IsShowSineLine",
	)

	var configD = base64Captcha.ConfigCharacter{
		Height: models.GetIntSetting("captcha_height", 60),
		Width:  models.GetIntSetting("captcha_width", 240),
		//const CaptchaModeNumber:数字,CaptchaModeAlphabet:字母,CaptchaModeArithmetic:算术,CaptchaModeNumberAlphabet:数字字母混合.
		Mode:               models.GetIntSetting("captcha_mode", 3),
		ComplexOfNoiseText: models.GetIntSetting("captcha_ComplexOfNoiseText", 0),
		ComplexOfNoiseDot:  models.GetIntSetting("captcha_ComplexOfNoiseDot", 0),
		IsShowHollowLine:   models.IsTrueVal(options["captcha_IsShowHollowLine"]),
		IsShowNoiseDot:     models.IsTrueVal(options["captcha_IsShowNoiseDot"]),
		IsShowNoiseText:    models.IsTrueVal(options["captcha_IsShowNoiseText"]),
		IsShowSlimeLine:    models.IsTrueVal(options["captcha_IsShowSlimeLine"]),
		IsShowSineLine:     models.IsTrueVal(options["captcha_IsShowSineLine"]),
		CaptchaLen:         models.GetIntSetting("captcha_CaptchaLen", 6),
	}
	idKeyD, capD := base64Captcha.GenerateCaptcha("", configD)
	utils.SetSession(c, map[string]interface{}{
		"captchaID": idKeyD,
	})
	base64stringD := base64Captcha.CaptchaWriteToBase64Encoding(capD)
	c.JSON(200, serializer.Response{
		Code: 0,
		Data: base64stringD,
	})
}

func SiteConfig(c *gin.Context) {
	siteConfig := models.GetSettingByNames(
		"siteName",
		"login_captcha",
		"reg_captcha",
		"email_active",
		"forget_captcha",
		"email_active",
		"themes",
		"defaultTheme",
		"home_view_method",
		"share_view_method",
		"authn_enabled",
		"captcha_ReCaptchaKey",
		"captcha_type",
		"captcha_TCaptcha_CaptchaAppId",
		"register_enabled",
	)
	user, _ := c.Get("user")
	if user, ok := user.(*models.User); ok {
		c.JSON(200, serializer.BuildSiteConfig(siteConfig, user))
		return
	}
	c.JSON(200, serializer.BuildSiteConfig(siteConfig, nil))
}
