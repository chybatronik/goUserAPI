// Package logging provides structured logging functionality using log/slog
package logging

import (
	"log/slog"
	"os"
)

// Logger wraps slog.Logger with additional application-specific functionality
type Logger struct {
	*slog.Logger
	service string
	version string
}

// NewStructuredLogger creates a new structured logger with JSON output
func NewStructuredLogger(level string, service, version string) *Logger {
	var logLevel slog.Level
	switch level {
	case "debug":
		logLevel = slog.LevelDebug
	case "warn":
		logLevel = slog.LevelWarn
	case "error":
		logLevel = slog.LevelError
	default:
		logLevel = slog.LevelInfo
	}

	// Create JSON handler for structured logging
	handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: logLevel,
	})

	logger := slog.New(handler)

	return &Logger{
		Logger:  logger,
		service: service,
		version: version,
	}
}

// WithRequestID adds request ID to the logger
func (l *Logger) WithRequestID(reqID string) *Logger {
	return &Logger{
		Logger:  l.Logger.With(slog.String("req_id", reqID)),
		service: l.service,
		version: l.version,
	}
}

// WithHTTPRequest adds HTTP request context to the logger
func (l *Logger) WithHTTPRequest(method, path string, statusCode int, latencyMs int64) *Logger {
	return &Logger{
		Logger: l.Logger.With(
			slog.String("method", method),
			slog.String("path", path),
			slog.Int("status", statusCode),
			slog.Int64("latency_ms", latencyMs),
		),
		service: l.service,
		version: l.version,
	}
}

// WithError adds error context to the logger
func (l *Logger) WithError(err error) *Logger {
	if err == nil {
		return l
	}
	return &Logger{
		Logger:  l.Logger.With(slog.String("error", err.Error())),
		service: l.service,
		version: l.version,
	}
}

// WithServiceContext adds service context to the logger
func (l *Logger) WithServiceContext() *Logger {
	return &Logger{
		Logger: l.Logger.With(
			slog.String("service", l.service),
			slog.String("version", l.version),
		),
		service: l.service,
		version: l.version,
	}
}

// Info logs an info message with structured context
func (l *Logger) Info(msg string, args ...any) {
	l.Logger.Info(msg, args...)
}

// Error logs an error message with structured context
func (l *Logger) Error(msg string, args ...any) {
	l.Logger.Error(msg, args...)
}

// Warn logs a warning message with structured context
func (l *Logger) Warn(msg string, args ...any) {
	l.Logger.Warn(msg, args...)
}

// Debug logs a debug message with structured context
func (l *Logger) Debug(msg string, args ...any) {
	l.Logger.Debug(msg, args...)
}

// Startup logs application startup information
func (l *Logger) Startup(msg string, args ...any) {
	l.WithServiceContext().Info(msg, args...)
}

// Request logs HTTP request completion
func (l *Logger) Request(reqID, method, path string, statusCode int, latencyMs int64) {
	l.WithRequestID(reqID).
		WithHTTPRequest(method, path, statusCode, latencyMs).
		Info("HTTP request completed")
}

// Database logs database-related operations
func (l *Logger) Database(msg string, args ...any) {
	l.Logger.Info("database: "+msg, args...)
}

// DatabaseError logs database errors
func (l *Logger) DatabaseError(msg string, err error) {
	l.WithError(err).Error("database: " + msg)
}

// HealthCheck logs health check operations
func (l *Logger) HealthCheck(msg string, args ...any) {
	l.Logger.Info("healthcheck: "+msg, args...)
}
