#!/bin/bash
#
# Restore script for maicivy
# Usage: ./restore.sh <backup_date>
# Example: ./restore.sh 20250104_120000
#
set -e

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"
BACKUP_DIR="${BACKUP_DIR:-$PROJECT_DIR/backups}"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

log_info() { echo -e "${GREEN}[INFO]${NC} $1"; }
log_warn() { echo -e "${YELLOW}[WARN]${NC} $1"; }
log_error() { echo -e "${RED}[ERROR]${NC} $1"; }

# Check arguments
if [ -z "$1" ]; then
    echo "Usage: $0 <backup_date>"
    echo "Example: $0 20250104_120000"
    echo ""
    echo "Available backups:"
    ls -la "$BACKUP_DIR"/*.sql.gz 2>/dev/null | awk '{print $NF}' | xargs -I {} basename {} | sed 's/postgres_/  /' | sed 's/.sql.gz//'
    exit 1
fi

DATE=$1
POSTGRES_BACKUP="$BACKUP_DIR/postgres_$DATE.sql.gz"
REDIS_BACKUP="$BACKUP_DIR/redis_$DATE.rdb"

# Verify backups exist
if [ ! -f "$POSTGRES_BACKUP" ]; then
    log_error "PostgreSQL backup not found: $POSTGRES_BACKUP"
    exit 1
fi

log_warn "=== WARNING ==="
log_warn "This will OVERWRITE the current database with backup from $DATE"
log_warn "Current data will be LOST!"
echo ""
read -p "Are you sure you want to continue? (yes/no): " CONFIRM

if [ "$CONFIRM" != "yes" ]; then
    log_info "Restore cancelled"
    exit 0
fi

# Stop application to prevent writes during restore
log_info "Stopping backend service..."
docker-compose stop backend 2>/dev/null || true

# Restore PostgreSQL
log_info "Restoring PostgreSQL from $POSTGRES_BACKUP..."

# Drop and recreate database
docker exec maicivy-postgres psql -U maicivy -d postgres -c "DROP DATABASE IF EXISTS maicivy_db;" 2>/dev/null || true
docker exec maicivy-postgres psql -U maicivy -d postgres -c "CREATE DATABASE maicivy_db;" 2>/dev/null

# Restore data
gunzip -c "$POSTGRES_BACKUP" | docker exec -i maicivy-postgres psql -U maicivy maicivy_db

log_info "PostgreSQL restore complete"

# Restore Redis (if backup exists)
if [ -f "$REDIS_BACKUP" ]; then
    log_info "Restoring Redis from $REDIS_BACKUP..."

    # Stop Redis, copy dump, restart
    docker-compose stop redis
    docker cp "$REDIS_BACKUP" maicivy-redis:/data/dump.rdb
    docker-compose start redis

    log_info "Redis restore complete"
else
    log_warn "Redis backup not found, skipping Redis restore"
fi

# Restart backend
log_info "Starting backend service..."
docker-compose start backend

# Health check
log_info "Waiting for services to be ready..."
sleep 10

if curl -sf http://localhost:8080/health >/dev/null 2>&1; then
    log_info "Health check passed"
else
    log_warn "Health check failed - please check logs"
fi

log_info "=== Restore completed ==="
log_info "Restored from backup: $DATE"
