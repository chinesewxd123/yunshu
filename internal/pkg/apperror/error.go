package apperror

import (
	"errors"
	"net/http"
)

type AppError struct {
	StatusCode int
	ErrorCode  string // 数字业务码字符串，兼容历史字段 error_code
	Reason     string // OneX 风格稳定枚举，如 UserNotFound
	Message    string
	Metadata   map[string]any
}

func (e *AppError) Error() string {
	return e.Message
}

func New(statusCode int, message string, code ...string) error {
	errCode := defaultCodeByStatus(statusCode)
	if len(code) > 0 && code[0] != "" {
		errCode = code[0]
	}
	return &AppError{
		StatusCode: statusCode,
		ErrorCode:  errCode,
		Reason:     legacyReason(errCode),
		Message:    message,
	}
}

// legacyReason 将历史字符串 error_code（如 BAD_REQUEST）映射为 OneX reason。
func legacyReason(errCode string) string {
	switch errCode {
	case "BAD_REQUEST":
		return "BadRequest"
	case "UNAUTHORIZED":
		return "Unauthorized"
	case "FORBIDDEN":
		return "Forbidden"
	case "NOT_FOUND":
		return "NotFound"
	case "CONFLICT":
		return "Conflict"
	case "TOO_MANY_REQUESTS":
		return "TooManyRequests"
	case "INTERNAL_ERROR":
		return "InternalError"
	default:
		if errCode == "" || errCode == "UNKNOWN_ERROR" {
			return "UnknownError"
		}
		return errCode
	}
}

func IsAppError(err error) (*AppError, bool) {
	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr, true
	}
	return nil, false
}

func BadRequest(message string) error {
	return New(http.StatusBadRequest, message)
}

func Unauthorized(message string) error {
	return New(http.StatusUnauthorized, message)
}

func Forbidden(message string) error {
	return New(http.StatusForbidden, message)
}

func NotFound(message string) error {
	return New(http.StatusNotFound, message)
}

func Conflict(message string) error {
	return New(http.StatusConflict, message)
}

func Internal(message string) error {
	return New(http.StatusInternalServerError, message)
}

func defaultCodeByStatus(statusCode int) string {
	switch statusCode {
	case http.StatusBadRequest:
		return "BAD_REQUEST"
	case http.StatusUnauthorized:
		return "UNAUTHORIZED"
	case http.StatusForbidden:
		return "FORBIDDEN"
	case http.StatusNotFound:
		return "NOT_FOUND"
	case http.StatusConflict:
		return "CONFLICT"
	case http.StatusTooManyRequests:
		return "TOO_MANY_REQUESTS"
	case http.StatusInternalServerError:
		return "INTERNAL_ERROR"
	default:
		return "UNKNOWN_ERROR"
	}
}
