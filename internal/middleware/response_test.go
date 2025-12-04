package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNewResponseWriter(t *testing.T) {
	w := httptest.NewRecorder()
	rw := NewResponseWriter(w)

	if rw.StatusCode() != http.StatusOK {
		t.Errorf("Expected initial status %d, got %d", http.StatusOK, rw.StatusCode())
	}
}

func TestResponseWriter_WriteHeader(t *testing.T) {
	w := httptest.NewRecorder()
	rw := NewResponseWriter(w)

	rw.WriteHeader(http.StatusNotFound)

	if rw.StatusCode() != http.StatusNotFound {
		t.Errorf("Expected status %d, got %d", http.StatusNotFound, rw.StatusCode())
	}

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected underlying writer status %d, got %d", http.StatusNotFound, w.Code)
	}
}

func TestResponseWriter_Write(t *testing.T) {
	w := httptest.NewRecorder()
	rw := NewResponseWriter(w)

	data := []byte("test data")
	n, err := rw.Write(data)

	if err != nil {
		t.Errorf("Expected no error on write, got %v", err)
	}

	if n != len(data) {
		t.Errorf("Expected written bytes %d, got %d", len(data), n)
	}

	if w.Body.String() != string(data) {
		t.Errorf("Expected body '%s', got '%s'", string(data), w.Body.String())
	}
}

func TestResponseWriter_Header(t *testing.T) {
	w := httptest.NewRecorder()
	rw := NewResponseWriter(w)

	rw.Header().Set("Content-Type", "application/json")

	if w.Header().Get("Content-Type") != "application/json" {
		t.Errorf("Expected Content-Type 'application/json', got '%s'", w.Header().Get("Content-Type"))
	}

	if rw.Header().Get("Content-Type") != "application/json" {
		t.Errorf("Expected ResponseWriter Content-Type 'application/json', got '%s'", rw.Header().Get("Content-Type"))
	}
}
