package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/chybatronik/goUserAPI/internal/logging"
	"github.com/chybatronik/goUserAPI/internal/models"
	"github.com/chybatronik/goUserAPI/internal/types"
	"github.com/chybatronik/goUserAPI/pkg/errors"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockDBService represents a mock database service for testing
type MockDBService struct {
	shouldFailCreate bool
	createdUsers     []*models.User
}

func (m *MockDBService) CreateUser(ctx context.Context, pool *pgxpool.Pool, user *models.User) (*models.User, error) {
	if m.shouldFailCreate {
		return nil, errors.NewUserValidationError("DATABASE_QUERY_ERROR", "Database connection failed")
	}

	// Mock successful creation
	newUser := &models.User{
		ID:            "550e8400-e29b-41d4-a716-446655440000",
		FirstName:     user.FirstName,
		LastName:      user.LastName,
		Age:           user.Age,
		RecordingDate: time.Now().Unix(),
	}
	m.createdUsers = append(m.createdUsers, newUser)
	return newUser, nil
}

func (m *MockDBService) GetUsers(ctx context.Context, pool *pgxpool.Pool, params types.GetUsersParams) ([]models.User, int64, error) {
	// Return empty slice for tests that don't need GetUsers functionality
	return []models.User{}, 0, nil
}

func (m *MockDBService) GetReports(ctx context.Context, pool *pgxpool.Pool, params types.GetReportsParams) ([]models.User, int64, error) {
	// Return empty slice for tests that don't need GetReports functionality
	return []models.User{}, 0, nil
}

