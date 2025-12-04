package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/chybatronik/goUserAPI/internal/logging"
	"github.com/chybatronik/goUserAPI/internal/models"
	"github.com/chybatronik/goUserAPI/internal/types"
	pkgerrors "github.com/chybatronik/goUserAPI/pkg/errors"
	"github.com/jackc/pgx/v5/pgxpool"
)

// MockDatabaseService implements DatabaseService for testing
type MockDatabaseService struct {
	users      []models.User
	totalCount int64
	err        error
}

func (m *MockDatabaseService) CreateUser(ctx context.Context, pool *pgxpool.Pool, user *models.User) (*models.User, error) {
	return nil, nil
}

func (m *MockDatabaseService) GetUsers(ctx context.Context, pool *pgxpool.Pool, params types.GetUsersParams) ([]models.User, int64, error) {
	return nil, 0, nil
}

func (m *MockDatabaseService) GetReports(ctx context.Context, pool *pgxpool.Pool, params types.GetReportsParams) ([]models.User, int64, error) {
	if m.err != nil {
		return nil, 0, m.err
	}
	return m.users, m.totalCount, nil
}

func setupTestReportHandler() *ReportHandler {
	logger := logging.NewStructuredLogger("info", "goUserAPI", "test")
	dbService := &MockDatabaseService{}
	return NewReportHandler(logger, nil, dbService)
}

func TestGetReports_DefaultParameters(t *testing.T) {
	handler := setupTestReportHandler()

	// Setup mock data
	mockUsers := []models.User{
		{ID: "1", FirstName: "John", LastName: "Doe", Age: 30, RecordingDate: time.Now().Unix()},
		{ID: "2", FirstName: "Jane", LastName: "Smith", Age: 25, RecordingDate: time.Now().Unix()},
	}
	dbService := handler.dbService.(*MockDatabaseService)
	dbService.users = mockUsers
	dbService.totalCount = 2

	// Create request with no parameters
	req := httptest.NewRequest(http.MethodGet, "/reports", nil)
	w := httptest.NewRecorder()

	handler.GetReports(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, resp.StatusCode)
	}

	var response GetReportsResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response.Count != 2 {
		t.Errorf("Expected count 2, got %d", response.Count)
	}

	if len(response.Users) != 2 {
		t.Errorf("Expected 2 users, got %d", len(response.Users))
	}

	if response.Pagination.HasMore {
		t.Error("Expected has_more to be false when offset+limit >= total_count")
	}
}

func TestGetReports_WithLimitAndOffset(t *testing.T) {
	handler := setupTestReportHandler()

	// Setup mock data
	mockUsers := []models.User{
		{ID: "1", FirstName: "John", LastName: "Doe", Age: 30, RecordingDate: time.Now().Unix()},
	}
	dbService := handler.dbService.(*MockDatabaseService)
	dbService.users = mockUsers
	dbService.totalCount = 5

	// Create request with limit and offset
	params := url.Values{}
	params.Add("limit", "1")
	params.Add("offset", "2")
	req := httptest.NewRequest(http.MethodGet, "/reports?"+params.Encode(), nil)
	w := httptest.NewRecorder()

	handler.GetReports(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, resp.StatusCode)
	}

	var response GetReportsResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response.Pagination.Limit != 1 {
		t.Errorf("Expected limit 1, got %d", response.Pagination.Limit)
	}

	if response.Pagination.Offset != 2 {
		t.Errorf("Expected offset 2, got %d", response.Pagination.Offset)
	}

	// With limit=1, offset=2, total_count=5: has_more should be true (2+1 < 5)
	if !response.Pagination.HasMore {
		t.Error("Expected has_more to be true when offset+limit < total_count")
	}
}

func TestGetReports_WithAgeFilters(t *testing.T) {
	handler := setupTestReportHandler()

	// Setup mock data
	mockUsers := []models.User{
		{ID: "1", FirstName: "John", LastName: "Doe", Age: 30, RecordingDate: time.Now().Unix()},
	}
	dbService := handler.dbService.(*MockDatabaseService)
	dbService.users = mockUsers
	dbService.totalCount = 1

	// Create request with age filters
	params := url.Values{}
	params.Add("min_age", "25")
	params.Add("max_age", "35")
	req := httptest.NewRequest(http.MethodGet, "/reports?"+params.Encode(), nil)
	w := httptest.NewRecorder()

	handler.GetReports(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, resp.StatusCode)
	}
}

