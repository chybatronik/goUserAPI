// Package logging provides standard field definitions for structured logging
package logging

import (
	"log/slog"
)

// Standard log field values and constants for structured logging
const (
	// Standard field names
	FieldTimestamp    = "ts"
	FieldLevel        = "level"
	FieldMessage      = "msg"
	FieldRequestID    = "req_id"
	FieldHTTPMethod   = "method"
	FieldHTTPPath     = "path"
	FieldHTTPStatus   = "status"
	FieldLatencyMs    = "latency_ms"
	FieldService      = "service"
	FieldVersion      = "version"
	FieldUptimeSec    = "uptime_seconds"
	FieldError        = "error"
	FieldResponseTime = "response_time_ms"
	FieldCheckName    = "check_name"
	FieldCheckStatus  = "check_status"

	// Log levels
	LevelDebug = "debug"
	LevelInfo  = "info"
	LevelWarn  = "warn"
	LevelError = "error"

	// Health check statuses
	StatusHealthy   = "healthy"
	StatusUnhealthy = "unhealthy"
	StatusOK        = "ok"
	StatusFailed    = "failed"
)

// StandardField provides helper functions for creating structured log fields
type StandardField struct{}

// NewStandardField creates a new StandardField instance
func NewStandardField() *StandardField {
	return &StandardField{}
}

// Timestamp adds timestamp field
func (sf *StandardField) Timestamp(ts int64) slog.Attr {
	return slog.Int64(FieldTimestamp, ts)
}

// RequestID adds request ID field
func (sf *StandardField) RequestID(reqID string) slog.Attr {
	return slog.String(FieldRequestID, reqID)
}

// HTTPMethod adds HTTP method field
func (sf *StandardField) HTTPMethod(method string) slog.Attr {
	return slog.String(FieldHTTPMethod, method)
}

// HTTPPath adds HTTP path field
func (sf *StandardField) HTTPPath(path string) slog.Attr {
	return slog.String(FieldHTTPPath, path)
}

// HTTPStatus adds HTTP status code field
func (sf *StandardField) HTTPStatus(status int) slog.Attr {
	return slog.Int(FieldHTTPStatus, status)
}

// LatencyMs adds latency in milliseconds field
func (sf *StandardField) LatencyMs(latency int64) slog.Attr {
	return slog.Int64(FieldLatencyMs, latency)
}

// Service adds service name field
func (sf *StandardField) Service(service string) slog.Attr {
	return slog.String(FieldService, service)
}

// Version adds version field
func (sf *StandardField) Version(version string) slog.Attr {
	return slog.String(FieldVersion, version)
}

// Uptime adds uptime in seconds field
func (sf *StandardField) Uptime(uptime int64) slog.Attr {
	return slog.Int64(FieldUptimeSec, uptime)
}

// Error adds error field
func (sf *StandardField) Error(err error) slog.Attr {
	if err == nil {
		return slog.String(FieldError, "")
	}
	return slog.String(FieldError, err.Error())
}

// ResponseTime adds response time in milliseconds field
func (sf *StandardField) ResponseTime(ms int64) slog.Attr {
	return slog.Int64(FieldResponseTime, ms)
}

// CheckStatus adds health check status field
func (sf *StandardField) CheckStatus(status string) slog.Attr {
	return slog.String(FieldCheckStatus, status)
}

// DatabaseStatus adds database-specific status field
func (sf *StandardField) DatabaseStatus(healthy bool, responseTimeMs int64, err error) []slog.Attr {
	attrs := []slog.Attr{
		slog.String("database_status", map[bool]string{true: StatusHealthy, false: StatusUnhealthy}[healthy]),
		slog.Int64("database_response_time_ms", responseTimeMs),
	}

	if err != nil {
		attrs = append(attrs, slog.String("database_error", err.Error()))
	}

	return attrs
}
