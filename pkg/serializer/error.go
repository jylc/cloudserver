package serializer

import (
	"errors"
	"github.com/gin-gonic/gin"
)

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
	// CodeNotFullySuccess 未完全成功
	CodeNotFullySuccess = 203
	// CodeCheckLogin 未登录
	CodeCheckLogin = 401
	// CodeNoPermissionErr 未授权访问
	CodeNoPermissionErr = 403
	// CodeNotFound 资源未找到
	CodeNotFound = 404
	// CodeConflict 资源冲突
	CodeConflict = 409
	// CodeUploadFailed 上传出错
	CodeUploadFailed = 40002
	// CodeCreateFolderFailed 目录创建失败
	CodeCreateFolderFailed = 40003
	// CodeSignExpired 签名过期
	CodeSignExpired = 40005
	// CodePolicyNotAllowed 当前存储策略不允许
	CodePolicyNotAllowed = 40006
	// CodeAdminRequired 非管理用户组
	CodeAdminRequired = 40008
	// CodeMasterNotFound 主机节点未注册
	CodeMasterNotFound = 40009
	// CodeUploadSessionExpired 上传会话已过期
	CodeUploadSessionExpired = 400011
	// CodeInvalidChunkIndex 无效的分片序号
	CodeInvalidChunkIndex = 400012
	// CodeInvalidContentLength 无效的正文长度
	CodeInvalidContentLength = 400013
	// CodeBatchSourceSize 超出批量获取外链限制
	CodeBatchSourceSize = 40014
	// CodeBatchAria2Size 超出最大 Aria2 任务数量限制
	CodeBatchAria2Size = 40015
	// CodeParentNotExist 父目录不存在
	CodeParentNotExist = 40016
	// CodeUserBaned 用户不活跃
	CodeUserBaned = 40017
	// CodeUserNotActivated 用户不活跃
	CodeUserNotActivated = 40018
	// CodeFeatureNotEnabled 此功能未开启
	CodeFeatureNotEnabled = 40019
	// CodeCredentialInvalid 凭证无效
	CodeCredentialInvalid = 40020
	// CodeUserNotFound 用户不存在
	CodeUserNotFound = 40021
	// Code2FACodeErr 二步验证代码错误
	Code2FACodeErr = 40022
	// CodeLoginSessionNotExist 登录会话不存在
	CodeLoginSessionNotExist = 40023
	// CodeInitializeAuthn 无法初始化 WebAuthn
	CodeInitializeAuthn = 40024
	// CodeWebAuthnCredentialError WebAuthn 凭证无效
	CodeWebAuthnCredentialError = 40025
	// CodeCaptchaError 验证码错误
	CodeCaptchaError = 40026
	// CodeFailedSendEmail 邮件发送失败
	CodeFailedSendEmail = 40028
	// CodeInvalidTempLink 临时链接无效
	CodeInvalidTempLink = 40029
	// CodeEmailExisted 邮箱已被使用
	CodeEmailExisted = 40032
	// CodeEmailSent 邮箱已重新发送
	CodeEmailSent = 40033
	// CodeUserCannotActivate 用户无法激活
	CodeUserCannotActivate = 40034
	//CodeParamErr 各种奇奇怪怪的参数错误
	CodeParamErr = 40001
	// CodeObjectExist 对象已存在
	CodeObjectExist = 40004
	// CodeGroupNotAllowed 用户组无法进行此操作
	CodeGroupNotAllowed = 40007
	// CodeDBError 数据库操作失败
	CodeDBError = 50001
	// CodeEncryptError 加密失败
	CodeEncryptError = 50002
	// CodeIOFailed IO操作失败
	CodeIOFailed = 50004
	// CodeInternalSetting 内部设置参数错误
	CodeInternalSetting = 50005
	// CodeCacheOperation 缓存操作失败
	CodeCacheOperation = 50006
	// CodeCallbackError 回调失败
	CodeCallbackError = 50007
	// CodeNotSet 未定错误，后续尝试从error中获取
	CodeNotSet = -1
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

func DBErr(msg string, err error) Response {
	if msg == "" {
		msg = "database operation failed"
	}
	return Err(CodeDBError, msg, err)
}

func NewError(code int, msg string, err error) AppError {
	return AppError{
		Code:     code,
		Msg:      msg,
		RawError: err,
	}
}

func NewErrorFromResponse(resp *Response) AppError {
	return AppError{
		Code:     resp.Code,
		Msg:      resp.Msg,
		RawError: errors.New(resp.Error),
	}
}
