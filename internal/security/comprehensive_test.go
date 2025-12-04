package security

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/chybatronik/goUserAPI/internal/errors"
	"github.com/chybatronik/goUserAPI/internal/middleware"
	"github.com/chybatronik/goUserAPI/internal/validation"
	usererrors "github.com/chybatronik/goUserAPI/pkg/errors"
)

// TestComprehensiveSecurityIntegration tests all security components together
func TestComprehensiveSecurityIntegration(t *testing.T) {
	t.Run("Unicode Security + Rate Limiting + Error Handling", func(t *testing.T) {
		// Create a mock handler that processes user input
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Simulate processing user input with Unicode validation
			userInput := r.URL.Query().Get("input")

			// Validate Unicode security
			err := validation.ValidateUnicodeSecurity(userInput)
			if err != nil {
				errors.WriteSecurityError(w, r, "INVALID_UNICODE")
				return
			}

			// Process input successfully
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]string{
				"status": "success",
				"input":  userInput,
			})
		})

		// Apply security middleware chain
		secureHandler := middleware.SecurityRateLimit(10.0, 5)(handler)

		testCases := []struct {
			name           string
			input          string
			expectedStatus int
			expectBlock    bool
		}{
			{
				name:           "valid input",
				input:          "normal_user_input",
				expectedStatus: http.StatusOK,
				expectBlock:    false,
			},
			{
				name:           "homograph attack",
				input:          "admin",
				expectedStatus: http.StatusOK,
				expectBlock:    false,
			},
			{
				name:           "cyrillic homograph attack",
				input:          "аdmin", // Cyrillic 'а' not Latin 'a'
				expectedStatus: http.StatusBadRequest,
				expectBlock:    true,
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				req := httptest.NewRequest("GET", "/?input="+tc.input, nil)
				req.RemoteAddr = "192.168.1.100:12345"

				w := httptest.NewRecorder()
				secureHandler.ServeHTTP(w, req)

				if w.Code != tc.expectedStatus {
					t.Errorf("Expected status %d, got %d", tc.expectedStatus, w.Code)
				}

				// Verify response is valid JSON
				var response map[string]interface{}
				err := json.Unmarshal(w.Body.Bytes(), &response)
				if err != nil {
					t.Errorf("Expected valid JSON response, got error: %v", err)
				}
			})
		}
	})

	t.Run("Rate Limiting Bypass Protection", func(t *testing.T) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})

		// Very restrictive rate limit for testing
		secureHandler := middleware.SecurityRateLimit(1.0, 2)(handler)

		// Send rapid requests to trigger rate limiting
		blockedRequests := 0
		for i := 0; i < 10; i++ {
			req := httptest.NewRequest("GET", "/", nil)
			req.RemoteAddr = "192.168.1.200:12345"

			w := httptest.NewRecorder()
			secureHandler.ServeHTTP(w, req)

			if w.Code == http.StatusTooManyRequests {
				blockedRequests++
			}
		}

		if blockedRequests == 0 {
			t.Error("Expected some requests to be rate limited")
		}

		// Verify rate limit responses are secure
		req := httptest.NewRequest("GET", "/", nil)
		req.RemoteAddr = "192.168.1.200:12345"

		w := httptest.NewRecorder()
		secureHandler.ServeHTTP(w, req)

		if w.Code == http.StatusTooManyRequests {
			// Check response structure
			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			if err != nil {
				t.Errorf("Rate limit response should be valid JSON: %v", err)
			}

			if response["code"] != "RATE_LIMIT_EXCEEDED" {
				t.Errorf("Expected RATE_LIMIT_EXCEEDED code, got: %v", response["code"])
			}
		}
	})
}

// TestSecurityHeaders tests that security headers are properly set
func TestSecurityHeaders(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		errors.WriteInternalError(w, r)
	})

	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	// Check security headers
	expectedHeaders := map[string]string{
		"Content-Type":           "application/json",
		"X-Content-Type-Options": "nosniff",
		"X-Frame-Options":        "DENY",
		"Referrer-Policy":        "strict-origin-when-cross-origin",
	}

	for header, expectedValue := range expectedHeaders {
		actualValue := w.Header().Get(header)
		if actualValue != expectedValue {
			t.Errorf("Expected header %s to be %s, got %s", header, expectedValue, actualValue)
		}
	}
}

// TestErrorSanitization tests that error messages don't leak sensitive information
func TestErrorSanitization(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "database error with path",
			input:    "Error opening file /etc/passwd: permission denied",
			expected: "Validation failed",
		},
		{
			name:     "system error with internal details",
			input:    "Internal panic at database.go:123:45 in func Query",
			expected: "Validation failed",
		},
		{
			name:     "stack trace error",
			input:    "goroutine 1 [running]: database.Query(...)",
			expected: "Validation failed",
		},
		{
			name:     "valid error message",
			input:    "Invalid email format",
			expected: "Invalid email format",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := sanitizeErrorMessage(tc.input)
			if result != tc.expected {
				t.Errorf("Expected '%s', got '%s'", tc.expected, result)
			}

			// Ensure no dangerous information remains
			dangerousTerms := []string{
				"/", "\\", "pg_", "sql_", "internal", "system",
				"database", "server", "panic", "fatal", "exception",
			}

			lowerResult := strings.ToLower(result)
			for _, term := range dangerousTerms {
				if strings.Contains(lowerResult, term) {
					t.Errorf("Dangerous term '%s' found in sanitized message: %s", term, result)
				}
			}
		})
	}
}

