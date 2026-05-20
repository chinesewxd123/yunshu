package apperror

import (
	"errors"
	"net/http"
	"testing"
)

func TestLegacyReasonOnNew(t *testing.T) {
	err := New(http.StatusNotFound, "missing")
	var app *AppError
	if !errors.As(err, &app) || app.Reason != "NotFound" {
		t.Fatalf("reason=%q", app.Reason)
	}
}
