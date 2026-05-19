package logger

// Init 初始化全局 Logger 并注册默认 context 提取器（对齐 onex log.Init）。
func Init(l *Logger, extra ...ContextExtractors) {
	SetDefault(l)
	for _, ext := range extra {
		RegisterContextExtractors(ext)
	}
}

// Sync 刷新日志缓冲；slog 标准输出一般无需 flush，保留接口便于进程退出时调用。
func Sync() {
	if defaultLogger == nil {
		return
	}
	defaultLogger.Sync()
}

// Sync 实例方法，便于与 onex 用法对齐。
func (l *Logger) Sync() {}
