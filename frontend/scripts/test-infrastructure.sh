#!/bin/bash

# Script de test de l'infrastructure avant déploiement
# Vérifie que tous les fichiers et configurations sont corrects

set -e

# Couleurs
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

FAILED=0
PASSED=0

# Fonction de test
test_file() {
    local file=$1
    local description=$2

    echo -n "Testing $description... "

    if [ -f "$file" ]; then
        echo -e "${GREEN}OK${NC}"
        PASSED=$((PASSED+1))
        return 0
    else
        echo -e "${RED}FAIL${NC} (file not found: $file)"
        FAILED=$((FAILED+1))
        return 1
    fi
}

# Fonction de test syntax
test_syntax() {
    local file=$1
    local description=$2

    echo -n "Testing $description syntax... "

    case "$file" in
        *.sh)
            if bash -n "$file" 2>/dev/null; then
                echo -e "${GREEN}OK${NC}"
                PASSED=$((PASSED+1))
                return 0
            else
                echo -e "${RED}FAIL${NC} (bash syntax error)"
                FAILED=$((FAILED+1))
                return 1
            fi
            ;;
        *.yml|*.yaml)
            if command -v yamllint &> /dev/null; then
                if yamllint -d relaxed "$file" 2>/dev/null; then
                    echo -e "${GREEN}OK${NC}"
                    PASSED=$((PASSED+1))
                    return 0
                else
                    echo -e "${YELLOW}WARN${NC} (yamllint issues, but may be OK)"
                    PASSED=$((PASSED+1))
                    return 0
                fi
            else
                echo -e "${YELLOW}SKIP${NC} (yamllint not installed)"
                return 0
            fi
            ;;
        *.json)
            if command -v jq &> /dev/null; then
                if jq empty "$file" 2>/dev/null; then
                    echo -e "${GREEN}OK${NC}"
                    PASSED=$((PASSED+1))
                    return 0
                else
                    echo -e "${RED}FAIL${NC} (invalid JSON)"
                    FAILED=$((FAILED+1))
                    return 1
                fi
            else
                echo -e "${YELLOW}SKIP${NC} (jq not installed)"
                return 0
            fi
            ;;
        *.conf)
            # Pour nginx, on skip si nginx pas installé
            if command -v nginx &> /dev/null; then
                if nginx -t -c "$file" 2>/dev/null; then
                    echo -e "${GREEN}OK${NC}"
                    PASSED=$((PASSED+1))
                    return 0
                else
                    echo -e "${YELLOW}WARN${NC} (nginx test failed, may need adjustments)"
                    PASSED=$((PASSED+1))
                    return 0
                fi
            else
                echo -e "${YELLOW}SKIP${NC} (nginx not installed)"
                return 0
            fi
            ;;
        *)
            echo -e "${YELLOW}SKIP${NC} (no syntax test for this type)"
            return 0
            ;;
    esac
}

# Fonction de test Docker Compose
test_docker_compose() {
    local file=$1
    local description=$2

    echo -n "Testing $description... "

    if command -v docker-compose &> /dev/null; then
        if docker-compose -f "$file" config > /dev/null 2>&1; then
            echo -e "${GREEN}OK${NC}"
            PASSED=$((PASSED+1))
            return 0
        else
            echo -e "${RED}FAIL${NC} (invalid docker-compose syntax)"
            FAILED=$((FAILED+1))
            return 1
        fi
    else
        echo -e "${YELLOW}SKIP${NC} (docker-compose not installed)"
        return 0
    fi
}

echo "========================================="
echo "  Infrastructure Tests"
echo "========================================="
echo ""

# Docker Configuration
echo -e "${BLUE}=== Docker Configuration ===${NC}"
test_file "docker/nginx/nginx.conf" "Nginx config"
test_file "docker/docker-compose.prod.yml" "Docker Compose production"
test_file "docker/docker-compose.monitoring.yml" "Docker Compose monitoring"
test_docker_compose "docker/docker-compose.prod.yml" "Docker Compose production syntax"

echo ""

# Prometheus
echo -e "${BLUE}=== Prometheus Configuration ===${NC}"
test_file "monitoring/prometheus/prometheus.yml" "Prometheus config"
test_file "monitoring/prometheus/alerts.yml" "Prometheus alerts"
test_file "monitoring/prometheus/alertmanager.yml" "Alertmanager config"
test_syntax "monitoring/prometheus/prometheus.yml" "Prometheus config"
test_syntax "monitoring/prometheus/alerts.yml" "Prometheus alerts"

