#!/bin/bash

# Production Deployment Script for Go User API
# =============================================

set -euo pipefail

# Configuration
PROJECT_NAME="${PROJECT_NAME:-goUserAPI}"
ENVIRONMENT="${ENVIRONMENT:-production}"
VERSION="${VERSION:-latest}"
COMPOSE_FILE="docker-compose.yml"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Log functions
log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Check prerequisites
check_prerequisites() {
    log_info "Checking deployment prerequisites..."

    # Check Docker
    if ! command -v docker &> /dev/null; then
        log_error "Docker is not installed or not in PATH"
        exit 1
    fi

    # Check Docker Compose
    if ! command -v docker-compose &> /dev/null && ! docker compose version &> /dev/null; then
        log_error "Docker Compose is not installed or not in PATH"
        exit 1
    fi

    # Check environment file
    local env_file=".env.${ENVIRONMENT}"
    if [[ ! -f "$env_file" ]]; then
        log_error "Environment file $env_file not found"
        exit 1
    fi

    # Check required files
    local required_files=("Dockerfile" "docker-compose.yml" ".env.example")
    for file in "${required_files[@]}"; do
        if [[ ! -f "$file" ]]; then
            log_error "Required file $file not found"
            exit 1
        fi
    done

    log_info "Prerequisites check passed"
}

# Validate environment configuration
validate_environment() {
    log_info "Validating environment configuration..."

    # Copy environment file if .env doesn't exist
    if [[ ! -f ".env" ]]; then
        log_warn ".env file not found, copying from .env.${ENVIRONMENT}"
        cp ".env.${ENVIRONMENT}" .env
    fi

    # Source environment file
    if [[ -f ".env" ]]; then
        set -a
        source .env
        set +a
    fi

    # Check required environment variables
    local required_vars=("DB_PASSWORD")
    for var in "${required_vars[@]}"; do
        if [[ -z "${!var:-}" ]]; then
            log_error "Required environment variable $var is not set"
            exit 1
        fi
    done

    log_info "Environment configuration validated"
}

# Build and deploy services
deploy_services() {
    log_info "Starting deployment of $PROJECT_NAME ($ENVIRONMENT)..."

    # Pull latest images
    log_info "Pulling latest images..."
    docker-compose -f "$COMPOSE_FILE" pull || log_warn "Some images could not be pulled (may be local builds)"

    # Build custom images
    log_info "Building application image..."
    docker-compose -f "$COMPOSE_FILE" build --no-cache

    # Stop existing services
    log_info "Stopping existing services..."
    docker-compose -f "$COMPOSE_FILE" down --remove-orphans

    # Start services
    log_info "Starting services..."
    docker-compose -f "$COMPOSE_FILE" up -d

    log_info "Deployment initiated"
}

# Wait for services to be healthy
wait_for_health() {
    log_info "Waiting for services to become healthy..."

    local max_attempts=30
    local attempt=1

    while [[ $attempt -le $max_attempts ]]; do
        log_info "Health check attempt $attempt/$max_attempts..."

        local healthy_services=$(docker-compose -f "$COMPOSE_FILE" ps --services --filter "status=running" | wc -l)
        local total_services=$(docker-compose -f "$COMPOSE_FILE" ps --services | wc -l)

        if [[ $healthy_services -eq $total_services ]]; then
            log_info "All services are healthy"
            return 0
        fi

        log_warn "Only $healthy_services/$total_services services are healthy"
        sleep 10
        ((attempt++))
    done

    log_error "Services did not become healthy within expected time"
    return 1
}

# Run health checks
run_health_checks() {
    log_info "Running comprehensive health checks..."

    # Check application health endpoint
    local app_port="${APP_PORT:-8080}"
    local health_url="http://localhost:${app_port}/health"

    log_info "Checking application health at $health_url..."

    if curl -f -s "$health_url" > /dev/null; then
        log_info "Application health check passed"
    else
        log_error "Application health check failed"
        return 1
    fi

    # Check database connectivity
    log_info "Checking database connectivity..."

    if docker-compose -f "$COMPOSE_FILE" exec -T postgres pg_isready -U "${DB_USER:-postgres}" -d "${DB_NAME:-postgres}"; then
        log_info "Database health check passed"
    else
        log_error "Database health check failed"
        return 1
    fi

    log_info "All health checks passed"
}

# Show deployment status
show_status() {
    log_info "Deployment status:"
    docker-compose -f "$COMPOSE_FILE" ps

    log_info "Service logs (last 10 lines):"
    docker-compose -f "$COMPOSE_FILE" logs --tail=10
}

# Cleanup old images and containers
cleanup() {
    log_info "Cleaning up old Docker resources..."

    # Remove unused images
    docker image prune -f

    # Remove unused containers
    docker container prune -f

    log_info "Cleanup completed"
}

# Main deployment process
main() {
    log_info "Starting deployment process for $PROJECT_NAME ($ENVIRONMENT)"

    check_prerequisites
    validate_environment
    deploy_services

    if wait_for_health; then
        if run_health_checks; then
            show_status
            cleanup
            log_info "Deployment completed successfully!"
            exit 0
        else
            log_error "Health checks failed"
            exit 1
        fi
    else
        log_error "Services failed to become healthy"
        log_error "Deployment logs:"
        docker-compose -f "$COMPOSE_FILE" logs
        exit 1
    fi
}

# Handle script arguments
case "${1:-}" in
    --status)
        show_status
        ;;
    --health-check)
        run_health_checks
        ;;
    --cleanup)
        cleanup
        ;;
    --rollback)
        log_info "Rolling back to previous deployment..."
        docker-compose -f "$COMPOSE_FILE" down
        # Here you would implement your rollback logic
        log_info "Rollback completed"
        ;;
    *)
        main
        ;;
esac