-- Rollback Migration 001: Drop schema_migrations table and indexes
-- Source: Story 1.2 Database Schema & Migration Setup
-- Rollback for 001_create_schema_migrations_table.sql

-- Drop the index first
DROP INDEX IF EXISTS idx_schema_migrations_executed_at;

-- Drop the schema_migrations table
DROP TABLE IF EXISTS schema_migrations;