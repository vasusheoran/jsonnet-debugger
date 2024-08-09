package utils

import (
	"io"
	"log/slog"
)

type CustomLogger struct {
	*slog.Logger
}

func NewCustomLogger(out io.Writer, prefix string) *CustomLogger {
	return &CustomLogger{slog.New(slog.NewJSONHandler(out, &slog.HandlerOptions{Level: slog.LevelDebug}))}
}
