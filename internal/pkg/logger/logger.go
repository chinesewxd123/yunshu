package logger

import (
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"yunshu/internal/config"
)

// Logger 三通道日志：info.log（Info+Warn）、error.log（Error+）、sql.log（仅 SQL）。
type Logger struct {
	Info  *slog.Logger
	Error *slog.Logger
	SQL   *slog.Logger
}

// New 按 config.Log 创建三通道 slog 实例。
func New(cfg config.LogConfig) *Logger {
	logDir := cfg.FilePath
	if logDir == "" {
		logDir = "./logs"
	}
	if err := os.MkdirAll(logDir, 0755); err != nil {
		panic("failed to create log directory: " + err.Error())
	}

	return &Logger{
		Info:  buildChannelLogger(cfg, logDir, channelInfo),
		Error: buildChannelLogger(cfg, logDir, channelError),
		SQL:   buildChannelLogger(cfg, logDir, channelSQL),
	}
}

// Biz 返回带 component 的业务日志器（默认 layer=service）。
func (l *Logger) Biz(component string) *Component {
	return &Component{log: l, component: component, layer: LayerService}
}

func buildChannelLogger(cfg config.LogConfig, logDir, channel string) *slog.Logger {
	writers := outputWriters(cfg, logDir, channel)
	multi := io.MultiWriter(writers...)

	minLevel := channelMinLevel(cfg, channel)
	opts := &slog.HandlerOptions{
		Level:     minLevel,
		AddSource: true,
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			switch a.Key {
			case slog.TimeKey:
				if t, ok := a.Value.Any().(time.Time); ok {
					return slog.String("timestamp", t.Format("2006-01-02 15:04:05.000"))
				}
			case slog.MessageKey:
				return slog.String("message", a.Value.String())
			case slog.SourceKey:
				return shortenSourceAttr(a)
			}
			return a
		},
	}

	var base slog.Handler
	if strings.EqualFold(cfg.Format, "json") {
		base = slog.NewJSONHandler(multi, opts)
	} else {
		base = slog.NewTextHandler(multi, opts)
	}
	return slog.New(wrapHandler(base, channel))
}

func outputWriters(cfg config.LogConfig, logDir, channel string) []io.Writer {
	switch strings.ToLower(cfg.Output) {
	case "file":
		return []io.Writer{openLogFile(logDir, channel)}
	case "both":
		return []io.Writer{os.Stdout, openLogFile(logDir, channel)}
	default:
		return []io.Writer{os.Stdout}
	}
}

func channelMinLevel(cfg config.LogConfig, channel string) slog.Level {
	switch channel {
	case channelSQL:
		if strings.EqualFold(cfg.Level, "debug") {
			return slog.LevelDebug
		}
		return slog.LevelInfo
	case channelError:
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

func openLogFile(logDir, channel string) *os.File {
	fileName := filepath.Join(logDir, channel+".log")
	f, err := os.OpenFile(fileName, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		panic("failed to open log file: " + err.Error())
	}
	return f
}

// shortenSourceAttr 缩短 source 路径，保留包内相对位置。
func shortenSourceAttr(a slog.Attr) slog.Attr {
	if a.Value.Kind() != slog.KindAny {
		return a
	}
	if src, ok := a.Value.Any().(*slog.Source); ok && src != nil {
		file := src.File
		if i := strings.LastIndex(file, "yunshu/"); i >= 0 {
			file = file[i+len("yunshu/"):]
		} else if i := strings.LastIndex(file, "yunshu\\"); i >= 0 {
			file = file[i+len("yunshu\\"):]
		}
		return slog.Any(slog.SourceKey, &slog.Source{Function: trimFuncName(src.Function), File: file, Line: src.Line})
	}
	return a
}

func trimFuncName(fn string) string {
	if i := strings.LastIndex(fn, "/"); i >= 0 {
		fn = fn[i+1:]
	}
	if i := strings.LastIndex(fn, "."); i >= 0 {
		return fn[i+1:]
	}
	return fn
}
