# 15. CI/CD & Deployment

## 📋 Métadonnées

- **Phase:** 6
- **Priorité:** 🟡 HAUTE
- **Complexité:** ⭐⭐⭐ (3/5)
- **Prérequis:** 14. INFRASTRUCTURE_PRODUCTION.md
- **Temps estimé:** 2-3 jours
- **Status:** 🔲 À faire

---

## 🎯 Objectif

Mettre en place un pipeline CI/CD complet automatisant les tests, la construction des images Docker, et le déploiement sur VPS. Utilisation de GitHub Actions pour l'intégration continue et le déploiement continu, avec support de Gitea comme repository principal.

**Résultats attendus:**
- Tests automatiques sur chaque push/PR
- Build automatique des images Docker
- Déploiement automatique sur le VPS en production
- Stratégie de rollback en cas d'échec
- Gestion sécurisée des secrets
- Notifications de statut des déploiements

---

## 🏗️ Architecture

### Vue d'Ensemble

```
┌─────────────────────────────────────────────────────────────────┐
│                         WORKFLOW CI/CD                          │
└─────────────────────────────────────────────────────────────────┘

┌──────────────┐
│ Developer    │
│ git push     │
└──────┬───────┘
       │
       ▼
┌──────────────────────────────────────────────────────────────┐
│ PHASE 1: CI - Tests & Quality Checks                        │
│ ┌──────────────┐  ┌──────────────┐  ┌──────────────┐       │
│ │ Backend      │  │ Frontend     │  │ Security     │       │
│ │ Tests        │  │ Tests        │  │ Scans        │       │
│ │ - go test    │  │ - npm test   │  │ - gosec      │       │
│ │ - golangci   │  │ - eslint     │  │ - npm audit  │       │
│ │ - go vet     │  │ - type check │  │ - trivy      │       │
│ └──────────────┘  └──────────────┘  └──────────────┘       │
└──────────────────────────────────────────────────────────────┘
       │
       ▼
┌──────────────────────────────────────────────────────────────┐
│ PHASE 2: Build - Docker Images                              │
│ ┌──────────────┐  ┌──────────────┐  ┌──────────────┐       │
│ │ Backend      │  │ Frontend     │  │ Nginx        │       │
│ │ Image        │  │ Image        │  │ Image        │       │
│ │ - Multi-stage│  │ - Next.js    │  │ - Config     │       │
│ │ - Cache      │  │ - Optimized  │  │ - SSL ready  │       │
│ └──────────────┘  └──────────────┘  └──────────────┘       │
│          │                │                  │               │
│          └────────────────┴──────────────────┘               │
│                           │                                  │
│                           ▼                                  │
│               Push to Docker Registry                        │
│               (GitHub Registry / Docker Hub)                 │
└──────────────────────────────────────────────────────────────┘
       │
       ▼
┌──────────────────────────────────────────────────────────────┐
│ PHASE 3: Deploy - Production VPS                            │
│ ┌──────────────┐  ┌──────────────┐  ┌──────────────┐       │
│ │ SSH Connect  │  │ Pull Images  │  │ Rolling      │       │
│ │ to VPS       │→ │ & Update     │→ │ Update       │       │
│ │              │  │ Compose      │  │ (zero down)  │       │
│ └──────────────┘  └──────────────┘  └──────────────┘       │
│                                              │               │
│                                              ▼               │
│                                    ┌──────────────┐          │
│                                    │ Health Check │          │
│                                    │ /health      │          │
│                                    └──────┬───────┘          │
│                                           │                  │
│                    ┌──────────────────────┴────────┐         │
│                    ▼                               ▼         │
│            ┌──────────────┐              ┌──────────────┐   │
│            │ Success      │              │ Rollback     │   │
│            │ Notify       │              │ to previous  │   │
│            └──────────────┘              └──────────────┘   │
└──────────────────────────────────────────────────────────────┘
```

### Design Decisions

**1. GitHub Actions vs Autres CI/CD:**
- ✅ GitHub Actions : intégration native GitHub, free tier généreux
- Gitea : repository principal mais utilise GitHub Actions pour CI/CD (mirror)
- Justification : simplicité, pas besoin d'infrastructure CI séparée

**2. Multi-stage Docker Builds:**
- Optimisation taille images (builder stage → runtime stage)
- Cache layers pour builds rapides
- Images de production minimales (distroless/alpine)

**3. Rolling Update Strategy:**
- `docker-compose up -d` avec timeout
- Health checks avant bascule complète
- Ancien container garde jusqu'à confirmation nouveau OK

**4. Secrets Management:**
- GitHub Secrets pour credentials sensibles
- Génération dynamique `.env` sur VPS (pas commité)
- SSH keys pour déploiement sécurisé

---

## 📦 Dépendances

### GitHub Actions Runners

- Ubuntu latest (GitHub-hosted runners)
- Docker pre-installed
- SSH client

### Outils Requis

**Sur VPS:**
```bash
# Docker & Docker Compose
docker --version  # 24.0+
docker-compose --version  # 2.20+

# SSH
ssh-keygen  # Génération clé déploiement
```

**Localement (développement):**
```bash
# Act (pour tester workflows localement)
brew install act  # macOS
# ou
curl https://raw.githubusercontent.com/nektos/act/master/install.sh | sudo bash
```

### Registres Docker

