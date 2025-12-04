package config

import (
	"errors"
	"fmt"
	"strings"
)

// Validate validates the configuration and returns any errors
func Validate(config *Config) error {
	var validationErrors []string

	// Validate database configuration
	if err := validateDatabaseConfig(&config.Database); err != nil {
		validationErrors = append(validationErrors, err.Error())
	}

	// Validate server configuration
	if err := validateServerConfig(&config.Server); err != nil {
		validationErrors = append(validationErrors, err.Error())
	}

	// Validate logging configuration
	if err := validateLoggingConfig(&config.Logging); err != nil {
		validationErrors = append(validationErrors, err.Error())
	}

	// Validate application configuration
	if err := validateApplicationConfig(&config.Application); err != nil {
		validationErrors = append(validationErrors, err.Error())
	}

	if len(validationErrors) > 0 {
		return fmt.Errorf("configuration validation failed: %s", strings.Join(validationErrors, "; "))
	}

	return nil
}

// validateDatabaseConfig validates database configuration
func validateDatabaseConfig(db *DatabaseConfig) error {
	if db.Host == "" {
		return errors.New("database host is required")
	}

	if db.Port <= 0 || db.Port > 65535 {
		return errors.New("database port must be between 1 and 65535")
	}

	if db.User == "" {
		return errors.New("database user is required")
	}

	if db.Password == "" && db.SSLMode != "disable" {
		return errors.New("database password is required when SSL is enabled")
	}

	if db.Database == "" {
		return errors.New("database name is required")
	}

	// Validate SSL mode
	validSSLModes := []string{"disable", "allow", "prefer", "require", "verify-ca", "verify-full"}
	validSSL := false
	for _, mode := range validSSLModes {
		if db.SSLMode == mode {
			validSSL = true
			break
		}
	}
	if !validSSL {
		return fmt.Errorf("invalid SSL mode: %s, must be one of: %s", db.SSLMode, strings.Join(validSSLModes, ", "))
	}

	if db.MaxConns <= 0 {
		return errors.New("database max connections must be positive")
	}

	if db.MinConns < 0 || db.MinConns > db.MaxConns {
		return errors.New("database min connections must be between 0 and max connections")
	}

	return nil
}

// validateServerConfig validates server configuration
func validateServerConfig(server *ServerConfig) error {
	if server.Port <= 0 || server.Port > 65535 {
		return errors.New("server port must be between 1 and 65535")
	}

	if server.ReadTimeout <= 0 {
		return errors.New("server read timeout must be positive")
	}

	if server.WriteTimeout <= 0 {
		return errors.New("server write timeout must be positive")
	}

	if server.IdleTimeout <= 0 {
		return errors.New("server idle timeout must be positive")
	}

	return nil
}

// validateLoggingConfig validates logging configuration
func validateLoggingConfig(logging *LoggingConfig) error {
	validLevels := []string{"debug", "info", "warn", "error"}
	validLevel := false
	for _, level := range validLevels {
		if logging.Level == level {
			validLevel = true
			break
		}
	}
	if !validLevel {
		return fmt.Errorf("invalid log level: %s, must be one of: %s", logging.Level, strings.Join(validLevels, ", "))
	}

	validFormats := []string{"json", "text"}
	validFormat := false
	for _, format := range validFormats {
		if logging.Format == format {
			validFormat = true
			break
		}
	}
	if !validFormat {
		return fmt.Errorf("invalid log format: %s, must be one of: %s", logging.Format, strings.Join(validFormats, ", "))
	}

	return nil
}

// validateApplicationConfig validates application configuration
func validateApplicationConfig(app *ApplicationConfig) error {
	validEnvironments := []string{"development", "staging", "production"}
	validEnv := false
	for _, env := range validEnvironments {
		if app.Environment == env {
			validEnv = true
			break
		}
	}
	if !validEnv {
		return fmt.Errorf("invalid environment: %s, must be one of: %s", app.Environment, strings.Join(validEnvironments, ", "))
	}

	if app.ShutdownTimeout <= 0 {
		return errors.New("shutdown timeout must be positive")
	}

	if app.RateLimitRequests <= 0 {
		return errors.New("rate limit requests must be positive")
	}

	if app.RateLimitWindow == "" {
		return errors.New("rate limit window is required")
	}

	return nil
}

