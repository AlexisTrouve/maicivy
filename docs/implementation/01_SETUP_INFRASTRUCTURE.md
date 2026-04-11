# 01. SETUP_INFRASTRUCTURE.md

## 📋 Métadonnées

- **Phase:** 1
- **Priorité:** 🔴 CRITIQUE
- **Complexité:** ⭐⭐ (2/5)
- **Prérequis:** Aucun
- **Temps estimé:** 1-2 jours
- **Status:** 🔲 À faire

---

## 🎯 Objectif

Mettre en place l'infrastructure Docker pour l'environnement de développement et déploiement du projet maicivy. Cela inclut 4 services (backend Go, frontend Next.js, PostgreSQL, Redis) avec configuration réseau, persistence des données, health checks et variables d'environnement.

---

## 🏗️ Architecture

### Vue d'Ensemble

```
┌─────────────────────────────────────────────────────────────┐
│                   Docker Network (maicivy)                  │
├─────────────────────────────────────────────────────────────┤
│                                                               │
│  ┌──────────────┐    ┌──────────────┐    ┌──────────────┐   │
│  │   Frontend   │    │   Backend    │    │   Nginx      │   │
│  │  (Next.js)   │    │    (Fiber)   │    │  (Reverse    │   │
│  │  :3000       │    │    :8080     │    │   Proxy)     │   │
│  │              │    │              │    │  :80, :443   │   │
│  └──────────────┘    └──────────────┘    └──────────────┘   │
│         │                   │                    │            │
│         │                   │                    │            │
│  ┌──────────────┐    ┌──────────────┐                        │
│  │  PostgreSQL  │    │    Redis     │                        │
│  │   :5432      │    │   :6379      │                        │
│  │  postgres-db │    │  redis-cache │                        │
│  └──────────────┘    └──────────────┘                        │
│         │                   │                                 │
│    [postgres-data]      [redis-data]                         │
│         │                   │                                 │
└─────────┼───────────────────┼─────────────────────────────────┘
          │                   │
       [Volume]            [Volume]
```

### Design Decisions

1. **Docker Compose** : choisi pour simplicité environnement de dev/staging
2. **4 Services** : backend, frontend, PostgreSQL, Redis (Nginx sera dans Phase 6)
3. **Volumes** : persistence données PostgreSQL et Redis
4. **Health Checks** : détection services non-healthy
5. **Network** : bridge network custom pour isolation + communication inter-services
6. **Environnement** : fichier `.env.example` documenté pour configuration

---

## 📦 Dépendances

### Outils Requis

- Docker 20.10+
- Docker Compose 2.0+
- Bash (pour scripts de vérification)

### Pas de dépendances Go/NPM à installer

Les dépendances des services sont installées via leurs Dockerfiles respectifs.

---

## 🔨 Implémentation

### Étape 1: Créer le fichier docker-compose.yml

**Description:** Configuration des 4 services avec volumes, health checks et variables d'environnement.

**Fichier:** `docker-compose.yml`

