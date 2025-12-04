// Package handlers provides health check HTTP handler tests
package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/chybatronik/goUserAPI/internal/logging"
)

// MockHealthChecker implements HealthChecker interface for testing
type MockHealthChecker struct {
	name   string
	status string
	err    error
	delay  time.Duration
}

func (m *MockHealthChecker) Name() string {
	return m.name
}

func (m *MockHealthChecker) CheckHealth(ctx context.Context) HealthCheck {
	if m.delay > 0 {
		time.Sleep(m.delay)
	}

	healthCheck := HealthCheck{
		ResponseTimeMs: m.delay.Milliseconds(),
	}

	if m.err != nil {
		healthCheck.Status = "unhealthy"
		healthCheck.Error = m.err.Error()
	} else {
		healthCheck.Status = "healthy"
	}

	return healthCheck
}

// MockHealthCheckerDatabase implements HealthCheckerDatabase interface for testing
type MockHealthCheckerDatabase struct {
	err   error
	delay time.Duration
}

func (m *MockHealthCheckerDatabase) Ping(ctx context.Context) error {
	if m.delay > 0 {
		time.Sleep(m.delay)
	}
	return m.err
}

func TestNewHealthHandler(t *testing.T) {
	service := "test-service"
	version := "1.0.0"
	logger := logging.NewStructuredLogger("info", service, version)

	handler := NewHealthHandler(service, version, logger)

	if handler.service != service {
		t.Errorf("Expected service %s, got %s", service, handler.service)
	}
	if handler.version != version {
		t.Errorf("Expected version %s, got %s", version, handler.version)
	}
	if handler.logger != logger {
		t.Error("Expected logger to be set")
	}
}

func TestHealthHandlerPing(t *testing.T) {
	logger := logging.NewStructuredLogger("info", "test-service", "1.0.0")
	handler := NewHealthHandler("test-service", "1.0.0", logger)

	req := httptest.NewRequest("GET", "/health?ping=true", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if response["status"] != "ok" {
		t.Errorf("Expected status 'ok', got '%s'", response["status"])
	}
	if response["ping"] != "pong" {
		t.Errorf("Expected ping 'pong', got '%s'", response["ping"])
	}
}

func TestHealthHandlerHealthy(t *testing.T) {
	logger := logging.NewStructuredLogger("info", "test-service", "1.0.0")
	handler := NewHealthHandler("goUserAPI", "1.0.0", logger)

	// Add healthy checker
	healthyChecker := &MockHealthChecker{
		name:   "test-check",
		status: "healthy",
	}
	handler.AddChecker(healthyChecker)

	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response HealthCheckResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if response.Status != "healthy" {
		t.Errorf("Expected status 'healthy', got '%s'", response.Status)
	}
	if response.Service != "goUserAPI" {
		t.Errorf("Expected service 'goUserAPI', got '%s'", response.Service)
	}
	if response.Version != "1.0.0" {
		t.Errorf("Expected version '1.0.0', got '%s'", response.Version)
	}
	if response.UptimeSeconds < 0 {
		t.Error("Expected positive uptime")
	}

	// Check checker results
	check, exists := response.Checks["test-check"]
	if !exists {
		t.Error("Expected test-check in checks")
	}
	if check.Status != "healthy" {
		t.Errorf("Expected checker status 'healthy', got '%s'", check.Status)
	}
	if check.ResponseTimeMs < 0 {
		t.Error("Expected positive response time")
	}
	if check.Error != "" {
		t.Errorf("Expected empty error, got '%s'", check.Error)
	}
}

func TestHealthHandlerUnhealthy(t *testing.T) {
	logger := logging.NewStructuredLogger("info", "test-service", "1.0.0")
	handler := NewHealthHandler("goUserAPI", "1.0.0", logger)

	// Add unhealthy checker
	unhealthyChecker := &MockHealthChecker{
		name:   "failing-check",
		status: "unhealthy",
		err:    &testError{msg: "connection failed"},
	}
	handler.AddChecker(unhealthyChecker)

	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("Expected status %d, got %d", http.StatusServiceUnavailable, w.Code)
	}

	var response HealthCheckResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if response.Status != "unhealthy" {
		t.Errorf("Expected status 'unhealthy', got '%s'", response.Status)
	}

	// Check checker results
	check, exists := response.Checks["failing-check"]
	if !exists {
		t.Error("Expected failing-check in checks")
	}
	if check.Status != "unhealthy" {
		t.Errorf("Expected checker status 'unhealthy', got '%s'", check.Status)
	}
	if check.Error != "connection failed" {
		t.Errorf("Expected error 'connection failed', got '%s'", check.Error)
	}
}

func TestNewDatabaseHealthChecker(t *testing.T) {
	logger := logging.NewStructuredLogger("info", "test-service", "1.0.0")
	mockDB := &MockHealthCheckerDatabase{}

	checker := NewDatabaseHealthChecker(mockDB, logger)

	if checker.Name() != "database" {
		t.Errorf("Expected name 'database', got '%s'", checker.Name())
	}
	if checker.pool != mockDB {
		t.Error("Expected pool to be set")
	}
	if checker.logger != logger {
		t.Error("Expected logger to be set")
	}
}

func TestDatabaseHealthCheckerHealthy(t *testing.T) {
	logger := logging.NewStructuredLogger("info", "test-service", "1.0.0")
	mockDB := &MockHealthCheckerDatabase{
		delay: 10 * time.Millisecond,
	}

	checker := NewDatabaseHealthChecker(mockDB, logger)

	ctx := context.Background()
	healthCheck := checker.CheckHealth(ctx)

	if healthCheck.Status != "healthy" {
		t.Errorf("Expected status 'healthy', got '%s'", healthCheck.Status)
	}
	if healthCheck.ResponseTimeMs < 10 {
		t.Errorf("Expected response time at least 10ms, got %d", healthCheck.ResponseTimeMs)
	}
	if healthCheck.Error != "" {
		t.Errorf("Expected empty error, got '%s'", healthCheck.Error)
	}
}

func TestDatabaseHealthCheckerUnhealthy(t *testing.T) {
	logger := logging.NewStructuredLogger("info", "test-service", "1.0.0")
	mockDB := &MockHealthCheckerDatabase{
		err: &testError{msg: "database connection failed"},
	}

	checker := NewDatabaseHealthChecker(mockDB, logger)

	ctx := context.Background()
	healthCheck := checker.CheckHealth(ctx)

	if healthCheck.Status != "unhealthy" {
		t.Errorf("Expected status 'unhealthy', got '%s'", healthCheck.Status)
	}
	if healthCheck.Error != "database connection failed" {
		t.Errorf("Expected error 'database connection failed', got '%s'", healthCheck.Error)
	}
}

func TestDatabaseHealthCheckerPing(t *testing.T) {
	logger := logging.NewStructuredLogger("info", "test-service", "1.0.0")
	mockDB := &MockHealthCheckerDatabase{}

	checker := NewDatabaseHealthChecker(mockDB, logger)

	ctx := context.Background()
	err := checker.pool.Ping(ctx)

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
}

// testError implements error interface for testing
type testError struct {
	msg string
}

func (e *testError) Error() string {
	return e.msg
}