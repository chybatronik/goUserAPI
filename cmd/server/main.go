// Package main provides the entry point for the goUserAPI service.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/chybatronik/goUserAPI/internal/config"
	"github.com/chybatronik/goUserAPI/internal/database"
	"github.com/chybatronik/goUserAPI/internal/handlers"
	"github.com/chybatronik/goUserAPI/internal/logging"
	"github.com/chybatronik/goUserAPI/internal/middleware"
	"github.com/chybatronik/goUserAPI/internal/models"
	"github.com/chybatronik/goUserAPI/internal/types"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	// Build information (set during build)
	Version   = "dev"
	BuildTime = ""
)

// DatabaseAdapter implements the handlers.DatabaseService interface
type DatabaseAdapter struct{}

// CreateUser implements the DatabaseService interface
func (da *DatabaseAdapter) CreateUser(ctx context.Context, pool *pgxpool.Pool, user *models.User) (*models.User, error) {
	return database.CreateUser(ctx, pool, user)
}

// GetUsers implements the DatabaseService interface
func (da *DatabaseAdapter) GetUsers(ctx context.Context, pool *pgxpool.Pool, params types.GetUsersParams) ([]models.User, int64, error) {
	return database.GetUsers(ctx, pool, params)
}

// GetReports implements the DatabaseService interface (Story 3.1)
func (da *DatabaseAdapter) GetReports(ctx context.Context, pool *pgxpool.Pool, params types.GetReportsParams) ([]models.User, int64, error) {
	return database.GetReports(ctx, pool, params)
}

func main() {
	// Initialize configuration first
	appConfig, err := config.Load()
	if err != nil {
		log.Fatalf("FATAL: Failed to load configuration: %v", err)
	}

	// Setup structured logging
	logger := setupStructuredLogging(appConfig)

	// Log startup events
	logStartupEvents(logger, appConfig)

	logger.Startup("Initializing database connection...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Create connection pool
	pool, err := database.NewConnectionPool(appConfig)
	if err != nil {
		logger.Error("Failed to create database connection pool", logging.FieldError, err)
		log.Fatalf("FATAL: Failed to create database connection pool: %v", err)
	}
	defer pool.Close()

	// Validate database connection
	if err := database.ValidateConnection(ctx, pool); err != nil {
		logger.Error("Database connection validation failed", logging.FieldError, err)
		log.Fatalf("FATAL: Database connection validation failed: %v", err)
	}

	logger.Database("Database connection established successfully")

	// Run automatic migrations before starting HTTP server
	logger.Startup("Running database migrations...")
	migrationRunner := database.NewMigrationRunner(pool, "./migrations")

	if err := migrationRunner.RunMigrations(ctx); err != nil {
		logger.Error("Database migration failed", logging.FieldError, err)
		log.Fatalf("FATAL: Database migration failed: %v", err)
	}

	logger.Database("Database migrations completed successfully")

	// Setup HTTP server with graceful shutdown
	server := setupHTTPServer(appConfig, pool, logger)

	// Start server in a goroutine
	go func() {
		logger.Startup("HTTP server starting",
			"host", appConfig.Server.Host,
			"port", appConfig.Server.Port,
		)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("HTTP server failed to start", logging.FieldError, err)
			log.Fatalf("FATAL: HTTP server failed to start: %v", err)
		}
	}()

	logger.Startup("goUserAPI service started successfully")

	// Graceful shutdown handling
	gracefulShutdown(server, pool, appConfig.Application.ShutdownTimeout, logger)
}

