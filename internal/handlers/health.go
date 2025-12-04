// Package handlers provides HTTP handlers for the goUserAPI service.
package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/chybatronik/goUserAPI/internal/logging"
)

// HealthCheckResponse represents the structured health check response format
// MANDATORY format from Story 1.4 requirements
type HealthCheckResponse struct {
	Status         string                    `json:"status"`         // healthy|unhealthy
	Timestamp      int64                     `json:"timestamp"`      // Unix timestamp
	Service        string                    `json:"service"`        // goUserAPI
	Version        string                    `json:"version"`        // 1.0.0
	UptimeSeconds  int64                     `json:"uptime_seconds"` // Uptime in seconds
	Checks         map[string]HealthCheck    `json:"checks"`
}

// HealthCheck represents individual health check result with timing
type HealthCheck struct {
	Status         string `json:"status"`         // healthy|unhealthy
	ResponseTimeMs int64  `json:"response_time_ms"` // Response time in ms
	Error          string `json:"error,omitempty"` // Only present if unhealthy
}

// HealthChecker interface for health check components
type HealthChecker interface {
	CheckHealth(ctx context.Context) HealthCheck
	Name() string
}

// HealthHandler provides health check functionality with performance metrics
type HealthHandler struct {
	checkers []HealthChecker
	startTime time.Time
	version   string
	service   string
	mu        sync.RWMutex
	logger    *logging.Logger
}

// NewHealthHandler creates a new health handler
func NewHealthHandler(service, version string, logger *logging.Logger) *HealthHandler {
	return &HealthHandler{
		checkers: make([]HealthChecker, 0),
		startTime: time.Now(),
		version:   version,
		service:   service,
		logger:    logger,
	}
}

// AddChecker adds a health checker to the handler
func (h *HealthHandler) AddChecker(checker HealthChecker) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.checkers = append(h.checkers, checker)
}

// ServeHTTP handles health check requests with proper format and performance tracking
func (h *HealthHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	ctx := r.Context()

	// Simple ping response for quick health checks
	if r.URL.Query().Get("ping") == "true" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{
			"status": "ok",
			"ping":   "pong",
		})
		return
	}

	// Build health check response
	response := HealthCheckResponse{
		Timestamp:     time.Now().Unix(),
		Service:       h.service,
		Version:       h.version,
		UptimeSeconds: int64(time.Since(h.startTime).Seconds()),
		Checks:        make(map[string]HealthCheck),
	}

	// Run all health checks with timing
	allHealthy := true
	h.mu.RLock()
	checkers := make([]HealthChecker, len(h.checkers))
	copy(checkers, h.checkers)
	h.mu.RUnlock()

	for _, checker := range checkers {
		healthCheck := checker.CheckHealth(ctx)
		response.Checks[checker.Name()] = healthCheck

		if healthCheck.Status != "healthy" {
			allHealthy = false
			h.logger.HealthCheck("health check failed",
				"check_name", checker.Name(),
				"check_status", healthCheck.Status,
				"error", healthCheck.Error,
			)
		}
	}

	// Determine overall status
	if allHealthy {
		response.Status = "healthy"
		w.WriteHeader(http.StatusOK)
		h.logger.HealthCheck("health check completed successfully",
			logging.FieldResponseTime, time.Since(start).Milliseconds(),
		)
	} else {
		response.Status = "unhealthy"
		w.WriteHeader(http.StatusServiceUnavailable)
		h.logger.HealthCheck("health check completed with failures",
			logging.FieldResponseTime, time.Since(start).Milliseconds(),
		)
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		h.logger.Error("failed to encode health check response", logging.FieldError, err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// DatabaseHealthChecker checks database connectivity with timing
type DatabaseHealthChecker struct {
	pool   HealthCheckerDatabase
	logger *logging.Logger
}

// HealthCheckerDatabase interface for database health checking
type HealthCheckerDatabase interface {
	Ping(ctx context.Context) error
}

// NewDatabaseHealthChecker creates a new database health checker
func NewDatabaseHealthChecker(pool HealthCheckerDatabase, logger *logging.Logger) *DatabaseHealthChecker {
	return &DatabaseHealthChecker{
		pool:   pool,
		logger: logger,
	}
}

// Name returns the checker name
func (d *DatabaseHealthChecker) Name() string {
	return "database"
}

// CheckHealth performs the database health check with timing
func (d *DatabaseHealthChecker) CheckHealth(ctx context.Context) HealthCheck {
	start := time.Now()

	err := d.pool.Ping(ctx)
	responseTime := time.Since(start).Milliseconds()

	healthCheck := HealthCheck{
		ResponseTimeMs: responseTime,
	}

	if err != nil {
		healthCheck.Status = "unhealthy"
		healthCheck.Error = err.Error()
		d.logger.DatabaseError("database health check failed", err)
	} else {
		healthCheck.Status = "healthy"
		d.logger.Database("database health check successful",
			logging.FieldResponseTime, responseTime,
		)
	}

	return healthCheck
}