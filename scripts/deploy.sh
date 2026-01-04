#!/bin/bash
#
# Deploy script for maicivy
# Usage: ./deploy.sh [--no-build]
#
set -e

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

log_info() { echo -e "${GREEN}[INFO]${NC} $1"; }
log_warn() { echo -e "${YELLOW}[WARN]${NC} $1"; }
log_error() { echo -e "${RED}[ERROR]${NC} $1"; }
log_step() { echo -e "${BLUE}[STEP]${NC} $1"; }

cd "$PROJECT_DIR"

echo ""
echo "=========================================="
echo "       maicivy Deployment Script"
echo "=========================================="
echo ""

# Step 1: Pull latest changes
log_step "1/5 - Pulling latest changes..."
git fetch origin main
git reset --hard origin/main
log_info "Git pull complete"

# Step 2: Backup before deploy (optional but recommended)
log_step "2/5 - Creating pre-deploy backup..."
if [ -f "$SCRIPT_DIR/backup.sh" ]; then
    "$SCRIPT_DIR/backup.sh" || log_warn "Backup failed, continuing anyway..."
else
    log_warn "Backup script not found, skipping backup"
fi

# Step 3: Build and start containers
log_step "3/5 - Building and starting containers..."
if [ "$1" == "--no-build" ]; then
    docker-compose up -d
else
    docker-compose up -d --build
fi
log_info "Containers started"

# Step 4: Cleanup
log_step "4/5 - Cleaning up Docker resources..."
docker system prune -f
log_info "Cleanup complete"

# Step 5: Health check
log_step "5/5 - Running health checks..."
sleep 10

HEALTH_OK=true

# Check backend
if curl -sf http://localhost:8080/health >/dev/null 2>&1; then
    log_info "Backend: OK"
else
    log_error "Backend: FAILED"
    HEALTH_OK=false
fi

# Check frontend
if curl -sf http://localhost:3000 >/dev/null 2>&1; then
    log_info "Frontend: OK"
else
    log_warn "Frontend: Not responding (might still be starting)"
fi

# Check database
if docker exec maicivy-postgres pg_isready -U maicivy >/dev/null 2>&1; then
    log_info "PostgreSQL: OK"
else
    log_error "PostgreSQL: FAILED"
    HEALTH_OK=false
fi

# Check Redis
if docker exec maicivy-redis redis-cli ping >/dev/null 2>&1; then
    log_info "Redis: OK"
else
    log_error "Redis: FAILED"
    HEALTH_OK=false
fi

echo ""
echo "=========================================="
if [ "$HEALTH_OK" = true ]; then
    log_info "Deployment completed successfully!"
else
    log_error "Deployment completed with errors!"
    log_warn "Check logs: docker-compose logs -f"
    exit 1
fi
echo "=========================================="
echo ""
