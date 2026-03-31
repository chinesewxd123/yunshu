package apperror

import "net/http"

type AppError struct {
	StatusCode int
	Message    string
}

func (e *AppError) Error() string {
	return e.Message
}

func New(statusCode int, message string) error {
	return &AppError{
		StatusCode: statusCode,
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