**Option 1 - GitHub Container Registry (recommandé):**
```bash
# Login
echo $GITHUB_TOKEN | docker login ghcr.io -u USERNAME --password-stdin

# Tag & Push
docker tag maicivy-backend:latest ghcr.io/username/maicivy-backend:latest
docker push ghcr.io/username/maicivy-backend:latest
```

**Option 2 - Docker Hub:**
```bash
# Login
docker login -u USERNAME

# Tag & Push
docker tag maicivy-backend:latest username/maicivy-backend:latest
docker push username/maicivy-backend:latest
```

---

## 🔨 Implémentation

### Étape 1: Structure Workflows GitHub Actions

**Description:** Créer les fichiers workflow dans `.github/workflows/`

**Structure:**

```
.github/
└── workflows/
    ├── ci.yml              # Tests & linting
    ├── deploy.yml          # Déploiement production
    ├── backup.yml          # Backup hebdomadaire
    └── security-scan.yml   # Scan sécurité quotidien
```

**Fichiers à créer:**

```bash
mkdir -p .github/workflows
```

---

### Étape 2: Workflow CI - Tests Automatisés

**Description:** Workflow exécuté sur chaque push et pull request pour valider le code

**Code: `.github/workflows/ci.yml`**

```yaml
name: CI - Tests & Quality

on:
  push:
    branches: [ main, develop ]
  pull_request:
    branches: [ main, develop ]

env:
  GO_VERSION: '1.21'
  NODE_VERSION: '20'

jobs:
  # ============================================
  # JOB 1: Backend Tests
  # ============================================
  backend-tests:
    name: Backend Tests (Go)
    runs-on: ubuntu-latest

    services:
      postgres:
        image: postgres:16-alpine
        env:
          POSTGRES_USER: testuser
          POSTGRES_PASSWORD: testpass
          POSTGRES_DB: testdb
        options: >-
          --health-cmd pg_isready
          --health-interval 10s
          --health-timeout 5s
          --health-retries 5
        ports:
          - 5432:5432

      redis:
        image: redis:7-alpine
        options: >-
          --health-cmd "redis-cli ping"
          --health-interval 10s
          --health-timeout 5s
          --health-retries 5
        ports:
          - 6379:6379

    steps:
      - name: Checkout code
        uses: actions/checkout@v4

      - name: Setup Go
        uses: actions/setup-go@v5
        with:
          go-version: ${{ env.GO_VERSION }}
          cache: true
          cache-dependency-path: backend/go.sum

      - name: Install dependencies
        working-directory: ./backend
        run: go mod download

      - name: Run go vet
        working-directory: ./backend
        run: go vet ./...

      - name: Run golangci-lint
        uses: golangci/golangci-lint-action@v3
        with:
          version: latest
          working-directory: ./backend
          args: --timeout=5m

      - name: Run tests
        working-directory: ./backend
        env:
          DATABASE_URL: postgres://testuser:testpass@localhost:5432/testdb?sslmode=disable
          REDIS_URL: redis://localhost:6379
        run: |
          go test -v -race -coverprofile=coverage.out -covermode=atomic ./...
          go tool cover -func=coverage.out

      - name: Upload coverage to Codecov
        uses: codecov/codecov-action@v3
        with:
          file: ./backend/coverage.out
          flags: backend
          fail_ci_if_error: false

  # ============================================
  # JOB 2: Frontend Tests
  # ============================================
  frontend-tests:
    name: Frontend Tests (Next.js)
    runs-on: ubuntu-latest

    steps:
      - name: Checkout code
        uses: actions/checkout@v4

      - name: Setup Node.js
        uses: actions/setup-node@v4
        with:
          node-version: ${{ env.NODE_VERSION }}
          cache: 'npm'
          cache-dependency-path: frontend/package-lock.json

      - name: Install dependencies
        working-directory: ./frontend
        run: npm ci

      - name: Run ESLint
        working-directory: ./frontend
        run: npm run lint

      - name: Type checking
        working-directory: ./frontend
        run: npm run type-check

      - name: Run tests
        working-directory: ./frontend
        run: npm test -- --coverage

      - name: Upload coverage to Codecov
        uses: codecov/codecov-action@v3
        with:
          file: ./frontend/coverage/coverage-final.json
          flags: frontend
          fail_ci_if_error: false

  # ============================================
  # JOB 3: Security Scans
  # ============================================
  security-scan:
    name: Security Scanning
    runs-on: ubuntu-latest

    steps:
      - name: Checkout code
        uses: actions/checkout@v4

      - name: Run Gosec Security Scanner (Go)
        uses: securego/gosec@master
        with:
          args: ./backend/...

      - name: Run npm audit (Frontend)
        working-directory: ./frontend
        run: npm audit --audit-level=moderate
        continue-on-error: true

      - name: Run Trivy vulnerability scanner
        uses: aquasecurity/trivy-action@master
        with:
          scan-type: 'fs'
          scan-ref: '.'
          format: 'sarif'
          output: 'trivy-results.sarif'

      - name: Upload Trivy results to GitHub Security
        uses: github/codeql-action/upload-sarif@v2
        with:
          sarif_file: 'trivy-results.sarif'

  # ============================================
  # JOB 4: Build Validation
  # ============================================
  build-validation:
    name: Build Validation (Docker)
    runs-on: ubuntu-latest
    needs: [backend-tests, frontend-tests]

    steps:
      - name: Checkout code
        uses: actions/checkout@v4

      - name: Set up Docker Buildx
        uses: docker/setup-buildx-action@v3

      - name: Build Backend Image
        uses: docker/build-push-action@v5
        with:
          context: ./backend
          file: ./backend/Dockerfile
          push: false
          tags: maicivy-backend:test
          cache-from: type=gha
          cache-to: type=gha,mode=max

      - name: Build Frontend Image
        uses: docker/build-push-action@v5
        with:
          context: ./frontend
          file: ./frontend/Dockerfile
          push: false
          tags: maicivy-frontend:test
          cache-from: type=gha
          cache-to: type=gha,mode=max
```

