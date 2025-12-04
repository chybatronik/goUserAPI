package errors

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestWriteSecureErrorResponse(t *testing.T) {
	testCases := []struct {
		name           string
		statusCode     int
		code           string
		message        string
		expectedBody   string
		expectedStatus int
	}{
		{
			name:           "validation error",
			statusCode:     http.StatusBadRequest,
			code:           "VALIDATION_ERROR",
			message:        "Request failed validation",
			expectedBody:   `{"error":"Request failed validation","code":"VALIDATION_ERROR"}`,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "rate limit error",
			statusCode:     http.StatusTooManyRequests,
			code:           "RATE_LIMIT_EXCEEDED",
			message:        "Too many requests",
			expectedBody:   `{"error":"Too many requests","code":"RATE_LIMIT_EXCEEDED"}`,
			expectedStatus: http.StatusTooManyRequests,
		},
		{
			name:           "server error",
			statusCode:     http.StatusInternalServerError,
			code:           "INTERNAL_ERROR",
			message:        "Internal server error",
			expectedBody:   `{"error":"Internal server error","code":"INTERNAL_ERROR"}`,
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name:           "not found error",
			statusCode:     http.StatusNotFound,
			code:           "NOT_FOUND",
			message:        "Resource not found",
			expectedBody:   `{"error":"Resource not found","code":"NOT_FOUND"}`,
			expectedStatus: http.StatusNotFound,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			w := httptest.NewRecorder()

			writeSecureErrorResponse(w, tc.statusCode, tc.code, tc.message)

			// Check status code
			if w.Code != tc.expectedStatus {
				t.Errorf("Expected status %d, got %d", tc.expectedStatus, w.Code)
			}

			// Check content type
			expectedContentType := "application/json"
			if contentType := w.Header().Get("Content-Type"); contentType != expectedContentType {
				t.Errorf("Expected Content-Type %s, got %s", expectedContentType, contentType)
			}

			// Check response body
			body := strings.TrimSpace(w.Body.String())
			if body != tc.expectedBody {
				t.Errorf("Expected body %s, got %s", tc.expectedBody, body)
			}

			// Parse JSON to verify structure
			var response ErrorResponse
			err := json.Unmarshal(w.Body.Bytes(), &response)
			if err != nil {
				t.Errorf("Failed to parse JSON response: %v", err)
			}

			if response.Error != tc.message {
				t.Errorf("Expected error message %s, got %s", tc.message, response.Error)
			}

			if response.Code != tc.code {
				t.Errorf("Expected error code %s, got %s", tc.code, response.Code)
			}
		})
	}
}

func TestErrorResponseStructure(t *testing.T) {
	w := httptest.NewRecorder()
	writeSecureErrorResponse(w, http.StatusBadRequest, "TEST_ERROR", "Test message")

	// Parse response to ensure it matches expected structure
	var response ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	if err != nil {
		t.Fatalf("Failed to parse JSON response: %v", err)
	}

	// Check that Details field is omitted when empty
	var rawResponse map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &rawResponse)
	if err != nil {
		t.Fatalf("Failed to parse raw JSON response: %v", err)
	}

	if _, exists := rawResponse["details"]; exists {
		t.Error("Expected 'details' field to be omitted when empty")
	}

	// Check required fields are present
	if response.Error == "" {
		t.Error("Expected 'error' field to be present")
	}

	if response.Code == "" {
		t.Error("Expected 'code' field to be present")
	}
}

func TestSecureResponseHeaders(t *testing.T) {
	w := httptest.NewRecorder()
	writeSecureErrorResponse(w, http.StatusTooManyRequests, "RATE_LIMIT_EXCEEDED", "Too many requests")

	// Check security headers
	expectedHeaders := map[string]string{
		"Content-Type": "application/json",
		"X-Content-Type-Options": "nosniff",
		"X-Frame-Options":         "DENY",
	}

	for header, expectedValue := range expectedHeaders {
		if actualValue := w.Header().Get(header); actualValue != expectedValue {
			t.Errorf("Expected header %s to be %s, got %s", header, expectedValue, actualValue)
		}
	}
}

func TestWriteSecureErrorResponseLogging(t *testing.T) {
	// Test that the function doesn't panic and produces a valid response
	// Logging is tested indirectly by ensuring the function executes without errors

	w := httptest.NewRecorder()

	// This should not panic and should produce a valid response
	writeSecureErrorResponse(w, http.StatusBadRequest, "TEST_ERROR", "Test message")

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}

	// Verify JSON response is valid
	var response ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	if err != nil {
		t.Errorf("Expected valid JSON response, got error: %v", err)
	}
}

func TestWriteSecureErrorResponseWithRequest(t *testing.T) {
	// Test with a mock request that has request ID
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("X-Request-ID", "test-request-id-123")

	w := httptest.NewRecorder()

	// Create a context with the request
	ctx := context.WithValue(req.Context(), "requestID", "test-request-id-123")
	req = req.WithContext(ctx)

	writeSecureErrorResponseWithRequest(w, req, http.StatusBadRequest, "TEST_ERROR", "Test message")

	// Verify response structure
	var response ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	if err != nil {
		t.Errorf("Expected valid JSON response, got error: %v", err)
	}

	if response.Error != "Test message" {
		t.Errorf("Expected error message 'Test message', got '%s'", response.Error)
	}

	if response.Code != "TEST_ERROR" {
		t.Errorf("Expected error code 'TEST_ERROR', got '%s'", response.Code)
	}
}

func TestWriteSecureErrorResponseLargeMessage(t *testing.T) {
	// Test with very large message to ensure it's handled safely
	largeMessage := strings.Repeat("This is a very long error message that should be handled safely. ", 100)

	w := httptest.NewRecorder()

	// This should not cause any issues
	writeSecureErrorResponse(w, http.StatusBadRequest, "LARGE_ERROR", largeMessage)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}

	// Verify response is still valid JSON
	var response ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	if err != nil {
		t.Errorf("Expected valid JSON response even with large message, got error: %v", err)
	}

	if response.Error != largeMessage {
		t.Errorf("Expected large error message to be preserved")
	}
}

func TestWriteSecureErrorResponseSpecialCharacters(t *testing.T) {
	// Test with special characters in message
	specialMessage := "Error with special chars: \"quotes\", <brackets>, &ampersand;, \\backslashes\\"

	w := httptest.NewRecorder()

	writeSecureErrorResponse(w, http.StatusBadRequest, "SPECIAL_CHARS", specialMessage)

	// Verify response is valid JSON
	var response ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	if err != nil {
		t.Errorf("Expected valid JSON response with special characters, got error: %v", err)
	}

	if response.Error != specialMessage {
		t.Errorf("Expected special characters to be preserved in error message")
	}
}