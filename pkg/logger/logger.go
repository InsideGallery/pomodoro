// Package logger provides structured logging for the application.
package logger

import (
	"log/slog"
	"os"
)

var log = slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{ //nolint:gochecknoglobals // singleton logger
	Level: slog.LevelWarn,
}))

// Warn logs a warning message.
func Warn(msg string, args ...any) {
	log.Warn(msg, args...)
}

// Error logs an error message.
func Error(msg string, args ...any) {
	log.Error(msg, args...)
}

// Debug logs a debug message.
func Debug(msg string, args ...any) {
	log.Debug(msg, args...)
}

// Info logs an info message.
func Info(msg string, args ...any) {
	log.Info(msg, args...)
}
