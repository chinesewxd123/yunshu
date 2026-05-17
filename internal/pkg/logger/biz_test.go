package logger

import (
	"errors"
	"testing"

	"yunshu/internal/config"
	"yunshu/internal/pkg/apperror"
)

func TestBizOpInternalVsClient(t *testing.T) {
	SetDefault(New(config.LogConfig{Level: "debug", Output: "console"}))
	b := Biz("test")

	b.Op("noop", nil)
	b.Op("client", apperror.BadRequest("bad"))
	b.Op("server", apperror.Internal("boom"))
}

func TestBizDisabledWithoutDefault(t *testing.T) {
	SetDefault(nil)
	b := Biz("test")
	b.Error("should not panic")
	b.Op("x", errors.New("e"))
}
