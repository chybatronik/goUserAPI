// Package database provides database functionality for the goUserAPI service.
package database

import (
	"context"
	"fmt"
	"time"

	"github.com/chybatronik/goUserAPI/internal/handlers"
	"github.com/jackc/pgx/v5/pgxpool"
)

// HealthChecker implements database health checking with timing
type HealthChecker struct {
	db *pgxpool.Pool
}

// NewHealthChecker creates a new database health checker
func NewHealthChecker(db *pgxpool.Pool) *HealthChecker {
	return &HealthChecker{db: db}
}

// Name implements the handlers.HealthChecker interface
func (h *HealthChecker) Name() string {
	return "database"
}

// CheckHealth checks database connectivity with timing
func (h *HealthChecker) CheckHealth(ctx context.Context) handlers.HealthCheck {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	start := time.Now()
	err := h.db.Ping(ctx)
	responseTime := time.Since(start).Milliseconds()

	healthCheck := handlers.HealthCheck{
		Status:         "healthy",
		ResponseTimeMs: responseTime,
	}

	if err != nil {
		healthCheck.Status = "unhealthy"
		healthCheck.Error = fmt.Sprintf("database connection failed: %v", err)
	}

	return healthCheck
}

// Ping implements the handlers.HealthCheckerDatabase interface for simple connectivity test
func (h *HealthChecker) Ping(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	return h.db.Ping(ctx)
}
