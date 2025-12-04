// Package middleware provides HTTP middleware components for request tracking
package middleware

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
)

// RequestIDKey is the context key for request ID
type RequestIDKey string

const (
	// RequestIDContextKey is the context key for storing request ID
	RequestIDContextKey RequestIDKey = "req_id"
	// RequestIDHeader is the HTTP header name for request ID
	RequestIDHeader = "X-Request-ID"
)

// GenerateRequestID generates a unique request ID using crypto/rand
func GenerateRequestID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		// Fallback to simpler ID generation if crypto/rand fails
		return fallbackRequestID()
	}
	return hex.EncodeToString(b)
}

// fallbackRequestID provides a fallback ID generation method
func fallbackRequestID() string {
	// Simple fallback using timestamp and counter
	// In production, crypto/rand should be preferred
	return "req_" + hex.EncodeToString([]byte("fallback"))
}

// GetRequestID extracts request ID from context
func GetRequestID(ctx context.Context) string {
	if reqID, ok := ctx.Value(RequestIDContextKey).(string); ok {
		return reqID
	}
	return ""
}

// SetRequestID adds request ID to context
func SetRequestID(ctx context.Context, reqID string) context.Context {
	return context.WithValue(ctx, RequestIDContextKey, reqID)
}

// RequestIDMiddleware ensures request ID is present and adds it to context
func RequestIDMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check if request ID exists in header
		reqID := r.Header.Get(RequestIDHeader)
		if reqID == "" {
			// Generate new request ID if not present
			reqID = GenerateRequestID()
			// Debug: log generated request ID
			fmt.Printf("[DEBUG] Generated request ID: %s for path: %s\n", reqID, r.URL.Path)
		}

		// Add request ID to response header for client correlation
		w.Header().Set(RequestIDHeader, reqID)

		// Add request ID to context
		ctx := SetRequestID(r.Context(), reqID)
		r = r.WithContext(ctx)

		// Debug: verify context has request ID
		if checkReqID := GetRequestID(ctx); checkReqID != reqID {
			fmt.Printf("[DEBUG] ERROR: Context request ID mismatch! expected: %s, got: %s\n", reqID, checkReqID)
		}

		// Continue with request processing
		next.ServeHTTP(w, r)
	})
}
