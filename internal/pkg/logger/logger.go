package logger

import (
	"log/slog"
	"os"
	"strings"

	"go-permission-system/internal/config"
)

func New(cfg config.LogConfig) *slog.Logger {
	opts := &slog.HandlerOptions{
		Level: parseLevel(cfg.Level),
	}

	if strings.EqualFold(cfg.Format, "json") {
		return slog.New(slog.NewJSONHandler(os.Stdout, opts))
	}
	return slog.New(slog.NewTextHandler(os.Stdout, opts))
}

func parseLevel(level string) slog.Level {
	switch strings.ToLower(level) {
	case "debug":
		return slog.LevelDebug
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
