// Package config provides environment variable loading and validation tests
package config

import (
	"os"
	"testing"
)

func TestValidateRequired(t *testing.T) {
	// Save original environment
	originalEnv := make(map[string]string)
	for _, envVar := range RequiredEnvironmentVariables {
		if value := os.Getenv(envVar); value != "" {
			originalEnv[envVar] = value
		}
		os.Unsetenv(envVar)
	}
	defer func() {
		// Restore original environment
		for key, value := range originalEnv {
			os.Setenv(key, value)
		}
		for _, envVar := range RequiredEnvironmentVariables {
			if _, exists := originalEnv[envVar]; !exists {
				os.Unsetenv(envVar)
			}
		}
	}()

	t.Run("missing required variables", func(t *testing.T) {
		errors := ValidateRequired()
		if len(errors) == 0 {
			t.Error("Expected validation errors for missing required variables")
		}

		// Check if all required variables are in errors
		errorMap := make(map[string]bool)
		for _, err := range errors {
			errorMap[err.Field] = true
		}

		for _, reqVar := range RequiredEnvironmentVariables {
			if !errorMap[reqVar] {
				t.Errorf("Expected validation error for missing required variable %s", reqVar)
			}
		}
	})

	t.Run("all required variables present", func(t *testing.T) {
		// Set all required variables
		for _, envVar := range RequiredEnvironmentVariables {
			os.Setenv(envVar, "test-value")
		}

		errors := ValidateRequired()
		if len(errors) != 0 {
			t.Errorf("Expected no validation errors, got %d errors: %v", len(errors), errors)
		}
	})
}

func TestValidatePort(t *testing.T) {
	testCases := []struct {
		name        string
		envValue    string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "valid port",
			envValue:    "8080",
			expectError: false,
		},
		{
			name:        "port 1",
			envValue:    "1",
			expectError: false,
		},
		{
			name:        "port 65535",
			envValue:    "65535",
			expectError: false,
		},
		{
			name:        "port 0 invalid",
			envValue:    "0",
			expectError: true,
			errorMsg:    "must be between 1 and 65535",
		},
		{
			name:        "port 65536 invalid",
			envValue:    "65536",
			expectError: true,
			errorMsg:    "must be between 1 and 65535",
		},
		{
			name:        "negative port",
			envValue:    "-1",
			expectError: true,
			errorMsg:    "must be between 1 and 65535",
		},
		{
			name:        "non-numeric port",
			envValue:    "invalid",
			expectError: true,
			errorMsg:    "must be a valid integer",
		},
		{
			name:        "empty port",
			envValue:    "",
			expectError: false, // Should not error if not set
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Set environment variable
			os.Setenv("TEST_PORT", tc.envValue)

			err := ValidatePort("TEST_PORT")

			if tc.expectError {
				if err == nil {
					t.Errorf("Expected validation error for port %s", tc.envValue)
				} else if tc.errorMsg != "" {
					if !contains(err.Error(), tc.errorMsg) {
						t.Errorf("Expected error message containing '%s', got '%s'", tc.errorMsg, err.Error())
					}
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error for port %s, got: %v", tc.envValue, err)
				}
			}

			// Clean up
			os.Unsetenv("TEST_PORT")
		})
	}
}

func TestValidateLogLevel(t *testing.T) {
	testCases := []struct {
		name        string
		level       string
		expectError bool
	}{
		{"debug level", "debug", false},
		{"info level", "info", false},
		{"warn level", "warn", false},
		{"error level", "error", false},
		{"invalid level", "invalid", true},
		{"empty level", "", false}, // Should not error if not set
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.level != "" {
				os.Setenv("LOG_LEVEL", tc.level)
			} else {
				os.Unsetenv("LOG_LEVEL")
			}

			err := ValidateLogLevel()

			if tc.expectError {
				if err == nil {
					t.Errorf("Expected validation error for log level %s", tc.level)
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error for log level %s, got: %v", tc.level, err)
				}
			}

			// Clean up
			os.Unsetenv("LOG_LEVEL")
		})
	}
}

func TestValidateEnvironmentType(t *testing.T) {
	testCases := []struct {
		name        string
		env         string
		expectError bool
	}{
		{"development", "development", false},
		{"staging", "staging", false},
		{"production", "production", false},
		{"test", "test", false},
		{"invalid env", "invalid", true},
		{"empty env", "", false}, // Should not error if not set
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.env != "" {
				os.Setenv("ENVIRONMENT", tc.env)
			} else {
				os.Unsetenv("ENVIRONMENT")
			}

			err := ValidateEnvironmentType()

			if tc.expectError {
				if err == nil {
					t.Errorf("Expected validation error for environment %s", tc.env)
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error for environment %s, got: %v", tc.env, err)
				}
			}

			// Clean up
			os.Unsetenv("ENVIRONMENT")
		})
	}
}

func TestLoadAndValidate(t *testing.T) {
	// Save and clean environment
	savedEnv := make(map[string]string)
	for _, key := range RequiredEnvironmentVariables {
		if value := os.Getenv(key); value != "" {
			savedEnv[key] = value
			os.Unsetenv(key)
		}
	}
	defer func() {
		for key, value := range savedEnv {
			os.Setenv(key, value)
		}
		for _, key := range RequiredEnvironmentVariables {
			if _, exists := savedEnv[key]; !exists {
				os.Unsetenv(key)
			}
		}
	}()

	t.Run("invalid configuration", func(t *testing.T) {
		// Don't set required variables
		_, err := LoadAndValidate()
		if err == nil {
			t.Error("Expected error for invalid configuration")
		}
	})

	t.Run("valid configuration", func(t *testing.T) {
		// Set required variables
		for _, key := range RequiredEnvironmentVariables {
			os.Setenv(key, "test-value")
		}

		env, err := LoadAndValidate()
		if err != nil {
			t.Errorf("Expected no error for valid configuration, got: %v", err)
		}

		if env == nil {
			t.Error("Expected environment map, got nil")
		}

		// Check if required variables are loaded
		for _, key := range RequiredEnvironmentVariables {
			if value, exists := env[key]; !exists || value != "test-value" {
				t.Errorf("Expected %s='test-value' in environment map", key)
			}
		}

		// Check if optional variables have defaults
		if env["APP_PORT"] != "8080" {
			t.Errorf("Expected default APP_PORT='8080', got '%s'", env["APP_PORT"])
		}
	})
}

// Helper function to check if string contains substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > len(substr) && hasSubstring(s, substr)))
}

func hasSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
