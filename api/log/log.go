package log

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"os"
	"sync"
)

var (
	defaultLogger *slog.Logger
	defaultLoggerMu sync.Mutex
)

// DefaultLogger returns the current default logger. Used by tests to verify initialization.
func DefaultLogger() *slog.Logger {
	defaultLoggerMu.Lock()
	defer defaultLoggerMu.Unlock()
	return defaultLogger
}

// SetLogger sets the package-level logger. Used by tests to inject a logger that writes to a buffer.
func SetLogger(l *slog.Logger) {
	defaultLoggerMu.Lock()
	defer defaultLoggerMu.Unlock()
	defaultLogger = l
}

// InitLog initializes the default logger with slog. In development (developmentEnv true)
// logs are human-readable text; otherwise JSON is used. address and protocol are ignored
// (no Graylog sender). appName and tag are added as attributes to every log line.
func InitLog(developmentEnv bool, address, protocol, appName, tag string) {
	defaultLoggerMu.Lock()
	defer defaultLoggerMu.Unlock()

	opts := &slog.HandlerOptions{Level: slog.LevelInfo}
	var handler slog.Handler
	if developmentEnv {
		handler = slog.NewTextHandler(os.Stdout, opts)
	} else {
		handler = slog.NewJSONHandler(os.Stdout, opts)
	}
	defaultLogger = slog.New(handler).With("app", appName, "tags", tag)
}

func logAt(ctx context.Context, level slog.Level, action, info string, msgCode int, message ...interface{}) {
	defaultLoggerMu.Lock()
	l := defaultLogger
	defaultLoggerMu.Unlock()
	if l == nil {
		log.Println("[log] default logger not initialized")
		return
	}
	base := MsgCode[msgCode]
	msg := base
	if len(message) > 0 {
		msg = base + " " + fmt.Sprint(message...)
	}
	l.Log(ctx, level, msg, "action", action, "info", info, "msg_code", msgCode)
}

// Info logs at INFO level with a single combined message (template + variadic args) and structured attributes.
func Info(action, info string, msgCode int, message ...interface{}) {
	logAt(context.Background(), slog.LevelInfo, action, info, msgCode, message...)
}

// Warning logs at WARN level with a single combined message and structured attributes.
func Warning(action, info string, msgCode int, message ...interface{}) {
	logAt(context.Background(), slog.LevelWarn, action, info, msgCode, message...)
}

// Error logs at ERROR level with a single combined message and structured attributes.
func Error(action, info string, msgCode int, message ...interface{}) {
	logAt(context.Background(), slog.LevelError, action, info, msgCode, message...)
}
