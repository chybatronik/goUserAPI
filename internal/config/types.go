// Package config provides configuration types and structures for the goUserAPI service.
package config

// Config represents the application configuration
type Config struct {
	Server        ServerConfig
	Database      DatabaseConfig
	Logging       LoggingConfig
	HealthCheck   HealthCheckConfig
	Application   ApplicationConfig
}

// ServerConfig holds HTTP server configuration
type ServerConfig struct {
	Port         int    // Server port number
	Host         string // Server host address
	ReadTimeout  int    // Read timeout in seconds
	WriteTimeout int    // Write timeout in seconds
	IdleTimeout  int    // Idle timeout in seconds
	Debug        bool   // Enable debug mode
}

// DatabaseConfig holds database configuration
type DatabaseConfig struct {
	Host     string // Database host address
	Port     int    // Database port number
	User     string // Database username
	Password string // Database password
	Database string // Database name
	SSLMode  string // SSL mode (disable, require, etc.)
	MaxConns int    // Maximum database connections
	MinConns int    // Minimum database connections
}

// LoggingConfig holds logging configuration
type LoggingConfig struct {
	Level  string // Log level (debug, info, warn, error)
	Format string // Log format (json, text)
}

// HealthCheckConfig holds health check configuration
type HealthCheckConfig struct {
	Enabled bool   // Enable health check endpoint
	Port    int    // Health check port (deprecated, uses APP_PORT)
	Host    string // Health check host (deprecated, uses APP_HOST)
}

// ApplicationConfig holds application-specific configuration
type ApplicationConfig struct {
	Environment       string // Environment (development, staging, production)
	ShutdownTimeout   int    // Shutdown timeout in seconds
	RateLimitRequests int    // Rate limit requests per window
	RateLimitWindow   string // Rate limit time window
	MetricsEnabled    bool   // Enable metrics collection
}