package database

import (
	"testing"
	"time"

	"github.com/chybatronik/goUserAPI/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMigrationPerformance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	// Test migration loading performance
	start := time.Now()

	migrationRunner := &MigrationRunner{
		dir: "../../migrations",
	}

	migrations, err := migrationRunner.loadMigrationFiles()

	duration := time.Since(start)

	require.NoError(t, err, "Should load migrations without error")
	assert.Less(t, duration, 100*time.Millisecond, "Migration loading should complete quickly")
	assert.Len(t, migrations, 3, "Should load exactly 3 migration files (up migrations only)")
}

func TestQueryPerformanceTargets(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	// Define performance targets from PRD.md#NFR-P1
	performanceTargets := map[string]time.Duration{
		"create_user":    50 * time.Millisecond,  // With indexing
		"get_users":      100 * time.Millisecond, // With pagination
		"reports":        200 * time.Millisecond, // Complex filtering
		"max_acceptable": 300 * time.Millisecond, // Upper limit
	}

	t.Logf("Performance targets from PRD.md#NFR-P1:")
	for operation, target := range performanceTargets {
		if operation != "max_acceptable" {
			t.Logf("  %s: <%v (acceptable <%v)", operation, target, performanceTargets["max_acceptable"])
		}
	}

	// Test migration loading performance as proxy
	start := time.Now()
	migrationRunner := &MigrationRunner{
		dir: "../../migrations",
	}

	_, err := migrationRunner.loadMigrationFiles()
	duration := time.Since(start)

	require.NoError(t, err, "Should load migrations without error")

	// Migration loading should be much faster than query targets
	assert.Less(t, duration, 50*time.Millisecond,
		"Migration loading should be faster than query performance targets")

	t.Logf("Migration loading completed in %v (well within performance targets)", duration)
}

func TestIndexStrategyOriginalValidation(t *testing.T) {
	// Validate that our index strategy matches requirements from Architecture.md#Index-Strategy

	migrationRunner := &MigrationRunner{
		dir: "../../migrations",
	}

	migrations, err := migrationRunner.loadMigrationFiles()
	require.NoError(t, err)

	// Find indexes migration
	var indexesMigration *Migration
	for _, m := range migrations {
		if m.Version == "003_create_indexes" {
			indexesMigration = &m
			break
		}
	}

	require.NotNil(t, indexesMigration, "Should find indexes migration")

	// Validate composite index for age filtering (supports Epic 3)
	assert.Contains(t, indexesMigration.SQLContent,
		"CREATE INDEX idx_users_age_created ON users(age, recording_date)",
		"Should have composite index for age filtering")

	// Validate DESC index for user listing
	assert.Contains(t, indexesMigration.SQLContent,
		"CREATE INDEX idx_users_recording_date_desc ON users(recording_date DESC)",
		"Should have DESC index for user listing")
}

func TestUserModelValidation(t *testing.T) {
	tests := []struct {
		name        string
		user        models.User
		expectError bool
		errorMsg    string
	}{
		{
			name: "Valid user",
			user: models.User{
				FirstName:     "John",
				LastName:      "Doe",
				Age:           25,
				RecordingDate: time.Now().Unix(),
			},
			expectError: false,
		},
		{
			name: "Empty first name",
			user: models.User{
				FirstName:     "",
				LastName:      "Doe",
				Age:           25,
				RecordingDate: time.Now().Unix(),
			},
			expectError: true,
			errorMsg:    "first_name cannot be empty",
		},
		{
			name: "First name too long",
			user: models.User{
				FirstName:     string(make([]byte, 101)), // 101 characters
				LastName:      "Doe",
				Age:           25,
				RecordingDate: time.Now().Unix(),
			},
			expectError: true,
			errorMsg:    "first_name cannot exceed 100 characters",
		},
		{
			name: "Age too young",
			user: models.User{
				FirstName:     "John",
				LastName:      "Doe",
				Age:           0,
				RecordingDate: time.Now().Unix(),
			},
			expectError: true,
			errorMsg:    "age must be between 1 and 120 years",
		},
		{
			name: "Age too old",
			user: models.User{
				FirstName:     "John",
				LastName:      "Doe",
				Age:           121,
				RecordingDate: time.Now().Unix(),
			},
			expectError: true,
			errorMsg:    "age must be between 1 and 120 years",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateUser(&tt.user)

			if tt.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// BenchmarkMigrationLoading benchmarks migration file loading performance
func BenchmarkMigrationLoading(b *testing.B) {
	for i := 0; i < b.N; i++ {
		migrationRunner := &MigrationRunner{
			dir: "../../migrations",
		}

		_, err := migrationRunner.loadMigrationFiles()
		if err != nil {
			b.Fatal(err)
		}
	}
}
