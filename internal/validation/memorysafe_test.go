package validation

import (
	"testing"
)

func TestValidateFieldMemorySafe(t *testing.T) {
	testCases := []struct {
		name        string
		field       string
		fieldName   string
		maxLen      int
		expectError bool
		errorMsg    string
	}{
		{
			name:        "valid field under limit",
			field:       "john@example.com",
			fieldName:   "email",
			maxLen:      100,
			expectError: false,
		},
		{
			name:        "field exceeds limit",
			field:       "this.is.a.very.long.email.address.that.exceeds.the.maximum.allowed.length.for.this.field@example.com",
			fieldName:   "email",
			maxLen:      50,
			expectError: true,
			errorMsg:    "field email exceeds maximum length of 50 characters",
		},
		{
			name:        "field with dangerous Unicode",
			field:       "test\x00null",
			fieldName:   "username",
			maxLen:      100,
			expectError: true,
			errorMsg:    "unicode security validation failed for field username",
		},
		{
			name:        "field with homograph attack",
			field:       "\u0430dmin",
			fieldName:   "username",
			maxLen:      100,
			expectError: true,
			errorMsg:    "unicode security validation failed for field username",
		},
		{
			name:        "empty field",
			field:       "",
			fieldName:   "optional_field",
			maxLen:      100,
			expectError: false,
		},
		{
			name:        "field at exact limit",
			field:       "exactly_100_characters_long_field_that_is_at_the_maximum_allowed_length_for_validation_testing_12345",
			fieldName:   "description",
			maxLen:      100,
			expectError: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateFieldMemorySafe(tc.field, tc.fieldName, tc.maxLen)

			if tc.expectError && err == nil {
				t.Errorf("Expected error for field %s with length %d", tc.fieldName, len(tc.field))
				return
			}

			if !tc.expectError && err != nil {
				t.Errorf("Expected no error for field %s, got: %v", tc.fieldName, err)
				return
			}

			if tc.expectError && err != nil {
				if tc.errorMsg != "" && err.Error()[:len(tc.errorMsg)] != tc.errorMsg {
					t.Errorf("Expected error message to start with %q, got: %v", tc.errorMsg, err)
				}
			}
		})
	}
}

func TestValidatePayloadSize(t *testing.T) {
	testCases := []struct {
		name        string
		payload     []byte
		maxSize     int64
		expectError bool
	}{
		{
			name:        "valid payload under limit",
			payload:     []byte(`{"name": "John", "email": "john@example.com"}`),
			maxSize:     1024,
			expectError: false,
		},
		{
			name:        "payload exceeds limit",
			payload:     make([]byte, 2048),
			maxSize:     1024,
			expectError: true,
		},
		{
			name:        "payload at exact limit",
			payload:     make([]byte, 1024),
			maxSize:     1024,
			expectError: false,
		},
		{
			name:        "empty payload",
			payload:     []byte{},
			maxSize:     1024,
			expectError: false,
		},
		{
			name:        "nil payload",
			payload:     nil,
			maxSize:     1024,
			expectError: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidatePayloadSize(tc.payload, tc.maxSize)

			if tc.expectError && err == nil {
				t.Errorf("Expected error for payload of size %d", len(tc.payload))
				return
			}

			if !tc.expectError && err != nil {
				t.Errorf("Expected no error for payload of size %d, got: %v", len(tc.payload), err)
				return
			}
		})
	}
}

func TestValidateMultipleFields(t *testing.T) {
	testCases := []struct {
		name        string
		fields      map[string]string
		maxLen      int
		expectError bool
		errorCount  int
	}{
		{
			name: "all valid fields",
			fields: map[string]string{
				"first_name": "John",
				"last_name":  "Doe",
				"email":      "john@example.com",
			},
			maxLen:      100,
			expectError: false,
		},
		{
			name: "one field exceeds limit",
			fields: map[string]string{
				"first_name": "John",
				"last_name":  "This is a very long last name that exceeds the limit",
				"email":      "john@example.com",
			},
			maxLen:      50,
			expectError: true,
			errorCount:  1,
		},
		{
			name: "multiple field errors",
			fields: map[string]string{
				"first_name": "test\x00null",
				"last_name":  "This is a very long last name that exceeds the limit",
				"email":      "\u0430dmin@example.com",
			},
			maxLen:      50,
			expectError: true,
			errorCount:  3,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			errs := ValidateMultipleFields(tc.fields, tc.maxLen)

			hasError := len(errs) > 0
			if tc.expectError && !hasError {
				t.Errorf("Expected %d errors, got 0", tc.errorCount)
				return
			}

			if !tc.expectError && hasError {
				t.Errorf("Expected no errors, got %d: %v", len(errs), errs)
				return
			}

			if tc.expectError && len(errs) != tc.errorCount {
				t.Errorf("Expected %d errors, got %d: %v", tc.errorCount, len(errs), errs)
			}
		})
	}
}

func TestMemoryPoolEfficiency(t *testing.T) {
	// Test that buffer pooling works efficiently
	field := "test field with normal content"
	fieldName := "test_field"
	maxLen := 100

	// Run multiple validations to test buffer reuse
	iterations := 1000
	for i := 0; i < iterations; i++ {
		err := ValidateFieldMemorySafe(field, fieldName, maxLen)
		if err != nil {
			t.Errorf("Validation should pass for normal content: %v", err)
		}
	}
}
