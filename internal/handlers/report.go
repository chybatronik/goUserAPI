// Package handlers provides HTTP request handlers for the goUserAPI service.
package handlers

import (
	"encoding/json"
	"fmt"
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

// ReportHandler handles HTTP requests for report operations
type ReportHandler struct {
	pool         *pgxpool.Pool
	logger       *logging.Logger
	dbService    DatabaseService
}

// NewReportHandler creates a new ReportHandler instance
func NewReportHandler(logger *logging.Logger, pool *pgxpool.Pool, dbService DatabaseService) *ReportHandler {
	return &ReportHandler{
		pool:      pool,
		logger:    logger,
		dbService: dbService,
	}
}

// writeErrorResponse writes a unified error response
func (h *ReportHandler) writeErrorResponse(w http.ResponseWriter, statusCode int, errCode, message, details string) {
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

// GetReportsRequestParams represents query parameters for GetReports
type GetReportsRequestParams struct {
	Limit     int
	Offset    int
	StartDate *int64
	EndDate   *int64
	MinAge    *int
	MaxAge    *int
}

// GetReportsResponse represents the response format for GetReports
type GetReportsResponse struct {
	Count      int64           `json:"count"`
	Users      []models.User   `json:"users"`
	Pagination PaginationInfo  `json:"pagination"`
}

// parseAndValidateReportsQueryParams parses and validates query parameters for GetReports
func (h *ReportHandler) parseAndValidateReportsQueryParams(r *http.Request) (*GetReportsRequestParams, error) {
	params := &GetReportsRequestParams{}

	// SECURITY: Apply Story 2.4 Unicode security validation only to string parameters
	// Numeric parameters don't need Unicode validation for performance
	// This prevents homograph attacks, control characters, and other Unicode-based attacks
	queryParams := r.URL.Query()
	for key, values := range queryParams {
		// Skip Unicode validation for known numeric parameters to improve performance
		isNumericParam := key == "limit" || key == "offset" || key == "start_date" || key == "end_date" || key == "min_age" || key == "max_age"

		for _, value := range values {
			if !isNumericParam {
				if err := validation.ValidateUnicodeSecurity(value); err != nil {
					return nil, pkgerrors.NewUserValidationError("UNSECURE_UNICODE_INPUT",
						fmt.Sprintf("Invalid unicode characters in parameter '%s'", key))
				}
			}
			if err := validation.ValidateFieldSecurity(value, key, 1000); err != nil {
				return nil, pkgerrors.NewUserValidationError("INVALID_PARAMETER_FORMAT",
					fmt.Sprintf("Invalid format for parameter '%s'", key))
			}
		}
	}

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

	// Parse start_date (optional)
	startDateStr := r.URL.Query().Get("start_date")
	if startDateStr != "" {
		startDate, err := strconv.ParseInt(startDateStr, 10, 64)
		if err != nil {
			return nil, pkgerrors.NewUserValidationError("INVALID_START_DATE_PARAMETER", "Invalid start_date parameter. Must be Unix timestamp")
		}
		params.StartDate = &startDate
	}

	// Parse end_date (optional)
	endDateStr := r.URL.Query().Get("end_date")
	if endDateStr != "" {
		endDate, err := strconv.ParseInt(endDateStr, 10, 64)
		if err != nil {
			return nil, pkgerrors.NewUserValidationError("INVALID_END_DATE_PARAMETER", "Invalid end_date parameter. Must be Unix timestamp")
		}
		params.EndDate = &endDate
	}

	// Parse min_age (optional)
	minAgeStr := r.URL.Query().Get("min_age")
	if minAgeStr != "" {
		minAge, err := strconv.Atoi(minAgeStr)
		if err != nil {
			return nil, pkgerrors.NewUserValidationError("INVALID_MIN_AGE_PARAMETER", "Invalid min_age parameter. Must be integer between 1 and 120")
		}
		params.MinAge = &minAge
	}

	// Parse max_age (optional)
	maxAgeStr := r.URL.Query().Get("max_age")
	if maxAgeStr != "" {
		maxAge, err := strconv.Atoi(maxAgeStr)
		if err != nil {
			return nil, pkgerrors.NewUserValidationError("INVALID_MAX_AGE_PARAMETER", "Invalid max_age parameter. Must be integer between 1 and 120")
		}
		params.MaxAge = &maxAge
	}

	return params, nil
}

// validateGetReportsParams validates parsed query parameters against business rules
func (h *ReportHandler) validateGetReportsParams(params *GetReportsRequestParams) error {
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

	// Validate age range if both provided
	if params.MinAge != nil && params.MaxAge != nil {
		if *params.MinAge < 1 || *params.MinAge > 120 {
			return pkgerrors.NewUserValidationError("INVALID_MIN_AGE_PARAMETER",
				"Invalid min_age parameter. Must be between 1 and 120")
		}
		if *params.MaxAge < 1 || *params.MaxAge > 120 {
			return pkgerrors.NewUserValidationError("INVALID_MAX_AGE_PARAMETER",
				"Invalid max_age parameter. Must be between 1 and 120")
		}
		if *params.MinAge > *params.MaxAge {
			return pkgerrors.NewUserValidationError("INVALID_AGE_RANGE",
				"Invalid age range: min_age cannot be greater than max_age")
		}
	}

	// Validate individual min_age
	if params.MinAge != nil {
		if *params.MinAge < 1 || *params.MinAge > 120 {
			return pkgerrors.NewUserValidationError("INVALID_MIN_AGE_PARAMETER",
				"Invalid min_age parameter. Must be between 1 and 120")
		}
	}

	// Validate individual max_age
	if params.MaxAge != nil {
		if *params.MaxAge < 1 || *params.MaxAge > 120 {
			return pkgerrors.NewUserValidationError("INVALID_MAX_AGE_PARAMETER",
				"Invalid max_age parameter. Must be between 1 and 120")
		}
	}

	return nil
}

// writeGetReportsResponse writes a successful GetReports response with pagination metadata
func (h *ReportHandler) writeGetReportsResponse(w http.ResponseWriter, users []models.User, totalCount int64, limit, offset int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	// Calculate has_more for pagination
	hasMore := int64(offset+limit) < totalCount

	response := GetReportsResponse{
		Count: totalCount,
		Users: users,
		Pagination: PaginationInfo{
			TotalCount: totalCount,
			Limit:      limit,
			Offset:     offset,
			HasMore:    hasMore,
		},
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		h.logger.Error("Failed to encode GetReports response",
			logging.FieldError, err,
			"total_count", totalCount,
			"limit", limit,
			"offset", offset,
		)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// extractRequestID extracts request ID from context
func (h *ReportHandler) extractRequestID(r *http.Request) string {
	if reqID := middleware.GetRequestID(r.Context()); reqID != "" {
		return reqID
	}
	return "unknown"
}

// GetReports handles report generation requests with optional filtering (Story 3.1)
func (h *ReportHandler) GetReports(w http.ResponseWriter, r *http.Request) {
	// Start timer for performance monitoring
	startTime := time.Now()
	reqID := h.extractRequestID(r)
	logger := h.logger.WithRequestID(reqID)

	logger.Info("Starting report generation request",
		"method", r.Method,
		"path", r.URL.Path,
		"query", r.URL.RawQuery,
		"remote_addr", r.RemoteAddr,
	)

	// Validate HTTP method - only GET is allowed
	if r.Method != http.MethodGet {
		logger.Warn("Invalid HTTP method for report generation",
			"method", r.Method,
			"expected_method", "GET",
		)
		h.writeErrorResponse(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED",
			"Only GET method is allowed", "")
		return
	}

	// Parse and validate query parameters
	params, err := h.parseAndValidateReportsQueryParams(r)
	if err != nil {
		logger.Warn("Failed to parse query parameters",
			"error", err.Error(),
			"query", r.URL.RawQuery,
		)
		// Safe type assertion with fallback
		if userErr, ok := err.(*pkgerrors.UserError); ok {
			h.writeErrorResponse(w, http.StatusBadRequest, userErr.Code, userErr.Message, "")
		} else {
			// Fallback for unexpected error types
			h.writeErrorResponse(w, http.StatusBadRequest, "INVALID_QUERY_PARAMETERS",
				"Invalid query parameters", err.Error())
		}
		return
	}

	// Validate parameters against business rules
	if err := h.validateGetReportsParams(params); err != nil {
		logger.Warn("Query parameter validation failed",
			"limit", params.Limit,
			"offset", params.Offset,
			"start_date", params.StartDate,
			"end_date", params.EndDate,
			"min_age", params.MinAge,
			"max_age", params.MaxAge,
			"error", err.Error(),
		)
		// Safe type assertion with fallback
		if userErr, ok := err.(*pkgerrors.UserError); ok {
			details := ""
			if strings.Contains(userErr.Message, "limit") {
				details = "parameter: limit, value: " + strconv.Itoa(params.Limit) + ", valid_range: 1-100"
			} else if strings.Contains(userErr.Message, "age") {
				details = "parameters: min_age, max_age, valid_range: 1-120"
			}
			h.writeErrorResponse(w, http.StatusBadRequest, userErr.Code, userErr.Message, details)
		} else {
			// Fallback for unexpected error types
			h.writeErrorResponse(w, http.StatusBadRequest, "VALIDATION_ERROR",
				"Parameter validation failed", err.Error())
		}
		return
	}

	// Prepare database parameters with Epic 3 defaults
	dbParams := types.GetReportsParams{
		Limit:     params.Limit,
		Offset:    params.Offset,
		StartDate: params.StartDate,
		EndDate:   params.EndDate,
		MinAge:    params.MinAge,
		MaxAge:    params.MaxAge,
	}

	logger.Info("Generating report from database",
		"limit", params.Limit,
		"offset", params.Offset,
		"start_date", params.StartDate,
		"end_date", params.EndDate,
		"min_age", params.MinAge,
		"max_age", params.MaxAge,
	)

	// Get reports from database
	users, totalCount, err := h.dbService.GetReports(r.Context(), h.pool, dbParams)
	if err != nil {
		logger.Error("Failed to generate report from database",
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

	// Log successful report generation
	logger.Info("Report generated successfully",
		"user_count", len(users),
		"total_count", totalCount,
		"limit", params.Limit,
		"offset", params.Offset,
	)

	// Write success response
	h.writeGetReportsResponse(w, users, totalCount, params.Limit, params.Offset)

	// Log request completion for performance monitoring
	duration := time.Since(startTime)
	logger.Info("Report generation request completed",
		"duration_ms", duration.Milliseconds(),
		"status_code", http.StatusOK,
		"user_count", len(users),
		"total_count", totalCount,
	)
}