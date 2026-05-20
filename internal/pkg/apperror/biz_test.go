package apperror

import (
	"errors"
	"net/http"
	"testing"
)

func TestNewBizReasonMetadata(t *testing.T) {
	err := NewBiz(http.StatusConflict, 20003, "EmailAlreadyRegistered", "该邮箱已被占用，请更换后重试")
	var app *AppError
	if !errors.As(err, &app) {
		t.Fatal("expected AppError")
	}
	if app.Reason != "EmailAlreadyRegistered" || app.ErrorCode != "20003" {
		t.Fatalf("reason/code: %+v", app)
	}

	withMD := WithMetadata(err, map[string]any{"email": "a@b.c"})
	if !errors.As(withMD, &app) || app.Metadata["email"] != "a@b.c" {
		t.Fatalf("metadata: %+v", app)
	}
}