func TestGetReports_WithDateFilters(t *testing.T) {
	handler := setupTestReportHandler()

	// Setup mock data
	mockUsers := []models.User{
		{ID: "1", FirstName: "John", LastName: "Doe", Age: 30, RecordingDate: time.Now().Unix()},
	}
	dbService := handler.dbService.(*MockDatabaseService)
	dbService.users = mockUsers
	dbService.totalCount = 1

	// Create request with date filters
	now := time.Now().Unix()
	params := url.Values{}
	params.Add("start_date", "1609459200") // 2021-01-01
	params.Add("end_date", strconv.FormatInt(now, 10))
	req := httptest.NewRequest(http.MethodGet, "/reports?"+params.Encode(), nil)
	w := httptest.NewRecorder()

	handler.GetReports(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, resp.StatusCode)
	}
}

func TestGetReports_InvalidLimit(t *testing.T) {
	handler := setupTestReportHandler()

	// Create request with invalid limit
	req := httptest.NewRequest(http.MethodGet, "/reports?limit=0", nil)
	w := httptest.NewRecorder()

	handler.GetReports(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, resp.StatusCode)
	}

	var errorResp ErrorResponse
	if err := json.NewDecoder(resp.Body).Decode(&errorResp); err != nil {
		t.Fatalf("Failed to decode error response: %v", err)
	}

	if errorResp.Code != "INVALID_LIMIT_PARAMETER" {
		t.Errorf("Expected error code 'INVALID_LIMIT_PARAMETER', got '%s'", errorResp.Code)
	}
}

func TestGetReports_InvalidOffset(t *testing.T) {
	handler := setupTestReportHandler()

	// Create request with invalid offset
	req := httptest.NewRequest(http.MethodGet, "/reports?offset=-1", nil)
	w := httptest.NewRecorder()

	handler.GetReports(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, resp.StatusCode)
	}
}

func TestGetReports_InvalidAgeRange(t *testing.T) {
	handler := setupTestReportHandler()

	// Create request with invalid age range (min > max)
	params := url.Values{}
	params.Add("min_age", "50")
	params.Add("max_age", "30")
	req := httptest.NewRequest(http.MethodGet, "/reports?"+params.Encode(), nil)
	w := httptest.NewRecorder()

	handler.GetReports(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, resp.StatusCode)
	}
}

