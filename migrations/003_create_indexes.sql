-- Migration 002: Create optimized indexes for query performance
-- Source: Story 1.2 Database Schema & Migration Setup
-- Architecture.md#Index-Strategy sections 175-176

-- Composite index for reports with age filtering (supports Epic 3)
-- Supports queries: WHERE age BETWEEN X AND Y ORDER BY recording_date DESC
CREATE INDEX idx_users_age_created ON users(age, recording_date);

-- DESC index for user listing (supports GET /users default sorting)
-- Optimizes queries: ORDER BY recording_date DESC LIMIT N
CREATE INDEX idx_users_recording_date_desc ON users(recording_date DESC);

-- Primary key index is automatically created by PostgreSQL
-- These indexes support the main query patterns from the PRD:
-- 1. GET /users with pagination (recording_date DESC)
-- 2. GET /reports with age filtering (age, recording_date composite)