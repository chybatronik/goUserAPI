package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestErrorHandler_Success(t *testing.T) {
	// Create a handler that returns success
	successHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("success"))
	})

	errorHandler := NewErrorHandler(successHandler)

	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()

	errorHandler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	if w.Body.String() != "success" {
		t.Errorf("Expected body 'success', got '%s'", w.Body.String())
	}
}

func TestErrorHandler_Error(t *testing.T) {
	// Create a handler that returns an error
	errorHandlerFunc := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	})

	errorHandler := NewErrorHandler(errorHandlerFunc)

	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()

	errorHandler.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
	}

	// Should contain JSON error response
	expectedContentType := "application/json"
	if w.Header().Get("Content-Type") != expectedContentType {
		t.Errorf("Expected content type '%s', got '%s'", expectedContentType, w.Header().Get("Content-Type"))
	}
}

func TestNewErrorHandler(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})
	errorHandler := NewErrorHandler(next)

	if errorHandler.next == nil {
		t.Error("Expected next handler to be set")
	}
}

func TestErrorHandler_PanicRecovery(t *testing.T) {
	// Create a handler that panics
	panicHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("test panic")
	})

	errorHandler := NewErrorHandler(panicHandler)

	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()

	errorHandler.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("Expected status %d after panic recovery, got %d", http.StatusInternalServerError, w.Code)
	}

	// Should contain error response JSON
	expectedContentType := "application/json"
	if w.Header().Get("Content-Type") != expectedContentType {
		t.Errorf("Expected content type '%s', got '%s'", expectedContentType, w.Header().Get("Content-Type"))
	}
}