func TestGetReports_InvalidMethod(t *testing.T) {
	handler := setupTestReportHandler()

	// Create POST request (should fail)
	req := httptest.NewRequest(http.MethodPost, "/reports", nil)
	w := httptest.NewRecorder()

	handler.GetReports(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusMethodNotAllowed {
		t.Errorf("Expected status %d, got %d", http.StatusMethodNotAllowed, resp.StatusCode)
	}

	var errorResp ErrorResponse
	if err := json.NewDecoder(resp.Body).Decode(&errorResp); err != nil {
		t.Fatalf("Failed to decode error response: %v", err)
	}

	if errorResp.Code != "METHOD_NOT_ALLOWED" {
		t.Errorf("Expected error code 'METHOD_NOT_ALLOWED', got '%s'", errorResp.Code)
	}
}

func TestGetReports_DatabaseError(t *testing.T) {
	handler := setupTestReportHandler()

	// Setup mock to return error
	dbService := handler.dbService.(*MockDatabaseService)
	dbService.err = pkgerrors.NewUserValidationError("DATABASE_ERROR", "Database connection failed")

	req := httptest.NewRequest(http.MethodGet, "/reports", nil)
	w := httptest.NewRecorder()

	handler.GetReports(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	// The error mapping may return different status codes based on the error type
	if resp.StatusCode != http.StatusInternalServerError && resp.StatusCode != http.StatusServiceUnavailable {
		t.Errorf("Expected status 500 or 503, got %d", resp.StatusCode)
	}
}

func TestParseAndValidateReportsQueryParams_ValidParameters(t *testing.T) {
	handler := setupTestReportHandler()

	// Create request with all valid parameters
	params := url.Values{}
	params.Add("limit", "10")
	params.Add("offset", "5")
	params.Add("start_date", "1609459200")
	params.Add("end_date", "1640995200")
	params.Add("min_age", "18")
	params.Add("max_age", "65")
	req := httptest.NewRequest(http.MethodGet, "/reports?"+params.Encode(), nil)

	parsedParams, err := handler.parseAndValidateReportsQueryParams(req)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if parsedParams.Limit != 10 {
		t.Errorf("Expected limit 10, got %d", parsedParams.Limit)
	}

	if parsedParams.Offset != 5 {
		t.Errorf("Expected offset 5, got %d", parsedParams.Offset)
	}

	if *parsedParams.StartDate != 1609459200 {
		t.Errorf("Expected start_date 1609459200, got %d", *parsedParams.StartDate)
	}

	if *parsedParams.EndDate != 1640995200 {
		t.Errorf("Expected end_date 1640995200, got %d", *parsedParams.EndDate)
	}

	if *parsedParams.MinAge != 18 {
		t.Errorf("Expected min_age 18, got %d", *parsedParams.MinAge)
	}

	if *parsedParams.MaxAge != 65 {
		t.Errorf("Expected max_age 65, got %d", *parsedParams.MaxAge)
	}
}

func TestParseAndValidateReportsQueryParams_DefaultValues(t *testing.T) {
	handler := setupTestReportHandler()

	// Create request with no parameters (should use defaults)
	req := httptest.NewRequest(http.MethodGet, "/reports", nil)

	parsedParams, err := handler.parseAndValidateReportsQueryParams(req)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if parsedParams.Limit != 20 {
		t.Errorf("Expected default limit 20, got %d", parsedParams.Limit)
	}

	if parsedParams.Offset != 0 {
		t.Errorf("Expected default offset 0, got %d", parsedParams.Offset)
	}

	if parsedParams.StartDate != nil {
		t.Error("Expected start_date to be nil when not provided")
	}

	if parsedParams.EndDate != nil {
		t.Error("Expected end_date to be nil when not provided")
	}

	if parsedParams.MinAge != nil {
		t.Error("Expected min_age to be nil when not provided")
	}

	if parsedParams.MaxAge != nil {
		t.Error("Expected max_age to be nil when not provided")
	}
}

func TestValidateGetReportsParams_ValidParameters(t *testing.T) {
	handler := setupTestReportHandler()

	params := &GetReportsRequestParams{
		Limit:  10,
		Offset: 0,
	}

	err := handler.validateGetReportsParams(params)
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}
}

func TestValidateGetReportsParams_InvalidLimit(t *testing.T) {
	handler := setupTestReportHandler()

	params := &GetReportsRequestParams{
		Limit:  0, // Invalid
		Offset: 0,
	}

	err := handler.validateGetReportsParams(params)
	if err == nil {
		t.Error("Expected error for invalid limit")
	}

	if !strings.Contains(err.Error(), "limit") {
		t.Errorf("Expected error to mention 'limit', got: %v", err)
	}
}

func TestValidateGetReportsParams_InvalidOffset(t *testing.T) {
	handler := setupTestReportHandler()

	params := &GetReportsRequestParams{
		Limit:  10,
		Offset: -1, // Invalid
	}

	err := handler.validateGetReportsParams(params)
	if err == nil {
		t.Error("Expected error for invalid offset")
	}
}

func TestValidateGetReportsParams_InvalidAgeRange(t *testing.T) {
	handler := setupTestReportHandler()

	minAge := 50
	maxAge := 30
	params := &GetReportsRequestParams{
		Limit:  10,
		Offset: 0,
		MinAge: &minAge,
		MaxAge: &maxAge,
	}

	err := handler.validateGetReportsParams(params)
	if err == nil {
		t.Error("Expected error for invalid age range")
	}

	if !strings.Contains(err.Error(), "min_age") || !strings.Contains(err.Error(), "max_age") {
		t.Errorf("Expected error to mention age parameters, got: %v", err)
	}
}
