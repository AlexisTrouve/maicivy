#!/bin/bash

# ============================================================================
# MAICIVY - Infrastructure Health Check Script
# ============================================================================
# Vérifie l'état de tous les services Docker
# Usage: ./scripts/health-check.sh
# ============================================================================

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
DOCKER_COMPOSE_FILE="docker-compose.yml"
REQUIRED_SERVICES=("postgres" "redis" "backend" "frontend")
MAX_RETRIES=10
RETRY_DELAY=2

# ============================================================================
# Functions
# ============================================================================

print_header() {
    echo -e "${BLUE}========================================${NC}"
    echo -e "${BLUE}$1${NC}"
    echo -e "${BLUE}========================================${NC}"
}

print_status() {
    local service=$1
    local status=$2

    if [ "$status" = "healthy" ]; then
        echo -e "${GREEN}✓${NC} $service: ${GREEN}HEALTHY${NC}"
    elif [ "$status" = "running" ]; then
        echo -e "${GREEN}✓${NC} $service: ${GREEN}RUNNING${NC}"
    else
        echo -e "${RED}✗${NC} $service: ${RED}$status${NC}"
    fi
}

check_docker_installed() {
    if ! command -v docker &> /dev/null; then
        echo -e "${RED}ERROR: Docker is not installed${NC}"
        exit 1
    fi
    echo -e "${GREEN}✓ Docker is installed${NC}"
}

check_docker_compose() {
    if ! command -v docker-compose &> /dev/null; then
        echo -e "${RED}ERROR: Docker Compose is not installed${NC}"
        exit 1
    fi
    echo -e "${GREEN}✓ Docker Compose is installed${NC}"
}

check_docker_running() {
    if ! docker info &> /dev/null; then
        echo -e "${RED}ERROR: Docker daemon is not running${NC}"
        exit 1
    fi
    echo -e "${GREEN}✓ Docker daemon is running${NC}"
}

check_service_health() {
    local service=$1
    local max_retries=$2
    local retry_count=0

    while [ $retry_count -lt $max_retries ]; do
        local health=$(docker inspect -f '{{.State.Health.Status}}' maicivy-$service 2>/dev/null || echo "no-healthcheck")
        local state=$(docker inspect -f '{{.State.Running}}' maicivy-$service 2>/dev/null || echo "false")

        if [ "$health" = "healthy" ]; then
            print_status "$service" "healthy"
            return 0
        elif [ "$health" = "starting" ]; then
            echo -e "${YELLOW}⟳${NC} $service: Starting... ($((retry_count + 1))/$max_retries)"
        elif [ "$state" = "true" ] && [ "$health" = "no-healthcheck" ]; then
            print_status "$service" "running"
            return 0
        else
            echo -e "${YELLOW}⟳${NC} $service: Not ready... ($((retry_count + 1))/$max_retries)"
        fi

        ((retry_count++))
        sleep $RETRY_DELAY
    done

    echo -e "${RED}✗${NC} $service: ${RED}FAILED${NC}"
    return 1
}

get_service_logs() {
    local service=$1
    echo -e "\n${YELLOW}Last 20 logs for $service:${NC}"
    docker logs --tail=20 maicivy-$service 2>&1 || echo "No logs available"
}

test_connectivity() {
    print_header "Testing Service Connectivity"

    # Test backend health endpoint
    echo -n "Testing Backend health endpoint... "
    if curl -s http://localhost:8080/health > /dev/null 2>&1; then
        echo -e "${GREEN}✓${NC}"
    else
        echo -e "${RED}✗${NC}"
    fi

    # Test PostgreSQL connection
    echo -n "Testing PostgreSQL connection... "
    if docker exec maicivy-postgres pg_isready -U maicivy > /dev/null 2>&1; then
        echo -e "${GREEN}✓${NC}"
    else
        echo -e "${RED}✗${NC}"
    fi

    # Test Redis connection
    echo -n "Testing Redis connection... "
    if docker exec maicivy-redis redis-cli ping > /dev/null 2>&1; then
        echo -e "${GREEN}✓${NC}"
    else
        echo -e "${RED}✗${NC}"
    fi

    # Test Frontend
    echo -n "Testing Frontend availability... "
    if curl -s http://localhost:3000 > /dev/null 2>&1; then
        echo -e "${GREEN}✓${NC}"
    else
        echo -e "${YELLOW}⟳${NC} (Frontend may still be building)"
    fi
}

# ============================================================================
# Main
# ============================================================================

main() {
    print_header "MAICIVY Infrastructure Health Check"

    echo -e "\n${BLUE}Step 1: Checking prerequisites${NC}"
    check_docker_installed
    check_docker_compose
    check_docker_running

    echo -e "\n${BLUE}Step 2: Checking Docker Compose file${NC}"
    if [ ! -f "$DOCKER_COMPOSE_FILE" ]; then
        echo -e "${RED}ERROR: $DOCKER_COMPOSE_FILE not found${NC}"
        exit 1
    fi
    echo -e "${GREEN}✓ $DOCKER_COMPOSE_FILE found${NC}"

    echo -e "\n${BLUE}Step 3: Checking service health${NC}"
    local all_healthy=true
    for service in "${REQUIRED_SERVICES[@]}"; do
        if ! check_service_health "$service" $MAX_RETRIES; then
            all_healthy=false
            get_service_logs "$service"
        fi
    done

    echo -e "\n${BLUE}Step 4: Testing connectivity${NC}"
    test_connectivity

    echo -e "\n${BLUE}Step 5: Container Information${NC}"
    docker-compose ps

    echo -e "\n${BLUE}Step 6: Network Information${NC}"
    echo "Containers connected to maicivy network:"
    docker network inspect maicivy -f '{{range .Containers}}  - {{.Name}} ({{.IPv4Address}}){{"\n"}}{{end}}'

    echo -e "\n${BLUE}Step 7: Volume Information${NC}"
    echo "Docker volumes:"
    docker volume ls | grep maicivy

    if [ "$all_healthy" = true ]; then
        echo -e "\n${GREEN}========================================${NC}"
        echo -e "${GREEN}✓ All services are healthy!${NC}"
        echo -e "${GREEN}========================================${NC}"
        echo -e "\n${BLUE}Access points:${NC}"
        echo "  Frontend:  http://localhost:3000"
        echo "  Backend:   http://localhost:8080"
        echo "  PostgreSQL: localhost:5432"
        echo "  Redis:     localhost:6379"
        exit 0
    else
        echo -e "\n${RED}========================================${NC}"
        echo -e "${RED}✗ Some services are not healthy${NC}"
        echo -e "${RED}========================================${NC}"
        exit 1
    fi
}

main "$@"
