package logger

import (
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"go-permission-system/internal/config"
)

type Logger struct {
	Info *slog.Logger
	Error *slog.Logger
	SQL *slog.Logger
}

func New(cfg config.LogConfig) *Logger {
	logDir := cfg.FilePath
	if logDir == "" {
		logDir = "./logs"
	}

	if err := os.MkdirAll(logDir, 0755); err != nil {
		panic("failed to create log directory: " + err.Error())
	}

	infoLogger := createLogger(cfg, "info", logDir)
	errorLogger := createLogger(cfg, "error", logDir)
	sqlLogger := createLogger(cfg, "sql", logDir)

	return &Logger{
		Info: infoLogger,
		Error: errorLogger,
		SQL: sqlLogger,
	}
}

func createLogger(cfg config.LogConfig, logType string, logDir string) *slog.Logger {
	var writers []io.Writer

	switch strings.ToLower(cfg.Output) {
	case "file":
		writers = []io.Writer{createFileWriter(logDir, logType)}
	case "both":
		writers = []io.Writer{os.Stdout, createFileWriter(logDir, logType)}
	default: // console
		writers = []io.Writer{os.Stdout}
	}

	multiWriter := io.MultiWriter(writers...)

	opts := &slog.HandlerOptions{
		Level: parseLevel(cfg.Level),
		AddSource: true,
	}

	if strings.EqualFold(cfg.Format, "json") {
		return slog.New(slog.NewJSONHandler(multiWriter, opts))
	}
	return slog.New(slog.NewTextHandler(multiWriter, opts))
}

func createFileWriter(logDir string, logType string) *os.File {
	fileName := filepath.Join(logDir, logType+".log")
	file, err := os.OpenFile(fileName, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		panic("failed to open log file: " + err.Error())
	}
	return file
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
