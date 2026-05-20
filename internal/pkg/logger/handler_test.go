package logger

import (
	"bytes"
	"context"
	"log/slog"
	"testing"
)

func TestLevelWindowHandler(t *testing.T) {
	var buf bytes.Buffer
	base := slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug})
	h := wrapHandler(base, channelInfo)
	log := slog.New(h)

	log.Debug("debug-msg")
	log.Info("info-msg")
	log.Warn("warn-msg")
	log.Error("error-msg")

	out := buf.String()
	if contains(out, "debug-msg") {
		t.Fatal("debug should not be in info channel")
	}
	if !contains(out, "info-msg") || !contains(out, "warn-msg") {
		t.Fatalf("info/warn missing: %s", out)
	}
	if contains(out, "error-msg") {
		t.Fatal("error should not be in info channel")
	}
}

func TestMinLevelErrorHandler(t *testing.T) {
	var buf bytes.Buffer
	base := slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug})
	h := wrapHandler(base, channelError)
	log := slog.New(h)

	log.Warn("warn-msg")
	log.Error("error-msg")

	out := buf.String()
	if contains(out, "warn-msg") {
		t.Fatal("warn should not be in error channel")
	}
	if !contains(out, "error-msg") {
		t.Fatalf("error missing: %s", out)
	}
}

func contains(s, sub string) bool {
	return len(sub) == 0 || (len(s) >= len(sub) && bytesContains([]byte(s), []byte(sub)))
}

func bytesContains(b, sub []byte) bool {
	return bytes.Index(b, sub) >= 0
}

func TestWrapHandlerEnabled(t *testing.T) {
	h := wrapHandler(slog.NewTextHandler(&bytes.Buffer{}, nil), channelInfo)
	if h.Enabled(context.Background(), slog.LevelDebug) {
		t.Fatal("debug disabled")
	}
	if !h.Enabled(context.Background(), slog.LevelInfo) {
		t.Fatal("info enabled")
	}
}
