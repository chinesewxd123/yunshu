package handler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"yunshu/internal/pkg/apperror"
	"yunshu/internal/pkg/response"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

func bindErrorMessage(err error) string {
	var verr validator.ValidationErrors
	if errors.As(err, &verr) && len(verr) > 0 {
		field := fieldLabel(verr[0].Field())
		return fmt.Sprintf("%s", validationMessage(field, verr[0]))
	}
	var typeErr *json.UnmarshalTypeError
	if errors.As(err, &typeErr) {
		if strings.EqualFold(typeErr.Field, "sort") {
			return "排序字段不能为空或非数字（若清空过「排序」，请填 0 或留默认值）"
		}
		return fmt.Sprintf("JSON 字段 %s 类型不正确", typeErr.Field)
	}
	var syn *json.SyntaxError
	if errors.As(err, &syn) {
		return "请求体 JSON 语法错误，请检查是否截断或未转义字符"
	}
	return "请求参数格式错误，请检查后重试"
}

func fieldLabel(field string) string {
	switch strings.ToLower(field) {
	case "username":
		return "用户名"
	case "email":
		return "邮箱"
	case "nickname":
		return "昵称"
	case "password":
		return "密码"
	case "oldpassword", "old_password":
		return "旧密码"
	case "newpassword", "new_password":
		return "新密码"
	case "code":
		return "验证码"
	case "scene":
		return "验证码场景"
	case "captchakey", "captcha_key":
		return "验证码键"
	case "name":
		return "名称"
	case "departmentid", "department_id":
		return "部门"
	case "parentid", "parent_id":
		return "上级部门"
	case "value":
		return "字典值"
	default:
		return field
	}
}

func validationMessage(field string, fe validator.FieldError) string {
	switch fe.Tag() {
	case "required":
		return field + "不能为空"
	case "email":
		return field + "格式不正确"
	case "numeric":
		return field + "必须为数字"
	case "len":
		return fmt.Sprintf("%s长度必须为%s位", field, fe.Param())
	case "min":
		return fmt.Sprintf("%s长度不能小于%s", field, fe.Param())
	case "max":
		return fmt.Sprintf("%s长度不能超过%s", field, fe.Param())
	case "oneof":
		return fmt.Sprintf("%s取值不合法（允许值：%s）", field, fe.Param())
	default:
		return field + "参数不合法"
	}
}

func handleQuery[T any, R any](c *gin.Context, call func(context.Context, T) (R, error)) {
	var req T
	if err := c.ShouldBindQuery(&req); err != nil {
		response.Error(c, apperror.BadRequest(bindErrorMessage(err)))
		return
	}
	data, err := call(c.Request.Context(), req)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, data)
}

func bindQuery[T any](c *gin.Context) (T, bool) {
	var req T
	if err := c.ShouldBindQuery(&req); err != nil {
		response.Error(c, apperror.BadRequest(bindErrorMessage(err)))
		return req, false
	}
	return req, true
}

func handleJSON[T any, R any](c *gin.Context, call func(context.Context, T) (R, error)) {
	var req T
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, apperror.BadRequest(bindErrorMessage(err)))
		return
	}
	data, err := call(c.Request.Context(), req)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, data)
}

func handleJSONCreated[T any, R any](c *gin.Context, call func(context.Context, T) (R, error)) {
	var req T
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, apperror.BadRequest(bindErrorMessage(err)))
		return
	}
	data, err := call(c.Request.Context(), req)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Created(c, data)
}

func handleQueryOK[T any](c *gin.Context, okData any, call func(context.Context, T) error) {
	var req T
	if err := c.ShouldBindQuery(&req); err != nil {
		response.Error(c, apperror.BadRequest(bindErrorMessage(err)))
		return
	}
	if err := call(c.Request.Context(), req); err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, okData)
}

func handleJSONOK[T any](c *gin.Context, okData any, call func(context.Context, T) error) {
	var req T
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, apperror.BadRequest(bindErrorMessage(err)))
		return
	}
	if err := call(c.Request.Context(), req); err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, okData)
}

func handleQueryWithKind[T any, R any](c *gin.Context, call func(context.Context, string, T) (R, error)) {
	kind := c.Query("kind")
	var req T
	if err := c.ShouldBindQuery(&req); err != nil {
		response.Error(c, apperror.BadRequest(bindErrorMessage(err)))
		return
	}
	data, err := call(c.Request.Context(), kind, req)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, data)
}

func handleQueryWithKindOK[T any](c *gin.Context, okData any, call func(context.Context, string, T) error) {
	kind := c.Query("kind")
	var req T
	if err := c.ShouldBindQuery(&req); err != nil {
		response.Error(c, apperror.BadRequest(bindErrorMessage(err)))
		return
	}
	if err := call(c.Request.Context(), kind, req); err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, okData)
}
