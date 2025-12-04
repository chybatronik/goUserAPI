// Package errors provides user-specific error definitions for goUserAPI
// Following unified error response format from Story 1.4: {"error": "message", "code": "ERROR_CODE"}
package errors

import (
	"fmt"
	"net/http"
)

// User-specific error codes (AC: #4)
const (
	// Validation errors (400 Bad Request)
	ErrCodeValidationFailed    = "USER_VALIDATION_FAILED"
	ErrCodeFirstNameEmpty      = "USER_FIRST_NAME_EMPTY"
	ErrCodeFirstNameTooLong    = "USER_FIRST_NAME_TOO_LONG"
	ErrCodeLastNameEmpty       = "USER_LAST_NAME_EMPTY"
	ErrCodeLastNameTooLong     = "USER_LAST_NAME_TOO_LONG"
	ErrCodeAgeInvalid          = "USER_AGE_INVALID"
	ErrCodeAgeTooYoung         = "USER_AGE_TOO_YOUNG"
	ErrCodeAgeTooOld           = "USER_AGE_TOO_OLD"
	ErrCodeUUIDInvalid         = "USER_UUID_INVALID"

	// Database constraint violations (409 Conflict)
	ErrCodeUserAlreadyExists   = "USER_ALREADY_EXISTS"
	ErrCodeUserNotFound        = "USER_NOT_FOUND"
	ErrCodeDuplicateEmail      = "USER_DUPLICATE_EMAIL" // Future use for email field

	// Database errors (500 Internal Server Error)
	ErrCodeDatabaseError       = "USER_DATABASE_ERROR"
	ErrCodeConnectionFailed    = "USER_CONNECTION_FAILED"
	ErrCodeTransactionFailed   = "USER_TRANSACTION_FAILED"
)

// UserError represents a user-specific error with HTTP status mapping
type UserError struct {
	Code       string `json:"code"`
	Message    string `json:"message"`
	HTTPStatus int    `json:"-"`
}

// Error implements the error interface
func (e *UserError) Error() string {
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// NewUserValidationError creates validation errors (400 Bad Request)
func NewUserValidationError(errCode, message string) *UserError {
	return &UserError{
		Code:       errCode,
		Message:    message,
		HTTPStatus: http.StatusBadRequest,
	}
}

// NewUserNotFoundError creates not found errors (404 Not Found)
func NewUserNotFoundError(userID string) *UserError {
	return &UserError{
		Code:       ErrCodeUserNotFound,
		Message:    fmt.Sprintf("User with ID '%s' not found", userID),
		HTTPStatus: http.StatusNotFound,
	}
}

// NewUserConflictError creates conflict errors (409 Conflict)
func NewUserConflictError(errCode, message string) *UserError {
	return &UserError{
		Code:       errCode,
		Message:    message,
		HTTPStatus: http.StatusConflict,
	}
}

// NewUserDatabaseError creates database errors (500 Internal Server Error)
func NewUserDatabaseError(message string) *UserError {
	return &UserError{
		Code:       ErrCodeDatabaseError,
		Message:    message,
		HTTPStatus: http.StatusInternalServerError,
	}
}

// GetHTTPStatus returns the HTTP status code for the error
func (e *UserError) GetHTTPStatus() int {
	return e.HTTPStatus
}

// IsUserError checks if error is a UserError
func IsUserError(err error) bool {
	_, ok := err.(*UserError)
	return ok
}

// GetUserError extracts UserError from error
func GetUserError(err error) (*UserError, bool) {
	userErr, ok := err.(*UserError)
	return userErr, ok
}

// MapValidationError maps validation errors to UserError (AC: #4)
func MapValidationError(fieldName, details string) *UserError {
	switch fieldName {
	case "first_name":
		if details == "empty" {
			return NewUserValidationError(ErrCodeFirstNameEmpty, "First name cannot be empty")
		}
		return NewUserValidationError(ErrCodeFirstNameTooLong, "First name cannot exceed 100 characters")
	case "last_name":
		if details == "empty" {
			return NewUserValidationError(ErrCodeLastNameEmpty, "Last name cannot be empty")
		}
		return NewUserValidationError(ErrCodeLastNameTooLong, "Last name cannot exceed 100 characters")
	case "age":
		if details == "too_young" {
			return NewUserValidationError(ErrCodeAgeTooYoung, "Age must be at least 1 year")
		}
		return NewUserValidationError(ErrCodeAgeTooOld, "Age cannot exceed 120 years")
	default:
		return NewUserValidationError(ErrCodeValidationFailed, fmt.Sprintf("Validation failed for field: %s", fieldName))
	}
}