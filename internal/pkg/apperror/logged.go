package apperror

import "errors"

// loggedMarker 表示错误已在 Service 层写入 error 级日志，边界层（HTTP/gRPC）应跳过重复记录。
type loggedMarker struct {
	cause error
}

func (e *loggedMarker) Error() string { return e.cause.Error() }
func (e *loggedMarker) Unwrap() error { return e.cause }

// MarkLogged 标记该错误链已在业务层记录，避免 API/gRPC 重复写 error.log。
func MarkLogged(err error) error {
	if err == nil {
		return nil
	}
	var m *loggedMarker
	if errors.As(err, &m) {
		return err
	}
	return &loggedMarker{cause: err}
}

// AlreadyLogged 判断错误是否已在业务层记录。
func AlreadyLogged(err error) bool {
	var m *loggedMarker
	return errors.As(err, &m)
}
