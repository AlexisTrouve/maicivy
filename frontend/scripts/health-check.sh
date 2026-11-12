#!/bin/bash

# ============================================
# maicivy - Health Check Script
# ============================================
# Usage: ./scripts/health-check.sh [OPTIONS]
# Options:
#   --continuous  Run continuous health checks
#   --interval N  Check interval in seconds (default: 30)
#   --timeout N   Timeout in seconds (default: 5)
# ============================================

set -u  # Exit on undefined variable

# ============================================
# Configuration
# ============================================
BACKEND_URL="http://localhost:8080"
FRONTEND_URL="http://localhost:3000"
NGINX_URL="http://localhost:80"

# Default values
CONTINUOUS=false
INTERVAL=30
TIMEOUT=5

# Colors
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# ============================================
# Functions
# ============================================

log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[OK]${NC} $1"
}

log_error() {
    echo -e "${RED}[FAIL]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

check_service() {
    local name=$1
    local url=$2
    local endpoint=$3

    local full_url="${url}${endpoint}"

    if curl -sf --max-time "$TIMEOUT" "$full_url" > /dev/null 2>&1; then
        log_success "$name is healthy ($full_url)"
        return 0
    else
        log_error "$name is unhealthy or unreachable ($full_url)"
        return 1
    fi
}

check_docker_container() {
    local container=$1

    if docker ps --filter "name=$container" --filter "status=running" --format "{{.Names}}" | grep -q "$container"; then
        local health=$(docker inspect --format='{{.State.Health.Status}}' "$container" 2>/dev/null || echo "no-healthcheck")

        if [ "$health" = "healthy" ]; then
            log_success "Container $container is running and healthy"
            return 0
        elif [ "$health" = "no-healthcheck" ]; then
            log_warning "Container $container is running (no health check configured)"
            return 0
        else
            log_error "Container $container is running but unhealthy (status: $health)"
            return 1
        fi
    else
        log_error "Container $container is not running"
        return 1
    fi
}

check_all_services() {
    local timestamp=$(date '+%Y-%m-%d %H:%M:%S')
    echo ""
    log_info "=== Health Check - $timestamp ==="
    echo ""

    local all_healthy=true

    # Check Docker containers
    log_info "Checking Docker containers..."
    check_docker_container "maicivy-backend" || all_healthy=false
    check_docker_container "maicivy-frontend" || all_healthy=false
    check_docker_container "maicivy-nginx" || all_healthy=false
    check_docker_container "maicivy-postgres" || all_healthy=false
    check_docker_container "maicivy-redis" || all_healthy=false

    echo ""

    # Check HTTP endpoints
    log_info "Checking HTTP endpoints..."
    check_service "Backend" "$BACKEND_URL" "/health" || all_healthy=false
    check_service "Frontend" "$FRONTEND_URL" "/" || all_healthy=false
    check_service "Nginx" "$NGINX_URL" "/" || all_healthy=false

    echo ""

    # Check database connectivity
    log_info "Checking database connectivity..."
    if docker exec maicivy-postgres pg_isready -U postgres > /dev/null 2>&1; then
        log_success "PostgreSQL is accepting connections"
    else
        log_error "PostgreSQL is not accepting connections"
        all_healthy=false
    fi

    # Check Redis connectivity
    if docker exec maicivy-redis redis-cli ping > /dev/null 2>&1; then
        log_success "Redis is responding to commands"
    else
        log_error "Redis is not responding to commands"
        all_healthy=false
    fi

    echo ""

    # Summary
    if [ "$all_healthy" = true ]; then
        log_success "=== All services are healthy ==="
        return 0
    else
        log_error "=== Some services are unhealthy ==="
        return 1
    fi
}

show_service_stats() {
    echo ""
    log_info "=== Service Statistics ==="
    echo ""

    # Container resource usage
    log_info "Container resource usage:"
    docker stats --no-stream --format "table {{.Container}}\t{{.CPUPerc}}\t{{.MemUsage}}\t{{.NetIO}}" \
        maicivy-backend maicivy-frontend maicivy-nginx maicivy-postgres maicivy-redis 2>/dev/null || log_warning "Could not fetch stats"

    echo ""

    # Disk usage
    log_info "Docker disk usage:"
    docker system df

    echo ""
}

continuous_monitoring() {
    log_info "Starting continuous health monitoring (interval: ${INTERVAL}s)"
    log_info "Press Ctrl+C to stop"
    echo ""

    local check_count=0
    local failure_count=0

    while true; do
        ((check_count++))

        if ! check_all_services; then
            ((failure_count++))
        fi

        echo ""
        log_info "Check #$check_count - Failures: $failure_count"
        echo ""

        sleep "$INTERVAL"
    done
}

# ============================================
# Parse Arguments
# ============================================

parse_args() {
    while [ $# -gt 0 ]; do
        case $1 in
            --continuous)
                CONTINUOUS=true
                ;;
            --interval)
                shift
                INTERVAL=$1
                ;;
            --timeout)
                shift
                TIMEOUT=$1
                ;;
            --stats)
                show_service_stats
                exit 0
                ;;
            --help)
                echo "Usage: $0 [OPTIONS]"
                echo ""
                echo "Options:"
                echo "  --continuous       Run continuous health checks"
                echo "  --interval N       Check interval in seconds (default: 30)"
                echo "  --timeout N        HTTP timeout in seconds (default: 5)"
                echo "  --stats           Show service statistics"
                echo "  --help            Show this help message"
                exit 0
                ;;
            *)
                log_error "Unknown option: $1"
                echo "Use --help for usage information"
                exit 1
                ;;
        esac
        shift
    done
}

# ============================================
# Main
# ============================================

main() {
    parse_args "$@"

    if [ "$CONTINUOUS" = true ]; then
        continuous_monitoring
    else
        check_all_services
        exit_code=$?

        if [ $exit_code -eq 0 ]; then
            exit 0
        else
            exit 1
        fi
    fi
}

# ============================================
# Execute
# ============================================
main "$@"
