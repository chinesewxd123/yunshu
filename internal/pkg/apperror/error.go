package apperror

import "net/http"

type AppError struct {
	StatusCode int
	ErrorCode  string
	Message    string
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
		Message:    message,
	}
}

func IsAppError(err error) (*AppError, bool) {
	appErr, ok := err.(*AppError)
	return appErr, ok
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
