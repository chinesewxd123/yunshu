package response

import (
	"net/http"

	"go-permission-system/internal/pkg/apperror"

	"github.com/gin-gonic/gin"
)

type Body struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
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
		c.JSON(appErr.StatusCode, Body{
			Code:    appErr.StatusCode,
			Message: appErr.Message,
		})
		return
	}

	c.JSON(http.StatusInternalServerError, Body{
		Code:    http.StatusInternalServerError,
		Message: "internal server error",
	})
}
