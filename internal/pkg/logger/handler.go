package logger

import (
	"context"
	"log/slog"
)

// levelWindowHandler 仅放行 [min, max] 级别（用于 info.log：Info～Warn）。
type levelWindowHandler struct {
	min slog.Level
	max slog.Level
	h   slog.Handler
}

func (h *levelWindowHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return level >= h.min && level <= h.max && h.h.Enabled(ctx, level)
}

func (h *levelWindowHandler) Handle(ctx context.Context, r slog.Record) error {
	if !h.Enabled(ctx, r.Level) {
		return nil
	}
	return h.h.Handle(ctx, r)
}

func (h *levelWindowHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &levelWindowHandler{min: h.min, max: h.max, h: h.h.WithAttrs(attrs)}
}

func (h *levelWindowHandler) WithGroup(name string) slog.Handler {
	return &levelWindowHandler{min: h.min, max: h.max, h: h.h.WithGroup(name)}
}

// minLevelHandler 仅放行 level >= min（用于 error.log、sql.log）。
type minLevelHandler struct {
	min slog.Level
	h   slog.Handler
}

func (h *minLevelHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return level >= h.min && h.h.Enabled(ctx, level)
}

func (h *minLevelHandler) Handle(ctx context.Context, r slog.Record) error {
	if !h.Enabled(ctx, r.Level) {
		return nil
	}
	return h.h.Handle(ctx, r)
}

func (h *minLevelHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &minLevelHandler{min: h.min, h: h.h.WithAttrs(attrs)}
}

func (h *minLevelHandler) WithGroup(name string) slog.Handler {
	return &minLevelHandler{min: h.min, h: h.h.WithGroup(name)}
}

func wrapHandler(h slog.Handler, channel string) slog.Handler {
	switch channel {
	case channelInfo:
		return &levelWindowHandler{min: slog.LevelInfo, max: slog.LevelWarn, h: h}
	case channelError:
		return &minLevelHandler{min: slog.LevelError, h: h}
	case channelSQL:
		return &minLevelHandler{min: slog.LevelDebug, h: h}
	default:
		return h
	}
}
