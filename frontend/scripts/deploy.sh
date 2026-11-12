#!/bin/bash

# ============================================
# maicivy - Production Deployment Script
# ============================================
# Usage: ./scripts/deploy.sh [OPTIONS]
# Options:
#   --rollback    Rollback to previous version
#   --backup      Backup current state before deploy
#   --dry-run     Simulate deployment without changes
# ============================================

set -e  # Exit on error
set -u  # Exit on undefined variable

# ============================================
# Configuration
# ============================================
DEPLOY_DIR="/opt/maicivy"
COMPOSE_FILE="$DEPLOY_DIR/docker-compose.yml"
BACKUP_DIR="$DEPLOY_DIR/backups"
TIMESTAMP=$(date +%Y%m%d_%H%M%S)

# Colors
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# ============================================
# Functions
# ============================================

log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

check_prerequisites() {
    log_info "Checking prerequisites..."

    # Check if Docker is running
    if ! docker info > /dev/null 2>&1; then
        log_error "Docker is not running!"
        exit 1
    fi

    # Check if docker-compose exists
    if ! command -v docker-compose &> /dev/null; then
        log_error "docker-compose is not installed!"
        exit 1
    fi

    # Check if deployment directory exists
    if [ ! -d "$DEPLOY_DIR" ]; then
        log_error "Deployment directory does not exist: $DEPLOY_DIR"
        exit 1
    fi

    log_success "Prerequisites check passed"
}

backup_current_state() {
    log_info "Backing up current state..."

    mkdir -p "$BACKUP_DIR"

    # Backup container IDs
    docker-compose -f "$COMPOSE_FILE" ps -q > "$BACKUP_DIR/containers_${TIMESTAMP}.txt"

    # Backup current images
    docker-compose -f "$COMPOSE_FILE" images --format json > "$BACKUP_DIR/images_${TIMESTAMP}.json"

    # Backup environment file (if exists)
    if [ -f "$DEPLOY_DIR/.env" ]; then
        cp "$DEPLOY_DIR/.env" "$BACKUP_DIR/env_${TIMESTAMP}.bak"
    fi

    log_success "Backup created: $BACKUP_DIR/*_${TIMESTAMP}.*"
}

pull_latest_images() {
    log_info "Pulling latest Docker images..."

    cd "$DEPLOY_DIR"
    docker-compose pull

    log_success "Images pulled successfully"
}

deploy_services() {
    log_info "Deploying services with rolling update..."

    cd "$DEPLOY_DIR"

    # Rolling update: recreate only changed containers
    docker-compose up -d --remove-orphans

    log_success "Services deployed"
}

health_check() {
    log_info "Running health checks..."

    local max_attempts=30
    local attempt=1
    local backend_healthy=false

    while [ $attempt -le $max_attempts ]; do
        log_info "Health check attempt $attempt/$max_attempts..."

        # Check backend health
        if curl -sf http://localhost:8080/health > /dev/null 2>&1; then
            backend_healthy=true
            log_success "Backend is healthy!"
            break
        fi

        sleep 2
        ((attempt++))
    done

    if [ "$backend_healthy" = false ]; then
        log_error "Health check failed after $max_attempts attempts!"
        return 1
    fi

    # Additional checks
    log_info "Checking container status..."
    docker-compose -f "$COMPOSE_FILE" ps

    return 0
}

rollback() {
    log_warning "Rolling back to previous version..."

    # Find latest backup
    local latest_backup=$(ls -t "$BACKUP_DIR"/containers_*.txt 2>/dev/null | head -n1)

    if [ -z "$latest_backup" ]; then
        log_error "No backup found for rollback!"
        exit 1
    fi

    log_info "Using backup: $latest_backup"

    # Stop current containers
    docker-compose -f "$COMPOSE_FILE" down

    # Restore previous containers
    while IFS= read -r container_id; do
        if [ -n "$container_id" ]; then
            log_info "Starting container: $container_id"
            docker start "$container_id" || log_warning "Could not start container $container_id"
        fi
    done < "$latest_backup"

    log_success "Rollback completed"
}

cleanup_old_images() {
    log_info "Cleaning up old Docker images..."

    # Remove dangling images
    docker image prune -f

    # Remove images older than 7 days
    docker image prune -af --filter "until=168h"

    log_success "Cleanup completed"
}

show_status() {
    log_info "Current deployment status:"
    echo ""
    docker-compose -f "$COMPOSE_FILE" ps
    echo ""
    docker-compose -f "$COMPOSE_FILE" logs --tail=20
}

# ============================================
# Main Deployment Flow
# ============================================

main() {
    log_info "Starting maicivy deployment..."
    echo ""

    # Parse arguments
    ROLLBACK_MODE=false
    BACKUP_ONLY=false
    DRY_RUN=false

    for arg in "$@"; do
        case $arg in
            --rollback)
                ROLLBACK_MODE=true
                ;;
            --backup)
                BACKUP_ONLY=true
                ;;
            --dry-run)
                DRY_RUN=true
                ;;
        esac
    done

    # Execute based on mode
    if [ "$ROLLBACK_MODE" = true ]; then
        rollback
        health_check
        exit 0
    fi

    check_prerequisites
    backup_current_state

    if [ "$BACKUP_ONLY" = true ]; then
        log_success "Backup completed. Exiting."
        exit 0
    fi

    if [ "$DRY_RUN" = true ]; then
        log_info "DRY RUN mode - no changes will be made"
        log_info "Would pull images and deploy..."
        exit 0
    fi

    pull_latest_images
    deploy_services

    if health_check; then
        log_success "✅ Deployment successful!"
        cleanup_old_images
        show_status
    else
        log_error "❌ Deployment failed!"
        log_warning "Initiating automatic rollback..."
        rollback

        if health_check; then
            log_warning "Rollback successful, system restored"
            exit 1
        else
            log_error "Rollback failed! Manual intervention required!"
            exit 2
        fi
    fi
}

# ============================================
# Execute
# ============================================
main "$@"