```yaml
version: '3.8'

services:
  # PostgreSQL - Base de données principale
  postgres:
    image: postgres:16-alpine
    container_name: maicivy-postgres
    environment:
      POSTGRES_USER: ${DB_USER:-maicivy}
      POSTGRES_PASSWORD: ${DB_PASSWORD:-dev-password-change-in-prod}
      POSTGRES_DB: ${DB_NAME:-maicivy_db}
      POSTGRES_INITDB_ARGS: "--encoding=UTF8 --locale=en_US.UTF-8"
    ports:
      - "${DB_PORT:-5432}:5432"
    volumes:
      - postgres-data:/var/lib/postgresql/data
      - ./backend/migrations:/docker-entrypoint-initdb.d:ro
    networks:
      - maicivy
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U ${DB_USER:-maicivy} -d ${DB_NAME:-maicivy_db}"]
      interval: 10s
      timeout: 5s
      retries: 5
      start_period: 30s
    restart: unless-stopped
    command: |
      postgres
      -c max_connections=100
      -c shared_buffers=256MB
      -c effective_cache_size=1GB
      -c maintenance_work_mem=64MB
      -c checkpoint_completion_target=0.9
      -c wal_buffers=16MB
      -c default_statistics_target=100

  # Redis - Cache et sessions
  redis:
    image: redis:7-alpine
    container_name: maicivy-redis
    ports:
      - "${REDIS_PORT:-6379}:6379"
    volumes:
      - redis-data:/data
      - ./docker/redis/redis.conf:/usr/local/etc/redis/redis.conf:ro
    networks:
      - maicivy
    healthcheck:
      test: ["CMD", "redis-cli", "ping"]
      interval: 10s
      timeout: 5s
      retries: 5
      start_period: 30s
    restart: unless-stopped
    command: redis-server /usr/local/etc/redis/redis.conf

  # Backend - API Go
  backend:
    build:
      context: ./backend
      dockerfile: Dockerfile
      args:
        - BUILDKIT_INLINE_CACHE=1
    container_name: maicivy-backend
    environment:
      # Database
      DB_HOST: postgres
      DB_PORT: 5432
      DB_USER: ${DB_USER:-maicivy}
      DB_PASSWORD: ${DB_PASSWORD:-dev-password-change-in-prod}
      DB_NAME: ${DB_NAME:-maicivy_db}
      DB_SSL_MODE: ${DB_SSL_MODE:-disable}
      # Redis
      REDIS_HOST: redis
      REDIS_PORT: 6379
      REDIS_PASSWORD: ${REDIS_PASSWORD:-}
      # Server
      SERVER_PORT: 8080
      SERVER_ENV: ${SERVER_ENV:-development}
      # API Keys (to be set in .env)
      CLAUDE_API_KEY: ${CLAUDE_API_KEY}
      OPENAI_API_KEY: ${OPENAI_API_KEY}
    ports:
      - "${BACKEND_PORT:-8080}:8080"
    depends_on:
      postgres:
        condition: service_healthy
      redis:
        condition: service_healthy
    networks:
      - maicivy
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:8080/health"]
      interval: 10s
      timeout: 5s
      retries: 5
      start_period: 30s
    restart: unless-stopped
    volumes:
      - ./backend:/app
      - backend-cache:/go/pkg/mod

  # Frontend - Next.js
  frontend:
    build:
      context: ./frontend
      dockerfile: Dockerfile
      args:
        - BUILDKIT_INLINE_CACHE=1
    container_name: maicivy-frontend
    environment:
      NEXT_PUBLIC_API_URL: ${NEXT_PUBLIC_API_URL:-http://localhost:8080}
      NODE_ENV: ${NODE_ENV:-development}
    ports:
      - "${FRONTEND_PORT:-3000}:3000"
    depends_on:
      backend:
        condition: service_healthy
    networks:
      - maicivy
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:3000"]
      interval: 15s
      timeout: 5s
      retries: 5
      start_period: 45s
    restart: unless-stopped
    volumes:
      - ./frontend:/app
      - frontend-node-modules:/app/node_modules

volumes:
  postgres-data:
    driver: local
  redis-data:
    driver: local
  backend-cache:
    driver: local
  frontend-node-modules:
    driver: local

networks:
  maicivy:
    driver: bridge
```

**Explications:**

- **postgres** : PostgreSQL 16 avec paramètres optimisés pour dev
- **redis** : Redis 7 avec configuration externe
- **backend** : Build image Go, dépend de postgres et redis en santé
- **frontend** : Build image Next.js, dépend de backend en santé
- **Volumes nommés** : persistence données et cache npm/go
- **Health checks** : vérifie état services (curl pour web, pg_isready pour DB, redis-cli pour cache)
- **Networks** : bridge custom "maicivy" pour communication inter-services
- **Environment** : variables injectées depuis .env ou defaults

---

### Étape 2: Créer le fichier .env.example

**Description:** Template des variables d'environnement documentées.

**Fichier:** `.env.example`

