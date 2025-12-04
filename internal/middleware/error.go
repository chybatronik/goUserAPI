// Package middleware provides HTTP middleware for the goUserAPI service.
package middleware

import (
	"encoding/json"
	"log"
	"net/http"
)

// ErrorResponse represents an error response structure
type ErrorResponse struct {
	Error   string `json:"error"`
	Code    int    `json:"code"`
	Message string `json:"message,omitempty"`
}

// ErrorHandler middleware handles errors and returns consistent JSON responses
type ErrorHandler struct {
	next http.Handler
}

// NewErrorHandler creates a new error handler middleware
func NewErrorHandler(next http.Handler) *ErrorHandler {
	return &ErrorHandler{
		next: next,
	}
}

// ServeHTTP implements the http.Handler interface with panic recovery
func (eh *ErrorHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Create a response writer that captures status codes
	wrapped := NewResponseWriter(w)

	// Add panic recovery
	defer func() {
		if rec := recover(); rec != nil {
			log.Printf("Panic recovered in error handler: %v", rec)
			eh.handleError(w, http.StatusInternalServerError, "Internal server error")
		}
	}()

	eh.next.ServeHTTP(wrapped, r)

	// If status code indicates an error AND no body was written, format the response
	// This prevents overwriting custom error responses from handlers
	if wrapped.StatusCode() >= 400 && !wrapped.HasBody() {
		eh.handleError(w, wrapped.StatusCode(), "Request failed")
	}
}

// handleError formats and writes error responses only if response hasn't been sent
func (eh *ErrorHandler) handleError(w http.ResponseWriter, statusCode int, message string) {
	// Try to write headers safely - if this fails, headers are already sent
	if !eh.tryWriteHeaders(w, statusCode) {
		log.Printf("Skipping error formatting - headers already sent for status %d", statusCode)
		return
	}

	// Write JSON error response
	errorResp := ErrorResponse{
		Error:   http.StatusText(statusCode),
		Code:    statusCode,
		Message: message,
	}

	if err := json.NewEncoder(w).Encode(errorResp); err != nil {
		log.Printf("Failed to encode error response: %v", err)
	}
}

// tryWriteHeaders attempts to write headers safely, returns false if headers already sent
func (eh *ErrorHandler) tryWriteHeaders(w http.ResponseWriter, statusCode int) bool {
	// Use a recover to catch panic if headers are already written
	defer func() {
		if rec := recover(); rec != nil {
			// Headers were already written
		}
	}()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	return true
}