**Explications:**

1. **Triggers:** S'exécute sur push (main/develop) et pull requests
2. **Services:** PostgreSQL et Redis pour tests d'intégration
3. **Jobs parallèles:** Backend, Frontend, Security scans en parallèle (rapidité)
4. **Coverage:** Upload vers Codecov pour tracking couverture
5. **Cache:** Utilisation cache GitHub pour dépendances (Go modules, npm)
6. **Security:** Gosec (Go), npm audit, Trivy (vulnérabilités images)

---

### Étape 3: Workflow Deploy - Déploiement Production

**Description:** Workflow de déploiement automatique vers VPS sur push main

**Code: `.github/workflows/deploy.yml`**

```yaml
name: Deploy to Production

on:
  push:
    branches: [ main ]
  workflow_dispatch:  # Permet déclenchement manuel

env:
  REGISTRY: ghcr.io
  IMAGE_PREFIX: ${{ github.repository_owner }}/maicivy

jobs:
  # ============================================
  # JOB 1: Build & Push Docker Images
  # ============================================
  build-and-push:
    name: Build & Push Images
    runs-on: ubuntu-latest
    permissions:
      contents: read
      packages: write

    outputs:
      backend-tag: ${{ steps.meta-backend.outputs.tags }}
      frontend-tag: ${{ steps.meta-frontend.outputs.tags }}
      nginx-tag: ${{ steps.meta-nginx.outputs.tags }}

    steps:
      - name: Checkout code
        uses: actions/checkout@v4

      - name: Set up Docker Buildx
        uses: docker/setup-buildx-action@v3

      - name: Log in to GitHub Container Registry
        uses: docker/login-action@v3
        with:
          registry: ${{ env.REGISTRY }}
          username: ${{ github.actor }}
          password: ${{ secrets.GITHUB_TOKEN }}

      # ============================================
      # Backend Image
      # ============================================
      - name: Extract metadata (Backend)
        id: meta-backend
        uses: docker/metadata-action@v5
        with:
          images: ${{ env.REGISTRY }}/${{ env.IMAGE_PREFIX }}-backend
          tags: |
            type=sha,prefix=,format=short
            type=raw,value=latest,enable={{is_default_branch}}

      - name: Build and push Backend
        uses: docker/build-push-action@v5
        with:
          context: ./backend
          file: ./backend/Dockerfile
          push: true
          tags: ${{ steps.meta-backend.outputs.tags }}
          labels: ${{ steps.meta-backend.outputs.labels }}
          cache-from: type=gha
          cache-to: type=gha,mode=max
          build-args: |
            BUILD_DATE=${{ github.event.head_commit.timestamp }}
            VCS_REF=${{ github.sha }}

      # ============================================
      # Frontend Image
      # ============================================
      - name: Extract metadata (Frontend)
        id: meta-frontend
        uses: docker/metadata-action@v5
        with:
          images: ${{ env.REGISTRY }}/${{ env.IMAGE_PREFIX }}-frontend
          tags: |
            type=sha,prefix=,format=short
            type=raw,value=latest,enable={{is_default_branch}}

      - name: Build and push Frontend
        uses: docker/build-push-action@v5
        with:
          context: ./frontend
          file: ./frontend/Dockerfile
          push: true
          tags: ${{ steps.meta-frontend.outputs.tags }}
          labels: ${{ steps.meta-frontend.outputs.labels }}
          cache-from: type=gha
          cache-to: type=gha,mode=max
          build-args: |
            BUILD_DATE=${{ github.event.head_commit.timestamp }}
            VCS_REF=${{ github.sha }}

      # ============================================
      # Nginx Image
      # ============================================
      - name: Extract metadata (Nginx)
        id: meta-nginx
        uses: docker/metadata-action@v5
        with:
          images: ${{ env.REGISTRY }}/${{ env.IMAGE_PREFIX }}-nginx
          tags: |
            type=sha,prefix=,format=short
            type=raw,value=latest,enable={{is_default_branch}}

      - name: Build and push Nginx
        uses: docker/build-push-action@v5
        with:
          context: ./docker/nginx
          file: ./docker/nginx/Dockerfile
          push: true
          tags: ${{ steps.meta-nginx.outputs.tags }}
          labels: ${{ steps.meta-nginx.outputs.labels }}
          cache-from: type=gha
          cache-to: type=gha,mode=max

  # ============================================
  # JOB 2: Deploy to VPS
  # ============================================
  deploy:
    name: Deploy to VPS
    runs-on: ubuntu-latest
    needs: build-and-push
    environment:
      name: production
      url: https://maicivy.yourdomain.com

    steps:
      - name: Checkout code
        uses: actions/checkout@v4

      - name: Setup SSH
        run: |
          mkdir -p ~/.ssh
          echo "${{ secrets.VPS_SSH_KEY }}" > ~/.ssh/deploy_key
          chmod 600 ~/.ssh/deploy_key
          ssh-keyscan -H ${{ secrets.VPS_HOST }} >> ~/.ssh/known_hosts

      - name: Create deployment script
        run: |
          cat > deploy.sh << 'EOF'
          #!/bin/bash
          set -e

          # Variables
          DEPLOY_DIR="/opt/maicivy"
          REGISTRY="${{ env.REGISTRY }}"
          IMAGE_PREFIX="${{ env.IMAGE_PREFIX }}"
          COMMIT_SHA="${{ github.sha }}"

          # Colors for output
          GREEN='\033[0;32m'
          RED='\033[0;31m'
          NC='\033[0m' # No Color

          echo -e "${GREEN}[1/6] Navigating to deployment directory...${NC}"
          cd $DEPLOY_DIR

          echo -e "${GREEN}[2/6] Logging into GitHub Container Registry...${NC}"
          echo "${{ secrets.GITHUB_TOKEN }}" | docker login $REGISTRY -u ${{ github.actor }} --password-stdin

          echo -e "${GREEN}[3/6] Pulling latest images...${NC}"
          docker pull ${REGISTRY}/${IMAGE_PREFIX}-backend:latest
          docker pull ${REGISTRY}/${IMAGE_PREFIX}-frontend:latest
          docker pull ${REGISTRY}/${IMAGE_PREFIX}-nginx:latest

          echo -e "${GREEN}[4/6] Backing up current deployment...${NC}"
          docker-compose ps -q > .last_deployment || true

          echo -e "${GREEN}[5/6] Deploying new version (rolling update)...${NC}"
          docker-compose up -d --no-deps --build

          echo -e "${GREEN}[6/6] Waiting for services to be healthy...${NC}"
          sleep 10

          # Health check
          if curl -f http://localhost:8080/health > /dev/null 2>&1; then
            echo -e "${GREEN}✅ Deployment successful! Backend is healthy.${NC}"

            # Cleanup old images
            docker image prune -af --filter "until=24h"

            exit 0
          else
            echo -e "${RED}❌ Health check failed! Rolling back...${NC}"

            # Rollback
            if [ -f .last_deployment ]; then
              docker-compose up -d $(cat .last_deployment)
              echo -e "${RED}Rollback completed.${NC}"
            fi

            exit 1
          fi
          EOF

          chmod +x deploy.sh

      - name: Execute deployment
        run: |
          scp -i ~/.ssh/deploy_key deploy.sh ${{ secrets.VPS_USER }}@${{ secrets.VPS_HOST }}:/tmp/
          ssh -i ~/.ssh/deploy_key ${{ secrets.VPS_USER }}@${{ secrets.VPS_HOST }} 'bash /tmp/deploy.sh'

      - name: Cleanup
        if: always()
        run: |
          rm -f ~/.ssh/deploy_key
          rm -f deploy.sh

      - name: Notify success
        if: success()
        uses: appleboy/discord-action@master
        with:
          webhook_id: ${{ secrets.DISCORD_WEBHOOK_ID }}
          webhook_token: ${{ secrets.DISCORD_WEBHOOK_TOKEN }}
          color: "#00FF00"
          message: |
            ✅ **Deployment Successful**

            **Repository:** ${{ github.repository }}
            **Branch:** ${{ github.ref_name }}
            **Commit:** ${{ github.sha }}
            **Author:** ${{ github.actor }}
            **Message:** ${{ github.event.head_commit.message }}

            🚀 Live at: https://maicivy.yourdomain.com

      - name: Notify failure
        if: failure()
        uses: appleboy/discord-action@master
        with:
          webhook_id: ${{ secrets.DISCORD_WEBHOOK_ID }}
          webhook_token: ${{ secrets.DISCORD_WEBHOOK_TOKEN }}
          color: "#FF0000"
          message: |
            ❌ **Deployment Failed**

            **Repository:** ${{ github.repository }}
            **Branch:** ${{ github.ref_name }}
            **Commit:** ${{ github.sha }}
            **Author:** ${{ github.actor }}

            Check logs: ${{ github.server_url }}/${{ github.repository }}/actions/runs/${{ github.run_id }}
```