```bash
# ============================================================================
# MAICIVY - Environment Configuration
# ============================================================================
# Copy this file to .env and update values for your environment
# IMPORTANT: NEVER commit .env to version control
# ============================================================================

# ============================================================================
# DATABASE - PostgreSQL Configuration
# ============================================================================

# Database connection credentials
DB_USER=maicivy
DB_PASSWORD=dev-password-change-in-prod
DB_NAME=maicivy_db
DB_HOST=postgres
DB_PORT=5432

# SSL mode: disable (dev), require (prod)
DB_SSL_MODE=disable

# Connection pool settings (optional, for production)
DB_MAX_CONNECTIONS=100
DB_MIN_CONNECTIONS=10
DB_CONNECTION_TIMEOUT=30

# ============================================================================
# REDIS - Cache & Session Store Configuration
# ============================================================================

REDIS_HOST=redis
REDIS_PORT=6379
REDIS_PASSWORD=
REDIS_DB=0

# Redis persistence: RDB snapshots interval (seconds)
REDIS_RDB_SAVE_INTERVAL=3600

# ============================================================================
# BACKEND - API Server Configuration
# ============================================================================

# Server settings
SERVER_PORT=8080
SERVER_ENV=development
# Values: development, staging, production

# Server timeouts (milliseconds)
SERVER_READ_TIMEOUT=30000
SERVER_WRITE_TIMEOUT=30000
SERVER_IDLE_TIMEOUT=120000

# CORS configuration (comma-separated URLs)
CORS_ALLOWED_ORIGINS=http://localhost:3000,http://localhost:8000

# ============================================================================
# AI SERVICES - API Keys
# ============================================================================

# Anthropic Claude API
CLAUDE_API_KEY=sk-ant-xxxxxxxxxxxxxxxxxxxxxxxxxxxxx

# OpenAI GPT-4 API
OPENAI_API_KEY=sk-xxxxxxxxxxxxxxxxxxxxxxxxxxxxx

# ============================================================================
# FRONTEND - Next.js Configuration
# ============================================================================

# API endpoint for frontend to call
NEXT_PUBLIC_API_URL=http://localhost:8080

# Node environment
NODE_ENV=development
# Values: development, production

# ============================================================================
# LOGGING & MONITORING
# ============================================================================

# Log level: debug, info, warn, error
LOG_LEVEL=info

# Prometheus metrics endpoint
METRICS_ENABLED=true
METRICS_PORT=9090

# ============================================================================
# RATE LIMITING
# ============================================================================

# Global rate limit (requests per minute per IP)
RATE_LIMIT_GLOBAL=100

# AI endpoint rate limit (requests per day per session)
RATE_LIMIT_AI=5

# Cooldown between AI generations (seconds)
AI_GENERATION_COOLDOWN=120

# ============================================================================
# ANALYTICS & TRACKING
# ============================================================================

# Track visitor information
TRACKING_ENABLED=true

# Data retention (days)
ANALYTICS_RETENTION_DAYS=90
ANALYTICS_AGGREGATION_RETENTION_DAYS=365

# ============================================================================
# OPTIONAL FEATURES
# ============================================================================

# GitHub API integration (for auto-importing projects)
GITHUB_API_TOKEN=ghp_xxxxxxxxxxxxxxxxxxxxxxxxxxxxx

# Clearbit API (for company info lookup)
CLEARBIT_API_KEY=xxxxxxxxxxxxxxxxxxxxxxxxxxxxx

# ============================================================================
# DEVELOPMENT
# ============================================================================

# Enable hot reload / debug mode
DEBUG_MODE=true

# Enable profiling
ENABLE_PROFILING=false

# ============================================================================
# PRODUCTION ONLY
# ============================================================================

# SSL/TLS certificates (for Nginx)
# SSL_CERT_PATH=/etc/letsencrypt/live/domain.com/fullchain.pem
# SSL_KEY_PATH=/etc/letsencrypt/live/domain.com/privkey.pem

# Database backup settings
# DB_BACKUP_ENABLED=true
# DB_BACKUP_SCHEDULE=0 2 * * *  # daily at 2 AM UTC

# ============================================================================
```

**Explications:**

- Groupes par section (Database, Redis, Backend, AI, Frontend, etc.)
- Defaults documentés avec commentaires
- Clés API à remplir
- Notes de sécurité (NEVER commit .env)
- Values pour dev/staging/prod clairement indiqués

---

### Étape 3: Créer configuration Redis

**Description:** Fichier de configuration Redis pour RDB snapshots et persistence.

**Fichier:** `docker/redis/redis.conf`

