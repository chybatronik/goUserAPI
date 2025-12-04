-- Rollback Migration 002: Drop users table and pgcrypto extension
-- Source: Story 1.2 Database Schema & Migration Setup
-- Rollback for 002_create_users_table.sql

-- Drop the users table
DROP TABLE IF EXISTS users;

-- Note: We don't drop the pgcrypto extension as other migrations might depend on it
-- In production, extension management should be done carefully