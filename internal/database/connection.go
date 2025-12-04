package database

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/chybatronik/goUserAPI/internal/config"
)

// NewConnectionPool creates a new PostgreSQL connection pool
// Source: Architecture.md#Database-Driver - pgx v5+ with connection pooling
func NewConnectionPool(appConfig *config.Config) (*pgxpool.Pool, error) {
	ctx := context.Background()

	// Build connection string from config
	connString := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		appConfig.Database.Host,
		appConfig.Database.Port,
		appConfig.Database.User,
		appConfig.Database.Password,
		appConfig.Database.Database,
		appConfig.Database.SSLMode)

	poolConfig, err := pgxpool.ParseConfig(connString)
	if err != nil {
		return nil, fmt.Errorf("unable to parse database config: %w", err)
	}

	// Configure connection pool using config values with performance optimization
	poolConfig.MaxConns = int32(appConfig.Database.MaxConns)
	poolConfig.MinConns = int32(appConfig.Database.MinConns)

	// Performance optimizations for user operations (AC #5)
	// Health check configuration for proactive connection validation
	poolConfig.HealthCheckPeriod = 1 * time.Minute  // Check connection health every minute
	poolConfig.MaxConnLifetime = 30 * time.Minute    // Recycle connections after 30 minutes
	poolConfig.MaxConnIdleTime = 5 * time.Minute     // Close idle connections after 5 minutes

	// Connection acquisition timeout for responsive error handling
	// Note: AcquireTimeout was removed in pgx v5.5+, using MaxConnIdleTime instead

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, fmt.Errorf("unable to create connection pool: %w", err)
	}

	// Test connection
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("unable to connect to database: %w", err)
	}

	return pool, nil
}

// ValidateConnection checks if database connection is working
func ValidateConnection(ctx context.Context, pool *pgxpool.Pool) error {
	if pool == nil {
		return fmt.Errorf("connection pool is nil")
	}
	return pool.Ping(ctx)
}