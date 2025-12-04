// Package handlers provides HTTP request handlers for the goUserAPI service.
package handlers

import (
	"context"
	"encoding/json"
	"io"
	"mime"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/chybatronik/goUserAPI/internal/errors"
	"github.com/chybatronik/goUserAPI/internal/logging"
	"github.com/chybatronik/goUserAPI/internal/middleware"
	"github.com/chybatronik/goUserAPI/internal/models"
	"github.com/chybatronik/goUserAPI/internal/types"
	"github.com/chybatronik/goUserAPI/internal/validation"
	"github.com/jackc/pgx/v5/pgxpool"
	pkgerrors "github.com/chybatronik/goUserAPI/pkg/errors"
)

// DatabaseService defines the interface for database operations
type DatabaseService interface {
	CreateUser(ctx context.Context, pool *pgxpool.Pool, user *models.User) (*models.User, error)
	GetUsers(ctx context.Context, pool *pgxpool.Pool, params types.GetUsersParams) ([]models.User, int64, error)
	GetReports(ctx context.Context, pool *pgxpool.Pool, params types.GetReportsParams) ([]models.User, int64, error)
}

// UserHandler handles HTTP requests for user operations
type UserHandler struct {
	pool         *pgxpool.Pool
	logger       *logging.Logger
	dbService    DatabaseService
}

// NewUserHandler creates a new UserHandler instance
func NewUserHandler(logger *logging.Logger, pool *pgxpool.Pool, dbService DatabaseService) *UserHandler {
	return &UserHandler{
		pool:      pool,
		logger:    logger,
		dbService: dbService,
	}
}

// CreateUserRequest represents the request body for creating a user
type CreateUserRequest struct {
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Age       int    `json:"age"`
}

// ErrorResponse represents the unified error response format
type ErrorResponse struct {
	Error   string `json:"error"`
	Code    string `json:"code"`
	Details string `json:"details,omitempty"`
}