// setupHTTPServer configures and returns an HTTP server with structured logging and middleware
func setupHTTPServer(appConfig *config.Config, pool *pgxpool.Pool, logger *logging.Logger) *http.Server {
	// Setup health check handler with structured logging
	healthHandler := handlers.NewHealthHandler("goUserAPI", Version, logger)

	// Add database health checker if enabled
	if appConfig.HealthCheck.Enabled {
		dbHealthChecker := database.NewHealthChecker(pool)
		healthHandler.AddChecker(dbHealthChecker)
	}

	// Setup user handler
	dbAdapter := &DatabaseAdapter{}
	userHandler := handlers.NewUserHandler(logger, pool, dbAdapter)

	// Setup report handler (Story 3.1)
	reportHandler := handlers.NewReportHandler(logger, pool, dbAdapter)

	// Create rate limiters once at startup (NOT per request)
	// Reports endpoint gets stricter rate limiting due to potential resource intensity
	reportsRateLimiter := middleware.SecurityRateLimit(50.0/60.0, 10) // 50 req/min, burst 10 (stricter than global 100/min)

	// Create router
	mux := http.NewServeMux()

	// Register routes
	mux.HandleFunc("/health", healthHandler.ServeHTTP)
	mux.HandleFunc("/users", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			userHandler.GetUsers(w, r)    // NEW from Story 2.3
		case http.MethodPost:
			userHandler.CreateUser(w, r)  // EXISTING from Story 2.2
		default:
			// Comprehensive Method Not Allowed response with proper headers
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("Allow", "GET, POST") // Explicitly list allowed methods
			w.WriteHeader(http.StatusMethodNotAllowed)

			errorResponse := map[string]interface{}{
				"error":   "Method not allowed",
				"code":    "METHOD_NOT_ALLOWED",
				"details": fmt.Sprintf("Method %s is not allowed. Supported methods: GET, POST", r.Method),
			}

			if err := json.NewEncoder(w).Encode(errorResponse); err != nil {
				// Fallback to simple error response if JSON encoding fails
				http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			}
		}
	})

	mux.HandleFunc("/reports", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			// SECURITY: Apply pre-created Story 2.4 endpoint-specific rate limiting for reports
			// Reports endpoint gets stricter rate limiting due to potential resource intensity
			reportsRateLimiter(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				reportHandler.GetReports(w, r)  // NEW from Story 3.1
			})).ServeHTTP(w, r)
		default:
			// Comprehensive Method Not Allowed response with proper headers
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("Allow", "GET") // Explicitly list allowed methods
			w.WriteHeader(http.StatusMethodNotAllowed)

			errorResponse := map[string]interface{}{
				"error":   "Method not allowed",
				"code":    "METHOD_NOT_ALLOWED",
				"details": fmt.Sprintf("Method %s is not allowed. Supported methods: GET", r.Method),
			}

			if err := json.NewEncoder(w).Encode(errorResponse); err != nil {
				// Fallback to simple error response if JSON encoding fails
				http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			}
		}
	})
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("goUserAPI is running"))
	})

	// Setup middleware chain with request ID, security, and structured logging
	// Order matters: Security -> RequestID -> Logging -> Router
	// Security middleware should be first to validate input and enforce rate limits
	handler := http.Handler(mux)
	handler = middleware.NewLoggingMiddleware(logger, handler)     // Apply logging last
	handler = middleware.RequestIDMiddleware(handler)             // Apply request ID second
	handler = middleware.SecurityRateLimit(100.0/60.0, 20)(handler) // Apply security rate limiting first (100 req/min, burst 20)

	// Configure server with timeouts
	server := &http.Server{
		Addr:         fmt.Sprintf("%s:%d", appConfig.Server.Host, appConfig.Server.Port),
		Handler:      handler,
		ReadTimeout:  time.Duration(appConfig.Server.ReadTimeout) * time.Second,
		WriteTimeout: time.Duration(appConfig.Server.WriteTimeout) * time.Second,
		IdleTimeout:  time.Duration(appConfig.Server.IdleTimeout) * time.Second,
	}

	return server
}

// gracefulShutdown handles graceful shutdown of the service with structured logging
func gracefulShutdown(server *http.Server, pool *pgxpool.Pool, shutdownTimeout int, logger *logging.Logger) {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Wait for signal
	sig := <-sigChan
	logger.Startup("Received signal, initiating graceful shutdown", "signal", sig.String())

	// Create shutdown context with timeout
	shutdownCtx, cancel := context.WithTimeout(context.Background(), time.Duration(shutdownTimeout)*time.Second)
	defer cancel()

	// Shutdown HTTP server
	logger.Startup("Shutting down HTTP server...")
	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.Error("HTTP server shutdown failed", logging.FieldError, err)
	} else {
		logger.Startup("HTTP server shutdown completed")
	}

	// Close database connections
	logger.Startup("Closing database connections...")
	pool.Close()
	logger.Startup("Database connections closed")

	logger.Startup("goUserAPI service shutdown completed")
}

// setupStructuredLogging initializes the structured logger based on configuration
func setupStructuredLogging(cfg *config.Config) *logging.Logger {
	logger := logging.NewStructuredLogger(
		cfg.Logging.Level,
		"goUserAPI",
		Version,
	)

	return logger.WithServiceContext()
}

// logStartupEvents logs comprehensive startup information
func logStartupEvents(logger *logging.Logger, cfg *config.Config) {
	logger.Startup("goUserAPI service starting up",
		"version", Version,
		"service", "goUserAPI",
	)

	logger.Startup("configuration loaded successfully",
		"environment", cfg.Application.Environment,
		"log_level", cfg.Logging.Level,
		"server_port", cfg.Server.Port,
		"server_host", cfg.Server.Host,
		"db_host", cfg.Database.Host,
		"db_port", cfg.Database.Port,
		"db_name", cfg.Database.Database,
		"health_check_enabled", cfg.HealthCheck.Enabled,
	)
}