**Explications:**

1. **Two-job strategy:** Build images → Deploy to VPS (séquentiel)
2. **GitHub Container Registry:** Stockage images (alternative: Docker Hub)
3. **Rolling Update:** `docker-compose up -d` avec backup container IDs
4. **Health Check:** Vérification `/health` endpoint avant validation
5. **Rollback automatique:** Si health check échoue, restaure anciens containers
6. **Notifications:** Discord webhook (success/failure)

---

### Étape 4: Script de Déploiement VPS

**Description:** Script shell exécuté sur le VPS pour gérer le déploiement

**Code: `scripts/deploy.sh`** (version standalone pour déploiement manuel)

```bash
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
```

**Explications:**

1. **Safe deployment:** Backup avant chaque déploiement
2. **Rollback automatique:** Si health check échoue
3. **Options flexibles:** `--rollback`, `--backup`, `--dry-run`
4. **Health checks robustes:** Retry 30 fois avec timeout
5. **Cleanup:** Suppression automatique vieilles images

---

### Étape 5: Configuration Secrets GitHub

**Description:** Configurer les secrets nécessaires dans GitHub repository

**Secrets à ajouter (Settings → Secrets and variables → Actions):**

```bash
# VPS Access
VPS_HOST=your-vps-ip-or-domain
VPS_USER=deploy
VPS_SSH_KEY=<contenu de la clé privée SSH>

# Docker Registry (si Docker Hub au lieu de GHCR)
DOCKER_USERNAME=your-docker-username
DOCKER_PASSWORD=your-docker-password

# Notifications (optionnel)
DISCORD_WEBHOOK_ID=your-webhook-id
DISCORD_WEBHOOK_TOKEN=your-webhook-token

# Application Secrets (pour génération .env sur VPS)
CLAUDE_API_KEY=your-claude-api-key
OPENAI_API_KEY=your-openai-api-key
JWT_SECRET=your-jwt-secret
DATABASE_PASSWORD=your-db-password
REDIS_PASSWORD=your-redis-password
```

