package config

import (
	"os"
	"testing"
)

func TestLoad_Defaults(t *testing.T) {
	// Set required environment variables for test
	os.Setenv("DB_HOST", "localhost")
	os.Setenv("DB_USER", "postgres")
	os.Setenv("DB_PASSWORD", "postgres")
	os.Setenv("DB_NAME", "postgres")

	// Clear optional environment variables to test defaults
	os.Unsetenv("SERVER_PORT")
	os.Unsetenv("APP_HOST")
	os.Unsetenv("LOG_LEVEL")

	config, err := Load()

	// Should succeed with default values
	if err != nil {
		t.Errorf("Expected no error with defaults, got %v", err)
	}

	if config == nil {
		t.Fatal("Expected config to be non-nil")
	}

	// Verify default values are applied
	if config.Database.Host != "localhost" {
		t.Errorf("Expected DB host 'localhost', got '%s'", config.Database.Host)
	}
	if config.Database.User != "postgres" {
		t.Errorf("Expected default DB user 'postgres', got '%s'", config.Database.User)
	}
	if config.Database.Password != "postgres" {
		t.Errorf("Expected default DB password 'postgres', got '%s'", config.Database.Password)
	}
	if config.Database.Database != "postgres" {
		t.Errorf("Expected default DB name 'postgres', got '%s'", config.Database.Database)
	}

	// Cleanup
	os.Unsetenv("DB_HOST")
	os.Unsetenv("DB_USER")
	os.Unsetenv("DB_PASSWORD")
	os.Unsetenv("DB_NAME")
}

func TestLoad_WithEnvironment(t *testing.T) {
	// Set required environment variables
	os.Setenv("DB_HOST", "testhost")
	os.Setenv("DB_USER", "testuser")
	os.Setenv("DB_PASSWORD", "testpass")
	os.Setenv("DB_NAME", "testdb")

	// Also test optional variables
	os.Setenv("APP_PORT", "9000")
	os.Setenv("LOG_LEVEL", "debug")

	config, err := Load()

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if config == nil {
		t.Fatal("Expected config to be non-nil")
	}

	if config.Server.Port != 9000 {
		t.Errorf("Expected server port 9000, got %d", config.Server.Port)
	}

	if config.Database.Host != "testhost" {
		t.Errorf("Expected database host 'testhost', got '%s'", config.Database.Host)
	}

	if config.Database.User != "testuser" {
		t.Errorf("Expected database user 'testuser', got '%s'", config.Database.User)
	}

	if config.Logging.Level != "debug" {
		t.Errorf("Expected log level 'debug', got '%s'", config.Logging.Level)
	}

	// Cleanup
	os.Unsetenv("DB_HOST")
	os.Unsetenv("DB_USER")
	os.Unsetenv("DB_PASSWORD")
	os.Unsetenv("DB_NAME")
	os.Unsetenv("APP_PORT")
	os.Unsetenv("LOG_LEVEL")
}

func TestGetEnv(t *testing.T) {
	// Test with existing environment variable
	os.Setenv("TEST_KEY", "test_value")
	defer os.Unsetenv("TEST_KEY")

	result := getEnv("TEST_KEY", "default")
	if result != "test_value" {
		t.Errorf("Expected 'test_value', got '%s'", result)
	}

	// Test with non-existing environment variable
	result = getEnv("NON_EXISTING_KEY", "default")
	if result != "default" {
		t.Errorf("Expected 'default', got '%s'", result)
	}
}

func TestGetEnvInt(t *testing.T) {
	// Test with valid integer
	os.Setenv("TEST_INT", "42")
	defer os.Unsetenv("TEST_INT")

	result := getEnvInt("TEST_INT", 10)
	if result != 42 {
		t.Errorf("Expected 42, got %d", result)
	}

	// Test with invalid integer
	os.Setenv("TEST_INT_INVALID", "not_a_number")
	defer os.Unsetenv("TEST_INT_INVALID")

	result = getEnvInt("TEST_INT_INVALID", 10)
	if result != 10 {
		t.Errorf("Expected default 10, got %d", result)
	}

	// Test with non-existing key
	result = getEnvInt("NON_EXISTING_INT", 20)
	if result != 20 {
		t.Errorf("Expected default 20, got %d", result)
	}
}

func TestGetEnvBool(t *testing.T) {
	// Test with true
	os.Setenv("TEST_BOOL_TRUE", "true")
	defer os.Unsetenv("TEST_BOOL_TRUE")

	result := getEnvBool("TEST_BOOL_TRUE", false)
	if !result {
		t.Error("Expected true, got false")
	}

	// Test with false
	os.Setenv("TEST_BOOL_FALSE", "false")
	defer os.Unsetenv("TEST_BOOL_FALSE")

	result = getEnvBool("TEST_BOOL_FALSE", true)
	if result {
		t.Error("Expected false, got true")
	}

	// Test with invalid value
	os.Setenv("TEST_BOOL_INVALID", "not_a_bool")
	defer os.Unsetenv("TEST_BOOL_INVALID")

	result = getEnvBool("TEST_BOOL_INVALID", true)
	if !result {
		t.Error("Expected default true, got false")
	}
}
