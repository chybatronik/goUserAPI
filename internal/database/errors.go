package database

import (
	"context"
	"database/sql/driver"
	"log"

	"github.com/chybatronik/goUserAPI/pkg/errors"
	"github.com/jackc/pgx/v5/pgconn"
)

// MapDatabaseErrorSecure maps database errors to secure user-facing errors
// Implements NFR-S3: Error Information Security - no internal details exposed
func MapDatabaseErrorSecure(err error) error {
	if err == nil {
		return nil
	}

	// PostgreSQL specific errors - NEVER expose internal details to users
	if pgErr, ok := err.(*pgconn.PgError); ok {
		// Log detailed error internally for debugging
		log.Printf("Database constraint violation - Code: %s, Table: %s, Column: %s, Constraint: %s, Message: %s",
			pgErr.Code, pgErr.TableName, pgErr.ColumnName, pgErr.ConstraintName, pgErr.Message)

		// Return GENERIC error to users based on error category
		switch pgErr.Code {
		// Constraint violations - return as validation errors
		case "23505", "23503", "23502", "23514": // unique, foreign key, not null, check constraints
			return errors.NewUserValidationError(errors.ErrCodeValidationFailed, "Request failed validation")
		default:
			return errors.NewUserDatabaseError("Database operation failed")
		}
	}

	// Connection errors - return generic service unavailable
	if isConnectionError(err) {
		log.Printf("Database connection error: %v", err)
		return &errors.UserError{
			Code:       errors.ErrCodeConnectionFailed,
			Message:    "Service temporarily unavailable",
			HTTPStatus: 503,
		}
	}

	// All other errors - generic database error message
	log.Printf("Database error: %v", err)
	return errors.NewUserDatabaseError("Database operation failed")
}

// isConnectionError checks if error is a connection-related error
func isConnectionError(err error) bool {
	if err == nil {
		return false
	}

	// Check for common connection error patterns
	if err == driver.ErrBadConn {
		return true
	}

	// Check for context errors (timeouts, cancellations)
	if err == context.DeadlineExceeded || err == context.Canceled {
		return true
	}

	// Check for connection-related PostgreSQL errors
	if pgErr, ok := err.(*pgconn.PgError); ok {
		switch pgErr.Code {
		case "08001", "08003", "08004", "08006", "08007", "08P01": // connection exceptions
			return true
		case "53XXX": // insufficient resources (including connection limits)
			return true
		}
	}

	return false
}

// MapTransactionErrorSecure maps transaction errors to secure responses
func MapTransactionErrorSecure(err error) error {
	if err == nil {
		return nil
	}

	log.Printf("Transaction error: %v", err)

	// Never expose transaction details to users
	return errors.NewUserDatabaseError("Transaction failed")
}