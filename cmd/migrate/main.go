// Package main provides CLI for manual database migration management.
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"time"

	"github.com/chybatronik/goUserAPI/internal/config"
	"github.com/chybatronik/goUserAPI/internal/database"
)

func main() {
	var (
		action      = flag.String("action", "up", "Migration action: up, down, status, rollback-last")
		target      = flag.String("target", "", "Target version for down migration")
		migrationsDir = flag.String("dir", "./migrations", "Migrations directory path")
		help        = flag.Bool("help", false, "Show help information")
	)
	flag.Parse()

	if *help {
		showHelp()
		return
	}

	log.Printf("Migration CLI - Action: %s, Directory: %s", *action, *migrationsDir)

	// Initialize database connection
	appConfig, err := config.Load()
	if err != nil {
		log.Fatalf("FATAL: Failed to load configuration: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Create connection pool
	pool, err := database.NewConnectionPool(appConfig)
	if err != nil {
		log.Fatalf("FATAL: Failed to create database connection pool: %v", err)
	}
	defer pool.Close()

	// Validate database connection
	if err := database.ValidateConnection(ctx, pool); err != nil {
		log.Fatalf("FATAL: Database connection validation failed: %v", err)
	}

	log.Println("Database connection established successfully")

	// Create migration runner
	migrationRunner := database.NewMigrationRunner(pool, *migrationsDir)

	// Execute requested action
	switch *action {
	case "up":
		log.Println("Running pending migrations...")
		if err := migrationRunner.RunMigrations(ctx); err != nil {
			log.Fatalf("FATAL: Migration failed: %v", err)
		}
		log.Println("Migrations completed successfully")

	case "down":
		if *target == "" {
			log.Fatalf("ERROR: --target version required for down migration")
		}
		log.Printf("Rolling back migrations to version: %s", *target)
		if err := migrationRunner.RunDownMigrations(ctx, *target); err != nil {
			log.Fatalf("FATAL: Rollback failed: %v", err)
		}
		log.Printf("Rollback to version %s completed successfully", *target)

	case "rollback-last":
		log.Println("Rolling back last migration...")
		if err := migrationRunner.RollbackLastMigration(ctx); err != nil {
			log.Fatalf("FATAL: Rollback failed: %v", err)
		}
		log.Println("Last migration rolled back successfully")

	case "status":
		showMigrationStatus(ctx, migrationRunner)

	default:
		log.Fatalf("ERROR: Unknown action: %s", *action)
	}
}

func showHelp() {
	fmt.Println("Migration CLI for goUserAPI")
	fmt.Println("")
	fmt.Println("Usage:")
	fmt.Println("  go run cmd/migrate/main.go [flags]")
	fmt.Println("")
	fmt.Println("Flags:")
	fmt.Println("  -action string")
	fmt.Println("        Migration action: up, down, status, rollback-last (default \"up\")")
	fmt.Println("  -target string")
	fmt.Println("        Target version for down migration")
	fmt.Println("  -dir string")
	fmt.Println("        Migrations directory path (default \"./migrations\")")
	fmt.Println("  -help")
	fmt.Println("        Show help information")
	fmt.Println("")
	fmt.Println("Examples:")
	fmt.Println("  go run cmd/migrate/main.go -action=up")
	fmt.Println("  go run cmd/migrate/main.go -action=status")
	fmt.Println("  go run cmd/migrate/main.go -action=down -target=001_create_schema_migrations_table")
	fmt.Println("  go run cmd/migrate/main.go -action=rollback-last")
}

func showMigrationStatus(ctx context.Context, runner *database.MigrationRunner) {
	log.Println("Checking migration status...")

	// Get executed migrations
	executedMigrations, err := runner.GetExecutedMigrations(ctx)
	if err != nil {
		log.Fatalf("ERROR: Failed to get migration status: %v", err)
	}

	// Get all migration files
	migrationFiles, err := runner.LoadMigrationFiles()
	if err != nil {
		log.Fatalf("ERROR: Failed to load migration files: %v", err)
	}

	fmt.Println("\nMigration Status:")
	fmt.Println("================")

	if len(executedMigrations) == 0 {
		fmt.Println("No migrations have been executed yet.")
	} else {
		fmt.Printf("Executed migrations (%d):\n", len(executedMigrations))
		for version := range executedMigrations {
			fmt.Printf("  ✓ %s\n", version)
		}
	}

	pendingCount := 0
	for _, migration := range migrationFiles {
		if !executedMigrations[migration.Version] {
			pendingCount++
		}
	}

	if pendingCount > 0 {
		fmt.Printf("\nPending migrations (%d):\n", pendingCount)
		for _, migration := range migrationFiles {
			if !executedMigrations[migration.Version] {
				fmt.Printf("  ○ %s (%s)\n", migration.Version, migration.Filename)
			}
		}
	} else {
		fmt.Println("\nAll migrations are up to date!")
	}

	fmt.Println("================")
}