**Setup SSH Key pour déploiement:**

```bash
# Sur votre machine locale
ssh-keygen -t ed25519 -C "github-actions-deploy" -f ~/.ssh/maicivy_deploy
# Ne pas mettre de passphrase (pour automation)

# Copier la clé publique sur le VPS
ssh-copy-id -i ~/.ssh/maicivy_deploy.pub deploy@your-vps-ip

# Tester la connexion
ssh -i ~/.ssh/maicivy_deploy deploy@your-vps-ip

# Ajouter la clé privée aux GitHub Secrets
cat ~/.ssh/maicivy_deploy
# Copier tout le contenu dans GitHub Secret: VPS_SSH_KEY
```

**Créer utilisateur deploy sur VPS:**

```bash
# SSH sur VPS en tant que root
ssh root@your-vps-ip

# Créer utilisateur deploy
useradd -m -s /bin/bash deploy

# Ajouter au groupe docker
usermod -aG docker deploy

# Créer répertoire déploiement
mkdir -p /opt/maicivy
chown deploy:deploy /opt/maicivy

# Setup sudoers (pour docker-compose)
echo "deploy ALL=(ALL) NOPASSWD: /usr/bin/docker-compose" >> /etc/sudoers.d/deploy
```

---

### Étape 6: Workflow Backup Automatisé

**Description:** Backup hebdomadaire automatique des données PostgreSQL et Redis

**Code: `.github/workflows/backup.yml`**

```yaml
name: Weekly Backup

on:
  schedule:
    # Every Sunday at 2 AM UTC
    - cron: '0 2 * * 0'
  workflow_dispatch:  # Manual trigger

jobs:
  backup:
    name: Backup Database & Redis
    runs-on: ubuntu-latest

    steps:
      - name: Setup SSH
        run: |
          mkdir -p ~/.ssh
          echo "${{ secrets.VPS_SSH_KEY }}" > ~/.ssh/deploy_key
          chmod 600 ~/.ssh/deploy_key
          ssh-keyscan -H ${{ secrets.VPS_HOST }} >> ~/.ssh/known_hosts

      - name: Execute backup script
        run: |
          ssh -i ~/.ssh/deploy_key ${{ secrets.VPS_USER }}@${{ secrets.VPS_HOST }} << 'EOF'

          # Variables
          BACKUP_DIR="/opt/maicivy/backups"
          TIMESTAMP=$(date +%Y%m%d_%H%M%S)

          # Create backup directory
          mkdir -p $BACKUP_DIR

          # PostgreSQL backup
          docker exec maicivy-postgres pg_dump -U postgres maicivy > $BACKUP_DIR/postgres_$TIMESTAMP.sql
          gzip $BACKUP_DIR/postgres_$TIMESTAMP.sql

          # Redis backup (RDB snapshot)
          docker exec maicivy-redis redis-cli SAVE
          docker cp maicivy-redis:/data/dump.rdb $BACKUP_DIR/redis_$TIMESTAMP.rdb

          # Cleanup old backups (keep last 4 weeks)
          find $BACKUP_DIR -name "*.sql.gz" -mtime +28 -delete
          find $BACKUP_DIR -name "*.rdb" -mtime +28 -delete

          echo "✅ Backup completed: $BACKUP_DIR/*_$TIMESTAMP.*"

          EOF

      - name: Notify backup completion
        if: success()
        uses: appleboy/discord-action@master
        with:
          webhook_id: ${{ secrets.DISCORD_WEBHOOK_ID }}
          webhook_token: ${{ secrets.DISCORD_WEBHOOK_TOKEN }}
          color: "#0099FF"
          message: |
            💾 **Weekly Backup Completed**

            **Timestamp:** $(date)
            **Status:** Success

            Database and Redis backups created successfully.
```

---

### Étape 7: Dockerfiles Optimisés pour Production

**Description:** Dockerfiles multi-stage optimisés pour images légères

**Backend Dockerfile: `backend/Dockerfile`**

```dockerfile
# ============================================
# Stage 1: Builder
# ============================================
FROM golang:1.21-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git gcc musl-dev

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build binary
RUN CGO_ENABLED=1 GOOS=linux go build \
    -ldflags="-w -s -X main.version=${VCS_REF} -X main.buildDate=${BUILD_DATE}" \
    -a -installsuffix cgo \
    -o maicivy-backend \
    ./cmd/main.go

# ============================================
# Stage 2: Runtime
# ============================================
FROM alpine:latest

# Install runtime dependencies
RUN apk add --no-cache ca-certificates tzdata

# Create non-root user
RUN addgroup -g 1000 app && \
    adduser -D -u 1000 -G app app

WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/maicivy-backend .

# Copy migrations (if needed)
COPY --from=builder /app/migrations ./migrations

# Change ownership
RUN chown -R app:app /app

USER app

EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=40s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

CMD ["./maicivy-backend"]
```

