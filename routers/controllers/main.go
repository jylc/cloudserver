package controllers

import (
	"encoding/json"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/jylc/cloudserver/models"
	"github.com/jylc/cloudserver/pkg/serializer"
)

func ParamErrorMsg(field string, tag string) string {
	fieldMap := map[string]string{
		"UserName": "Email",
		"Password": "Password",
		"Path":     "Path",
		"SourceID": "Source resource",
		"URL":      "URL",
		"Nick":     "NickName",
	}

	tagMap := map[string]string{
		"required": "cannot be empty",
		"min":      "too short",
		"max":      "too long",
		"email":    "format error",
	}
	fieldVal, findField := fieldMap[field]
	if !findField {
		fieldVal = field
	}
	tagVal, findTag := tagMap[tag]
	if findTag {
		return fieldVal + " " + tagVal
	}
	return ""
}

func ErrorResponse(err error) serializer.Response {
	if ve, ok := err.(validator.ValidationErrors); ok {
		for _, e := range ve {
			return serializer.ParamErr(
				ParamErrorMsg(e.Field(), e.Tag()),
				err)
		}
	}
	if _, ok := err.(*json.UnmarshalTypeError); ok {
		return serializer.ParamErr("JSON marshall error", err)
	}
	return serializer.ParamErr("Parameter error", err)
}

func CurrentUser(c *gin.Context) *models.User {
	if user, _ := c.Get("user"); user != nil {
		if u, ok := user.(*models.User); ok {
			return u
		}
	}
	return nil
}