```bash
# ============================================================================
# MAICIVY Redis Configuration
# ============================================================================

# ============================================================================
# GENERAL
# ============================================================================

# Accept connections from all interfaces
bind 0.0.0.0

# TCP listen port
port 6379

# Close client connection after N seconds of inactivity
timeout 0

# TCP backlog setting
tcp-backlog 511

# Specify the logging verbosity level
# debug, verbose, notice, warning
loglevel notice

# ============================================================================
# PERSISTENCE
# ============================================================================

# Save the DB on disk (RDB snapshots)
# Format: save <seconds> <changes>
# Save if 900 seconds passed and at least 1 key changed
save 900 1
# Save if 300 seconds passed and at least 10 keys changed
save 300 10
# Save if 60 seconds passed and at least 10000 keys changed
save 60 10000

# Compress string objects using LZF when dump .rdb databases
rdbcompression yes

# The filename for the DB
dbfilename dump.rdb

# The working directory
dir /data

# AOF (Append-Only File) - alternative persistence mechanism
appendonly no
# appendonly yes  # Uncomment for AOF in production
# appendfsync everysec  # Write to disk every second

# ============================================================================
# MEMORY MANAGEMENT
# ============================================================================

# Set a memory usage limit
# maxmemory <bytes>
# maxmemory 256mb

# Memory eviction policy when max-memory reached
# Policy: noeviction, allkeys-lru, allkeys-lfu, volatile-lru, volatile-lfu
# maxmemory-policy noeviction

# ============================================================================
# SLOWLOG
# ============================================================================

# Log queries slower than specified microseconds
slowlog-log-slower-than 10000

# Max length of slowlog
slowlog-max-len 128

# ============================================================================
# REPLICATION
# ============================================================================

# Master/replica setup not needed for dev
# Uncomment and configure for replication in production
# replicaof <masterhost> <masterport>
# masterauth <password>

# ============================================================================
```

**Explications:**

- RDB snapshots : sauvegarde automatique selon conditions
- Persistence directory : `/data` (volume monté)
- Compression active pour économiser espace
- Slowlog pour debugging performances
- Commentaires pour AOF et replication en production

---

### Étape 4: Créer scripts de vérification

**Description:** Scripts bash pour vérifier état et santé de l'infrastructure.

**Fichier:** `scripts/health-check.sh`

```bash
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
```

**Explications:**

- Vérifie Docker et Docker Compose installés
- Contrôle l'état de chaque service avec retries
- Teste la connectivité (health endpoints, pg_isready, redis-cli)
- Affiche logs en cas de problème
- Affiche infos réseaux et volumes
- Exit code 0 (succès) ou 1 (échec) pour scripting

**Fichier:** `scripts/health-check.ps1` (pour Windows)

```powershell
# ============================================================================
# MAICIVY - Infrastructure Health Check Script (PowerShell - Windows)
# ============================================================================
# Usage: .\scripts\health-check.ps1
# ============================================================================

$ErrorActionPreference = "Stop"

# Configuration
$DOCKER_COMPOSE_FILE = "docker-compose.yml"
$REQUIRED_SERVICES = @("postgres", "redis", "backend", "frontend")
$MAX_RETRIES = 10
$RETRY_DELAY = 2

function Write-StatusOk {
    param([string]$message)
    Write-Host "✓ $message" -ForegroundColor Green
}

function Write-StatusError {
    param([string]$message)
    Write-Host "✗ $message" -ForegroundColor Red
}

function Write-StatusWarn {
    param([string]$message)
    Write-Host "⟳ $message" -ForegroundColor Yellow
}

function Write-Header {
    param([string]$message)
    Write-Host "========================================" -ForegroundColor Blue
    Write-Host $message -ForegroundColor Blue
    Write-Host "========================================" -ForegroundColor Blue
}

# Check Docker
if (-not (Get-Command docker -ErrorAction SilentlyContinue)) {
    Write-StatusError "Docker is not installed"
    exit 1
}
Write-StatusOk "Docker is installed"

# Check Docker Compose
if (-not (Get-Command docker-compose -ErrorAction SilentlyContinue)) {
    Write-StatusError "Docker Compose is not installed"
    exit 1
}
Write-StatusOk "Docker Compose is installed"

# Main checks
Write-Header "MAICIVY Infrastructure Health Check"

# Check docker-compose.yml exists
if (-not (Test-Path $DOCKER_COMPOSE_FILE)) {
    Write-StatusError "$DOCKER_COMPOSE_FILE not found"
    exit 1
}
Write-StatusOk "$DOCKER_COMPOSE_FILE found"

# Check service health
Write-Host "`nChecking service health..." -ForegroundColor Blue

