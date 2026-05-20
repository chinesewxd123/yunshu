package agent

import (
	"fmt"
	"log/slog"
	"os"
)

// agent 进程独立运行于目标机，使用 stderr 结构化日志（不依赖平台 logx.Init）。
var agentLog = slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo}))

func logInfo(msg string, attrs ...any) {
	agentLog.Info(msg, attrs...)
}

func logDebug(enabled bool, msg string, attrs ...any) {
	if !enabled {
		return
	}
	agentLog.Debug(msg, attrs...)
}

// logInfof 兼容旧调用；新代码请优先 logInfo + KV。
func logInfof(format string, args ...any) {
	logInfo(fmt.Sprintf(format, args...))
}

func logDebugf(enabled bool, format string, args ...any) {
	if !enabled {
		return
	}
	logDebug(enabled, fmt.Sprintf(format, args...))
}