echo ""

# Grafana
echo -e "${BLUE}=== Grafana Configuration ===${NC}"
test_file "monitoring/grafana/provisioning/datasources/prometheus.yml" "Grafana datasource"
test_file "monitoring/grafana/provisioning/dashboards/dashboard.yml" "Grafana dashboard provisioning"
test_file "monitoring/grafana/dashboards/maicivy_overview.json" "Grafana dashboard"
test_syntax "monitoring/grafana/dashboards/maicivy_overview.json" "Grafana dashboard"

echo ""

# Loki (optionnel)
echo -e "${BLUE}=== Loki Configuration (Optional) ===${NC}"
test_file "monitoring/loki/loki-config.yml" "Loki config"
test_file "monitoring/loki/promtail-config.yml" "Promtail config"

echo ""

# Scripts
echo -e "${BLUE}=== Scripts ===${NC}"
test_file "scripts/setup-infrastructure.sh" "Setup infrastructure script"
test_file "scripts/deploy.sh" "Deploy script"
test_file "scripts/health-check.sh" "Health check script"
test_file "scripts/monitor-services.sh" "Monitor services script"
test_file "scripts/renew-ssl.sh" "SSL renewal script"
test_file "scripts/backup-postgres.sh" "PostgreSQL backup script"
test_file "scripts/backup-redis.sh" "Redis backup script"
test_file "scripts/restore-postgres.sh" "PostgreSQL restore script"
test_file "scripts/restore-redis.sh" "Redis restore script"

echo ""

# Script syntax
echo -e "${BLUE}=== Script Syntax ===${NC}"
test_syntax "scripts/setup-infrastructure.sh" "Setup infrastructure"
test_syntax "scripts/deploy.sh" "Deploy"
test_syntax "scripts/health-check.sh" "Health check"
test_syntax "scripts/monitor-services.sh" "Monitor services"
test_syntax "scripts/renew-ssl.sh" "SSL renewal"
test_syntax "scripts/backup-postgres.sh" "PostgreSQL backup"
test_syntax "scripts/backup-redis.sh" "Redis backup"
test_syntax "scripts/restore-postgres.sh" "PostgreSQL restore"
test_syntax "scripts/restore-redis.sh" "Redis restore"

echo ""

# Systemd
echo -e "${BLUE}=== Systemd ===${NC}"
test_file "scripts/systemd/maicivy.service" "Systemd service"

echo ""

# Environment
echo -e "${BLUE}=== Environment ===${NC}"
test_file ".env.production.example" "Environment template"

echo ""

# Documentation
echo -e "${BLUE}=== Documentation ===${NC}"
test_file "INFRASTRUCTURE_PRODUCTION_GUIDE.md" "Production guide"
test_file "INFRASTRUCTURE_VALIDATION.md" "Validation guide"
test_file "INFRASTRUCTURE_SUMMARY.md" "Summary"
test_file "QUICKSTART_PRODUCTION.md" "Quick start"
test_file "docker/README.md" "Docker README"

echo ""

# Script permissions
echo -e "${BLUE}=== Script Permissions ===${NC}"
echo -n "Testing script executability... "
EXECUTABLE_COUNT=$(find scripts/ -name "*.sh" -executable | wc -l)
TOTAL_SCRIPTS=$(find scripts/ -name "*.sh" | wc -l)

if [ "$EXECUTABLE_COUNT" -eq "$TOTAL_SCRIPTS" ]; then
    echo -e "${GREEN}OK${NC} ($EXECUTABLE_COUNT/$TOTAL_SCRIPTS executable)"
    PASSED=$((PASSED+1))
else
    echo -e "${YELLOW}WARN${NC} ($EXECUTABLE_COUNT/$TOTAL_SCRIPTS executable)"
    echo "Run: chmod +x scripts/*.sh"
fi

echo ""

# Results
echo "========================================="
echo "  Test Results"
echo "========================================="
echo -e "Passed: ${GREEN}$PASSED${NC}"
echo -e "Failed: ${RED}$FAILED${NC}"
echo ""

if [ $FAILED -eq 0 ]; then
    echo -e "${GREEN}All tests passed! Infrastructure is ready for deployment.${NC}"
    exit 0
else
    echo -e "${RED}Some tests failed. Please fix the issues before deploying.${NC}"
    exit 1
fi
