// Package errors provides secure error handling utilities
package errors

import (
	"github.com/chybatronik/goUserAPI/pkg/errors"
	pgerr "github.com/jackc/pgx/v5/pgconn"
)

// MapDatabaseErrorSecure maps database errors to secure user-facing errors
// Implements NFR-S3: Error Information Security - no internal details exposed
func MapDatabaseErrorSecure(err error) error {
	if err == nil {
		return nil
	}

	// PostgreSQL specific errors - NEVER expose internal details to users
	if pgErr, ok := err.(*pgerr.PgError); ok {
		// Log detailed error internally (would be done by caller)
		// logger.Warn("Database constraint violation",
		//     "constraint", pgErr.Constraint,
		//     "table", pgErr.Table,
		//     "column", pgErr.Column,
		// )

		// Return GENERIC error to users
		switch pgErr.Code {
		case "23505", "23503", "23502", "23514":
			return errors.NewUserValidationError("VALIDATION_ERROR", "Request failed validation")
		default:
			return errors.NewUserDatabaseError("Database operation failed")
		}
	}

	// Connection errors - return SERVICE_UNAVAILABLE error with proper status
	if isConnectionError(err) {
		// logger.Error("Database connection error", "error", err.Error())
		// Create error with 503 status code for connection issues
		connErr := &errors.UserError{
			Code:       "SERVICE_UNAVAILABLE",
			Message:    "Service temporarily unavailable",
			HTTPStatus: 503, // Service Unavailable
		}
		return connErr
	}

	// Default: always return generic message
	return errors.NewUserDatabaseError("Database operation failed")
}

// isConnectionError checks if error is a connection-related error
func isConnectionError(err error) bool {
	if err == nil {
		return false
	}

	errStr := err.Error()
	return contains(errStr, []string{
		"connection",
		"connect",
		"timeout",
		"network",
		"unreachable",
		"refused",
		"failed to connect",
	})
}

func contains(str string, substrings []string) bool {
	for _, substr := range substrings {
		if len(str) >= len(substr) {
			for i := 0; i <= len(str)-len(substr); i++ {
				if str[i:i+len(substr)] == substr {
					return true
				}
			}
		}
	}
	return false
}
