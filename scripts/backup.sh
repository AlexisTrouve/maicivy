#!/bin/bash
#
# Backup script for maicivy
# Usage: ./backup.sh [--upload]
#
set -e

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"
BACKUP_DIR="${BACKUP_DIR:-$PROJECT_DIR/backups}"
RETENTION_DAYS="${RETENTION_DAYS:-7}"
DATE=$(date +%Y%m%d_%H%M%S)

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

log_info() { echo -e "${GREEN}[INFO]${NC} $1"; }
log_warn() { echo -e "${YELLOW}[WARN]${NC} $1"; }
log_error() { echo -e "${RED}[ERROR]${NC} $1"; }

# Create backup directory
mkdir -p "$BACKUP_DIR"

log_info "Starting backup - $DATE"
log_info "Backup directory: $BACKUP_DIR"

# Backup PostgreSQL
log_info "Backing up PostgreSQL..."
if docker exec maicivy-postgres pg_dump -U maicivy maicivy_db 2>/dev/null | gzip > "$BACKUP_DIR/postgres_$DATE.sql.gz"; then
    SIZE=$(du -h "$BACKUP_DIR/postgres_$DATE.sql.gz" | cut -f1)
    log_info "PostgreSQL backup complete: postgres_$DATE.sql.gz ($SIZE)"
else
    log_error "PostgreSQL backup failed!"
    exit 1
fi

# Backup Redis
log_info "Backing up Redis..."
if docker exec maicivy-redis redis-cli BGSAVE >/dev/null 2>&1; then
    sleep 2
    if docker cp maicivy-redis:/data/dump.rdb "$BACKUP_DIR/redis_$DATE.rdb" 2>/dev/null; then
        SIZE=$(du -h "$BACKUP_DIR/redis_$DATE.rdb" | cut -f1)
        log_info "Redis backup complete: redis_$DATE.rdb ($SIZE)"
    else
        log_warn "Redis backup failed - container might not have data"
    fi
else
    log_warn "Redis BGSAVE failed - skipping Redis backup"
fi

# Backup .env file (without secrets in filename)
log_info "Backing up configuration..."
if [ -f "$PROJECT_DIR/backend/.env" ]; then
    cp "$PROJECT_DIR/backend/.env" "$BACKUP_DIR/env_$DATE.bak"
    log_info "Configuration backup complete: env_$DATE.bak"
fi

# Cleanup old backups
log_info "Cleaning up backups older than $RETENTION_DAYS days..."
DELETED=$(find "$BACKUP_DIR" -type f -mtime +$RETENTION_DAYS -delete -print | wc -l)
log_info "Deleted $DELETED old backup files"

# Optional: Upload to remote storage
if [ "$1" == "--upload" ] && [ -n "$S3_BUCKET" ]; then
    log_info "Uploading to S3..."
    aws s3 cp "$BACKUP_DIR/postgres_$DATE.sql.gz" "$S3_BUCKET/postgres/"
    aws s3 cp "$BACKUP_DIR/redis_$DATE.rdb" "$S3_BUCKET/redis/" 2>/dev/null || true
    log_info "Upload complete"
fi

# Summary
log_info "=== Backup Summary ==="
log_info "Date: $DATE"
log_info "Location: $BACKUP_DIR"
ls -lh "$BACKUP_DIR"/*_$DATE.* 2>/dev/null || true
log_info "=== Backup completed successfully ==="
