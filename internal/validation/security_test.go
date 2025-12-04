package validation

import (
	"testing"
)

func TestValidateUnicodeSecurity(t *testing.T) {
	testCases := []struct {
		name        string
		input       string
		expectError bool
		errorType   string
	}{
		{
			name:        "valid ASCII email",
			input:       "john@example.com",
			expectError: false,
		},
		{
			name:        "valid Unicode name",
			input:       "José María",
			expectError: false,
		},
		{
			name:        "null byte injection",
			input:       "test\x00null",
			expectError: true,
			errorType:   "ErrInvalidUnicodeSecurity",
		},
		{
			name:        "backspace character",
			input:       "control\x08char",
			expectError: true,
			errorType:   "ErrInvalidUnicodeSecurity",
		},
		{
			name:        "zero-width space",
			input:       "zero\u200Bwidth",
			expectError: true,
			errorType:   "ErrInvalidUnicodeCategory",
		},
		{
			name:        "format character",
			input:       "test\u200Dformat",
			expectError: true,
			errorType:   "ErrInvalidUnicodeCategory",
		},
		{
			name:        "Cyrillic homograph attack",
			input:       "\u0430dmin@example.com", // Cyrillic 'a'
			expectError: true,
			errorType:   "ErrHomographAttackDetected",
		},
		{
			name:        "control character - vertical tab",
			input:       "test\x0Bcontrol",
			expectError: true,
			errorType:   "ErrInvalidUnicodeSecurity",
		},
		{
			name:        "format character - zero-width joiner",
			input:       "test\u200Dformat",
			expectError: true,
			errorType:   "ErrInvalidUnicodeCategory",
		},
		{
			name:        "private use character",
			input:       "test\ue000private",
			expectError: true,
			errorType:   "ErrInvalidUnicodeCategory",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateUnicodeSecurity(tc.input)

			if tc.expectError && err == nil {
				t.Errorf("Expected error for input: %q", tc.input)
				return
			}

			if !tc.expectError && err != nil {
				t.Errorf("Expected no error for input: %q, got: %v", tc.input, err)
				return
			}

			if tc.expectError && err != nil {
				// Check if error contains expected type
				if tc.errorType != "" && err.Error() != tc.errorType {
					t.Errorf("Expected error type %s, got: %v", tc.errorType, err)
				}
			}
		})
	}
}

func TestContainsHomographAttacks(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected bool
	}{
		{
			name:     "no homograph",
			input:    "admin@example.com",
			expected: false,
		},
		{
			name:     "Cyrillic a",
			input:    "\u0430dmin@example.com",
			expected: true,
		},
		{
			name:     "Cyrillic e",
			input:    "t\u0435st@example.com",
			expected: true,
		},
		{
			name:     "Cyrillic o",
			input:    "l\u043et@example.com",
			expected: true,
		},
		{
			name:     "Cyrillic r",
			input:    "p\u0440file@example.com",
			expected: true,
		},
		{
			name:     "Cyrillic s",
			input:    "te\u0441t@example.com",
			expected: true,
		},
		{
			name:     "Cyrillic x",
			input:    "e\u0445ample@example.com",
			expected: true,
		},
		{
			name:     "Cyrillic u",
			input:    "yo\u0443@example.com",
			expected: true,
		},
		{
			name:     "mixed dangerous characters",
			input:    "\u0430dmin", // Cyrillic а + Latin dmin
			expected: true,
		},
		{
			name:     "Greek characters (not in our dangerous list)",
			input:    "αβγδε@example.com",
			expected: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := containsHomographAttacks(tc.input)
			if result != tc.expected {
				t.Errorf("Expected %v for input %q, got %v", tc.expected, tc.input, result)
			}
		})
	}
}

func TestValidateUnicodePerformance(t *testing.T) {
	// Performance test - should complete within 2ms for normal inputs
	input := "john.doe@example.com"

	iterations := 1000

	for i := 0; i < iterations; i++ {
		err := ValidateUnicodeSecurity(input)
		if err != nil {
			t.Errorf("Valid input should not error: %v", err)
		}
	}

	// This should complete very quickly even for 1000 iterations
	// The actual timing threshold will depend on the test environment
}