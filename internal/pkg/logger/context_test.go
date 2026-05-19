package logger

import (
	"bytes"
	"context"
	"log/slog"
	"strings"
	"testing"
)

func TestWithRequestIDInLogs(t *testing.T) {
	var buf bytes.Buffer
	h := slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo})
	Info := slog.New(wrapHandler(h, channelInfo))

	l := &Logger{Info: Info, Error: Info, SQL: Info}
	SetDefault(l)

	ctx := WithRequestID(context.Background(), "req-abc")
	Biz("test").W(ctx).Infow("hello", "k", "v")

	if !strings.Contains(buf.String(), "req-abc") {
		t.Fatalf("expected request_id in log, got: %s", buf.String())
	}
}

func TestWithUserInLogs(t *testing.T) {
	var buf bytes.Buffer
	h := slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo})
	Info := slog.New(wrapHandler(h, channelInfo))
	l := &Logger{Info: Info, Error: Info, SQL: Info}
	SetDefault(l)

	ctx := WithUser(context.Background(), 42, "alice")
	Biz("test").W(ctx).Infow("user action")

	out := buf.String()
	if !strings.Contains(out, "alice") || !strings.Contains(out, "42") {
		t.Fatalf("expected user fields in log, got: %s", out)
	}
}
