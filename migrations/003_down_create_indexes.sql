-- Rollback Migration 003: Drop indexes created for user table
-- Source: Story 1.2 Database Schema & Migration Setup
-- Rollback for 003_create_indexes.sql

DROP INDEX IF EXISTS idx_users_age_created;
DROP INDEX IF EXISTS idx_users_recording_date_desc;