// Package config provides configuration loading and environment management
package config

import (
	"fmt"
	"log"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

// ValidationError represents a configuration validation error
type ValidationError struct {
	Field   string
	Value   string
	Message string
}

func (e ValidationError) Error() string {
	return fmt.Sprintf("validation failed for %s='%s': %s", e.Field, e.Value, e.Message)
}

// ValidationErrors represents multiple validation errors
type ValidationErrors []ValidationError

func (ve ValidationErrors) Error() string {
	if len(ve) == 0 {
		return ""
	}

	msg := "configuration validation errors:\n"
	for _, err := range ve {
		msg += fmt.Sprintf("  - %s\n", err.Error())
	}
	return msg
}

// RequiredEnvironmentVariables defines required environment variables
var RequiredEnvironmentVariables = []string{
	"DB_HOST",
	"DB_USER",
	"DB_PASSWORD",
	"DB_NAME",
}

// OptionalEnvironmentVariables defines optional environment variables with defaults
var OptionalEnvironmentVariables = map[string]string{
	"DB_PORT":              "5432",
	"DB_SSL_MODE":          "disable",
	"DB_MAX_CONNECTIONS":   "25",
	"DB_MIN_CONNS":         "5",
	"APP_HOST":             "0.0.0.0",
	"APP_PORT":             "8080",
	"LOG_LEVEL":            "info",
	"LOG_FORMAT":           "json",
	"ENVIRONMENT":          "development",
	"SERVER_DEBUG":         "false",
	"SERVER_READ_TIMEOUT":  "30",
	"SERVER_WRITE_TIMEOUT": "30",
	"SERVER_IDLE_TIMEOUT":  "120",
	"SHUTDOWN_TIMEOUT":     "30",
	"RATE_LIMIT_REQUESTS":  "100",
	"RATE_LIMIT_WINDOW":    "1m",
	"METRICS_ENABLED":      "false",
	"HEALTH_CHECK_ENABLED": "true",
}

// ValidateRequired validates required environment variables
func ValidateRequired() ValidationErrors {
	var errors ValidationErrors

	for _, envVar := range RequiredEnvironmentVariables {
		if value := os.Getenv(envVar); value == "" {
			errors = append(errors, ValidationError{
				Field:   envVar,
				Value:   "",
				Message: "required environment variable is not set",
			})
		}
	}

	return errors
}

// ValidatePort validates that a port number is in valid range
func ValidatePort(envVar string) error {
	portStr := os.Getenv(envVar)
	if portStr == "" {
		return nil // skip validation if not set
	}

	port, err := strconv.Atoi(portStr)
	if err != nil {
		return ValidationError{
			Field:   envVar,
			Value:   portStr,
			Message: "must be a valid integer",
		}
	}

	if port < 1 || port > 65535 {
		return ValidationError{
			Field:   envVar,
			Value:   portStr,
			Message: "must be between 1 and 65535",
		}
	}

	return nil
}

// ValidateLogLevel validates log level value
func ValidateLogLevel() error {
	level := os.Getenv("LOG_LEVEL")
	if level == "" {
		return nil
	}

	validLevels := map[string]bool{
		"debug": true,
		"info":  true,
		"warn":  true,
		"error": true,
	}

	if !validLevels[level] {
		return ValidationError{
			Field:   "LOG_LEVEL",
			Value:   level,
			Message: "must be one of: debug, info, warn, error",
		}
	}

	return nil
}

// ValidateEnvironmentType validates environment type
func ValidateEnvironmentType() error {
	env := os.Getenv("ENVIRONMENT")
	if env == "" {
		return nil
	}

	validEnvs := map[string]bool{
		"development": true,
		"staging":     true,
		"production":  true,
		"test":        true,
	}

	if !validEnvs[env] {
		return ValidationError{
			Field:   "ENVIRONMENT",
			Value:   env,
			Message: "must be one of: development, staging, production, test",
		}
	}

	return nil
}

// ValidateAll performs comprehensive configuration validation
func ValidateAll() error {
	var errors ValidationErrors

	// Validate required variables
	if requiredErrs := ValidateRequired(); len(requiredErrs) > 0 {
		errors = append(errors, requiredErrs...)
	}

	// Validate port numbers
	portVars := []string{"DB_PORT", "APP_PORT"}
	for _, portVar := range portVars {
		if err := ValidatePort(portVar); err != nil {
			if validationErr, ok := err.(ValidationError); ok {
				errors = append(errors, validationErr)
			}
		}
	}

	// Validate log level
	if err := ValidateLogLevel(); err != nil {
		if validationErr, ok := err.(ValidationError); ok {
			errors = append(errors, validationErr)
		}
	}

	// Validate environment
	if err := ValidateEnvironmentType(); err != nil {
		if validationErr, ok := err.(ValidationError); ok {
			errors = append(errors, validationErr)
		}
	}

	if len(errors) > 0 {
		return errors
	}

	return nil
}

// LoadAndValidate loads environment variables and validates configuration
func LoadAndValidate() (map[string]string, error) {
	// Create environment map
	env := make(map[string]string)

	// Load required variables
	for _, key := range RequiredEnvironmentVariables {
		value := os.Getenv(key)
		if value != "" {
			env[key] = value
		}
	}

	// Load optional variables with defaults
	for key, defaultValue := range OptionalEnvironmentVariables {
		value := os.Getenv(key)
		if value == "" {
			value = defaultValue
		}
		env[key] = value
	}

	// Validate configuration
	if err := ValidateAll(); err != nil {
		return nil, fmt.Errorf("configuration validation failed: %w", err)
	}

	return env, nil
}

// Load loads and validates configuration from environment variables
func Load() (*Config, error) {
	// 1. Load .env file if it exists
	if err := godotenv.Load(); err != nil {
		log.Printf("Warning: Could not load .env file: %v", err)
	}

	// 2. Pre-load environment variable validation
	if _, err := LoadAndValidate(); err != nil {
		return nil, fmt.Errorf("environment validation failed: %w", err)
	}

	// 3. Load configuration with defaults
	config := &Config{
		Server: ServerConfig{
			Port:         getEnvInt("APP_PORT", 8080),
			Host:         getEnv("APP_HOST", "0.0.0.0"),
			ReadTimeout:  getEnvInt("SERVER_READ_TIMEOUT", 30),
			WriteTimeout: getEnvInt("SERVER_WRITE_TIMEOUT", 30),
			IdleTimeout:  getEnvInt("SERVER_IDLE_TIMEOUT", 120),
			Debug:        getEnvBool("SERVER_DEBUG", false),
		},
		Database: DatabaseConfig{
			Host:     getEnv("DB_HOST", "localhost"),
			Port:     getEnvInt("DB_PORT", 5432),
			User:     getEnv("DB_USER", "postgres"),
			Password: getEnv("DB_PASSWORD", "postgres"),
			Database: getEnv("DB_NAME", "postgres"),
			SSLMode:  getEnv("DB_SSL_MODE", "disable"),
			MaxConns: getEnvInt("DB_MAX_CONNECTIONS", 25),
			MinConns: getEnvInt("DB_MIN_CONNS", 5),
		},
		Logging: LoggingConfig{
			Level:  getEnv("LOG_LEVEL", "info"),
			Format: getEnv("LOG_FORMAT", "json"),
		},
		HealthCheck: HealthCheckConfig{
			Enabled: getEnvBool("HEALTH_CHECK_ENABLED", true),
			Port:    getEnvInt("APP_PORT", 8080),
			Host:    getEnv("APP_HOST", "0.0.0.0"),
		},
		Application: ApplicationConfig{
			Environment:       getEnv("ENVIRONMENT", "development"),
			ShutdownTimeout:   getEnvInt("SHUTDOWN_TIMEOUT", 30),
			RateLimitRequests: getEnvInt("RATE_LIMIT_REQUESTS", 100),
			RateLimitWindow:   getEnv("RATE_LIMIT_WINDOW", "1m"),
			MetricsEnabled:    getEnvBool("METRICS_ENABLED", false),
		},
	}

	// 4. Post-load configuration validation
	if err := Validate(config); err != nil {
		return nil, fmt.Errorf("configuration validation failed: %w", err)
	}

	return config, nil
}

// getEnv gets environment variable with default value
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getEnvInt gets environment variable as integer with default value
func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

// getEnvBool gets environment variable as boolean with default value
func getEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolValue, err := strconv.ParseBool(value); err == nil {
			return boolValue
		}
	}
	return defaultValue
}
