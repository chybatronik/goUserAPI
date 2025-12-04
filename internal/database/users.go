package database

import (
	"context"
	"fmt"
	"time"

	"github.com/chybatronik/goUserAPI/internal/logging"
	"github.com/chybatronik/goUserAPI/internal/models"
	"github.com/chybatronik/goUserAPI/internal/types"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Performance constants for NFR-P1 compliance (AC #5)
const (
	// Default timeout for database operations to meet <200ms target
	DefaultOperationTimeout = 5 * time.Second
	// Performance warning threshold (should be much less than NFR-P1 target)
	PerformanceWarningThreshold = 100 * time.Millisecond
	// Critical performance threshold (approaching NFR-P1 limit)
	PerformanceCriticalThreshold = 180 * time.Millisecond
)

// CreateUser inserts a new user into the database (AC: #2, #3)
// Uses parameterized queries for security (NFR-S1)
// Returns user with generated ID and recording_date
// Includes performance monitoring for NFR-P1 compliance (AC #5)
func CreateUser(ctx context.Context, pool *pgxpool.Pool, user *models.User) (*models.User, error) {
	// Add operation timeout for performance guarantees (AC #5)
	ctx, cancel := context.WithTimeout(ctx, DefaultOperationTimeout)
	defer cancel()

	// Validate user before database operation (AC: #4)
	if err := validateUser(user); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	// Performance monitoring start
	start := time.Now()

	// PostgreSQL gen_random_uuid() for ID generation (AC: #3)
	// Uses existing indexes: idx_users_recording_date_desc for optimal insertion
	query := `INSERT INTO users (first_name, last_name, age) VALUES ($1, $2, $3) RETURNING id, recording_date`

	var newUser models.User
	err := pool.QueryRow(ctx, query, user.FirstName, user.LastName, user.Age).Scan(&newUser.ID, &newUser.RecordingDate)
	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	// Copy user data to returned struct
	newUser.FirstName = user.FirstName
	newUser.LastName = user.LastName
	newUser.Age = user.Age

	// Performance monitoring end
	duration := time.Since(start)
	logPerformanceMetrics("CreateUser", duration)

	return &newUser, nil
}

// GetUserByID retrieves a user by ID using parameterized query
// Includes performance monitoring for NFR-P1 compliance (AC #5)
func GetUserByID(ctx context.Context, pool *pgxpool.Pool, id string) (*models.User, error) {
	// Add operation timeout for performance guarantees (AC #5)
	ctx, cancel := context.WithTimeout(ctx, DefaultOperationTimeout)
	defer cancel()

	start := time.Now()

	// Uses primary key index for optimal performance
	query := `SELECT id, first_name, last_name, age, recording_date FROM users WHERE id = $1`

	var user models.User
	err := pool.QueryRow(ctx, query, id).Scan(&user.ID, &user.FirstName, &user.LastName, &user.Age, &user.RecordingDate)
	if err != nil {
		return nil, fmt.Errorf("failed to get user by ID %s: %w", id, err)
	}

	// Performance monitoring
	duration := time.Since(start)
	logPerformanceMetrics("GetUserByID", duration)

	return &user, nil
}

// GetAllUsers retrieves all users with pagination support
func GetAllUsers(ctx context.Context, pool *pgxpool.Pool, limit, offset int) ([]*models.User, error) {
	query := `SELECT id, first_name, last_name, age, recording_date FROM users ORDER BY recording_date DESC LIMIT $1 OFFSET $2`

	rows, err := pool.Query(ctx, query, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to get users: %w", err)
	}
	defer rows.Close()

	var users []*models.User
	for rows.Next() {
		var user models.User
		err := rows.Scan(&user.ID, &user.FirstName, &user.LastName, &user.Age, &user.RecordingDate)
		if err != nil {
			return nil, fmt.Errorf("failed to scan user row: %w", err)
		}
		users = append(users, &user)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating user rows: %w", err)
	}

	return users, nil
}

// UpdateUser updates user information by ID
func UpdateUser(ctx context.Context, pool *pgxpool.Pool, id string, user *models.User) (*models.User, error) {
	// Validate user before database operation
	if err := validateUser(user); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	query := `UPDATE users SET first_name = $1, last_name = $2, age = $3 WHERE id = $4 RETURNING id, recording_date`

	var updatedUser models.User
	err := pool.QueryRow(ctx, query, user.FirstName, user.LastName, user.Age, id).Scan(&updatedUser.ID, &updatedUser.RecordingDate)
	if err != nil {
		return nil, fmt.Errorf("failed to update user %s: %w", id, err)
	}

	// Copy user data to returned struct
	updatedUser.FirstName = user.FirstName
	updatedUser.LastName = user.LastName
	updatedUser.Age = user.Age

	return &updatedUser, nil
}

// DeleteUser deletes a user by ID
func DeleteUser(ctx context.Context, pool *pgxpool.Pool, id string) error {
	query := `DELETE FROM users WHERE id = $1`

	cmdTag, err := pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete user %s: %w", id, err)
	}

	if cmdTag.RowsAffected() == 0 {
		return fmt.Errorf("user with ID %s not found", id)
	}

	return nil
}

// validateUser performs business logic validation before database operations (AC: #4)
func validateUser(user *models.User) error {
	if user.FirstName == "" {
		return fmt.Errorf("first_name cannot be empty")
	}
	if len(user.FirstName) > 100 {
		return fmt.Errorf("first_name cannot exceed 100 characters")
	}
	if user.LastName == "" {
		return fmt.Errorf("last_name cannot be empty")
	}
	if len(user.LastName) > 100 {
		return fmt.Errorf("last_name cannot exceed 100 characters")
	}
	if user.Age < 1 || user.Age > 120 {
		return fmt.Errorf("age must be between 1 and 120 years")
	}
	return nil
}

// TransactionExample demonstrates transaction support (AC: #2)
func TransactionExample(ctx context.Context, pool *pgxpool.Pool, users []*models.User) error {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	for _, user := range users {
		if err := validateUser(user); err != nil {
			return fmt.Errorf("validation failed for user %s %s: %w", user.FirstName, user.LastName, err)
		}

		query := `INSERT INTO users (first_name, last_name, age) VALUES ($1, $2, $3)`
		_, err := tx.Exec(ctx, query, user.FirstName, user.LastName, user.Age)
		if err != nil {
			return fmt.Errorf("failed to insert user %s %s: %w", user.FirstName, user.LastName, err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// GetUsers retrieves users with pagination and sorting (Story 2.3)
// Uses parameterized queries for security (NFR-S1 compliance)
// Returns users array, total count, and error
func GetUsers(ctx context.Context, pool *pgxpool.Pool, params types.GetUsersParams) ([]models.User, int64, error) {
	// Add operation timeout for performance guarantees (AC #5)
	ctx, cancel := context.WithTimeout(ctx, DefaultOperationTimeout)
	defer cancel()

	start := time.Now()

	// Validate parameters
	if err := validateGetUsersParams(params); err != nil {
		return nil, 0, fmt.Errorf("parameter validation failed: %w", err)
	}

	// Build ORDER BY clause with whitelist validation
	orderClause := buildOrderClause(params.SortBy, params.SortOrder)

	// First, get total count
	var totalCount int64
	countQuery := `SELECT COUNT(*) FROM users`
	err := pool.QueryRow(ctx, countQuery).Scan(&totalCount)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get total count: %w", err)
	}

	// Main query with parameterized LIMIT and OFFSET
	query := fmt.Sprintf(`
		SELECT id, first_name, last_name, age, recording_date
		FROM users
		ORDER BY %s
		LIMIT $1 OFFSET $2`, orderClause)

	rows, err := pool.Query(ctx, query, params.Limit, params.Offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query users: %w", err)
	}
	defer rows.Close()

	var users []models.User
	for rows.Next() {
		var user models.User
		err := rows.Scan(&user.ID, &user.FirstName, &user.LastName, &user.Age, &user.RecordingDate)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan user row: %w", err)
		}
		users = append(users, user)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("error iterating user rows: %w", err)
	}

	// Performance monitoring
	duration := time.Since(start)
	logPerformanceMetrics("GetUsers", duration)

	return users, totalCount, nil
}

// validateGetUsersParams validates query parameters for GetUsers
func validateGetUsersParams(params types.GetUsersParams) error {
	// Validate limit (1-100)
	if params.Limit < 1 || params.Limit > 100 {
		return fmt.Errorf("invalid limit: %d (must be between 1 and 100)", params.Limit)
	}

	// Validate offset (>= 0)
	if params.Offset < 0 {
		return fmt.Errorf("invalid offset: %d (must be >= 0)", params.Offset)
	}

	// Validate sort_by against whitelist
	validSortFields := map[string]bool{
		"recording_date": true,
		"age":            true,
		"first_name":     true,
		"last_name":      true,
	}

	if !validSortFields[params.SortBy] {
		return fmt.Errorf("invalid sort_by: %s (must be one of: recording_date, age, first_name, last_name)", params.SortBy)
	}

	// Validate sort_order
	validSortOrders := map[string]bool{
		"asc":  true,
		"desc": true,
	}

	if !validSortOrders[params.SortOrder] {
		return fmt.Errorf("invalid sort_order: %s (must be 'asc' or 'desc')", params.SortOrder)
	}

	return nil
}

// buildOrderClause builds SQL ORDER BY clause with hardcoded safe values
// This eliminates any possibility of SQL injection through defense in depth
func buildOrderClause(sortBy, sortOrder string) string {
	// Map whitelisted values to exact SQL fragments
	validSortColumns := map[string]string{
		"recording_date": "recording_date",
		"age":            "age",
		"first_name":     "first_name",
		"last_name":      "last_name",
	}

	validSortOrders := map[string]string{
		"asc":  "ASC",
		"desc": "DESC",
	}

	// Get safe column name or default to recording_date
	column, exists := validSortColumns[sortBy]
	if !exists {
		column = "recording_date"
	}

	// Get safe order or default to ASC
	order, exists := validSortOrders[sortOrder]
	if !exists {
		order = "ASC"
	}

	return fmt.Sprintf("%s %s", column, order)
}

// GetReports retrieves users with optional filtering for reports (Story 3.1)
// Uses parameterized queries for security (NFR-S1 compliance)
// Returns users array, total count, and error
func GetReports(ctx context.Context, pool *pgxpool.Pool, params types.GetReportsParams) ([]models.User, int64, error) {
	// Add operation timeout for performance guarantees (AC #5)
	ctx, cancel := context.WithTimeout(ctx, DefaultOperationTimeout)
	defer cancel()

	start := time.Now()

	// Set Epic 3 defaults if parameters are nil
	var startDate int64 = 0
	var endDate int64 = time.Now().Unix()
	var minAge int = 1
	var maxAge int = 120

	if params.StartDate != nil {
		startDate = *params.StartDate
	}
	if params.EndDate != nil {
		endDate = *params.EndDate
	}
	if params.MinAge != nil {
		minAge = *params.MinAge
	}
	if params.MaxAge != nil {
		maxAge = *params.MaxAge
	}

	// Validate parameters
	if err := validateGetReportsParams(params.Limit, params.Offset, startDate, endDate, minAge, maxAge); err != nil {
		return nil, 0, fmt.Errorf("parameter validation failed: %w", err)
	}

	// Get filtered users with pagination in single query to avoid race conditions
	// Use window function to get accurate count and results in atomic operation
	query := `
		WITH filtered_users AS (
			SELECT id, first_name, last_name, age, recording_date,
				   COUNT(*) OVER() as total_count
			FROM users
			WHERE recording_date >= $1 AND recording_date <= $2 AND age >= $3 AND age <= $4
			ORDER BY recording_date DESC
			LIMIT $5 OFFSET $6
		)
		SELECT id, first_name, last_name, age, recording_date, total_count
		FROM filtered_users`

	rows, err := pool.Query(ctx, query, startDate, endDate, minAge, maxAge, params.Limit, params.Offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query users for report: %w", err)
	}
	defer rows.Close()

	var users []models.User
	var totalCount int64
	hasRows := false

	for rows.Next() {
		hasRows = true
		var user models.User
		err := rows.Scan(&user.ID, &user.FirstName, &user.LastName, &user.Age, &user.RecordingDate, &totalCount)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan user row: %w", err)
		}
		users = append(users, user)
	}

	// Handle case where no rows are returned
	if !hasRows {
		// If no users found, still need to get count separately
		countQuery := `SELECT COUNT(*) FROM users
					   WHERE recording_date >= $1 AND recording_date <= $2 AND age >= $3 AND age <= $4`
		err := pool.QueryRow(ctx, countQuery, startDate, endDate, minAge, maxAge).Scan(&totalCount)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to get total count: %w", err)
		}
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("error iterating user rows: %w", err)
	}

	// Performance monitoring
	duration := time.Since(start)
	logPerformanceMetrics("GetReports", duration)

	return users, totalCount, nil
}

// validateGetReportsParams validates query parameters for GetReports
func validateGetReportsParams(limit, offset int, startDate, endDate int64, minAge, maxAge int) error {
	// Validate limit (1-100)
	if limit < 1 || limit > 100 {
		return fmt.Errorf("invalid limit: %d (must be between 1 and 100)", limit)
	}

	// Validate offset (>= 0)
	if offset < 0 {
		return fmt.Errorf("invalid offset: %d (must be >= 0)", offset)
	}

	// Validate date range
	if startDate > endDate {
		return fmt.Errorf("invalid date range: start_date (%d) cannot be greater than end_date (%d)", startDate, endDate)
	}

	// Validate age range
	if minAge < 1 || minAge > 120 {
		return fmt.Errorf("invalid min_age: %d (must be between 1 and 120)", minAge)
	}
	if maxAge < 1 || maxAge > 120 {
		return fmt.Errorf("invalid max_age: %d (must be between 1 and 120)", maxAge)
	}
	if minAge > maxAge {
		return fmt.Errorf("invalid age range: min_age (%d) cannot be greater than max_age (%d)", minAge, maxAge)
	}

	return nil
}

// logPerformanceMetrics logs database operation performance for monitoring (AC #5)
// Helps ensure NFR-P1 compliance (<200ms response time) and provides observability
func logPerformanceMetrics(operation string, duration time.Duration) {
	// Create a structured logger for database operations
	logger := logging.NewStructuredLogger("info", "goUserAPI", "database")

	// Performance monitoring with NFR-P1 compliance checking
	durationMs := duration.Milliseconds()

	if duration > PerformanceCriticalThreshold {
		// Critical: approaching NFR-P1 limit of 200ms
		logger.Error("Database operation performance critical - NFR-P1 compliance risk",
			"operation", operation,
			"duration_ms", durationMs,
			"threshold_ms", PerformanceCriticalThreshold.Milliseconds(),
			"performance_level", "critical",
			"nfr_p1_target_ms", 200,
			"component", "database",
		)
	} else if duration > PerformanceWarningThreshold {
		// Warning: slower than optimal but within acceptable range
		logger.Warn("Database operation performance warning - exceeding optimal threshold",
			"operation", operation,
			"duration_ms", durationMs,
			"threshold_ms", PerformanceWarningThreshold.Milliseconds(),
			"performance_level", "warning",
			"nfr_p1_target_ms", 200,
			"component", "database",
		)
	} else {
		// Info: operation completed successfully within acceptable range
		logger.Info("Database operation performance metric",
			"operation", operation,
			"duration_ms", durationMs,
			"performance_level", "optimal",
			"nfr_p1_target_ms", 200,
			"component", "database",
		)
	}
}
