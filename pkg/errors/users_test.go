package errors

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// GREEN TEST: Test user error creation and HTTP status mapping (AC: #4)
func TestUserErrorCreation(t *testing.T) {
	tests := []struct {
		name       string
		errCode    string
		message    string
		httpStatus int
	}{
		{
			name:       "validation error",
			errCode:    ErrCodeFirstNameEmpty,
			message:    "First name cannot be empty",
			httpStatus: http.StatusBadRequest,
		},
		{
			name:       "not found error",
			errCode:    ErrCodeUserNotFound,
			message:    "User with ID '123' not found",
			httpStatus: http.StatusNotFound,
		},
		{
			name:       "database error",
			errCode:    ErrCodeDatabaseError,
			message:    "Database connection failed",
			httpStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			userErr := &UserError{
				Code:       tt.errCode,
				Message:    tt.message,
				HTTPStatus: tt.httpStatus,
			}

			assert.Equal(t, tt.errCode, userErr.Code)
			assert.Equal(t, tt.message, userErr.Message)
			assert.Equal(t, tt.httpStatus, userErr.HTTPStatus)
			assert.Equal(t, tt.httpStatus, userErr.GetHTTPStatus())
		})
	}
}

// GREEN TEST: Test validation error mapping (AC: #4)
func TestMapValidationError(t *testing.T) {
	tests := []struct {
		name         string
		fieldName    string
		details      string
		expectedCode string
		expectedMsg  string
	}{
		{
			name:         "empty first name",
			fieldName:    "first_name",
			details:      "empty",
			expectedCode: ErrCodeFirstNameEmpty,
			expectedMsg:  "First name cannot be empty",
		},
		{
			name:         "first name too long",
			fieldName:    "first_name",
			details:      "too_long",
			expectedCode: ErrCodeFirstNameTooLong,
			expectedMsg:  "First name cannot exceed 100 characters",
		},
		{
			name:         "empty last name",
			fieldName:    "last_name",
			details:      "empty",
			expectedCode: ErrCodeLastNameEmpty,
			expectedMsg:  "Last name cannot be empty",
		},
		{
			name:         "last name too long",
			fieldName:    "last_name",
			details:      "too_long",
			expectedCode: ErrCodeLastNameTooLong,
			expectedMsg:  "Last name cannot exceed 100 characters",
		},
		{
			name:         "age too young",
			fieldName:    "age",
			details:      "too_young",
			expectedCode: ErrCodeAgeTooYoung,
			expectedMsg:  "Age must be at least 1 year",
		},
		{
			name:         "age too old",
			fieldName:    "age",
			details:      "too_old",
			expectedCode: ErrCodeAgeTooOld,
			expectedMsg:  "Age cannot exceed 120 years",
		},
		{
			name:         "unknown field",
			fieldName:    "unknown",
			details:      "some_error",
			expectedCode: ErrCodeValidationFailed,
			expectedMsg:  "Validation failed for field: unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			userErr := MapValidationError(tt.fieldName, tt.details)

			assert.Equal(t, tt.expectedCode, userErr.Code)
			assert.Equal(t, tt.expectedMsg, userErr.Message)
			assert.Equal(t, http.StatusBadRequest, userErr.HTTPStatus)
		})
	}
}

// GREEN TEST: Test convenience constructors (AC: #4)
func TestUserErrorConstructors(t *testing.T) {
	// Test validation error constructor
	validationErr := NewUserValidationError(ErrCodeAgeInvalid, "Age validation failed")
	assert.Equal(t, ErrCodeAgeInvalid, validationErr.Code)
	assert.Equal(t, "Age validation failed", validationErr.Message)
	assert.Equal(t, http.StatusBadRequest, validationErr.HTTPStatus)

	// Test not found error constructor
	notFoundErr := NewUserNotFoundError("123e4567-e89b-12d3-a456-426614174000")
	assert.Equal(t, ErrCodeUserNotFound, notFoundErr.Code)
	assert.Contains(t, notFoundErr.Message, "123e4567-e89b-12d3-a456-426614174000")
	assert.Equal(t, http.StatusNotFound, notFoundErr.HTTPStatus)

	// Test conflict error constructor
	conflictErr := NewUserConflictError(ErrCodeUserAlreadyExists, "User already exists")
	assert.Equal(t, ErrCodeUserAlreadyExists, conflictErr.Code)
	assert.Equal(t, "User already exists", conflictErr.Message)
	assert.Equal(t, http.StatusConflict, conflictErr.HTTPStatus)

	// Test database error constructor
	dbErr := NewUserDatabaseError("Connection pool exhausted")
	assert.Equal(t, ErrCodeDatabaseError, dbErr.Code)
	assert.Equal(t, "Connection pool exhausted", dbErr.Message)
	assert.Equal(t, http.StatusInternalServerError, dbErr.HTTPStatus)
}

// GREEN TEST: Test error interface implementation
func TestUserErrorInterface(t *testing.T) {
	userErr := NewUserValidationError(ErrCodeFirstNameEmpty, "First name required")

	// Test Error() method
	errorMsg := userErr.Error()
	expected := "USER_FIRST_NAME_EMPTY: First name required"
	assert.Equal(t, expected, errorMsg)
}

// GREEN TEST: Test error type checking utilities
func TestErrorUtilities(t *testing.T) {
	userErr := NewUserNotFoundError("test-id")
	genericErr := fmt.Errorf("generic error")

	// Test IsUserError
	assert.True(t, IsUserError(userErr))
	assert.False(t, IsUserError(genericErr))

	// Test GetUserError
	extracted, ok := GetUserError(userErr)
	require.True(t, ok)
	assert.Equal(t, userErr, extracted)

	extracted, ok = GetUserError(genericErr)
	require.False(t, ok)
	assert.Nil(t, extracted)
}

// GREEN TEST: Test all error constants are defined (AC: #4)
func TestErrorConstants(t *testing.T) {
	// Validation error constants
	assert.Equal(t, "USER_VALIDATION_FAILED", ErrCodeValidationFailed)
	assert.Equal(t, "USER_FIRST_NAME_EMPTY", ErrCodeFirstNameEmpty)
	assert.Equal(t, "USER_FIRST_NAME_TOO_LONG", ErrCodeFirstNameTooLong)
	assert.Equal(t, "USER_LAST_NAME_EMPTY", ErrCodeLastNameEmpty)
	assert.Equal(t, "USER_LAST_NAME_TOO_LONG", ErrCodeLastNameTooLong)
	assert.Equal(t, "USER_AGE_INVALID", ErrCodeAgeInvalid)
	assert.Equal(t, "USER_AGE_TOO_YOUNG", ErrCodeAgeTooYoung)
	assert.Equal(t, "USER_AGE_TOO_OLD", ErrCodeAgeTooOld)
	assert.Equal(t, "USER_UUID_INVALID", ErrCodeUUIDInvalid)

	// Database constraint violation constants
	assert.Equal(t, "USER_ALREADY_EXISTS", ErrCodeUserAlreadyExists)
	assert.Equal(t, "USER_NOT_FOUND", ErrCodeUserNotFound)
	assert.Equal(t, "USER_DUPLICATE_EMAIL", ErrCodeDuplicateEmail)

	// Database error constants
	assert.Equal(t, "USER_DATABASE_ERROR", ErrCodeDatabaseError)
	assert.Equal(t, "USER_CONNECTION_FAILED", ErrCodeConnectionFailed)
	assert.Equal(t, "USER_TRANSACTION_FAILED", ErrCodeTransactionFailed)
}
