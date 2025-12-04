-- Migration 001: Create schema_migrations table for tracking migration versions
-- Source: Story 1.2 Database Schema & Migration Setup
-- Architecture.md#Migration-Execution-Process sections

-- Create the schema_migrations table to track migration versions
CREATE TABLE IF NOT EXISTS schema_migrations (
    version VARCHAR(255) PRIMARY KEY,
    executed_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    checksum VARCHAR(64)
);

-- Add index for faster migration version lookups
CREATE INDEX IF NOT EXISTS idx_schema_migrations_executed_at ON schema_migrations(executed_at);