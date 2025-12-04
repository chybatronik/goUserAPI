package errors

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strings"
)

// ErrorResponse represents a standardized error response
type ErrorResponse struct {
	Error   string `json:"error"`
	Code    string `json:"code"`
	Details string `json:"details,omitempty"`
}

// getRequestID extracts request ID from context
func getRequestID(ctx context.Context) string {
	if requestID, ok := ctx.Value("requestID").(string); ok {
		return requestID
	}
	return ""
}

// writeSecureErrorResponse writes a secure error response
func writeSecureErrorResponse(w http.ResponseWriter, statusCode int, code, message string) {
	// NEVER include internal details in user-facing errors
	response := ErrorResponse{
		Error: message,
		Code:  code,
	}

	// Log detailed error internally for debugging
	log.Printf("API error response - Status: %d, Code: %s, Message: %s",
		statusCode, code, message)

	// Set security headers
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("X-Frame-Options", "DENY")
	w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")

	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(response)
}

// writeSecureErrorResponseWithRequest writes a secure error response with request context
func writeSecureErrorResponseWithRequest(w http.ResponseWriter, r *http.Request, statusCode int, code, message string) {
	// Get request ID for tracing
	requestID := getRequestID(r.Context())

	// NEVER include internal details in user-facing errors
	response := ErrorResponse{
		Error: message,
		Code:  code,
	}

	// Log detailed error internally with request ID for debugging
	log.Printf("API error response - RequestID: %s, Status: %d, Code: %s, Message: %s, Path: %s, Method: %s",
		requestID, statusCode, code, message, r.URL.Path, r.Method)

	// Set security headers
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("X-Frame-Options", "DENY")
	w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")

	// Include request ID in response header if available
	if requestID != "" {
		w.Header().Set("X-Request-ID", requestID)
	}

	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(response)
}

// WriteValidationError writes a validation error response (400 Bad Request)
func WriteValidationError(w http.ResponseWriter, r *http.Request, field, message string) {
	sanitizedMessage := sanitizeErrorMessage(message)
	writeSecureErrorResponseWithRequest(w, r, http.StatusBadRequest, "VALIDATION_ERROR", sanitizedMessage)
}

// WriteNotFoundError writes a not found error response (404 Not Found)
func WriteNotFoundError(w http.ResponseWriter, r *http.Request, resource string) {
	message := "Resource not found"
	if resource != "" {
		message = resource + " not found"
	}
	writeSecureErrorResponseWithRequest(w, r, http.StatusNotFound, "NOT_FOUND", message)
}

// WriteRateLimitError writes a rate limit error response (429 Too Many Requests)
func WriteRateLimitError(w http.ResponseWriter, r *http.Request) {
	writeSecureErrorResponseWithRequest(w, r, http.StatusTooManyRequests, "RATE_LIMIT_EXCEEDED", "Too many requests")
}

// WriteInternalError writes an internal server error response (500 Internal Server Error)
func WriteInternalError(w http.ResponseWriter, r *http.Request) {
	// Generic message for internal errors to avoid information leakage
	writeSecureErrorResponseWithRequest(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error")
}

// WriteServiceUnavailableError writes a service unavailable error response (503 Service Unavailable)
func WriteServiceUnavailableError(w http.ResponseWriter, r *http.Request) {
	writeSecureErrorResponseWithRequest(w, r, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "Service temporarily unavailable")
}

// WriteUnauthorizedError writes an unauthorized error response (401 Unauthorized)
func WriteUnauthorizedError(w http.ResponseWriter, r *http.Request) {
	writeSecureErrorResponseWithRequest(w, r, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required")
}

// WriteForbiddenError writes a forbidden error response (403 Forbidden)
func WriteForbiddenError(w http.ResponseWriter, r *http.Request) {
	writeSecureErrorResponseWithRequest(w, r, http.StatusForbidden, "FORBIDDEN", "Access denied")
}

// sanitizeErrorMessage removes potentially dangerous information from error messages
func sanitizeErrorMessage(message string) string {
	// Store original to check for dangerous terms before modification
	originalMessage := message

	// Remove potential file paths
	message = strings.ReplaceAll(message, "/", "_")

	// Remove potential database identifiers
	message = strings.ReplaceAll(message, "pg_", "")
	message = strings.ReplaceAll(message, "sql_", "")

	// Remove potential system information
	dangerousTerms := []string{
		"internal", "system", "database", "server", "stack trace",
		"panic", "fatal", "exception", "error code", "line",
		"file:", "at line", "in function",
	}

	lowerOriginal := strings.ToLower(originalMessage)
	for _, term := range dangerousTerms {
		if strings.Contains(lowerOriginal, strings.ToLower(term)) {
			// Replace with generic message
			message = "Validation failed"
			break
		}
	}

	// Limit message length to prevent information leakage
	if len(message) > 200 {
		message = "Validation failed with invalid input"
	}

	return strings.TrimSpace(message)
}

// WriteCustomError writes a custom error response with provided status code
func WriteCustomError(w http.ResponseWriter, r *http.Request, statusCode int, code, message string) {
	sanitizedMessage := sanitizeErrorMessage(message)
	writeSecureErrorResponseWithRequest(w, r, statusCode, code, sanitizedMessage)
}

// WriteSecurityError writes a security-related error response
func WriteSecurityError(w http.ResponseWriter, r *http.Request, securityCode string) {
	securityMessages := map[string]string{
		"INVALID_UNICODE":     "Invalid input characters detected",
		"RATE_LIMIT_EXCEEDED": "Too many requests",
		"VALIDATION_ERROR":    "Invalid input provided",
		"SECURITY_VIOLATION":  "Security validation failed",
		"BLOCKED_REQUEST":     "Request blocked for security reasons",
	}

	message := securityMessages[securityCode]
	if message == "" {
		message = "Security validation failed"
	}

	writeSecureErrorResponseWithRequest(w, r, http.StatusBadRequest, securityCode, message)
}
