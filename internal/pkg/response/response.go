package response

import (
	"net/http"

	"yunshu/internal/pkg/apperror"

	"github.com/gin-gonic/gin"
)

// Body 成功响应体。
type Body struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

// ErrorBody 错误响应体（对齐 [OneX 错误规范](https://konglingfei.com/onex/convention/error.html)）。
type ErrorBody struct {
	Code      int            `json:"code"`
	Reason    string         `json:"reason"`
	Message   string         `json:"message"`
	ErrorCode string         `json:"error_code,omitempty"`
	Metadata  map[string]any `json:"metadata,omitempty"`
}

func Success(c *gin.Context, data any) {
	c.JSON(http.StatusOK, Body{
		Code:    http.StatusOK,
		Message: "success",
		Data:    data,
	})
}

func Created(c *gin.Context, data any) {
	c.JSON(http.StatusCreated, Body{
		Code:    http.StatusCreated,
		Message: "success",
		Data:    data,
	})
}

func Error(c *gin.Context, err error) {
	logHTTPError(c, err)
	if appErr, ok := apperror.IsAppError(err); ok {
		c.Set("error_code", appErr.ErrorCode)
		c.Set("error_message", appErr.Message)
		c.JSON(appErr.StatusCode, ErrorBody{
			Code:      appErr.StatusCode,
			Reason:    appErr.Reason,
			Message:   appErr.Message,
			ErrorCode: appErr.ErrorCode,
			Metadata:  appErr.Metadata,
		})
		return
	}

	c.Set("error_code", "INTERNAL_ERROR")
	c.Set("error_message", "服务器内部错误")
	c.JSON(http.StatusInternalServerError, ErrorBody{
		Code:      http.StatusInternalServerError,
		Reason:    "InternalError",
		Message:   "服务器内部错误",
		ErrorCode: "INTERNAL_ERROR",
	})
}
