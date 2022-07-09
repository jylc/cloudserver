package serializer

import "github.com/gin-gonic/gin"

type AppError struct {
	Code     int
	Msg      string
	RawError error
}

func (err *AppError) WithError(raw error) AppError {
	err.RawError = raw
	return *err
}

func (err AppError) Error() string {
	return err.Msg
}

const (
	// CodeUserBaned 用户不活跃
	CodeUserBaned = 40017
	// CodeUserNotActivated 用户不活跃
	CodeUserNotActivated = 40018
	// CodeFeatureNotEnabled 此功能未开启
	CodeFeatureNotEnabled = 40019
	// CodeCredentialInvalid 凭证无效
	CodeCredentialInvalid = 40020
	// CodeCaptchaError 验证码错误
	CodeCaptchaError = 40026
	// CodeFailedSendEmail 邮件发送失败
	CodeFailedSendEmail = 40028
	// CodeEmailExisted 邮箱已被使用
	CodeEmailExisted = 40032
	// CodeEmailSent 邮箱已重新发送
	CodeEmailSent = 40033
	//CodeParamErr 各种奇奇怪怪的参数错误
	CodeParamErr = 40001
	// CodeEncryptError 加密失败
	CodeEncryptError = 50002
)

func Err(errCode int, msg string, err error) Response {
	if appError, ok := err.(AppError); ok {
		errCode = appError.Code
		err = appError.RawError
		msg = appError.Msg
	}

	res := Response{
		Code: errCode,
		Msg:  msg,
	}
	if err != nil && gin.Mode() != gin.ReleaseMode {
		res.Error = err.Error()
	}
	return res
}

func ParamErr(msg string, err error) Response {
	if msg == "" {
		msg = "parameter error"
	}
	return Err(CodeParamErr, msg, err)
}
