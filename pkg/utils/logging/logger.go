package logging

import (
	"context"
	"io"
	"log/slog"
	"os"
	"strings"
	"sync"

	"github.com/m-mizutani/clog"
)

type contextKey struct{}

var (
	loggerKey       = contextKey{}
	defaultLogger   *slog.Logger
	defaultLoggerMu sync.RWMutex
)

func init() {
	defaultLogger = New("info", os.Stdout)
}

// parseLevel converts a string level to slog.Level
func parseLevel(level string) slog.Level {
	switch strings.ToLower(level) {
	case "debug":
		return slog.LevelDebug
	case "info":
		return slog.LevelInfo
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		defaultLogger.Warn("invalid log level", "level", level)
		return slog.LevelInfo
	}
}

// New creates a new slog.Logger with the specified level string
// Accepts: "debug", "info", "warn", "warning", "error" (case-insensitive)
func New(level string, w io.Writer) *slog.Logger {
	if w == nil {
		w = os.Stdout
	}

	// Force console output with colors
	handler := clog.New(
		clog.WithWriter(w),
		clog.WithLevel(parseLevel(level)),
		clog.WithTimeFmt("15:04:05"),
		clog.WithSource(false),
		clog.WithAttrHook(clog.GoerrHook),
	)

	return slog.New(handler)
}

// Default returns the default logger
func Default() *slog.Logger {
	defaultLoggerMu.RLock()
	defer defaultLoggerMu.RUnlock()
	return defaultLogger
}

// SetDefault sets the default logger
func SetDefault(logger *slog.Logger) {
	defaultLoggerMu.Lock()
	defer defaultLoggerMu.Unlock()
	defaultLogger = logger
}

// With returns a new context with the logger attached
func With(ctx context.Context, logger *slog.Logger) context.Context {
	return context.WithValue(ctx, loggerKey, logger)
}

// From retrieves the logger from the context
// If no logger is found, it returns the default logger
func From(ctx context.Context) *slog.Logger {
	if logger, ok := ctx.Value(loggerKey).(*slog.Logger); ok {
		return logger
	}
	return Default()
}