**Frontend Dockerfile: `frontend/Dockerfile`**

```dockerfile
# ============================================
# Stage 1: Dependencies
# ============================================
FROM node:20-alpine AS deps

WORKDIR /app

# Install dependencies based on lock file
COPY package.json package-lock.json ./
RUN npm ci --only=production

# ============================================
# Stage 2: Builder
# ============================================
FROM node:20-alpine AS builder

WORKDIR /app

COPY package.json package-lock.json ./
RUN npm ci

COPY . .

# Build Next.js app
ENV NEXT_TELEMETRY_DISABLED 1
RUN npm run build

# ============================================
# Stage 3: Runner
# ============================================
FROM node:20-alpine AS runner

WORKDIR /app

ENV NODE_ENV production
ENV NEXT_TELEMETRY_DISABLED 1

# Create non-root user
RUN addgroup -g 1001 -S nodejs && \
    adduser -S nextjs -u 1001

# Copy necessary files
COPY --from=builder /app/public ./public
COPY --from=builder /app/.next/standalone ./
COPY --from=builder /app/.next/static ./.next/static

USER nextjs

EXPOSE 3000

ENV PORT 3000

CMD ["node", "server.js"]
```

**Nginx Dockerfile: `docker/nginx/Dockerfile`**

```dockerfile
FROM nginx:1.25-alpine

# Remove default config
RUN rm /etc/nginx/conf.d/default.conf

# Copy custom config
COPY nginx.conf /etc/nginx/nginx.conf
COPY conf.d/ /etc/nginx/conf.d/

# Create SSL directory (certificates will be mounted)
RUN mkdir -p /etc/nginx/ssl

# Health check
HEALTHCHECK --interval=30s --timeout=3s --retries=3 \
    CMD wget --quiet --tries=1 --spider http://localhost:80/health || exit 1

EXPOSE 80 443

CMD ["nginx", "-g", "daemon off;"]
```

---

### Étape 8: Docker Compose pour Production

**Description:** Configuration Docker Compose optimisée pour production

**Code: `docker-compose.yml`** (version production)

```yaml
version: '3.9'

services:
  # ============================================
  # Backend (Go + Fiber)
  # ============================================
  backend:
    image: ghcr.io/${GITHUB_REPOSITORY_OWNER}/maicivy-backend:latest
    container_name: maicivy-backend
    restart: unless-stopped
    environment:
      - DATABASE_URL=postgres://${POSTGRES_USER}:${POSTGRES_PASSWORD}@postgres:5432/${POSTGRES_DB}?sslmode=disable
      - REDIS_URL=redis://redis:6379
      - CLAUDE_API_KEY=${CLAUDE_API_KEY}
      - OPENAI_API_KEY=${OPENAI_API_KEY}
      - JWT_SECRET=${JWT_SECRET}
      - LOG_LEVEL=info
      - ENVIRONMENT=production
    depends_on:
      postgres:
        condition: service_healthy
      redis:
        condition: service_healthy
    networks:
      - maicivy-network
    healthcheck:
      test: ["CMD", "wget", "--quiet", "--tries=1", "--spider", "http://localhost:8080/health"]
      interval: 30s
      timeout: 10s
      retries: 3
      start_period: 40s

  # ============================================
  # Frontend (Next.js)
  # ============================================
  frontend:
    image: ghcr.io/${GITHUB_REPOSITORY_OWNER}/maicivy-frontend:latest
    container_name: maicivy-frontend
    restart: unless-stopped
    environment:
      - NEXT_PUBLIC_API_URL=http://backend:8080
    depends_on:
      - backend
    networks:
      - maicivy-network
    healthcheck:
      test: ["CMD", "wget", "--quiet", "--tries=1", "--spider", "http://localhost:3000"]
      interval: 30s
      timeout: 10s
      retries: 3

  # ============================================
  # Nginx (Reverse Proxy)
  # ============================================
  nginx:
    image: ghcr.io/${GITHUB_REPOSITORY_OWNER}/maicivy-nginx:latest
    container_name: maicivy-nginx
    restart: unless-stopped
    ports:
      - "80:80"
      - "443:443"
    volumes:
      - ./ssl:/etc/nginx/ssl:ro
      - ./logs/nginx:/var/log/nginx
    depends_on:
      - backend
      - frontend
    networks:
      - maicivy-network
    healthcheck:
      test: ["CMD", "wget", "--quiet", "--tries=1", "--spider", "http://localhost:80/health"]
      interval: 30s
      timeout: 5s
      retries: 3

  # ============================================
  # PostgreSQL
  # ============================================
  postgres:
    image: postgres:16-alpine
    container_name: maicivy-postgres
    restart: unless-stopped
    environment:
      - POSTGRES_USER=${POSTGRES_USER}
      - POSTGRES_PASSWORD=${POSTGRES_PASSWORD}
      - POSTGRES_DB=${POSTGRES_DB}
    volumes:
      - postgres-data:/var/lib/postgresql/data
      - ./backups/postgres:/backups
    networks:
      - maicivy-network
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U ${POSTGRES_USER}"]
      interval: 10s
      timeout: 5s
      retries: 5

  # ============================================
  # Redis
  # ============================================
  redis:
    image: redis:7-alpine
    container_name: maicivy-redis
    restart: unless-stopped
    command: redis-server --appendonly yes --requirepass ${REDIS_PASSWORD}
    volumes:
      - redis-data:/data
      - ./backups/redis:/backups
    networks:
      - maicivy-network
    healthcheck:
      test: ["CMD", "redis-cli", "ping"]
      interval: 10s
      timeout: 5s
      retries: 5

  # ============================================
  # Prometheus (Monitoring)
  # ============================================
  prometheus:
    image: prom/prometheus:latest
    container_name: maicivy-prometheus
    restart: unless-stopped
    volumes:
      - ./monitoring/prometheus/prometheus.yml:/etc/prometheus/prometheus.yml:ro
      - prometheus-data:/prometheus
    command:
      - '--config.file=/etc/prometheus/prometheus.yml'
      - '--storage.tsdb.path=/prometheus'
      - '--storage.tsdb.retention.time=15d'
    networks:
      - maicivy-network
    ports:
      - "9090:9090"

  # ============================================
  # Grafana (Dashboards)
  # ============================================
  grafana:
    image: grafana/grafana:latest
    container_name: maicivy-grafana
    restart: unless-stopped
    environment:
      - GF_SECURITY_ADMIN_PASSWORD=${GRAFANA_PASSWORD}
      - GF_SERVER_ROOT_URL=https://maicivy.yourdomain.com/grafana
      - GF_AUTH_ANONYMOUS_ENABLED=true
      - GF_AUTH_ANONYMOUS_ORG_ROLE=Viewer
    volumes:
      - grafana-data:/var/lib/grafana
      - ./monitoring/grafana/dashboards:/etc/grafana/provisioning/dashboards:ro
    networks:
      - maicivy-network
    ports:
      - "3001:3000"
    depends_on:
      - prometheus

networks:
  maicivy-network:
    driver: bridge

volumes:
  postgres-data:
  redis-data:
  prometheus-data:
  grafana-data:
```

