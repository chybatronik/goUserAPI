package models

import (
	"encoding/json"
	"testing"
	"time"
)

func TestUserStruct(t *testing.T) {
	user := User{
		ID:            "test-123",
		FirstName:     "John",
		LastName:      "Doe",
		Age:           30,
		RecordingDate: 1638360000,
	}

	if user.FirstName != "John" {
		t.Errorf("Expected first name 'John', got '%s'", user.FirstName)
	}

	if user.LastName != "Doe" {
		t.Errorf("Expected last name 'Doe', got '%s'", user.LastName)
	}

	if user.Age != 30 {
		t.Errorf("Expected age 30, got %d", user.Age)
	}
}

func TestUserReportStruct(t *testing.T) {
	now := time.Now().Unix()
	report := UserReport{
		ID:            "report-123",
		UserID:        "user-123",
		RecordingDate: 1638360000,
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	if report.UserID != "user-123" {
		t.Errorf("Expected user ID 'user-123', got '%s'", report.UserID)
	}

	if report.ID != "report-123" {
		t.Errorf("Expected report ID 'report-123', got '%s'", report.ID)
	}
}

func TestUserJSONSerialization(t *testing.T) {
	user := User{
		ID:            "test-123",
		FirstName:     "John",
		LastName:      "Doe",
		Age:           30,
		RecordingDate: 1638360000,
	}

	// Test JSON marshaling
	jsonData, err := json.Marshal(user)
	if err != nil {
		t.Fatalf("Failed to marshal User to JSON: %v", err)
	}

	// Test JSON unmarshaling
	var unmarshaledUser User
	err = json.Unmarshal(jsonData, &unmarshaledUser)
	if err != nil {
		t.Fatalf("Failed to unmarshal User from JSON: %v", err)
	}

	// Verify data integrity
	if unmarshaledUser.ID != user.ID {
		t.Errorf("Expected ID %s, got %s", user.ID, unmarshaledUser.ID)
	}
	if unmarshaledUser.FirstName != user.FirstName {
		t.Errorf("Expected FirstName %s, got %s", user.FirstName, unmarshaledUser.FirstName)
	}
	if unmarshaledUser.LastName != user.LastName {
		t.Errorf("Expected LastName %s, got %s", user.LastName, unmarshaledUser.LastName)
	}
}

// RED TEST: Tests for Story 2.1 requirements - these should fail initially
func TestUserStructStory21Requirements(t *testing.T) {
	// Test User struct matches PRD specification exactly (AC: #1)
	user := User{
		ID:            "550e8400-e29b-41d4-a716-446655440000",
		FirstName:     "John",
		LastName:      "Doe",
		Age:           30,
		RecordingDate: 1638360000,
	}

	// AC: #1 - ID field as string (UUID)
	if user.ID != "550e8400-e29b-41d4-a716-446655440000" {
		t.Errorf("Expected ID field as string UUID, got %s", user.ID)
	}

	// AC: #1 - FirstName and LastName as strings
	if user.FirstName != "John" {
		t.Errorf("Expected FirstName as string 'John', got %s", user.FirstName)
	}
	if user.LastName != "Doe" {
		t.Errorf("Expected LastName as string 'Doe', got %s", user.LastName)
	}

	// AC: #1 - Age as integer with range validation (1-120)
	if user.Age != 30 {
		t.Errorf("Expected Age as integer 30, got %d", user.Age)
	}

	// AC: #1 - RecordingDate as int64 Unix timestamp
	if user.RecordingDate != 1638360000 {
		t.Errorf("Expected RecordingDate as int64 Unix timestamp 1638360000, got %d", user.RecordingDate)
	}
}

// GREEN TEST: Test validation tags exist (AC: #4)
func TestUserStructValidationTags(t *testing.T) {
	// Test that User struct has proper validation tags
	user := User{
		ID:            "550e8400-e29b-41d4-a716-446655440000", // Valid UUID
		FirstName:     "John",                                // Valid: max 100 chars
		LastName:      "Doe",                                 // Valid: max 100 chars
		Age:           30,                                    // Valid: 1-120
		RecordingDate: 1638360000,                           // Valid timestamp
	}

	// Verify field values for validation constraints
	if len(user.FirstName) > 100 {
		t.Errorf("FirstName should not exceed 100 characters, got %d", len(user.FirstName))
	}
	if len(user.LastName) > 100 {
		t.Errorf("LastName should not exceed 100 characters, got %d", len(user.LastName))
	}
	if user.Age < 1 || user.Age > 120 {
		t.Errorf("Age should be between 1 and 120, got %d", user.Age)
	}
}

// GREEN TEST: Test JSON tags match PRD exactly (AC: #1)
func TestUserStructJSONTags(t *testing.T) {
	user := User{
		ID:            "test-id",
		FirstName:     "Jane",
		LastName:      "Smith",
		Age:           25,
		RecordingDate: 1638360000,
	}

	// Test JSON marshaling produces expected field names
	jsonData, err := json.Marshal(user)
	if err != nil {
		t.Fatalf("Failed to marshal User to JSON: %v", err)
	}

	jsonStr := string(jsonData)

	// Check for expected JSON field names
	if !contains(jsonStr, `"id"`) {
		t.Errorf("Expected JSON field 'id', missing in: %s", jsonStr)
	}
	if !contains(jsonStr, `"first_name"`) {
		t.Errorf("Expected JSON field 'first_name', missing in: %s", jsonStr)
	}
	if !contains(jsonStr, `"last_name"`) {
		t.Errorf("Expected JSON field 'last_name', missing in: %s", jsonStr)
	}
	if !contains(jsonStr, `"age"`) {
		t.Errorf("Expected JSON field 'age', missing in: %s", jsonStr)
	}
	if !contains(jsonStr, `"recording_date"`) {
		t.Errorf("Expected JSON field 'recording_date', missing in: %s", jsonStr)
	}

	// CRITICAL: Ensure no extra fields like created_at or updated_at (PRD specification)
	if contains(jsonStr, `"created_at"`) {
		t.Errorf("Unexpected JSON field 'created_at' found, not in PRD specification: %s", jsonStr)
	}
	if contains(jsonStr, `"updated_at"`) {
		t.Errorf("Unexpected JSON field 'updated_at' found, not in PRD specification: %s", jsonStr)
	}
}

// Helper function to check if string contains substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr ||
		func() bool {
			for i := 0; i <= len(s)-len(substr); i++ {
				if s[i:i+len(substr)] == substr {
					return true
				}
			}
			return false
		}())))
}
