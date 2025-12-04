package database

import (
	"os"
	"testing"

	"github.com/chybatronik/goUserAPI/internal/config"
	"github.com/stretchr/testify/assert"
)

func TestLoadMigrationFiles(t *testing.T) {
	// Test that migration files can be loaded correctly
	migrationRunner := &MigrationRunner{
		dir: "../../migrations",
	}

	migrations, err := migrationRunner.loadMigrationFiles()
	assert.NoError(t, err, "Should load migration files without error")
	assert.Len(t, migrations, 3, "Should load exactly 3 migration files (up migrations only)")

	// All loaded migrations are up migrations (down migrations are filtered out in loadMigrationFiles)
	assert.Len(t, migrations, 3, "Should have 3 up migrations")
	assert.Equal(t, "001_create_schema_migrations_table", migrations[0].Version, "First migration should create schema_migrations table")
	assert.Equal(t, "002_create_users_table", migrations[1].Version, "Second migration should be 002_create_users_table")
	assert.Equal(t, "003_create_indexes", migrations[2].Version, "Third migration should be 003_create_indexes")

	// Verify schema_migrations table creation
	assert.Contains(t, migrations[0].SQLContent, "CREATE TABLE", "First migration should create schema_migrations table")
	assert.Contains(t, migrations[0].SQLContent, "schema_migrations", "First migration should create schema_migrations table")

	// Verify users table creation
	assert.Contains(t, migrations[1].SQLContent, "CREATE TABLE users", "Second migration should contain CREATE TABLE users")
	assert.Contains(t, migrations[1].SQLContent, "gen_random_uuid", "Second migration should contain UUID generation")
	assert.Contains(t, migrations[1].SQLContent, "age >= 1 AND age <= 120", "Second migration should contain age constraint")

	// Verify indexes creation
	assert.Contains(t, migrations[2].SQLContent, "CREATE INDEX idx_users_age_created", "Third migration should create age index")
	assert.Contains(t, migrations[2].SQLContent, "CREATE INDEX idx_users_recording_date_desc", "Third migration should create recording_date index")

	// Note: Down migrations are not loaded by loadMigrationFiles() - they are only used for rollbacks
	// This is intentional design to keep migration execution simple and safe
}

func TestConfigFromEnv(t *testing.T) {
	// Set required environment variables for testing
	testVars := map[string]string{
		"DB_HOST":     "localhost",
		"DB_USER":     "postgres",
		"DB_PASSWORD": "postgres",
		"DB_NAME":     "postgres",
	}

	// Save original values and set test values
	originalVars := make(map[string]string)
	for key, value := range testVars {
		if original := os.Getenv(key); original != "" {
			originalVars[key] = original
		}
		os.Setenv(key, value)
	}

	// Restore environment variables after test
	defer func() {
		for key, original := range originalVars {
			if original != "" {
				os.Setenv(key, original)
			} else {
				os.Unsetenv(key)
			}
		}
	}()

	appConfig, err := config.Load()
	assert.NoError(t, err, "Should load config without error")
	assert.NotNil(t, appConfig, "Config should not be nil")

	dbConfig := appConfig.Database
	assert.Equal(t, "localhost", dbConfig.Host, "Default host should be localhost")
	assert.Equal(t, 5432, dbConfig.Port, "Default port should be 5432")
	assert.Equal(t, "postgres", dbConfig.User, "Default user should be postgres")
	assert.Equal(t, "postgres", dbConfig.Password, "Default password should be postgres")
	assert.Equal(t, "postgres", dbConfig.Database, "Default database should be postgres")
	assert.Equal(t, "disable", dbConfig.SSLMode, "Default SSL mode should be disable")
}

func TestDatabaseConfigValidation(t *testing.T) {
	// Set required environment variables for testing
	testVars := map[string]string{
		"DB_HOST":     "localhost",
		"DB_USER":     "postgres",
		"DB_PASSWORD": "postgres",
		"DB_NAME":     "postgres",
	}

	// Save original values and set test values
	originalVars := make(map[string]string)
	for key, value := range testVars {
		if original := os.Getenv(key); original != "" {
			originalVars[key] = original
		}
		os.Setenv(key, value)
	}

	// Restore environment variables after test
	defer func() {
		for key, original := range originalVars {
			if original != "" {
				os.Setenv(key, original)
			} else {
				os.Unsetenv(key)
			}
		}
	}()

	appConfig, err := config.Load()
	assert.NoError(t, err, "Should load config without error")

	dbConfig := appConfig.Database
	assert.Greater(t, dbConfig.MaxConns, 0, "Max connections should be greater than 0")
	assert.GreaterOrEqual(t, dbConfig.MinConns, 0, "Min connections should be non-negative")
	assert.LessOrEqual(t, dbConfig.MinConns, dbConfig.MaxConns, "Min connections should not exceed max connections")
}