$all_healthy = $true
foreach ($service in $REQUIRED_SERVICES) {
    $retry_count = 0
    $is_healthy = $false

    while ($retry_count -lt $MAX_RETRIES) {
        try {
            $health = docker inspect -f '{{.State.Health.Status}}' maicivy-$service 2>$null
            $state = docker inspect -f '{{.State.Running}}' maicivy-$service 2>$null

            if ($health -eq "healthy" -or $state -eq "true") {
                Write-StatusOk "$service"
                $is_healthy = $true
                break
            }
        }
        catch {
            # Ignore errors
        }

        Write-StatusWarn "$service not ready ($($retry_count + 1)/$MAX_RETRIES)"
        $retry_count++
        Start-Sleep -Seconds $RETRY_DELAY
    }

    if (-not $is_healthy) {
        Write-StatusError "$service"
        $all_healthy = $false
    }
}

# Show container status
Write-Host "`nContainer status:" -ForegroundColor Blue
docker-compose ps

if ($all_healthy) {
    Write-Host "`nAll services are healthy!" -ForegroundColor Green
    Write-Host "`nAccess points:" -ForegroundColor Blue
    Write-Host "  Frontend:  http://localhost:3000"
    Write-Host "  Backend:   http://localhost:8080"
    Write-Host "  PostgreSQL: localhost:5432"
    Write-Host "  Redis:     localhost:6379"
    exit 0
} else {
    Write-Host "`nSome services are not healthy" -ForegroundColor Red
    exit 1
}
```

---

### Étape 5: Créer Dockerfiles pour backend et frontend

**Description:** Dockerfiles multistage pour optimiser taille images.

**Fichier:** `backend/Dockerfile`

```dockerfile
# ============================================================================
# MAICIVY Backend - Go API Server
# ============================================================================
# Multistage build for optimized image size
# ============================================================================

# Stage 1: Build
FROM golang:1.22-alpine AS builder

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache git gcc musl-dev ca-certificates

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build binary
RUN CGO_ENABLED=1 GOOS=linux go build -a -installsuffix cgo \
    -ldflags="-s -w" -o backend ./cmd/main.go

# Stage 2: Runtime
FROM alpine:3.18

WORKDIR /app

# Install runtime dependencies
RUN apk add --no-cache ca-certificates curl postgresql-client

# Copy binary from builder
COPY --from=builder /app/backend .

# Create non-root user
RUN addgroup -g 1000 appuser && adduser -D -u 1000 -G appuser appuser
USER appuser

# Expose port
EXPOSE 8080

# Health check
HEALTHCHECK --interval=10s --timeout=5s --retries=5 --start-period=30s \
    CMD curl -f http://localhost:8080/health || exit 1

# Run
CMD ["./backend"]
```

**Explications:**

- Stage 1 (builder) : compile l'app Go
- Stage 2 (runtime) : image minimale Alpine
- Utilisateur non-root pour sécurité
- HEALTHCHECK pour Docker Compose

**Fichier:** `frontend/Dockerfile`

```dockerfile
# ============================================================================
# MAICIVY Frontend - Next.js Application
# ============================================================================
# Multistage build for optimized image size
# ============================================================================

# Stage 1: Dependencies
FROM node:20-alpine AS deps

WORKDIR /app

# Copy package files
COPY package.json package-lock.json ./

# Install dependencies
RUN npm ci

# Stage 2: Build
FROM node:20-alpine AS builder

WORKDIR /app

# Copy dependencies from deps stage
COPY --from=deps /app/node_modules ./node_modules

# Copy source code
COPY . .

# Build Next.js app
RUN npm run build

# Stage 3: Runtime
FROM node:20-alpine AS runner

WORKDIR /app

ENV NODE_ENV=production

# Copy public files
COPY --from=builder /app/public ./public

# Copy .next build
COPY --from=builder /app/.next/standalone ./
COPY --from=builder /app/.next/static ./.next/static

# Create non-root user
RUN addgroup -g 1000 appuser && adduser -D -u 1000 -G appuser appuser
USER appuser

# Expose port
EXPOSE 3000

ENV PORT=3000

# Health check
HEALTHCHECK --interval=15s --timeout=5s --retries=5 --start-period=45s \
    CMD wget --quiet --tries=1 --spider http://localhost:3000 || exit 1

