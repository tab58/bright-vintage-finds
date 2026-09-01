package logger

import (
	"log/slog"
	"os"
)

type Config struct {
	LogLevel slog.Level
	Handlers []slog.Handler
}

func NewLogger(cfg Config) *slog.Logger {
	defaultHandler := slog.NewJSONHandler(
		os.Stderr,
		&slog.HandlerOptions{Level: cfg.LogLevel},
	)

	return slog.New(defaultHandler)
}
