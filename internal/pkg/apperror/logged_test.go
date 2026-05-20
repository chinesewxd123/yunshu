package apperror

import (
	"errors"
	"net/http"
	"testing"
)

func TestMarkLoggedSkipsBoundaryDuplicate(t *testing.T) {
	root := Internal("boom")
	wrapped := MarkLogged(root)
	if !AlreadyLogged(wrapped) {
		t.Fatal("expected AlreadyLogged")
	}
	app, ok := IsAppError(wrapped)
	if !ok || app.StatusCode != http.StatusInternalServerError {
		t.Fatalf("IsAppError: ok=%v status=%d", ok, app.StatusCode)
	}
}

func TestIsAppErrorUnwraps(t *testing.T) {
	root := BadRequest("bad")
	wrapped := errors.Join(MarkLogged(root), errors.New("noise"))
	app, ok := IsAppError(wrapped)
	if !ok || app.StatusCode != http.StatusBadRequest {
		t.Fatalf("got ok=%v status=%d", ok, app.StatusCode)
	}
}