# Run
CMD ["node", "server.js"]
```

**Explications:**

- Stage 1 (deps) : installe node_modules
- Stage 2 (builder) : build l'app Next.js
- Stage 3 (runner) : image légère sans sources
- Non-root user pour sécurité

---

## 🧪 Tests

### Tests de Configuration

**Description:** Vérifier que la configuration est correcte avant démarrage.

```bash
# Vérifier syntax docker-compose
docker-compose config

# Vérifier les volumes
docker volume ls | grep maicivy

# Vérifier les networks
docker network ls | grep maicivy
```

### Tests de Connectivité

```bash
# Lancer les services
docker-compose up -d

# Attendre que services soient sains
./scripts/health-check.sh  # Linux/Mac
# ou
.\scripts\health-check.ps1  # Windows

# Test PostgreSQL
docker exec maicivy-postgres psql -U maicivy -d maicivy_db -c "SELECT version();"

# Test Redis
docker exec maicivy-redis redis-cli INFO server

# Test Backend
curl http://localhost:8080/health

# Test Frontend
curl http://localhost:3000
```

### Commandes Utiles

```bash
# Afficher logs des services
docker-compose logs -f

# Logs d'un service spécifique
docker-compose logs -f backend
docker-compose logs -f postgres

# Accéder à une shell PostgreSQL
docker exec -it maicivy-postgres psql -U maicivy -d maicivy_db

# Accéder à Redis CLI
docker exec -it maicivy-redis redis-cli

# Arrêter les services
docker-compose down

# Arrêter et supprimer volumes
docker-compose down -v

# Rebuilder les images
docker-compose build --no-cache

# Redémarrer un service
docker-compose restart backend
```

---

## ⚠️ Points d'Attention

- **⚠️ Sécurité .env:** Jamais committer `.env` au répo. Utiliser `.env.example` comme template.
- **⚠️ Mots de passe dev:** Les defaults dans `.env.example` sont pour DEV SEULEMENT. Générer des mots de passe forts pour production.
- **⚠️ Volumes Docker:** Les volumes `postgres-data` et `redis-data` persisten entre `docker-compose down`. Utiliser `-v` pour les supprimer.
- **⚠️ Ports en conflit:** Vérifier que 3000, 5432, 6379, 8080 sont libres. Modifier dans `.env` si besoin.
- **⚠️ Memory limits:** Redis par défaut n'a pas de limite mémoire. Ajouter `maxmemory` en production.
- **⚠️ PostgreSQL locale:** Si PostgreSQL installée localement, conflit sur port 5432. Arrêter service local ou modifier port dans `.env`.
- **💡 Hot reload dev:** Backend et frontend volumes permettent hot reload. Modifier code et voir changements sans rebuild.
- **💡 Cleanup:** Exécuter `docker-compose down -v` avant un fresh start pour nettoyer.
- **💡 Production:** Pour production, utiliser images prébuilt (pas volumes source code), secrets management, monitoring.

---

## 📚 Ressources

- [Docker Compose Official Docs](https://docs.docker.com/compose/)
- [PostgreSQL Docker Image](https://hub.docker.com/_/postgres)
- [Redis Docker Image](https://hub.docker.com/_/redis)
- [Docker Networking Guide](https://docs.docker.com/network/)
- [Docker Best Practices](https://docs.docker.com/develop/develop-images/dockerfile_best-practices/)
- [Health Checks Documentation](https://docs.docker.com/compose/compose-file/#healthcheck)

---

## ✅ Checklist de Complétion

- [ ] Créer `docker-compose.yml` avec 4 services
- [ ] Créer `.env.example` documenté
- [ ] Créer `docker/redis/redis.conf`
- [ ] Créer `backend/Dockerfile` multistage
- [ ] Créer `frontend/Dockerfile` multistage
- [ ] Créer `scripts/health-check.sh` (Linux/Mac)
- [ ] Créer `scripts/health-check.ps1` (Windows)
- [ ] Vérifier syntax: `docker-compose config`
- [ ] Lancer services: `docker-compose up -d`
- [ ] Tester santé: `./scripts/health-check.sh`
- [ ] Tester PostgreSQL connection
- [ ] Tester Redis connection
- [ ] Documenter dans README (accès services, ports, etc.)
- [ ] Tester shutdown/restart gracieux
- [ ] Commit et push

---

**Dernière mise à jour:** 2025-12-08
**Auteur:** Alexi
