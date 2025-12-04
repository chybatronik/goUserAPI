-- Migration 001: Create users table with exact PRD structure
-- Source: Story 1.2 Database Schema & Migration Setup
-- Architecture.md sections 190-210

-- Enable pgcrypto extension for UUID generation
CREATE EXTENSION IF NOT EXISTS pgcrypto;

-- Create users table with exact structure from PRD and Architecture
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),  -- Auto-generated UUID [Source: Architecture.md#Index-Strategy]
    first_name VARCHAR(100) NOT NULL,                 -- Max 100 chars, required [Source: PRD.md#FR13]
    last_name VARCHAR(100) NOT NULL,                  -- Max 100 chars, required [Source: PRD.md#FR13]
    age INTEGER CHECK (age >= 1 AND age <= 120),      -- Age constraint 1-120 [Source: Architecture.md#Input-Validation]
    recording_date BIGINT DEFAULT EXTRACT(EPOCH FROM NOW())::BIGINT  -- Unix timestamp [Source: PRD.md#FR3]
);

-- Add indexes for this migration will be in 002_create_indexes.sql