// writeErrorResponse writes a unified error response
func (h *UserHandler) writeErrorResponse(w http.ResponseWriter, statusCode int, errCode, message, details string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	errorResp := ErrorResponse{
		Error:   message,
		Code:    errCode,
		Details: details,
	}

	if err := json.NewEncoder(w).Encode(errorResp); err != nil {
		h.logger.Error("Failed to encode error response",
			logging.FieldError, err,
			"error_code", errCode,
			"status_code", statusCode,
		)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// validateContentType validates that the request has application/json content type
func (h *UserHandler) validateContentType(r *http.Request) error {
	contentType := r.Header.Get("Content-Type")
	if contentType == "" {
		return pkgerrors.NewUserValidationError("INVALID_CONTENT_TYPE", "Content-Type header is required")
	}

	// Parse media type to handle content types like "application/json; charset=utf-8"
	mediaType, _, err := mime.ParseMediaType(contentType)
	if err != nil {
		return pkgerrors.NewUserValidationError("INVALID_CONTENT_TYPE", "Invalid Content-Type header format")
	}

	if mediaType != "application/json" {
		return pkgerrors.NewUserValidationError("INVALID_CONTENT_TYPE", "Content-Type must be application/json")
	}
	return nil
}

// parseRequestBody parses and validates the JSON request body
func (h *UserHandler) parseRequestBody(r *http.Request) (*CreateUserRequest, error) {
	// Check for empty request body
	if r.Body == nil {
		return nil, pkgerrors.NewUserValidationError("EMPTY_REQUEST_BODY", "Request body cannot be empty")
	}

	// Read and limit request body size (1MB limit)
	defer r.Body.Close()
	body, err := io.ReadAll(io.LimitReader(r.Body, 1048576))
	if err != nil {
		return nil, pkgerrors.NewUserValidationError("EMPTY_REQUEST_BODY", "Failed to read request body")
	}

	// Check for empty body after read
	if len(body) == 0 {
		return nil, pkgerrors.NewUserValidationError("EMPTY_REQUEST_BODY", "Request body cannot be empty")
	}

	// Parse JSON
	var req CreateUserRequest
	if err := json.Unmarshal(body, &req); err != nil {
		return nil, pkgerrors.NewUserValidationError("INVALID_JSON", "Invalid JSON format")
	}

	return &req, nil
}

// validateUserRequest validates the user request data with security checks
func (h *UserHandler) validateUserRequest(req *CreateUserRequest) error {
	// Check for missing required fields
	if req.FirstName == "" {
		return pkgerrors.NewUserValidationError("MISSING_REQUIRED_FIELD", "Missing required field: first_name")
	}
	if req.LastName == "" {
		return pkgerrors.NewUserValidationError("MISSING_REQUIRED_FIELD", "Missing required field: last_name")
	}

	// Trim whitespace from string fields
	req.FirstName = strings.TrimSpace(req.FirstName)
	req.LastName = strings.TrimSpace(req.LastName)

	// Validate fields are not empty after trimming
	if req.FirstName == "" {
		return pkgerrors.NewUserValidationError("EMPTY_FIELD_AFTER_TRIM", "First name cannot be empty after removing whitespace")
	}
	if req.LastName == "" {
		return pkgerrors.NewUserValidationError("EMPTY_FIELD_AFTER_TRIM", "Last name cannot be empty after removing whitespace")
	}

	// SECURITY: Unicode security validation (NFR-S2 compliance)
	if err := validation.ValidateUnicodeSecurity(req.FirstName); err != nil {
		h.logger.Warn("Unicode security validation failed for first_name",
			"field", "first_name",
			"error", err.Error(),
		)
		return pkgerrors.NewUserValidationError("UNICODE_SECURITY_VIOLATION", "Invalid characters in first name")
	}

	if err := validation.ValidateUnicodeSecurity(req.LastName); err != nil {
		h.logger.Warn("Unicode security validation failed for last_name",
			"field", "last_name",
			"error", err.Error(),
		)
		return pkgerrors.NewUserValidationError("UNICODE_SECURITY_VIOLATION", "Invalid characters in last name")
	}

	// Validate field lengths (after security validation)
	if len(req.FirstName) > 100 {
		return pkgerrors.NewUserValidationError("INVALID_FIELD_LENGTH", "First name cannot exceed 100 characters")
	}
	if len(req.LastName) > 100 {
		return pkgerrors.NewUserValidationError("INVALID_FIELD_LENGTH", "Last name cannot exceed 100 characters")
	}

	// Validate age range
	if req.Age < 1 || req.Age > 120 {
		return pkgerrors.NewUserValidationError("INVALID_AGE_RANGE", "Age must be between 1 and 120")
	}

	return nil
}

// convertToModel converts request to User model
func (h *UserHandler) convertToModel(req *CreateUserRequest) *models.User {
	return &models.User{
		FirstName: req.FirstName,
		LastName:  req.LastName,
		Age:       req.Age,
	}
}

// writeSuccessResponse writes a successful user creation response
func (h *UserHandler) writeSuccessResponse(w http.ResponseWriter, user *models.User) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)

	if err := json.NewEncoder(w).Encode(user); err != nil {
		h.logger.Error("Failed to encode success response",
			logging.FieldError, err,
			"user_id", user.ID,
		)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// extractRequestID extracts request ID from context
func (h *UserHandler) extractRequestID(r *http.Request) string {
	if reqID := middleware.GetRequestID(r.Context()); reqID != "" {
		return reqID
	}
	return "unknown"
}

// CreateUser handles user creation requests
func (h *UserHandler) CreateUser(w http.ResponseWriter, r *http.Request) {
	// Start timer for performance monitoring
	startTime := time.Now()
	reqID := h.extractRequestID(r)
	logger := h.logger.WithRequestID(reqID)

	logger.Info("Starting user creation request",
		"method", r.Method,
		"path", r.URL.Path,
		"remote_addr", r.RemoteAddr,
	)

	// Validate HTTP method - only POST is allowed
	if r.Method != http.MethodPost {
		logger.Warn("Invalid HTTP method for user creation",
			"method", r.Method,
			"expected_method", "POST",
		)
		h.writeErrorResponse(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Only POST method is allowed", "")
		return
	}

	// Validate Content-Type header
	if err := h.validateContentType(r); err != nil {
		logger.Warn("Invalid Content-Type header",
			"content_type", r.Header.Get("Content-Type"),
			"error", err.Error(),
		)
		h.writeErrorResponse(w, http.StatusBadRequest, "INVALID_CONTENT_TYPE", err.Error(), "Content-Type header must be 'application/json'")
		return
	}

	// Parse and validate request body
	req, err := h.parseRequestBody(r)
	if err != nil {
		logger.Warn("Failed to parse request body",
			"error", err.Error(),
		)
		// Safe type assertion with fallback
		if userErr, ok := err.(*pkgerrors.UserError); ok {
			h.writeErrorResponse(w, http.StatusBadRequest, userErr.Code, userErr.Message, "")
		} else {
			h.writeErrorResponse(w, http.StatusBadRequest, "INVALID_REQUEST_BODY",
				"Invalid request body format", err.Error())
		}
		return
	}

	// Validate user input fields
	if err := h.validateUserRequest(req); err != nil {
		logger.Warn("User input validation failed",
			"first_name", req.FirstName,
			"last_name", req.LastName,
			"age", req.Age,
			"error", err.Error(),
		)
		// Safe type assertion with fallback
		if userErr, ok := err.(*pkgerrors.UserError); ok {
			details := ""
			if strings.Contains(userErr.Message, "first_name") {
				details = "field: first_name"
			} else if strings.Contains(userErr.Message, "last_name") {
				details = "field: last_name"
			}
			h.writeErrorResponse(w, http.StatusBadRequest, userErr.Code, userErr.Message, details)
		} else {
			h.writeErrorResponse(w, http.StatusBadRequest, "VALIDATION_ERROR",
				"User input validation failed", err.Error())
		}
		return
	}

	// Convert request to User model
	user := h.convertToModel(req)

	logger.Info("Creating user in database",
		"first_name", user.FirstName,
		"last_name", user.LastName,
		"age", user.Age,
	)

	// Create user in database
	createdUser, err := h.dbService.CreateUser(r.Context(), h.pool, user)
	if err != nil {
		logger.Error("Failed to create user in database",
			logging.FieldError, err,
			"first_name", user.FirstName,
			"last_name", user.LastName,
		)

		// SECURITY: Use secure error mapping to prevent information leakage (NFR-S3 compliance)
		secureErr := errors.MapDatabaseErrorSecure(err)
		if userErr, ok := secureErr.(*pkgerrors.UserError); ok {
			h.writeErrorResponse(w, userErr.GetHTTPStatus(), userErr.Code, userErr.Message, "")
		} else {
			h.writeErrorResponse(w, http.StatusInternalServerError, "DATABASE_ERROR", "Database operation failed", "")
		}
		return
	}

	// Log successful user creation
	logger.Info("User created successfully",
		"user_id", createdUser.ID,
		"first_name", createdUser.FirstName,
		"last_name", createdUser.LastName,
		"age", createdUser.Age,
		"recording_date", createdUser.RecordingDate,
	)

	// Write success response
	h.writeSuccessResponse(w, createdUser)

	// Log request completion for performance monitoring
	duration := time.Since(startTime)
	logger.Info("User creation request completed",
		"duration_ms", duration.Milliseconds(),
		"status_code", http.StatusCreated,
		"user_id", createdUser.ID,
	)
}

// GetUsersRequestParams represents query parameters for GetUsers
type GetUsersRequestParams struct {
	Limit     int
	Offset    int
	SortBy    string
	SortOrder string
}

// GetUsersResponse represents the response format for GetUsers
type GetUsersResponse struct {
	Users      []models.User   `json:"users"`
	Pagination PaginationInfo `json:"pagination"`
}

// PaginationInfo represents pagination metadata
type PaginationInfo struct {
	TotalCount int64 `json:"total_count"`
	Limit      int   `json:"limit"`
	Offset     int   `json:"offset"`
	HasMore    bool  `json:"has_more"`
}

// parseAndValidateQueryParams parses and validates query parameters
func (h *UserHandler) parseAndValidateQueryParams(r *http.Request) (*GetUsersRequestParams, error) {
	params := &GetUsersRequestParams{}

	// Parse limit with default
	limitStr := r.URL.Query().Get("limit")
	if limitStr == "" {
		params.Limit = 20 // default
	} else {
		limit, err := strconv.Atoi(limitStr)
		if err != nil {
			return nil, pkgerrors.NewUserValidationError("INVALID_LIMIT_PARAMETER", "Invalid limit parameter. Must be between 1 and 100")
		}
		params.Limit = limit
	}

	// Parse offset with default
	offsetStr := r.URL.Query().Get("offset")
	if offsetStr == "" {
		params.Offset = 0 // default
	} else {
		offset, err := strconv.Atoi(offsetStr)
		if err != nil {
			return nil, pkgerrors.NewUserValidationError("INVALID_OFFSET_PARAMETER", "Invalid offset parameter. Must be >= 0")
		}
		params.Offset = offset
	}

	// Parse sort_by with default
	sortBy := r.URL.Query().Get("sort_by")
	if sortBy == "" {
		params.SortBy = "recording_date" // default
	} else {
		params.SortBy = sortBy
	}

	// Parse sort_order with default
	sortOrder := r.URL.Query().Get("sort_order")
	if sortOrder == "" {
		params.SortOrder = "desc" // default for recording_date
	} else {
		params.SortOrder = sortOrder
	}

	return params, nil
}

// validateGetUsersParams validates parsed query parameters against business rules
func (h *UserHandler) validateGetUsersParams(params *GetUsersRequestParams) error {
	// Validate limit range (1-100)
	if params.Limit < 1 || params.Limit > 100 {
		return pkgerrors.NewUserValidationError("INVALID_LIMIT_PARAMETER",
			"Invalid limit parameter. Must be between 1 and 100")
	}

	// Validate offset range (>= 0)
	if params.Offset < 0 {
		return pkgerrors.NewUserValidationError("INVALID_OFFSET_PARAMETER",
			"Invalid offset parameter. Must be >= 0")
	}

	// Validate sort_by against whitelist
	validSortFields := []string{"recording_date", "age", "first_name", "last_name"}
	isValidSortField := false
	for _, field := range validSortFields {
		if params.SortBy == field {
			isValidSortField = true
			break
		}
	}
	if !isValidSortField {
		return pkgerrors.NewUserValidationError("INVALID_SORT_FIELD",
			"Invalid sort_by parameter. Must be one of: recording_date, age, first_name, last_name")
	}

	// Validate sort_order
	if params.SortOrder != "asc" && params.SortOrder != "desc" {
		return pkgerrors.NewUserValidationError("INVALID_SORT_ORDER",
			"Invalid sort_order parameter. Must be 'asc' or 'desc'")
	}

	return nil
}

// writeGetUsersResponse writes a successful GetUsers response with pagination metadata
func (h *UserHandler) writeGetUsersResponse(w http.ResponseWriter, users []models.User, totalCount int64, limit, offset int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	// Calculate has_more for pagination
	hasMore := int64(offset+limit) < totalCount

	response := GetUsersResponse{
		Users: users,
		Pagination: PaginationInfo{
			TotalCount: totalCount,
			Limit:      limit,
			Offset:     offset,
			HasMore:    hasMore,
		},
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		h.logger.Error("Failed to encode GetUsers response",
			logging.FieldError, err,
			"total_count", totalCount,
			"limit", limit,
			"offset", offset,
		)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// GetUsers handles user retrieval requests with pagination and sorting (Story 2.3)
func (h *UserHandler) GetUsers(w http.ResponseWriter, r *http.Request) {
	// Start timer for performance monitoring
	startTime := time.Now()
	reqID := h.extractRequestID(r)
	logger := h.logger.WithRequestID(reqID)

	logger.Info("Starting user retrieval request",
		"method", r.Method,
		"path", r.URL.Path,
		"query", r.URL.RawQuery,
		"remote_addr", r.RemoteAddr,
	)

	// Validate HTTP method - only GET is allowed
	if r.Method != http.MethodGet {
		logger.Warn("Invalid HTTP method for user retrieval",
			"method", r.Method,
			"expected_method", "GET",
		)
		h.writeErrorResponse(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED",
			"Only GET method is allowed", "")
		return
	}

	// Parse and validate query parameters
	params, err := h.parseAndValidateQueryParams(r)
	if err != nil {
		logger.Warn("Failed to parse query parameters",
			"error", err.Error(),
			"query", r.URL.RawQuery,
		)
		// Safe type assertion with fallback
		if userErr, ok := err.(*pkgerrors.UserError); ok {
			h.writeErrorResponse(w, http.StatusBadRequest, userErr.Code, userErr.Message, "")
		} else {
			h.writeErrorResponse(w, http.StatusBadRequest, "INVALID_QUERY_PARAMETERS",
				"Invalid query parameters", err.Error())
		}
		return
	}

	// Validate parameters against business rules
	if err := h.validateGetUsersParams(params); err != nil {
		logger.Warn("Query parameter validation failed",
			"limit", params.Limit,
			"offset", params.Offset,
			"sort_by", params.SortBy,
			"sort_order", params.SortOrder,
			"error", err.Error(),
		)
		// Safe type assertion with fallback
		if userErr, ok := err.(*pkgerrors.UserError); ok {
			details := ""
			if strings.Contains(userErr.Message, "limit") {
				details = "parameter: limit, value: " + strconv.Itoa(params.Limit) + ", valid_range: 1-100"
			} else if strings.Contains(userErr.Message, "sort_by") {
				details = "parameter: sort_by, value: " + params.SortBy + ", allowed_fields: recording_date,age,first_name,last_name"
			}
			h.writeErrorResponse(w, http.StatusBadRequest, userErr.Code, userErr.Message, details)
		} else {
			h.writeErrorResponse(w, http.StatusBadRequest, "VALIDATION_ERROR",
				"Query parameter validation failed", err.Error())
		}
		return
	}

	// Prepare database parameters
	dbParams := types.GetUsersParams{
		Limit:     params.Limit,
		Offset:    params.Offset,
		SortBy:    params.SortBy,
		SortOrder: params.SortOrder,
	}

	logger.Info("Retrieving users from database",
		"limit", params.Limit,
		"offset", params.Offset,
		"sort_by", params.SortBy,
		"sort_order", params.SortOrder,
	)

	// Get users from database
	users, totalCount, err := h.dbService.GetUsers(r.Context(), h.pool, dbParams)
	if err != nil {
		logger.Error("Failed to retrieve users from database",
			logging.FieldError, err,
			"limit", params.Limit,
			"offset", params.Offset,
		)

		// SECURITY: Use secure error mapping to prevent information leakage (NFR-S3 compliance)
		secureErr := errors.MapDatabaseErrorSecure(err)
		if userErr, ok := secureErr.(*pkgerrors.UserError); ok {
			h.writeErrorResponse(w, userErr.GetHTTPStatus(), userErr.Code, userErr.Message, "")
		} else {
			h.writeErrorResponse(w, http.StatusInternalServerError, "DATABASE_ERROR", "Database operation failed", "")
		}
		return
	}

	// Log successful retrieval
	logger.Info("Users retrieved successfully",
		"user_count", len(users),
		"total_count", totalCount,
		"limit", params.Limit,
		"offset", params.Offset,
	)

	// Write success response
	h.writeGetUsersResponse(w, users, totalCount, params.Limit, params.Offset)

	// Log request completion for performance monitoring
	duration := time.Since(startTime)
	logger.Info("User retrieval request completed",
		"duration_ms", duration.Milliseconds(),
		"status_code", http.StatusOK,
		"user_count", len(users),
		"total_count", totalCount,
	)
}
