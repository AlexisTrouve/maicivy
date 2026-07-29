#!/bin/bash

# Script de monitoring en temps réel des services
# Affiche l'état de tous les services maicivy

set -e

# Couleurs
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# Fonction pour afficher l'état d'un service
check_docker_service() {
    local service_name=$1
    local container_name=$2

    echo -n "  $service_name: "

    if docker ps --filter "name=$container_name" --format "{{.Status}}" | grep -q "Up"; then
        echo -e "${GREEN}RUNNING${NC}"

        # Vérifier health check si disponible
        health=$(docker inspect --format='{{.State.Health.Status}}' $container_name 2>/dev/null || echo "no-health-check")
        if [ "$health" != "no-health-check" ]; then
            if [ "$health" = "healthy" ]; then
                echo -e "    Health: ${GREEN}$health${NC}"
            else
                echo -e "    Health: ${YELLOW}$health${NC}"
            fi
        fi
    else
        echo -e "${RED}STOPPED${NC}"
    fi
}

# Fonction pour afficher stats Docker
show_docker_stats() {
    echo ""
    echo -e "${BLUE}=== Docker Stats ===${NC}"
    docker stats --no-stream --format "table {{.Name}}\t{{.CPUPerc}}\t{{.MemUsage}}\t{{.NetIO}}" \
        maicivy_postgres maicivy_redis maicivy_backend maicivy_frontend maicivy_nginx maicivy_prometheus maicivy_grafana 2>/dev/null || echo "No containers running"
}

# Fonction pour afficher disk usage
show_disk_usage() {
    echo ""
    echo -e "${BLUE}=== Disk Usage ===${NC}"
    df -h / | tail -n 1 | awk '{print "  Root: " $3 " / " $2 " (" $5 " used)"}'

    if [ -d "/opt/maicivy/backups" ]; then
        backup_size=$(du -sh /opt/maicivy/backups 2>/dev/null | awk '{print $1}')
        echo "  Backups: $backup_size"
    fi
}

# Fonction pour afficher memory usage
show_memory_usage() {
    echo ""
    echo -e "${BLUE}=== Memory Usage ===${NC}"
    free -h | grep Mem | awk '{print "  Total: " $2 " | Used: " $3 " | Free: " $4}'
}

# Clear screen et afficher header
clear
echo "========================================="
echo "    maicivy - Services Monitor"
echo "========================================="
echo ""

# Services Docker
echo -e "${BLUE}=== Docker Services ===${NC}"
check_docker_service "PostgreSQL     " "maicivy_postgres"
check_docker_service "Redis          " "maicivy_redis"
check_docker_service "Backend        " "maicivy_backend"
check_docker_service "Frontend       " "maicivy_frontend"
check_docker_service "Nginx          " "maicivy_nginx"
check_docker_service "Prometheus     " "maicivy_prometheus"
check_docker_service "Grafana        " "maicivy_grafana"
check_docker_service "Node Exporter  " "maicivy_node_exporter"

# Stats
show_docker_stats
show_disk_usage
show_memory_usage

# Nginx status
echo ""
echo -e "${BLUE}=== Nginx Status ===${NC}"
if systemctl is-active --quiet nginx; then
    echo -e "  Nginx: ${GREEN}ACTIVE${NC}"
else
    echo -e "  Nginx: ${RED}INACTIVE${NC}"
fi

# SSL Certificates
echo ""
echo -e "${BLUE}=== SSL Certificates ===${NC}"
if [ -d "/etc/letsencrypt/live/maicivy.com" ]; then
    expiry=$(openssl x509 -enddate -noout -in /etc/letsencrypt/live/maicivy.com/fullchain.pem 2>/dev/null | cut -d= -f2)
    if [ -n "$expiry" ]; then
        echo "  Expiry: $expiry"
    else
        echo -e "  ${YELLOW}Unable to check expiry${NC}"
    fi
else
    echo -e "  ${YELLOW}No certificates found${NC}"
fi

# Recent logs (errors only)
echo ""
echo -e "${BLUE}=== Recent Errors ===${NC}"
error_count=$(docker logs maicivy_backend --tail 100 2>/dev/null | grep -i error | wc -l || echo 0)
if [ "$error_count" -gt 0 ]; then
    echo -e "  Backend errors (last 100 lines): ${YELLOW}$error_count${NC}"
else
    echo -e "  Backend errors (last 100 lines): ${GREEN}0${NC}"
fi

# URLs
echo ""
echo -e "${BLUE}=== Service URLs ===${NC}"
echo "  Frontend:   https://maicivy.com"
echo "  API:        https://maicivy.com/api"
echo "  Health:     https://maicivy.com/health"
echo "  Grafana:    https://analytics.maicivy.com"
echo "  Prometheus: http://localhost:9090"

echo ""
echo "========================================="
echo "Timestamp: $(date)"
echo "========================================="