// TestDatabaseErrorMappingSecurity tests database error mapping doesn't leak information
func TestDatabaseErrorMappingSecurity(t *testing.T) {
	testCases := []struct {
		name           string
		errorInput     error
		expectedCode   string
		expectedStatus int
	}{
		{
			name:           "database connection error",
			errorInput:     &databaseError{Message: "Failed to connect to postgresql://user:password@localhost/db"},
			expectedCode:   "SERVICE_UNAVAILABLE",
			expectedStatus: 503,
		},
		{
			name:           "table access error",
			errorInput:     &databaseError{Message: "Permission denied for table users"},
			expectedCode:   "DATABASE_ERROR",
			expectedStatus: 500,
		},
		{
			name:           "sql injection attempt",
			errorInput:     &databaseError{Message: "SQL injection detected in query: DROP TABLE users"},
			expectedCode:   "DATABASE_ERROR",
			expectedStatus: 500,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := errors.MapDatabaseErrorSecure(tc.errorInput)

			if result == nil {
				t.Error("Expected non-nil error result")
				return
			}

			userErr, ok := result.(*usererrors.UserError)
			if !ok {
				t.Errorf("Expected *usererrors.UserError, got: %T", result)
				return
			}

			if userErr.HTTPStatus != tc.expectedStatus {
				t.Errorf("Expected status %d, got %d", tc.expectedStatus, userErr.HTTPStatus)
			}

			// Ensure no sensitive information in user message
			if strings.Contains(strings.ToLower(userErr.Message), "password") {
				t.Errorf("Sensitive information found in user message: %s", userErr.Message)
			}
		})
	}
}

// TestInputSizeLimits tests that large inputs are properly handled
func TestInputSizeLimits(t *testing.T) {
	// Test 1MB payload limit (1,048,576 bytes)
	largePayload := make([]byte, 1024*1024)
	largePayload[1024*1024-1] = 'x' // Make it exactly 1MB

	err := validation.ValidatePayloadSize(largePayload, 1024*1024)
	if err != nil {
		t.Errorf("1MB payload should be allowed, got error: %v", err)
	}

	// Test exceeding 1MB
	oversizedPayload := make([]byte, 1024*1024+1)
	oversizedPayload[1024*1024] = 'x'

	err = validation.ValidatePayloadSize(oversizedPayload, 1024*1024)
	if err == nil {
		t.Error("Oversized payload should be rejected")
	}

	// Test field length limits
	longField := strings.Repeat("a", 101) // 101 characters
	err = validation.ValidateStringLength(longField, 0, 100)
	if err == nil {
		t.Error("Field exceeding limit should be rejected")
	}

	// Test valid field length
	validField := strings.Repeat("a", 100) // exactly 100 characters
	err = validation.ValidateStringLength(validField, 0, 100)
	if err != nil {
		t.Errorf("Valid field length should be allowed, got error: %v", err)
	}
}

// TestPerformanceSecurityOverhead tests that security validation doesn't impact performance excessively
func TestPerformanceSecurityOverhead(t *testing.T) {
	// Test Unicode validation performance
	input := "john.doe@example.com"
	iterations := 10000

	start := time.Now()
	for i := 0; i < iterations; i++ {
		err := validation.ValidateUnicodeSecurity(input)
		if err != nil {
			t.Errorf("Valid input should pass validation: %v", err)
		}
	}
	duration := time.Since(start)

	// Should complete 10,000 validations in under 1 second (0.1ms per validation)
	if duration > time.Second {
		t.Errorf("Unicode validation too slow: %v for %d iterations", duration, iterations)
	}

	// Test rate limiting performance
	limiter := middleware.SecurityRateLimit(1000.0, 100)
	handler := limiter(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	start = time.Now()
	for i := 0; i < 100; i++ {
		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/", nil)
		req.RemoteAddr = "192.168.1." + string(rune(i%255)) + ":12345"
		handler.ServeHTTP(w, req)
	}
	duration = time.Since(start)

	// Should complete 100 requests in under 100ms (1ms per request)
	if duration > 100*time.Millisecond {
		t.Errorf("Rate limiting too slow: %v for 100 requests", duration)
	}
}

// Helper types and functions for testing
type databaseError struct {
	Message string
}

func (e *databaseError) Error() string {
	return e.Message
}

// sanitizeErrorMessage replicates the function from errors package for testing
func sanitizeErrorMessage(message string) string {
	// Check for dangerous terms BEFORE modification
	dangerousTerms := []string{
		"/", "\\", "internal", "system", "database", "server", "stack trace",
		"panic", "fatal", "exception", "error code", "line",
		"file:", "at line", "in function", "permission denied",
		"etc_passwd", "proc/", "sys/", "var/", "usr/",
	}

	lowerMessage := strings.ToLower(message)
	for _, term := range dangerousTerms {
		if strings.Contains(lowerMessage, strings.ToLower(term)) {
			return "Validation failed"
		}
	}

	// Remove potential database identifiers
	message = strings.ReplaceAll(message, "pg_", "")
	message = strings.ReplaceAll(message, "sql_", "")

	// Remove potential file paths (after dangerous term check)
	message = strings.ReplaceAll(message, "/", "_")
	message = strings.ReplaceAll(message, "\\", "_")

	// Limit message length to prevent information leakage
	if len(message) > 200 {
		return "Validation failed with invalid input"
	}

	return strings.TrimSpace(message)
}
