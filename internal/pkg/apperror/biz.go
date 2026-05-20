package apperror

import (
	"strconv"
)

// NewBiz 构造带 HTTP 状态、数字业务码、OneX 风格 reason 的业务错误。
// message 面向用户（可为中文）；reason 为稳定枚举字符串（英文 PascalCase）。
func NewBiz(httpStatus, bizCode int, reason, message string) error {
	if reason == "" {
		reason = "Unknown"
	}
	return &AppError{
		StatusCode: httpStatus,
		ErrorCode:  strconv.Itoa(bizCode),
		Reason:     reason,
		Message:    message,
	}
}

// WithMetadata 为 AppError 附加 metadata（返回新 error，不修改原 error）。
func WithMetadata(err error, md map[string]any) error {
	appErr, ok := IsAppError(err)
	if !ok {
		return err
	}
	cp := *appErr
	if cp.Metadata == nil {
		cp.Metadata = make(map[string]any, len(md))
	}
	for k, v := range md {
		cp.Metadata[k] = v
	}
	return &cp
}