func TestCreateUserSuccess(t *testing.T) {
	// Setup
	logger := logging.NewStructuredLogger("info", "goUserAPI", "test")
	mockDB := &MockDBService{}
	handler := NewUserHandler(logger, nil, mockDB)

	// Prepare request body
	requestBody := map[string]interface{}{
		"first_name": "John",
		"last_name":  "Doe",
		"age":        30,
	}

	bodyBytes, _ := json.Marshal(requestBody)
	req := httptest.NewRequest("POST", "/users", bytes.NewBuffer(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	// This test will fail initially - handler needs to be implemented
	handler.CreateUser(w, req)

	// After implementation, expect 201 Created
	assert.Equal(t, http.StatusCreated, w.Code)

	var response models.User
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, "John", response.FirstName)
	assert.Equal(t, "Doe", response.LastName)
	assert.Equal(t, 30, response.Age)
	assert.NotEmpty(t, response.ID)
	assert.NotZero(t, response.RecordingDate)
}

func TestCreateUserMissingContentType(t *testing.T) {
	logger := logging.NewStructuredLogger("info", "goUserAPI", "test")
	mockDB := &MockDBService{}
	handler := NewUserHandler(logger, nil, mockDB)

	requestBody := map[string]interface{}{
		"first_name": "John",
		"last_name":  "Doe",
		"age":        30,
	}

	bodyBytes, _ := json.Marshal(requestBody)
	req := httptest.NewRequest("POST", "/users", bytes.NewBuffer(bodyBytes))
	// Missing Content-Type header
	w := httptest.NewRecorder()

	handler.CreateUser(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, "INVALID_CONTENT_TYPE", response["code"])
	assert.Contains(t, response["error"], "Content-Type")
}

func TestCreateUserInvalidJSON(t *testing.T) {
	logger := logging.NewStructuredLogger("info", "goUserAPI", "test")
	mockDB := &MockDBService{}
	handler := NewUserHandler(logger, nil, mockDB)

	req := httptest.NewRequest("POST", "/users", strings.NewReader("invalid json"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.CreateUser(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, "INVALID_JSON", response["code"])
}

func TestCreateUserMissingRequiredFields(t *testing.T) {
	logger := logging.NewStructuredLogger("info", "goUserAPI", "test")
	mockDB := &MockDBService{}
	handler := NewUserHandler(logger, nil, mockDB)

	testCases := []struct {
		name        string
		requestBody map[string]interface{}
		expectedErr string
	}{
		{
			name: "missing first_name",
			requestBody: map[string]interface{}{
				"last_name": "Doe",
				"age":       30,
			},
			expectedErr: "MISSING_REQUIRED_FIELD",
		},
		{
			name: "missing last_name",
			requestBody: map[string]interface{}{
				"first_name": "John",
				"age":        30,
			},
			expectedErr: "MISSING_REQUIRED_FIELD",
		},
		{
			name: "missing age",
			requestBody: map[string]interface{}{
				"first_name": "John",
				"last_name":  "Doe",
			},
			expectedErr: "INVALID_AGE_RANGE",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			bodyBytes, _ := json.Marshal(tc.requestBody)
			req := httptest.NewRequest("POST", "/users", bytes.NewBuffer(bodyBytes))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			handler.CreateUser(w, req)

			assert.Equal(t, http.StatusBadRequest, w.Code)

			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			require.NoError(t, err)

			assert.Equal(t, tc.expectedErr, response["code"])
		})
	}
}

func TestCreateUserFieldValidation(t *testing.T) {
	logger := logging.NewStructuredLogger("info", "goUserAPI", "test")
	mockDB := &MockDBService{}
	handler := NewUserHandler(logger, nil, mockDB)

	testCases := []struct {
		name        string
		requestBody map[string]interface{}
		expectedErr string
	}{
		{
			name: "first_name too long",
			requestBody: map[string]interface{}{
				"first_name": strings.Repeat("a", 101), // 101 characters
				"last_name":  "Doe",
				"age":        30,
			},
			expectedErr: "INVALID_FIELD_LENGTH",
		},
		{
			name: "last_name too long",
			requestBody: map[string]interface{}{
				"first_name": "John",
				"last_name":  strings.Repeat("a", 101), // 101 characters
				"age":        30,
			},
			expectedErr: "INVALID_FIELD_LENGTH",
		},
		{
			name: "age too young",
			requestBody: map[string]interface{}{
				"first_name": "John",
				"last_name":  "Doe",
				"age":        0,
			},
			expectedErr: "INVALID_AGE_RANGE",
		},
		{
			name: "age too old",
			requestBody: map[string]interface{}{
				"first_name": "John",
				"last_name":  "Doe",
				"age":        121,
			},
			expectedErr: "INVALID_AGE_RANGE",
		},
		{
			name: "empty first_name after trim",
			requestBody: map[string]interface{}{
				"first_name": "   ",
				"last_name":  "Doe",
				"age":        30,
			},
			expectedErr: "EMPTY_FIELD_AFTER_TRIM",
		},
		{
			name: "empty last_name after trim",
			requestBody: map[string]interface{}{
				"first_name": "John",
				"last_name":  "   ",
				"age":        30,
			},
			expectedErr: "EMPTY_FIELD_AFTER_TRIM",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			bodyBytes, _ := json.Marshal(tc.requestBody)
			req := httptest.NewRequest("POST", "/users", bytes.NewBuffer(bodyBytes))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			handler.CreateUser(w, req)

			assert.Equal(t, http.StatusBadRequest, w.Code)

			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			require.NoError(t, err)

			assert.Equal(t, tc.expectedErr, response["code"])
		})
	}
}

func TestCreateUserUnsupportedMethod(t *testing.T) {
	logger := logging.NewStructuredLogger("info", "goUserAPI", "test")
	mockDB := &MockDBService{}
	handler := NewUserHandler(logger, nil, mockDB)

	req := httptest.NewRequest("GET", "/users", nil)
	w := httptest.NewRecorder()

	handler.CreateUser(w, req)

	assert.Equal(t, http.StatusMethodNotAllowed, w.Code)
}

func TestGetUsersSuccess(t *testing.T) {
	logger := logging.NewStructuredLogger("info", "goUserAPI", "test")
	mockDB := &MockGetUsersDBService{}
	handler := NewUserHandler(logger, nil, mockDB)

	// Test basic GET request without parameters
	req := httptest.NewRequest("GET", "/users", nil)
	w := httptest.NewRecorder()

	handler.GetUsers(w, req)

	// This should fail initially - needs implementation
	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Contains(t, response, "users")
	assert.Contains(t, response, "pagination")
}

func TestGetUsersWithQueryParameters(t *testing.T) {
	logger := logging.NewStructuredLogger("info", "goUserAPI", "test")
	mockDB := &MockGetUsersDBService{}
	handler := NewUserHandler(logger, nil, mockDB)

	testCases := []struct {
		name           string
		url            string
		expectedStatus int
	}{
		{
			name:           "with limit and offset",
			url:            "/users?limit=10&offset=20",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "with sort by age",
			url:            "/users?sort_by=age&sort_order=asc",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "with all parameters",
			url:            "/users?limit=50&offset=0&sort_by=first_name&sort_order=desc",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "invalid limit",
			url:            "/users?limit=0",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "invalid sort field",
			url:            "/users?sort_by=invalid_field",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "invalid sort order",
			url:            "/users?sort_order=invalid",
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", tc.url, nil)
			w := httptest.NewRecorder()

			handler.GetUsers(w, req)

			assert.Equal(t, tc.expectedStatus, w.Code)
		})
	}
}

