#!/bin/bash

# Database Backup Script for PostgreSQL
# =====================================

set -euo pipefail

# Configuration
DB_HOST="${DB_HOST:-postgres}"
DB_PORT="${DB_PORT:-5432}"
DB_USER="${DB_USER:-postgres}"
DB_PASSWORD="${DB_PASSWORD:-}"
DB_NAME="${DB_NAME:-postgres}"

BACKUP_DIR="${BACKUP_DIR:-/backups}"
TIMESTAMP=$(date +"%Y%m%d_%H%M%S")
BACKUP_FILE="${DB_NAME}_backup_${TIMESTAMP}.sql"
COMPRESSED_FILE="${BACKUP_FILE}.gz"

# Ensure backup directory exists
mkdir -p "$BACKUP_DIR"

# Log function
log() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] $1"
}

# Check prerequisites
check_prerequisites() {
    log "Checking prerequisites..."

    if ! command -v pg_dump &> /dev/null; then
        log "ERROR: pg_dump is not available"
        exit 1
    fi

    if [[ -z "$DB_PASSWORD" ]]; then
        log "ERROR: DB_PASSWORD environment variable is required"
        exit 1
    fi
}

# Test database connection
test_connection() {
    log "Testing database connection..."

    PGPASSWORD="$DB_PASSWORD" pg_isready -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" || {
        log "ERROR: Cannot connect to database"
        exit 1
    }

    log "Database connection successful"
}

# Perform backup
perform_backup() {
    log "Starting backup of database: $DB_NAME"

    # Export password for pg_dump
    export PGPASSWORD="$DB_PASSWORD"

    # Perform backup
    if pg_dump -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" \
        --no-password \
        --verbose \
        --clean \
        --if-exists \
        --create \
        --format=custom \
        --file="$BACKUP_DIR/$BACKUP_FILE"; then

        log "Backup completed successfully: $BACKUP_DIR/$BACKUP_FILE"
    else
        log "ERROR: Backup failed"
        exit 1
    fi

    # Compress backup
    if gzip "$BACKUP_DIR/$BACKUP_FILE"; then
        log "Backup compressed: $BACKUP_DIR/$COMPRESSED_FILE"
    else
        log "WARNING: Compression failed, keeping uncompressed backup"
        mv "$BACKUP_DIR/$BACKUP_FILE" "$BACKUP_DIR/${BACKUP_FILE}.uncompressed"
    fi
}

# Cleanup old backups
cleanup_old_backups() {
    local retention_days="${BACKUP_RETENTION_DAYS:-7}"

    log "Cleaning up backups older than $retention_days days..."

    find "$BACKUP_DIR" -name "${DB_NAME}_backup_*.sql.gz" -type f -mtime "+$retention_days" -delete || \
    find "$BACKUP_DIR" -name "${DB_NAME}_backup_*.sql.uncompressed" -type f -mtime "+$retention_days" -delete || \
        log "WARNING: Cleanup failed or no old backups found"

    log "Cleanup completed"
}

# Verify backup
verify_backup() {
    log "Verifying backup file..."

    local backup_file="$BACKUP_DIR/$COMPRESSED_FILE"

    if [[ -f "$backup_file" ]]; then
        local file_size=$(stat -f%z "$backup_file" 2>/dev/null || stat -c%s "$backup_file" 2>/dev/null || echo "unknown")
        log "Backup verified. File size: $file_size bytes"
        return 0
    else
        log "ERROR: Backup file not found"
        return 1
    fi
}

# Main execution
main() {
    log "Starting database backup process..."

    check_prerequisites
    test_connection
    perform_backup
    cleanup_old_backups

    if verify_backup; then
        log "Backup process completed successfully"
        exit 0
    else
        log "Backup process completed with warnings"
        exit 1
    fi
}

# Handle script arguments
case "${1:-}" in
    --test-connection)
        test_connection
        ;;
    --cleanup)
        cleanup_old_backups
        ;;
    *)
        main
        ;;
esac