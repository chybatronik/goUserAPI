package database

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Migration represents a database migration
type Migration struct {
	Version    string
	Filename   string
	SQLContent string
	Checksum   string
}

// calculateChecksum computes SHA256 checksum of migration content
func calculateChecksum(content string) string {
	hash := sha256.Sum256([]byte(content))
	return hex.EncodeToString(hash[:])
}

// MigrationRunner executes database migrations
// Source: Architecture.md#Migration-Execution-Process
type MigrationRunner struct {
	db  *pgxpool.Pool
	dir string
}

// NewMigrationRunner creates a new migration runner
func NewMigrationRunner(db *pgxpool.Pool, migrationsDir string) *MigrationRunner {
	return &MigrationRunner{
		db:  db,
		dir: migrationsDir,
	}
}

// RunMigrations executes all pending migrations in order
// Source: Architecture.md#Migration-Execution-Process
func (m *MigrationRunner) RunMigrations(ctx context.Context) error {
	log.Printf("[MIGRATION] Starting migration execution from directory: %s", m.dir)
	startTime := time.Now()

	// Create migrations table if not exists
	if err := m.createMigrationsTable(ctx); err != nil {
		log.Printf("[MIGRATION] ERROR: Failed to create migrations table: %v", err)
		return fmt.Errorf("failed to create migrations table: %w", err)
	}
	log.Printf("[MIGRATION] Schema migrations table verified")

	// Load migration files
	migrations, err := m.loadMigrationFiles()
	if err != nil {
		log.Printf("[MIGRATION] ERROR: Failed to load migration files: %v", err)
		return fmt.Errorf("failed to load migration files: %w", err)
	}
	log.Printf("[MIGRATION] Loaded %d migration files", len(migrations))

	// Get executed migrations
	executedMigrations, err := m.getExecutedMigrations(ctx)
	if err != nil {
		log.Printf("[MIGRATION] ERROR: Failed to get executed migrations: %v", err)
		return fmt.Errorf("failed to get executed migrations: %w", err)
	}
	log.Printf("[MIGRATION] Found %d already executed migrations", len(executedMigrations))

	// Run pending migrations
	pendingCount := 0
	for _, migration := range migrations {
		if !m.isMigrationExecuted(migration.Version, executedMigrations) {
			pendingCount++
			log.Printf("[MIGRATION] Executing migration: %s (%s) [checksum: %s]", migration.Version, migration.Filename, migration.Checksum[:16]+"...")

			if err := m.executeMigration(ctx, migration); err != nil {
				log.Printf("[MIGRATION] ERROR: Failed to execute migration %s: %v", migration.Version, err)
				return fmt.Errorf("failed to execute migration %s: %w", migration.Version, err)
			}
			log.Printf("[MIGRATION] SUCCESS: Migration %s executed successfully [checksum: %s]", migration.Version, migration.Checksum[:16]+"...")
		}
	}

	// Verify integrity of all migrations (both executed and pending)
	if err := m.verifyMigrationIntegrity(ctx, migrations); err != nil {
		log.Printf("[MIGRATION] ERROR: Migration integrity verification failed: %v", err)
		return fmt.Errorf("migration integrity verification failed: %w", err)
	}

	duration := time.Since(startTime)
	log.Printf("[MIGRATION] COMPLETED: %d migrations executed in %v", pendingCount, duration)
	return nil
}