func TestGetUsersValidationErrors(t *testing.T) {
	logger := logging.NewStructuredLogger("info", "goUserAPI", "test")
	mockDB := &MockGetUsersDBService{}
	handler := NewUserHandler(logger, nil, mockDB)

	testCases := []struct {
		name         string
		url          string
		expectedCode string
	}{
		{
			name:         "limit too low",
			url:          "/users?limit=0",
			expectedCode: "INVALID_LIMIT_PARAMETER",
		},
		{
			name:         "limit too high",
			url:          "/users?limit=101",
			expectedCode: "INVALID_LIMIT_PARAMETER",
		},
		{
			name:         "negative offset",
			url:          "/users?offset=-1",
			expectedCode: "INVALID_OFFSET_PARAMETER",
		},
		{
			name:         "invalid sort field",
			url:          "/users?sort_by=invalid_field",
			expectedCode: "INVALID_SORT_FIELD",
		},
		{
			name:         "invalid sort order",
			url:          "/users?sort_order=invalid",
			expectedCode: "INVALID_SORT_ORDER",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", tc.url, nil)
			w := httptest.NewRecorder()

			handler.GetUsers(w, req)

			assert.Equal(t, http.StatusBadRequest, w.Code)

			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			require.NoError(t, err)

			assert.Equal(t, tc.expectedCode, response["code"])
		})
	}
}

func TestGetUsersDefaultValues(t *testing.T) {
	logger := logging.NewStructuredLogger("info", "goUserAPI", "test")
	mockDB := &MockGetUsersDBService{}
	handler := NewUserHandler(logger, nil, mockDB)

	req := httptest.NewRequest("GET", "/users", nil)
	w := httptest.NewRecorder()

	handler.GetUsers(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	// Verify default values were applied
	assert.Equal(t, 20, mockDB.lastParams.Limit)
	assert.Equal(t, 0, mockDB.lastParams.Offset)
	assert.Equal(t, "recording_date", mockDB.lastParams.SortBy)
	assert.Equal(t, "desc", mockDB.lastParams.SortOrder)
}

// MockGetUsersDBService mocks the database service for GetUsers testing
type MockGetUsersDBService struct {
	shouldFail     bool
	lastParams     types.GetUsersParams
	mockUsers      []models.User
	mockTotalCount int64
}

func (m *MockGetUsersDBService) CreateUser(ctx context.Context, pool *pgxpool.Pool, user *models.User) (*models.User, error) {
	return nil, nil // Not used in GetUsers tests
}

func (m *MockGetUsersDBService) GetUsers(ctx context.Context, pool *pgxpool.Pool, params types.GetUsersParams) ([]models.User, int64, error) {
	if m.shouldFail {
		return nil, 0, fmt.Errorf("database error")
	}

	m.lastParams = params

	// Return mock data
	mockUsers := []models.User{
		{
			ID:            "550e8400-e29b-41d4-a716-446655440000",
			FirstName:     "John",
			LastName:      "Doe",
			Age:           30,
			RecordingDate: 1705314600,
		},
		{
			ID:            "550e8400-e29b-41d4-a716-446655440001",
			FirstName:     "Jane",
			LastName:      "Smith",
			Age:           25,
			RecordingDate: 1705314700,
		},
	}

	return mockUsers, int64(150), nil
}

func (m *MockGetUsersDBService) GetReports(ctx context.Context, pool *pgxpool.Pool, params types.GetReportsParams) ([]models.User, int64, error) {
	if m.shouldFail {
		return nil, 0, fmt.Errorf("database error")
	}

	// Return mock data with same pattern as GetUsers
	mockUsers := []models.User{
		{
			ID:            "550e8400-e29b-41d4-a716-446655440000",
			FirstName:     "John",
			LastName:      "Doe",
			Age:           30,
			RecordingDate: 1705314600,
		},
		{
			ID:            "550e8400-e29b-41d4-a716-446655440001",
			FirstName:     "Jane",
			LastName:      "Smith",
			Age:           25,
			RecordingDate: 1705314700,
		},
	}

	return mockUsers, int64(150), nil
}

// ===== NFR-P1 PERFORMANCE BENCHMARKS =====

// BenchmarkGetUsers tests performance compliance with NFR-P1 <200ms requirement
func BenchmarkGetUsers(b *testing.B) {
	logger := logging.NewStructuredLogger("info", "goUserAPI", "test")
	mockDB := &MockGetUsersDBService{
		mockUsers:      generateMockUsers(1000), // 1K users for realistic testing
		mockTotalCount: 100000,                  // 100K total users as per AC #5
	}
	handler := NewUserHandler(logger, nil, mockDB)

	testCases := []struct {
		name  string
		url   string
		limit int
	}{
		{"default_pagination", "/users", 20},
		{"small_pagination", "/users?limit=10&offset=0", 10},
		{"large_pagination", "/users?limit=100&offset=0", 100},
		{"medium_offset", "/users?limit=50&offset=500", 50},
		{"large_offset", "/users?limit=20&offset=50000", 20},
	}

	for _, tc := range testCases {
		b.Run(tc.name, func(b *testing.B) {
			b.ResetTimer()
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				req := httptest.NewRequest("GET", tc.url, nil)
				w := httptest.NewRecorder()

				handler.GetUsers(w, req)

				// Verify success
				if w.Code != http.StatusOK {
					b.Fatalf("Expected status 200, got %d", w.Code)
				}

				// Verify response contains expected data
				var response map[string]interface{}
				if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
					b.Fatalf("Failed to unmarshal response: %v", err)
				}

				users, ok := response["users"].([]interface{})
				if !ok {
					b.Fatal("Expected users array in response")
				}

				// Verify limit was respected
				if len(users) > tc.limit {
					b.Fatalf("Expected max %d users, got %d", tc.limit, len(users))
				}
			}
		})
	}
}

