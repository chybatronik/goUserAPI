package database

import (
	"fmt"
	"strings"
	"testing"

	"github.com/chybatronik/goUserAPI/internal/types"
)

func TestGetUsersSQLInjectionProtection(t *testing.T) {
	// Test cases that should be blocked by validation
	injectionAttempts := []types.GetUsersParams{
		{
			SortBy:    "id; DROP TABLE users; --",
			SortOrder: "asc",
			Limit:     10,
			Offset:    0,
		},
		{
			SortBy:    "first_name'; UPDATE users SET first_name = 'HACKED'; --",
			SortOrder: "asc",
			Limit:     10,
			Offset:    0,
		},
		{
			SortBy:    "(SELECT CASE WHEN (1=1) THEN id ELSE id END)",
			SortOrder: "asc",
			Limit:     10,
			Offset:    0,
		},
		{
			SortBy:    "id UNION SELECT password FROM admin_users --",
			SortOrder: "asc",
			Limit:     10,
			Offset:    0,
		},
		{
			SortBy:    "1 OR 1=1",
			SortOrder: "asc",
			Limit:     10,
			Offset:    0,
		},
		{
			SortBy:    "recording_date",
			SortOrder: "asc'; DROP TABLE users; --",
			Limit:     10,
			Offset:    0,
		},
		{
			SortBy:    "recording_date",
			SortOrder: "1; DELETE FROM users WHERE 1=1; --",
			Limit:     10,
			Offset:    0,
		},
		{
			SortBy:    "age",
			SortOrder: "CASE WHEN (SELECT COUNT(*) FROM users) > 0 THEN id ELSE id END",
			Limit:     10,
			Offset:    0,
		},
	}

	for i, params := range injectionAttempts {
		t.Run(fmt.Sprintf("injection_attempt_%d", i), func(t *testing.T) {
			// Test parameter validation - should reject all injection attempts
			err := validateGetUsersParams(params)
			if err == nil {
				t.Errorf("Expected validation to reject injection attempt: %+v", params)
			}
		})
	}
}

func TestGetUsersValidParameters(t *testing.T) {
	// Test valid parameters that should pass validation
	validParams := []types.GetUsersParams{
		{
			SortBy:    "recording_date",
			SortOrder: "asc",
			Limit:     10,
			Offset:    0,
		},
		{
			SortBy:    "recording_date",
			SortOrder: "desc",
			Limit:     50,
			Offset:    100,
		},
		{
			SortBy:    "age",
			SortOrder: "asc",
			Limit:     25,
			Offset:    75,
		},
		{
			SortBy:    "first_name",
			SortOrder: "desc",
			Limit:     1,
			Offset:    0,
		},
		{
			SortBy:    "last_name",
			SortOrder: "asc",
			Limit:     100,
			Offset:    200,
		},
	}

	for i, params := range validParams {
		t.Run(fmt.Sprintf("valid_params_%d", i), func(t *testing.T) {
			// Test parameter validation - should accept all valid parameters
			err := validateGetUsersParams(params)
			if err != nil {
				t.Errorf("Expected validation to accept valid parameters: %+v, got error: %v", params, err)
			}
		})
	}
}

func TestBuildOrderClauseSecurity(t *testing.T) {
	// Test that buildOrderClause produces safe SQL even with malicious inputs
	testCases := []struct {
		name      string
		sortBy    string
		sortOrder string
		expected  string
	}{
		// Valid inputs
		{
			name:      "valid recording_date asc",
			sortBy:    "recording_date",
			sortOrder: "asc",
			expected:  "recording_date ASC",
		},
		{
			name:      "valid age desc",
			sortBy:    "age",
			sortOrder: "desc",
			expected:  "age DESC",
		},
		// Malicious inputs that should be sanitized
		{
			name:      "malicious sort field with SQL injection",
			sortBy:    "id; DROP TABLE users; --",
			sortOrder: "asc",
			expected:  "recording_date ASC", // Falls back to default
		},
		{
			name:      "malicious sort order with SQL injection",
			sortBy:    "recording_date",
			sortOrder: "desc; DELETE FROM users; --",
			expected:  "recording_date ASC", // Falls back to default
		},
		{
			name:      "both fields malicious",
			sortBy:    "1; UPDATE users SET first_name = 'HACKED'; --",
			sortOrder: "CASE WHEN (SELECT COUNT(*) FROM users) > 0 THEN id ELSE id END",
			expected:  "recording_date ASC", // Falls back to defaults
		},
		{
			name:      "empty inputs",
			sortBy:    "",
			sortOrder: "",
			expected:  "recording_date ASC", // Falls back to defaults
		},
		{
			name:      "invalid sort field",
			sortBy:    "nonexistent_column",
			sortOrder: "asc",
			expected:  "recording_date ASC", // Falls back to default
		},
		{
			name:      "invalid sort order",
			sortBy:    "recording_date",
			sortOrder: "invalid_order",
			expected:  "recording_date ASC", // Falls back to default
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := buildOrderClause(tc.sortBy, tc.sortOrder)
			if result != tc.expected {
				t.Errorf("Expected '%s', got '%s'", tc.expected, result)
			}

			// Additional security check: ensure no dangerous SQL keywords
			dangerousKeywords := []string{"DROP", "DELETE", "UPDATE", "INSERT", "ALTER", "EXEC", "UNION", "--", ";", "'"}
			resultUpper := strings.ToUpper(result)
			for _, keyword := range dangerousKeywords {
				if strings.Contains(resultUpper, keyword) {
					t.Errorf("Dangerous keyword '%s' found in result: %s", keyword, result)
				}
			}
		})
	}
}

