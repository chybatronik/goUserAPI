// Package middleware provides request tracking middleware tests
package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGenerateRequestID(t *testing.T) {
	// Generate multiple request IDs to test uniqueness
	ids := make(map[string]bool)
	for i := 0; i < 100; i++ {
		id := GenerateRequestID()
		if id == "" {
			t.Error("GenerateRequestID() returned empty string")
		}
		if ids[id] {
			t.Errorf("GenerateRequestID() generated duplicate ID: %s", id)
		}
		ids[id] = true
	}
}

func TestGetRequestID(t *testing.T) {
	ctx := context.Background()

	// Test with no request ID in context
	if reqID := GetRequestID(ctx); reqID != "" {
		t.Errorf("Expected empty request ID, got %s", reqID)
	}

	// Test with request ID in context
	reqID := "test-req-123"
	ctx = context.WithValue(ctx, RequestIDContextKey, reqID)

	if gotReqID := GetRequestID(ctx); gotReqID != reqID {
		t.Errorf("Expected request ID %s, got %s", reqID, gotReqID)
	}
}

func TestSetRequestID(t *testing.T) {
	ctx := context.Background()
	reqID := "test-req-456"

	ctxWithReqID := SetRequestID(ctx, reqID)
	if gotReqID := GetRequestID(ctxWithReqID); gotReqID != reqID {
		t.Errorf("Expected request ID %s, got %s", reqID, gotReqID)
	}

	// Original context should be unchanged
	if originalReqID := GetRequestID(ctx); originalReqID != "" {
		t.Errorf("Expected empty request ID in original context, got %s", originalReqID)
	}
}

func TestResponseWriter(t *testing.T) {
	// Test default status code
	rw := NewResponseWriter(httptest.NewRecorder())
	if rw.StatusCode() != http.StatusOK {
		t.Errorf("Expected default status code %d, got %d", http.StatusOK, rw.StatusCode())
	}

	// Test setting status code
	expectedStatus := http.StatusNotFound
	rw.WriteHeader(expectedStatus)
	if rw.StatusCode() != expectedStatus {
		t.Errorf("Expected status code %d, got %d", expectedStatus, rw.StatusCode())
	}
}

func TestRequestIDMiddleware(t *testing.T) {
	// Create a test handler that checks request ID
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reqID := GetRequestID(r.Context())
		if reqID == "" {
			t.Error("Request ID not found in context")
		}

		// Check if response header is set
		responseReqID := w.Header().Get(RequestIDHeader)
		if responseReqID != reqID {
			t.Errorf("Response header request ID %s doesn't match context request ID %s", responseReqID, reqID)
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// Create middleware chain
	handler := RequestIDMiddleware(testHandler)

	// Test without request ID header
	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	responseReqID := w.Header().Get(RequestIDHeader)
	if responseReqID == "" {
		t.Error("Request ID header not set in response")
	}

	// Test with existing request ID header
	existingReqID := "existing-req-123"
	req.Header.Set(RequestIDHeader, existingReqID)
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	responseReqID = w.Header().Get(RequestIDHeader)
	if responseReqID != existingReqID {
		t.Errorf("Expected request ID header %s, got %s", existingReqID, responseReqID)
	}
}