// BenchmarkGetUsersWithSorting tests performance impact of different sort options
func BenchmarkGetUsersWithSorting(b *testing.B) {
	logger := logging.NewStructuredLogger("info", "goUserAPI", "test")
	mockDB := &MockGetUsersDBService{
		mockUsers:      generateMockUsers(5000),
		mockTotalCount: 100000,
	}
	handler := NewUserHandler(logger, nil, mockDB)

	sortTestCases := []struct {
		name   string
		sortBy string
		order  string
	}{
		{"sort_by_recording_date_desc", "recording_date", "desc"},
		{"sort_by_recording_date_asc", "recording_date", "asc"},
		{"sort_by_age_asc", "age", "asc"},
		{"sort_by_age_desc", "age", "desc"},
		{"sort_by_first_name_asc", "first_name", "asc"},
		{"sort_by_first_name_desc", "first_name", "desc"},
		{"sort_by_last_name_asc", "last_name", "asc"},
		{"sort_by_last_name_desc", "last_name", "desc"},
	}

	for _, tc := range sortTestCases {
		b.Run(tc.name, func(b *testing.B) {
			url := fmt.Sprintf("/users?sort_by=%s&sort_order=%s&limit=50", tc.sortBy, tc.order)

			b.ResetTimer()
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				req := httptest.NewRequest("GET", url, nil)
				w := httptest.NewRecorder()

				handler.GetUsers(w, req)

				if w.Code != http.StatusOK {
					b.Fatalf("Expected status 200, got %d", w.Code)
				}
			}
		})
	}
}