func TestGetUsersParameterBoundaries(t *testing.T) {
	// Test boundary conditions for limit and offset
	boundaryTests := []struct {
		name        string
		params      types.GetUsersParams
		expectError bool
	}{
		{
			name: "minimum valid limit",
			params: types.GetUsersParams{
				SortBy:    "recording_date",
				SortOrder: "asc",
				Limit:     1,
				Offset:    0,
			},
			expectError: false,
		},
		{
			name: "maximum valid limit",
			params: types.GetUsersParams{
				SortBy:    "recording_date",
				SortOrder: "asc",
				Limit:     100,
				Offset:    0,
			},
			expectError: false,
		},
		{
			name: "invalid limit - zero",
			params: types.GetUsersParams{
				SortBy:    "recording_date",
				SortOrder: "asc",
				Limit:     0,
				Offset:    0,
			},
			expectError: true,
		},
		{
			name: "invalid limit - negative",
			params: types.GetUsersParams{
				SortBy:    "recording_date",
				SortOrder: "asc",
				Limit:     -1,
				Offset:    0,
			},
			expectError: true,
		},
		{
			name: "invalid limit - too large",
			params: types.GetUsersParams{
				SortBy:    "recording_date",
				SortOrder: "asc",
				Limit:     101,
				Offset:    0,
			},
			expectError: true,
		},
		{
			name: "valid offset - zero",
			params: types.GetUsersParams{
				SortBy:    "recording_date",
				SortOrder: "asc",
				Limit:     10,
				Offset:    0,
			},
			expectError: false,
		},
		{
			name: "valid offset - positive",
			params: types.GetUsersParams{
				SortBy:    "recording_date",
				SortOrder: "asc",
				Limit:     10,
				Offset:    1000,
			},
			expectError: false,
		},
		{
			name: "invalid offset - negative",
			params: types.GetUsersParams{
				SortBy:    "recording_date",
				SortOrder: "asc",
				Limit:     10,
				Offset:    -1,
			},
			expectError: true,
		},
	}

	for _, tt := range boundaryTests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateGetUsersParams(tt.params)
			if tt.expectError && err == nil {
				t.Errorf("Expected validation error for params: %+v", tt.params)
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no validation error for params: %+v, got: %v", tt.params, err)
			}
		})
	}
}

// TestGetUsersWithMockDB would test with a real database connection
// but for now we focus on validation and SQL injection protection
func TestGetUsersValidationIntegration(t *testing.T) {
	// Test that GetUsers properly validates parameters before building SQL
	invalidParams := types.GetUsersParams{
		SortBy:    "invalid_field",
		SortOrder: "asc",
		Limit:     10,
		Offset:    0,
	}

	// This would require a real database connection to fully test
	// For now, we test the validation function directly
	err := validateGetUsersParams(invalidParams)
	if err == nil {
		t.Error("Expected validation error for invalid sort field")
	}

	// Test with valid params
	validParams := types.GetUsersParams{
		SortBy:    "recording_date",
		SortOrder: "desc",
		Limit:     10,
		Offset:    0,
	}

	err = validateGetUsersParams(validParams)
	if err != nil {
		t.Errorf("Expected no validation error for valid params, got: %v", err)
	}
}