---

## 🧪 Tests

### Tests Locaux Workflows

**Test avec Act (simule GitHub Actions localement):**

```bash
# Install Act
brew install act  # macOS
# ou
curl https://raw.githubusercontent.com/nektos/act/master/install.sh | sudo bash

# Tester workflow CI
act -j backend-tests

# Tester workflow deploy (dry-run)
act -j deploy --secret-file .secrets
```

**Fichier `.secrets` (pour tests locaux Act):**

```env
VPS_HOST=localhost
VPS_USER=testuser
VPS_SSH_KEY=fake-key-for-testing
GITHUB_TOKEN=fake-token
```

### Tests de Déploiement

**Test script de déploiement en dry-run:**

```bash
# Sur VPS
./scripts/deploy.sh --dry-run

# Test avec backup seulement
./scripts/deploy.sh --backup

# Test rollback
./scripts/deploy.sh --rollback
```

### Tests Health Checks

**Test endpoints santé:**

```bash
# Backend health
curl http://localhost:8080/health

# Expected response:
# {"status":"ok","services":{"database":"up","redis":"up"}}

# Frontend health
curl http://localhost:3000

# Nginx health
curl http://localhost:80/health
```

---

## ⚠️ Points d'Attention

### 1. Secrets Management

⚠️ **CRITICAL:** Ne JAMAIS commiter de secrets dans le code
- Utiliser GitHub Secrets pour credentials
- Générer `.env` dynamiquement sur VPS
- Rotation régulière des secrets (API keys, passwords)

💡 **Best Practice:** Utiliser HashiCorp Vault pour secrets en production avancée

### 2. SSH Key Security

⚠️ **Piège:** Clé SSH avec passphrase bloque automation
- Créer une clé dédiée SANS passphrase pour CI/CD
- Limiter les permissions (utilisateur `deploy` avec droits restreints)
- Restreindre accès SSH par IP (GitHub Actions IPs)

### 3. Docker Registry Limits

⚠️ **Docker Hub Rate Limits:**
- Free tier: 100 pulls / 6h (anonyme)
- Authenticated: 200 pulls / 6h
- Solution: Utiliser GitHub Container Registry (ghcr.io) - gratuit et illimité pour public repos

### 4. Rolling Updates

⚠️ **Zero-downtime deployment:**
- `docker-compose up -d` ne garantit PAS toujours zero downtime
- Utiliser health checks obligatoires
- Considérer Blue/Green deployment pour garantie 100%

💡 **Alternative:** Docker Swarm ou Kubernetes pour orchestration avancée

### 5. Database Migrations

⚠️ **Critical:** Gérer migrations AVANT déploiement nouvelle version
- Inclure step migration dans workflow deploy
- Tester migrations sur staging d'abord
- Avoir plan rollback migrations (down migrations)

```yaml
# Ajouter dans deploy.yml avant deploy_services
- name: Run database migrations
  run: |
    ssh -i ~/.ssh/deploy_key ${{ secrets.VPS_USER }}@${{ secrets.VPS_HOST }} \
    'docker exec maicivy-backend ./maicivy-backend migrate up'
```

### 6. Disk Space Management