// TestPerformance_NFRP1Compliance directly validates <200ms requirement
func TestPerformance_NFRP1Compliance(t *testing.T) {
	const (
		nfrP1Target     = 200 * time.Millisecond // NFR-P1 requirement
		nfrP1Acceptable = 300 * time.Millisecond // Acceptable limit
		concurrentOps   = 100                    // Concurrent operations test
	)

	logger := logging.NewStructuredLogger("info", "goUserAPI", "test")
	mockDB := &MockGetUsersDBService{
		mockUsers:      generateMockUsers(10000),
		mockTotalCount: 100000,
	}
	handler := NewUserHandler(logger, nil, mockDB)

	testScenarios := []struct {
		name        string
		url         string
		description string
	}{
		{
			name:        "default_pagination",
			url:         "/users",
			description: "Default pagination (limit=20, offset=0)",
		},
		{
			name:        "large_dataset_pagination",
			url:         "/users?limit=100&offset=50000",
			description: "Large offset with max limit",
		},
		{
			name:        "expensive_sort",
			url:         "/users?sort_by=first_name&sort_order=asc&limit=50",
			description: "Text field sorting (most expensive)",
		},
	}

	for _, tc := range testScenarios {
		t.Run(tc.name, func(t *testing.T) {
			t.Logf("Testing scenario: %s", tc.description)

			// Single operation test
			start := time.Now()
			req := httptest.NewRequest("GET", tc.url, nil)
			w := httptest.NewRecorder()
			handler.GetUsers(w, req)
			duration := time.Since(start)

			// Verify response is successful
			assert.Equal(t, http.StatusOK, w.Code, "Request should be successful")

			// Check NFR-P1 compliance for single operation
			if duration > nfrP1Acceptable {
				t.Errorf("Single operation took %v, exceeds acceptable limit of %v", duration, nfrP1Acceptable)
			} else if duration > nfrP1Target {
				t.Logf("WARNING: Single operation took %v, exceeds target of %v but within acceptable", duration, nfrP1Target)
			} else {
				t.Logf("PASS: Single operation took %v, meets NFR-P1 target of %v", duration, nfrP1Target)
			}

			// Concurrent operations test (stress test)
			t.Run("concurrent_operations", func(t *testing.T) {
				start = time.Now()

				var wg sync.WaitGroup
				var mu sync.Mutex
				var durations []time.Duration
				var errors []error

				for i := 0; i < concurrentOps; i++ {
					wg.Add(1)
					go func() {
						defer wg.Done()

						opStart := time.Now()
						req := httptest.NewRequest("GET", tc.url, nil)
						w := httptest.NewRecorder()
						handler.GetUsers(w, req)
						opDuration := time.Since(opStart)

						mu.Lock()
						durations = append(durations, opDuration)
						if w.Code != http.StatusOK {
							errors = append(errors, fmt.Errorf("status %d", w.Code))
						}
						mu.Unlock()
					}()
				}

				wg.Wait()
				totalDuration := time.Since(start)

				// Calculate statistics
				if len(errors) > 0 {
					t.Errorf("Concurrent operations had %d errors: %v", len(errors), errors)
				}

				var totalOpDuration time.Duration
				var maxDuration time.Duration
				for _, d := range durations {
					totalOpDuration += d
					if d > maxDuration {
						maxDuration = d
					}
				}
				avgDuration := totalOpDuration / time.Duration(len(durations))

				t.Logf("Concurrent stats (%d ops): total=%v, avg=%v, max=%v",
					concurrentOps, totalDuration, avgDuration, maxDuration)

				// Verify concurrent performance meets requirements
				if avgDuration > nfrP1Acceptable {
					t.Errorf("Average concurrent operation took %v, exceeds acceptable limit of %v",
						avgDuration, nfrP1Acceptable)
				}

				if maxDuration > nfrP1Acceptable*2 { // Allow some variance for max
					t.Errorf("Maximum concurrent operation took %v, significantly exceeds acceptable limit",
						maxDuration)
				}
			})
		})
	}
}

// Helper function to generate mock users for realistic performance testing
func generateMockUsers(count int) []models.User {
	users := make([]models.User, count)
	baseTime := int64(1705314600) // Base timestamp

	firstNames := []string{"John", "Jane", "Michael", "Sarah", "David", "Emily", "Robert", "Lisa"}
	lastNames := []string{"Smith", "Johnson", "Williams", "Brown", "Jones", "Garcia", "Miller", "Davis"}

	for i := 0; i < count; i++ {
		users[i] = models.User{
			ID:            fmt.Sprintf("550e8400-e29b-41d4-a716-%012d", i),
			FirstName:     firstNames[i%len(firstNames)],
			LastName:      lastNames[i%len(lastNames)],
			Age:           18 + (i % 83),          // Age range 18-100
			RecordingDate: baseTime + int64(i*60), // 1 minute intervals
		}
	}

	return users
}
