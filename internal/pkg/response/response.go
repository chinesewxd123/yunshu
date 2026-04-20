package response

import (
	"net/http"

	"go-permission-system/internal/pkg/apperror"

	"github.com/gin-gonic/gin"
)

type Body struct {
	Code      int    `json:"code"`
	ErrorCode string `json:"error_code,omitempty"`
	Message   string `json:"message"`
	Data      any    `json:"data,omitempty"`
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
	if appErr, ok := apperror.IsAppError(err); ok {
		c.Set("error_code", appErr.ErrorCode)
		c.Set("error_message", appErr.Message)
		c.JSON(appErr.StatusCode, Body{
			Code:      appErr.StatusCode,
			ErrorCode: appErr.ErrorCode,
			Message:   appErr.Message,
		})
		return
	}

	c.Set("error_code", "INTERNAL_ERROR")
	c.Set("error_message", "服务器内部错误")
	c.JSON(http.StatusInternalServerError, Body{
		Code:      http.StatusInternalServerError,
		ErrorCode: "INTERNAL_ERROR",
		Message:   "服务器内部错误",
	})
}
