package logger

import (
	"context"
	"log/slog"
)

type ContextHandler func(ctx context.Context, r *slog.Record) error

type ContextProcessorHandler struct {
	handler         slog.Handler
	contextHandlers []ContextHandler
}

func (h *ContextProcessorHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.handler.Enabled(ctx, level)
}

func (h *ContextProcessorHandler) Handle(ctx context.Context, r slog.Record) error {
	for _, handler := range h.contextHandlers {
		if err := handler(ctx, &r); err != nil {
			return err
		}
	}
	return h.handler.Handle(ctx, r)
}

func (h *ContextProcessorHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &ContextProcessorHandler{handler: h.handler.WithAttrs(attrs)}
}

func (h *ContextProcessorHandler) WithGroup(name string) slog.Handler {
	return &ContextProcessorHandler{handler: h.handler.WithGroup(name)}
}

func NewContextProcessorHandler(handler slog.Handler, contextHandlers ...ContextHandler) slog.Handler {
	return &ContextProcessorHandler{
		handler:         handler,
		contextHandlers: contextHandlers,
	}
}