⚠️ **Docker accumule images/volumes rapidement:**
- Cleanup automatique dans script deploy
- Monitoring disk usage (Prometheus + alert si > 80%)
- Cron job cleanup hebdomadaire

```bash
# Cron job sur VPS
0 3 * * 0 docker system prune -af --volumes --filter "until=168h"
```

### 7. Logs Rotation

⚠️ **Logs peuvent remplir disque:**

```yaml
# Ajouter dans docker-compose.yml pour chaque service
logging:
  driver: "json-file"
  options:
    max-size: "10m"
    max-file: "3"
```

### 8. Build Cache Invalidation

⚠️ **Cache GitHub Actions peut causer builds obsolètes:**
- Utiliser `cache-from/cache-to type=gha` (GitHub Actions cache)
- Invalidation automatique sur changement dependencies
- Option manuelle: workflow dispatch avec `--no-cache`

### 9. Environment Variables Order

⚠️ **Docker Compose résolution variables:**
1. `.env` file dans même directory que `docker-compose.yml`
2. Variables d'environnement shell
3. Defaults dans `docker-compose.yml`

💡 **Solution:** Générer `.env` depuis GitHub Secrets dans script deploy

### 10. Health Check Timeouts

⚠️ **Services lents à démarrer:**
- Ajuster `start_period` dans healthcheck (ex: 60s pour backend avec migrations)
- Retry logic suffisant (3-5 retries)
- Timeout adapté selon service (backend > frontend)

---

## 📚 Ressources

### Documentation Officielle

- [GitHub Actions](https://docs.github.com/en/actions) - Workflows CI/CD
- [Docker Multi-stage Builds](https://docs.docker.com/build/building/multi-stage/) - Optimisation images
- [Docker Compose](https://docs.docker.com/compose/) - Orchestration containers
- [GitHub Container Registry](https://docs.github.com/en/packages/working-with-a-github-packages-registry/working-with-the-container-registry) - Registry images

### Outils

- [Act](https://github.com/nektos/act) - Tester workflows GitHub Actions localement
- [Hadolint](https://github.com/hadolint/hadolint) - Linter Dockerfiles
- [docker-compose-wait](https://github.com/ufoscout/docker-compose-wait) - Attendre services ready
- [Renovate](https://github.com/renovatebot/renovate) - Auto-update dependencies

### Tutoriels

- [GitHub Actions CI/CD Pipeline](https://www.freecodecamp.org/news/what-is-ci-cd/) - FreeCodeCamp
- [Docker Best Practices](https://docs.docker.com/develop/dev-best-practices/) - Docker
- [Zero-downtime Deployments](https://www.digitalocean.com/community/tutorials/how-to-automate-deployments-to-digitalocean-kubernetes-with-circleci) - DigitalOcean

### Sécurité

- [GitHub Actions Security Best Practices](https://docs.github.com/en/actions/security-guides/security-hardening-for-github-actions)
- [Docker Security](https://cheatsheetseries.owasp.org/cheatsheets/Docker_Security_Cheat_Sheet.html) - OWASP

---

## ✅ Checklist de Complétion

### Configuration

- [ ] `.github/workflows/ci.yml` créé et testé
- [ ] `.github/workflows/deploy.yml` créé et testé
- [ ] `.github/workflows/backup.yml` créé
- [ ] GitHub Secrets configurés (VPS_HOST, VPS_USER, VPS_SSH_KEY, etc.)
- [ ] SSH key déploiement généré et installé sur VPS
- [ ] Utilisateur `deploy` créé sur VPS avec permissions Docker

### Dockerfiles

- [ ] `backend/Dockerfile` multi-stage optimisé
- [ ] `frontend/Dockerfile` multi-stage optimisé
- [ ] `docker/nginx/Dockerfile` configuré
- [ ] Health checks ajoutés à tous les Dockerfiles
- [ ] Non-root users configurés dans images

### Scripts

- [ ] `scripts/deploy.sh` créé et testé
- [ ] Permissions exécution (`chmod +x scripts/deploy.sh`)
- [ ] Rollback testé manuellement
- [ ] Backup script testé

### Docker Compose

- [ ] `docker-compose.yml` production configuré
- [ ] Health checks configurés pour tous services
- [ ] Volumes persistants configurés
- [ ] Networks isolés
- [ ] Logging limits configurés

### Tests

- [ ] CI workflow passe (backend tests)
- [ ] CI workflow passe (frontend tests)
- [ ] Security scans passent
- [ ] Build images réussi
- [ ] Deploy dry-run réussi
- [ ] Deploy réel testé (staging ou production)
- [ ] Rollback testé et fonctionnel

### Documentation

- [ ] README.md mis à jour avec badges CI/CD
- [ ] Documentation déploiement manuel
- [ ] Documentation rollback
- [ ] Runbook incidents

### Production

- [ ] Première déploiement production réussi
- [ ] Health checks fonctionnels en production
- [ ] Monitoring déploiements configuré
- [ ] Notifications Discord/Slack configurées
- [ ] Backups automatiques testés
- [ ] Plan de rollback documenté

### Optimisations

- [ ] Cache GitHub Actions configuré
- [ ] Images Docker optimisées (taille minimale)
- [ ] Build times < 5 min (CI)
- [ ] Deploy times < 3 min
- [ ] Zero-downtime deployment vérifié

---

**Dernière mise à jour:** 2025-12-08
**Auteur:** Alexi
