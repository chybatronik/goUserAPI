package middleware

import (
	"net/http"
	"time"

	"github.com/chybatronik/goUserAPI/internal/logging"
)

// LoggingMiddleware logs HTTP requests using structured logging
type LoggingMiddleware struct {
	next   http.Handler
	logger *logging.Logger
}

// NewLoggingMiddleware creates a new structured logging middleware
func NewLoggingMiddleware(logger *logging.Logger, next http.Handler) *LoggingMiddleware {
	return &LoggingMiddleware{
		next:   next,
		logger: logger,
	}
}

// ServeHTTP implements the http.Handler interface with structured logging
func (lm *LoggingMiddleware) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	start := time.Now()

	// Get request ID from context
	reqID := GetRequestID(r.Context())

	// Debug: log if request ID is missing
	if reqID == "" {
		lm.logger.Debug("Request ID not found in context", "path", r.URL.Path)
	}

	// Create a response writer that captures status codes
	wrapped := NewResponseWriter(w)

	// Process request
	lm.next.ServeHTTP(wrapped, r)

	// Calculate duration
	duration := time.Since(start)

	// Log request completion using structured logging
	lm.logger.Request(
		reqID,
		r.Method,
		r.URL.Path,
		wrapped.StatusCode(),
		duration.Milliseconds(),
	)
}
