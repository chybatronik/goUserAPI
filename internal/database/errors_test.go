package database

import (
	"context"
	"database/sql/driver"
	"errors"
	"fmt"
	"strings"
	"testing"

	usererrors "github.com/chybatronik/goUserAPI/pkg/errors"
	"github.com/jackc/pgx/v5/pgconn"
)

func TestMapDatabaseErrorSecure(t *testing.T) {
	testCases := []struct {
		name           string
		inputErr       error
		expectedCode   string
		expectedStatus int
		expectLog      bool
	}{
		{
			name:           "nil error",
			inputErr:       nil,
			expectedCode:   "",
			expectedStatus: 0,
			expectLog:      false,
		},
		{
			name: "unique constraint violation",
			inputErr: &pgconn.PgError{
				Code:           "23505",
				ConstraintName: "users_email_key",
				TableName:      "users",
			},
			expectedCode:   usererrors.ErrCodeValidationFailed,
			expectedStatus: 400,
			expectLog:      true,
		},
		{
			name: "foreign key constraint violation",
			inputErr: &pgconn.PgError{
				Code:           "23503",
				ConstraintName: "users_role_id_fkey",
				TableName:      "users",
			},
			expectedCode:   usererrors.ErrCodeValidationFailed,
			expectedStatus: 400,
			expectLog:      true,
		},
		{
			name: "not null constraint violation",
			inputErr: &pgconn.PgError{
				Code:       "23502",
				ColumnName: "first_name",
				TableName:  "users",
			},
			expectedCode:   usererrors.ErrCodeValidationFailed,
			expectedStatus: 400,
			expectLog:      true,
		},
		{
			name: "check constraint violation",
			inputErr: &pgconn.PgError{
				Code:           "23514",
				ConstraintName: "users_age_check",
				TableName:      "users",
			},
			expectedCode:   usererrors.ErrCodeValidationFailed,
			expectedStatus: 400,
			expectLog:      true,
		},
		{
			name:           "connection error",
			inputErr:       driver.ErrBadConn,
			expectedCode:   usererrors.ErrCodeConnectionFailed,
			expectedStatus: 503,
			expectLog:      true,
		},
		{
			name:           "generic database error",
			inputErr:       errors.New("generic database error"),
			expectedCode:   usererrors.ErrCodeDatabaseError,
			expectedStatus: 500,
			expectLog:      true,
		},
		{
			name: "unknown PostgreSQL error",
			inputErr: &pgconn.PgError{
				Code:    "99999",
				Message: "unknown error",
			},
			expectedCode:   usererrors.ErrCodeDatabaseError,
			expectedStatus: 500,
			expectLog:      true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := MapDatabaseErrorSecure(tc.inputErr)

			if tc.inputErr == nil {
				if result != nil {
					t.Errorf("Expected nil result for nil input, got: %v", result)
				}
				return
			}

			if result == nil {
				t.Errorf("Expected UserError for input: %v", tc.inputErr)
				return
			}

			userErr, ok := result.(*usererrors.UserError)
			if !ok {
				t.Errorf("Expected *errors.UserError, got: %T", result)
				return
			}

			if userErr.Code != tc.expectedCode {
				t.Errorf("Expected error code %s, got: %s", tc.expectedCode, userErr.Code)
			}

			if userErr.HTTPStatus != tc.expectedStatus {
				t.Errorf("Expected HTTP status %d, got: %d", tc.expectedStatus, userErr.HTTPStatus)
			}
		})
	}
}

func TestIsConnectionError(t *testing.T) {
	testCases := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "bad connection error",
			err:      driver.ErrBadConn,
			expected: true,
		},
		{
			name:     "connection timeout",
			err:      context.DeadlineExceeded,
			expected: true,
		},
		{
			name:     "connection cancelled",
			err:      context.Canceled,
			expected: true,
		},
		{
			name:     "generic error",
			err:      errors.New("some other error"),
			expected: false,
		},
		{
			name:     "nil error",
			err:      nil,
			expected: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := isConnectionError(tc.err)
			if result != tc.expected {
				t.Errorf("Expected %v for error %v, got: %v", tc.expected, tc.err, result)
			}
		})
	}
}

func TestSecureErrorMapping(t *testing.T) {
	// Test that sensitive database details are never exposed
	sensitiveErrors := []error{
		&pgconn.PgError{
			Code:       "23505",
			Message:    "duplicate key value violates unique constraint \"users_email_key\"",
			Detail:     "Key (email)=(admin@company.com) already exists.",
			TableName:  "users",
			ColumnName: "email",
		},
		&pgconn.PgError{
			Code:       "42703",
			Message:    "column \"secret_column\" does not exist",
			Hint:       "Perhaps you meant to reference the column \"public_column\".",
			TableName:  "users",
			SchemaName: "public",
		},
		&pgconn.PgError{
			Code:       "42501",
			Message:    "permission denied for table users",
			SchemaName: "public",
			TableName:  "users",
		},
	}

	for i, err := range sensitiveErrors {
		t.Run(fmt.Sprintf("sensitive_error_%d", i), func(t *testing.T) {
			result := MapDatabaseErrorSecure(err)

			if result == nil {
				t.Error("Expected error mapping for sensitive database error")
				return
			}

			userErr, ok := result.(*usererrors.UserError)
			if !ok {
				t.Errorf("Expected *errors.UserError, got: %T", result)
				return
			}

			// Ensure no sensitive information is exposed in user message
			sensitiveKeywords := []string{
				"users_email_key",
				"admin@company.com",
				"secret_column",
				"public_column",
				"permission denied",
				"RLS policy",
			}

			for _, keyword := range sensitiveKeywords {
				if strings.Contains(strings.ToLower(userErr.Message), strings.ToLower(keyword)) {
					t.Errorf("Sensitive keyword '%s' found in user message: %s", keyword, userErr.Message)
				}
			}
		})
	}
}
