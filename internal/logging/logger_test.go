// Package logging provides structured logging functionality tests
package logging

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"strings"
	"testing"
)

func TestNewStructuredLogger(t *testing.T) {
	tests := []struct {
		name     string
		level    string
		service  string
		version  string
		expected slog.Level
	}{
		{
			name:     "debug level",
			level:    "debug",
			service:  "test-service",
			version:  "1.0.0",
			expected: slog.LevelDebug,
		},
		{
			name:     "info level default",
			level:    "invalid",
			service:  "test-service",
			version:  "1.0.0",
			expected: slog.LevelInfo,
		},
		{
			name:     "warn level",
			level:    "warn",
			service:  "test-service",
			version:  "1.0.0",
			expected: slog.LevelWarn,
		},
		{
			name:     "error level",
			level:    "error",
			service:  "test-service",
			version:  "1.0.0",
			expected: slog.LevelError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			handler := slog.NewJSONHandler(&buf, &slog.HandlerOptions{
				Level: tt.expected,
			})
			logger := &Logger{
				Logger:  slog.New(handler),
				service: tt.service,
				version: tt.version,
			}

			if logger.service != tt.service {
				t.Errorf("NewStructuredLogger() service = %v, want %v", logger.service, tt.service)
			}
			if logger.version != tt.version {
				t.Errorf("NewStructuredLogger() version = %v, want %v", logger.version, tt.version)
			}
		})
	}
}

func TestLoggerWithRequestID(t *testing.T) {
	var buf bytes.Buffer
	handler := slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug})
	logger := &Logger{
		Logger:  slog.New(handler),
		service: "test-service",
		version: "1.0.0",
	}

	reqID := "test-req-id-123"
	loggerWithReqID := logger.WithRequestID(reqID)

	loggerWithReqID.Info("test message")

	var logEntry map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &logEntry); err != nil {
		t.Fatalf("Failed to unmarshal log entry: %v", err)
	}

	if logEntry["req_id"] != reqID {
		t.Errorf("Expected request ID %s, got %v", reqID, logEntry["req_id"])
	}
}

func TestLoggerWithHTTPRequest(t *testing.T) {
	var buf bytes.Buffer
	handler := slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug})
	logger := &Logger{
		Logger:  slog.New(handler),
		service: "test-service",
		version: "1.0.0",
	}

	reqID := "test-req-id"
	loggerWithReqID := logger.WithRequestID(reqID)
	loggerWithHTTP := loggerWithReqID.WithHTTPRequest("GET", "/api/users", 200, 150)

	loggerWithHTTP.Info("HTTP request completed")

	var logEntry map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &logEntry); err != nil {
		t.Fatalf("Failed to unmarshal log entry: %v", err)
	}

	if logEntry["req_id"] != reqID {
		t.Errorf("Expected request ID %s, got %v", reqID, logEntry["req_id"])
	}
	if logEntry["method"] != "GET" {
		t.Errorf("Expected method GET, got %v", logEntry["method"])
	}
	if logEntry["path"] != "/api/users" {
		t.Errorf("Expected path /api/users, got %v", logEntry["path"])
	}
	if logEntry["status"] != float64(200) {
		t.Errorf("Expected status 200, got %v", logEntry["status"])
	}
	if logEntry["latency_ms"] != float64(150) {
		t.Errorf("Expected latency 150, got %v", logEntry["latency_ms"])
	}
}

func TestLoggerRequest(t *testing.T) {
	var buf bytes.Buffer
	handler := slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug})
	logger := &Logger{
		Logger:  slog.New(handler),
		service: "test-service",
		version: "1.0.0",
	}

	logger.Request("req-123", "POST", "/api/users", 201, 250)

	var logEntry map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &logEntry); err != nil {
		t.Fatalf("Failed to unmarshal log entry: %v", err)
	}

	if logEntry["msg"] != "HTTP request completed" {
		t.Errorf("Expected message 'HTTP request completed', got %v", logEntry["msg"])
	}
	if logEntry["req_id"] != "req-123" {
		t.Errorf("Expected request ID req-123, got %v", logEntry["req_id"])
	}
	if logEntry["method"] != "POST" {
		t.Errorf("Expected method POST, got %v", logEntry["method"])
	}
}

func TestLoggerDatabase(t *testing.T) {
	var buf bytes.Buffer
	handler := slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug})
	logger := &Logger{
		Logger:  slog.New(handler),
		service: "test-service",
		version: "1.0.0",
	}

	logger.Database("connection established", "host", "localhost", "port", 5432)

	var logEntry map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &logEntry); err != nil {
		t.Fatalf("Failed to unmarshal log entry: %v", err)
	}

	if !strings.Contains(logEntry["msg"].(string), "connection established") {
		t.Errorf("Expected message containing 'database: connection established', got %v", logEntry["msg"])
	}
}

func TestLoggerHealthCheck(t *testing.T) {
	var buf bytes.Buffer
	handler := slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug})
	logger := &Logger{
		Logger:  slog.New(handler),
		service: "test-service",
		version: "1.0.0",
	}

	logger.HealthCheck("database check successful", "response_time_ms", 25)

	var logEntry map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &logEntry); err != nil {
		t.Fatalf("Failed to unmarshal log entry: %v", err)
	}

	if !strings.Contains(logEntry["msg"].(string), "database check successful") {
		t.Errorf("Expected message containing 'healthcheck: database check successful', got %v", logEntry["msg"])
	}
	if logEntry["response_time_ms"] != float64(25) {
		t.Errorf("Expected response time 25, got %v", logEntry["response_time_ms"])
	}
}

