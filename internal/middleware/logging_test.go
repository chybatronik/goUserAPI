package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/chybatronik/goUserAPI/internal/logging"
)

func TestLoggingMiddleware(t *testing.T) {
	// Create a mock logger
	logger := logging.NewStructuredLogger("info", "test-service", "1.0.0")

	// Create a handler that returns success
	successHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("success"))
	})

	loggingMiddleware := NewLoggingMiddleware(logger, successHandler)

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	loggingMiddleware.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	if w.Body.String() != "success" {
		t.Errorf("Expected body 'success', got '%s'", w.Body.String())
	}
}

func TestNewLoggingMiddleware(t *testing.T) {
	logger := logging.NewStructuredLogger("info", "test-service", "1.0.0")
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})
	loggingMiddleware := NewLoggingMiddleware(logger, next)

	if loggingMiddleware.next == nil {
		t.Error("Expected next handler to be set")
	}
	if loggingMiddleware.logger == nil {
		t.Error("Expected logger to be set")
	}
}