// createMigrationsTable creates the table to track migration execution
func (m *MigrationRunner) createMigrationsTable(ctx context.Context) error {
	// First create or upgrade the table using migration 001
	if err := m.executeMigrationSQL(ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version VARCHAR(255) PRIMARY KEY,
			executed_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
			checksum VARCHAR(64)
		)
	`); err != nil {
		return err
	}

	// Add index for faster migration version lookups
	if err := m.executeMigrationSQL(ctx, `
		CREATE INDEX IF NOT EXISTS idx_schema_migrations_executed_at
		ON schema_migrations(executed_at)
	`); err != nil {
		return err
	}

	// Add checksum column if table existed without it (backwards compatibility)
	if err := m.executeMigrationSQL(ctx, `
		ALTER TABLE schema_migrations
		ADD COLUMN IF NOT EXISTS checksum VARCHAR(64)
	`); err != nil {
		return err
	}

	// Update legacy rows with placeholder checksum
	if err := m.executeMigrationSQL(ctx, `
		UPDATE schema_migrations
		SET checksum = 'legacy_migration_no_checksum_available'
		WHERE checksum IS NULL
	`); err != nil {
		return err
	}

	return nil
}

// executeMigrationSQL executes a single SQL statement for table creation/modification
func (m *MigrationRunner) executeMigrationSQL(ctx context.Context, sql string) error {
	_, err := m.db.Exec(ctx, sql)
	return err
}

// LoadMigrationFiles loads migration files from the migrations directory (public method)
func (m *MigrationRunner) LoadMigrationFiles() ([]Migration, error) {
	return m.loadMigrationFiles()
}

// loadMigrationFiles loads migration files from the migrations directory
func (m *MigrationRunner) loadMigrationFiles() ([]Migration, error) {
	entries, err := os.ReadDir(m.dir)
	if err != nil {
		return nil, fmt.Errorf("failed to read migrations directory: %w", err)
	}

	var migrations []Migration
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sql") {
			continue
		}

		filename := entry.Name()

		// Skip down migrations - they should only be used for rollbacks
		if strings.Contains(filename, "_down_") {
			continue
		}

		version := strings.TrimSuffix(filename, ".sql")

		filePath := filepath.Join(m.dir, filename)
		content, err := os.ReadFile(filePath)
		if err != nil {
			return nil, fmt.Errorf("failed to read migration file %s: %w", filename, err)
		}

		contentStr := string(content)
		checksum := calculateChecksum(contentStr)

		migrations = append(migrations, Migration{
			Version:    version,
			Filename:   filename,
			SQLContent: contentStr,
			Checksum:   checksum,
		})
	}

	// Sort migrations by version (001_, 002_, etc.)
	sort.Slice(migrations, func(i, j int) bool {
		return migrations[i].Version < migrations[j].Version
	})

	return migrations, nil
}

// GetExecutedMigrations retrieves list of executed migrations from database (public method)
func (m *MigrationRunner) GetExecutedMigrations(ctx context.Context) (map[string]bool, error) {
	return m.getExecutedMigrations(ctx)
}

// getExecutedMigrations retrieves list of executed migrations from database
func (m *MigrationRunner) getExecutedMigrations(ctx context.Context) (map[string]bool, error) {
	rows, err := m.db.Query(ctx, "SELECT version FROM schema_migrations")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	executed := make(map[string]bool)
	for rows.Next() {
		var version string
		if err := rows.Scan(&version); err != nil {
			return nil, err
		}
		executed[version] = true
	}

	return executed, rows.Err()
}

// isMigrationExecuted checks if a migration has already been executed
func (m *MigrationRunner) isMigrationExecuted(version string, executed map[string]bool) bool {
	return executed[version]
}

// executeMigration executes a single migration
func (m *MigrationRunner) executeMigration(ctx context.Context, migration Migration) error {
	tx, err := m.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	// Execute migration SQL
	if _, err := tx.Exec(ctx, migration.SQLContent); err != nil {
		return fmt.Errorf("failed to execute migration SQL: %w", err)
	}

	// Record migration as executed with checksum
	if _, err := tx.Exec(ctx,
		"INSERT INTO schema_migrations (version, checksum) VALUES ($1, $2)",
		migration.Version, migration.Checksum); err != nil {
		return fmt.Errorf("failed to record migration: %w", err)
	}

	return tx.Commit(ctx)
}

// verifyMigrationIntegrity verifies that executed migrations match their checksums
func (m *MigrationRunner) verifyMigrationIntegrity(ctx context.Context, migrations []Migration) error {
	log.Printf("[MIGRATION] Verifying migration integrity...")

	// Get checksums from database for executed migrations
	type ExecutedMigration struct {
		Version  string
		Checksum string
	}

	rows, err := m.db.Query(ctx, "SELECT version, checksum FROM schema_migrations ORDER BY version")
	if err != nil {
		return fmt.Errorf("failed to query executed migrations: %w", err)
	}
	defer rows.Close()

	var executedMigrations []ExecutedMigration
	for rows.Next() {
		var em ExecutedMigration
		if err := rows.Scan(&em.Version, &em.Checksum); err != nil {
			return fmt.Errorf("failed to scan executed migration: %w", err)
		}
		executedMigrations = append(executedMigrations, em)
	}

	if err := rows.Err(); err != nil {
		return fmt.Errorf("error iterating executed migrations: %w", err)
	}

	// Verify each executed migration against current file checksum
	for _, executed := range executedMigrations {
		// Find corresponding migration in current migrations
		var currentMigration *Migration
		for _, mig := range migrations {
			if mig.Version == executed.Version {
				currentMigration = &mig
				break
			}
		}

		if currentMigration == nil {
			return fmt.Errorf("migration %s found in database but not in migrations directory", executed.Version)
		}

		// Verify checksum (skip legacy migrations without checksums)
		if executed.Checksum == "legacy_migration_no_checksum_available" {
			log.Printf("[MIGRATION] INFO: Migration %s is legacy migration (no original checksum)", executed.Version)
		} else if currentMigration.Checksum != executed.Checksum {
			log.Printf("[MIGRATION] WARNING: Migration %s checksum mismatch!", executed.Version)
			log.Printf("[MIGRATION]   Database: %s", executed.Checksum)
			log.Printf("[MIGRATION]   File:     %s", currentMigration.Checksum)
			return fmt.Errorf("migration %s has been modified after execution (checksum mismatch)", executed.Version)
		}
	}

	log.Printf("[MIGRATION] Migration integrity verified (%d migrations)", len(executedMigrations))
	return nil
}

// RollbackLastMigration rolls back the last executed migration
func (m *MigrationRunner) RollbackLastMigration(ctx context.Context) error {
	log.Printf("[MIGRATION] Starting rollback of last migration")

	// Get executed migrations in reverse order
	executedMigrations, err := m.getExecutedMigrations(ctx)
	if err != nil {
		log.Printf("[MIGRATION] ERROR: Failed to get executed migrations: %v", err)
		return fmt.Errorf("failed to get executed migrations: %w", err)
	}

	if len(executedMigrations) == 0 {
		log.Printf("[MIGRATION] No migrations to rollback")
		return nil
	}

	// Find the last executed migration
	var lastMigration string
	for version := range executedMigrations {
		if version > lastMigration {
			lastMigration = version
		}
	}

	log.Printf("[MIGRATION] Rolling back migration: %s", lastMigration)

	// Load down migration file - look for corresponding down migration
	var downFilename string
	switch lastMigration {
	case "001_create_schema_migrations_table":
		downFilename = "001_down_create_schema_migrations_table.sql"
	case "002_create_users_table":
		downFilename = "002_down_create_users_table.sql"
	case "003_create_indexes":
		downFilename = "003_down_create_indexes.sql"
	default:
		// For other migrations, try to find the corresponding down file
		files, err := os.ReadDir(m.dir)
		if err == nil {
			for _, file := range files {
				if strings.Contains(file.Name(), "_down_") && strings.Contains(file.Name(), lastMigration) {
					downFilename = file.Name()
					break
				}
			}
		}
		if downFilename == "" {
			downFilename = fmt.Sprintf("%s_down.sql", lastMigration)
		}
	}

	filePath := filepath.Join(m.dir, downFilename)
	content, err := os.ReadFile(filePath)
	if err != nil {
		log.Printf("[MIGRATION] ERROR: Failed to read rollback file %s: %v", downFilename, err)
		return fmt.Errorf("failed to read rollback file %s: %w", downFilename, err)
	}

	// Execute rollback in transaction
	tx, err := m.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// Execute rollback SQL
	if _, err := tx.Exec(ctx, string(content)); err != nil {
		log.Printf("[MIGRATION] ERROR: Failed to execute rollback SQL: %v", err)
		return fmt.Errorf("failed to execute rollback SQL: %w", err)
	}

	// Remove migration from tracking
	if _, err := tx.Exec(ctx, "DELETE FROM schema_migrations WHERE version = $1", lastMigration); err != nil {
		log.Printf("[MIGRATION] ERROR: Failed to remove migration from tracking: %v", err)
		return fmt.Errorf("failed to remove migration from tracking: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		log.Printf("[MIGRATION] ERROR: Failed to commit rollback transaction: %v", err)
		return fmt.Errorf("failed to commit rollback transaction: %w", err)
	}

	log.Printf("[MIGRATION] SUCCESS: Migration %s rolled back successfully", lastMigration)
	return nil
}

// RunDownMigrations rolls back all migrations to a specific version
func (m *MigrationRunner) RunDownMigrations(ctx context.Context, targetVersion string) error {
	log.Printf("[MIGRATION] Starting rollback to version: %s", targetVersion)
	startTime := time.Now()

	// Get executed migrations
	executedMigrations, err := m.getExecutedMigrations(ctx)
	if err != nil {
		log.Printf("[MIGRATION] ERROR: Failed to get executed migrations: %v", err)
		return fmt.Errorf("failed to get executed migrations: %w", err)
	}

	// Get migrations to rollback (in reverse order)
	migrationFiles, err := m.loadMigrationFiles()
	if err != nil {
		log.Printf("[MIGRATION] ERROR: Failed to load migration files: %v", err)
		return fmt.Errorf("failed to load migration files: %w", err)
	}

	// Sort migrations in reverse order for rollback
	sort.Slice(migrationFiles, func(i, j int) bool {
		return migrationFiles[i].Version > migrationFiles[j].Version
	})

	rollbackCount := 0
	for _, migration := range migrationFiles {
		if !strings.Contains(migration.Version, "_down_") {
			if executedMigrations[migration.Version] && migration.Version > targetVersion {
				log.Printf("[MIGRATION] Rolling back migration: %s", migration.Version)

				// Find corresponding down migration
				var downFilename string
				switch migration.Version {
				case "001_create_schema_migrations_table":
					downFilename = "001_down_create_schema_migrations_table.sql"
				case "002_create_users_table":
					downFilename = "002_down_create_users_table.sql"
				case "003_create_indexes":
					downFilename = "003_down_create_indexes.sql"
				default:
					// Try to find the corresponding down file
					files, err := os.ReadDir(m.dir)
					if err == nil {
						for _, file := range files {
							if strings.Contains(file.Name(), "_down_") && strings.Contains(file.Name(), migration.Version) {
								downFilename = file.Name()
								break
							}
						}
					}
					if downFilename == "" {
						downFilename = fmt.Sprintf("%s_down.sql", migration.Version)
					}
				}

				filePath := filepath.Join(m.dir, downFilename)
				content, err := os.ReadFile(filePath)
				if err != nil {
					log.Printf("[MIGRATION] ERROR: Failed to read rollback file %s: %v", downFilename, err)
					return fmt.Errorf("failed to read rollback file %s: %w", downFilename, err)
				}

				// Execute rollback
				tx, err := m.db.Begin(ctx)
				if err != nil {
					return fmt.Errorf("failed to begin transaction: %w", err)
				}
				defer tx.Rollback(ctx)

				if _, err := tx.Exec(ctx, string(content)); err != nil {
					log.Printf("[MIGRATION] ERROR: Failed to execute rollback SQL for %s: %v", migration.Version, err)
					return fmt.Errorf("failed to execute rollback SQL: %w", err)
				}

				if _, err := tx.Exec(ctx, "DELETE FROM schema_migrations WHERE version = $1", migration.Version); err != nil {
					log.Printf("[MIGRATION] ERROR: Failed to remove migration %s from tracking: %v", migration.Version, err)
					return fmt.Errorf("failed to remove migration from tracking: %w", err)
				}

				if err := tx.Commit(ctx); err != nil {
					log.Printf("[MIGRATION] ERROR: Failed to commit rollback transaction for %s: %v", migration.Version, err)
					return fmt.Errorf("failed to commit rollback transaction: %w", err)
				}

				rollbackCount++
				log.Printf("[MIGRATION] SUCCESS: Migration %s rolled back", migration.Version)
			}
		}
	}

	duration := time.Since(startTime)
	log.Printf("[MIGRATION] COMPLETED: Rolled back %d migrations to version %s in %v", rollbackCount, targetVersion, duration)
	return nil
}