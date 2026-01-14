#!/bin/bash

# ============================================
# maicivy - Rollback Script
# ============================================
# Usage: ./scripts/rollback.sh [TIMESTAMP]
# ============================================

set -e  # Exit on error
set -u  # Exit on undefined variable

# ============================================
# Configuration
# ============================================
DEPLOY_DIR="/opt/maicivy"
COMPOSE_FILE="$DEPLOY_DIR/docker-compose.yml"
BACKUP_DIR="$DEPLOY_DIR/backups"

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

list_available_backups() {
    log_info "Available backups:"
    echo ""

    local backups=$(ls -t "$BACKUP_DIR"/containers_*.txt 2>/dev/null)

    if [ -z "$backups" ]; then
        log_error "No backups found!"
        exit 1
    fi

    local index=1
    for backup in $backups; do
        local timestamp=$(basename "$backup" | sed 's/containers_\(.*\)\.txt/\1/')
        local date=$(echo "$timestamp" | sed 's/\([0-9]\{8\}\)_\([0-9]\{6\}\)/\1 \2/')
        echo "  [$index] $date"
        ((index++))
    done

    echo ""
}

get_latest_backup() {
    local latest=$(ls -t "$BACKUP_DIR"/containers_*.txt 2>/dev/null | head -n1)

    if [ -z "$latest" ]; then
        log_error "No backup found!"
        exit 1
    fi

    echo "$latest"
}

get_backup_by_timestamp() {
    local timestamp=$1
    local backup="$BACKUP_DIR/containers_${timestamp}.txt"

    if [ ! -f "$backup" ]; then
        log_error "Backup not found: $backup"
        exit 1
    fi

    echo "$backup"
}

rollback_to_backup() {
    local backup_file=$1

    log_warning "Rolling back using: $backup_file"

    # Extract timestamp from filename
    local timestamp=$(basename "$backup_file" | sed 's/containers_\(.*\)\.txt/\1/')
    log_info "Backup timestamp: $timestamp"

    # Stop current containers
    log_info "Stopping current containers..."
    cd "$DEPLOY_DIR"
    docker-compose down

    # Restore previous containers
    log_info "Restoring containers from backup..."
    local container_count=0

    while IFS= read -r container_id; do
        if [ -n "$container_id" ]; then
            log_info "Starting container: $container_id"
            if docker start "$container_id"; then
                ((container_count++))
            else
                log_warning "Could not start container: $container_id (may have been removed)"
            fi
        fi
    done < "$backup_file"

    if [ $container_count -eq 0 ]; then
        log_error "No containers could be restored!"
        log_warning "Attempting to restore from images backup..."
        restore_from_images_backup "$timestamp"
    else
        log_success "Restored $container_count containers"
    fi
}

restore_from_images_backup() {
    local timestamp=$1
    local images_backup="$BACKUP_DIR/images_${timestamp}.json"

    if [ ! -f "$images_backup" ]; then
        log_error "Images backup not found: $images_backup"
        exit 1
    fi

    log_info "Restoring from images backup: $images_backup"

    # Restore environment file
    local env_backup="$BACKUP_DIR/env_${timestamp}.bak"
    if [ -f "$env_backup" ]; then
        log_info "Restoring environment file..."
        cp "$env_backup" "$DEPLOY_DIR/.env"
    fi

    # Restart services with docker-compose
    cd "$DEPLOY_DIR"
    log_info "Starting services with docker-compose..."
    docker-compose up -d

    log_success "Services restarted from images backup"
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

    return 0
}

show_status() {
    log_info "Current deployment status:"
    echo ""
    docker-compose -f "$COMPOSE_FILE" ps
    echo ""
    log_info "Recent logs:"
    docker-compose -f "$COMPOSE_FILE" logs --tail=20
}

# ============================================
# Main Rollback Flow
# ============================================

main() {
    log_warning "=== maicivy Rollback Utility ==="
    echo ""

    # Parse arguments
    local backup_timestamp=""

    if [ $# -gt 0 ]; then
        backup_timestamp=$1
    fi

    # Determine which backup to use
    local backup_file=""

    if [ -n "$backup_timestamp" ]; then
        # Use specified timestamp
        backup_file=$(get_backup_by_timestamp "$backup_timestamp")
        log_info "Using specified backup: $backup_timestamp"
    else
        # List available backups
        list_available_backups

        # Use latest backup
        backup_file=$(get_latest_backup)
        log_info "Using latest backup"
    fi

    # Confirm rollback
    log_warning "This will rollback to: $(basename $backup_file)"
    read -p "Are you sure you want to proceed? (yes/no): " confirm

    if [ "$confirm" != "yes" ]; then
        log_info "Rollback cancelled"
        exit 0
    fi

    # Execute rollback
    rollback_to_backup "$backup_file"

    # Health check
    if health_check; then
        log_success "✅ Rollback successful!"
        show_status
        exit 0
    else
        log_error "❌ Rollback completed but health check failed!"
        log_error "Manual intervention required!"
        show_status
        exit 1
    fi
}

# ============================================
# Execute
# ============================================
main "$@"
