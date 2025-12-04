// Package types provides shared types for health checking
package types

import "context"

// HealthCheck represents the health check result
type HealthCheck struct {
	Status    string            `json:"status"`
	Timestamp int64             `json:"timestamp"`
	Details   map[string]string `json:"details,omitempty"`
}

// HealthChecker defines the interface for health check implementations
type HealthChecker interface {
	CheckHealth(ctx context.Context) HealthCheck
}

// HealthCheckerDatabase defines additional interface for database-specific health checks
type HealthCheckerDatabase interface {
	HealthChecker
	Ping(ctx context.Context) error
}
