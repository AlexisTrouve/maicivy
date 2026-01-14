# CI/CD Guide - maicivy

Complete guide for CI/CD pipeline, deployment, and operations.

---

## Table of Contents

1. [Overview](#overview)
2. [Architecture](#architecture)
3. [GitHub Actions Workflows](#github-actions-workflows)
4. [Deployment Process](#deployment-process)
5. [Rollback Procedure](#rollback-procedure)
6. [Secrets Management](#secrets-management)
7. [Monitoring & Health Checks](#monitoring--health-checks)
8. [Troubleshooting](#troubleshooting)

---

## Overview

### Pipeline Stages

```
┌─────────────┐
│   Git Push  │
└──────┬──────┘
       │
       ▼
┌────────────────────────────────────┐
│  Stage 1: CI - Tests & Quality    │
│  - Backend tests (Go)              │
│  - Frontend tests (Next.js)        │
│  - Security scans                  │
│  - Build validation                │
└──────┬─────────────────────────────┘
       │
       ▼
┌────────────────────────────────────┐
│  Stage 2: Build - Docker Images   │
│  - Build backend image             │
│  - Build frontend image            │
│  - Build nginx image               │
│  - Push to GitHub Registry         │
└──────┬─────────────────────────────┘
       │
       ▼
┌────────────────────────────────────┐
│  Stage 3: Deploy - Production VPS │
│  - SSH to VPS                      │
│  - Pull images                     │
│  - Rolling update                  │
│  - Health checks                   │
│  - Rollback on failure             │
└────────────────────────────────────┘
```

### Key Features

- **Automated Testing**: Every push triggers comprehensive test suite
- **Security Scanning**: Gosec, npm audit, Trivy vulnerability scanning
- **Multi-stage Docker Builds**: Optimized image sizes (<50MB backend, <150MB frontend)
- **Zero-downtime Deployment**: Rolling updates with health checks
- **Automatic Rollback**: Failed deployments trigger automatic rollback
- **Notifications**: Discord/Slack notifications on deployment status

---

## Architecture

### Repositories

- **Primary**: Gitea (private development)
- **Mirror**: GitHub (public showcase + CI/CD runner)

### Docker Registry

- **GitHub Container Registry (ghcr.io)**: Default, free for public repos
- **Alternative**: Docker Hub (requires authentication)

### VPS Setup

```
/opt/maicivy/
├── docker-compose.yml
├── .env
├── backups/
│   ├── postgres/
│   └── redis/
├── logs/
│   └── nginx/
├── ssl/
│   ├── fullchain.pem
│   └── privkey.pem
└── monitoring/
    ├── prometheus/
    └── grafana/
```

---

## GitHub Actions Workflows

### 1. CI Workflow (`.github/workflows/ci.yml`)

**Triggers:**
- Push to `main`, `develop`
- Pull requests to `main`, `develop`

**Jobs:**

#### Backend Tests
```yaml
- Setup Go 1.21
- Run go vet
- Run golangci-lint
- Run tests with coverage
- Upload coverage to Codecov
```

**Services:**
- PostgreSQL 16
- Redis 7

#### Frontend Tests
```yaml
- Setup Node.js 20
- Run ESLint
- Type checking (TypeScript)
- Run Jest tests with coverage
- Upload coverage to Codecov
```

#### Security Scan
```yaml
- Gosec (Go security scanner)
- npm audit (Node.js dependencies)
- Trivy (container vulnerability scanner)
```

#### Build Validation
```yaml
- Build backend Docker image (test)
- Build frontend Docker image (test)
- Validate images
```

**Expected Duration:** 5-8 minutes

---

### 2. Deploy Workflow (`.github/workflows/deploy.yml`)

**Triggers:**
- Push to `main` branch
- Manual trigger (workflow_dispatch)

**Jobs:**

#### Build & Push Images

```yaml
1. Checkout code
2. Setup Docker Buildx
3. Login to GitHub Container Registry
4. Build & push backend image
   - Tag: latest, <commit-sha>
   - Build args: BUILD_DATE, VCS_REF
5. Build & push frontend image
6. Build & push nginx image
```

**Registry:** `ghcr.io/<username>/maicivy-{backend,frontend,nginx}`

#### Deploy to VPS

```yaml
1. Setup SSH keys
2. Create deployment script
3. SCP script to VPS
4. Execute deployment:
   - Login to registry
   - Pull latest images
   - Backup current containers
   - Rolling update (docker-compose up -d)
   - Health check (curl /health)
   - Cleanup old images
5. Rollback on failure
6. Send notifications (Discord)
```

**Expected Duration:** 3-5 minutes

---

### 3. Backup Workflow (`.github/workflows/backup.yml`)

**Triggers:**
- Weekly (Sunday 2 AM UTC)
- Manual trigger

**Process:**
```bash
1. SSH to VPS
2. PostgreSQL dump (pg_dump)
3. Redis snapshot (SAVE + copy dump.rdb)
4. Compress backups (gzip)
5. Cleanup old backups (>28 days)
6. Send notification
```

**Retention:** 4 weeks (28 days)

---

## Deployment Process

### Automatic Deployment (on push to main)

1. **Push code to main branch**
   ```bash
   git add .
   git commit -m "feat: add new feature"
   git push origin main
   ```

2. **GitHub Actions automatically:**
   - Runs CI tests
   - Builds Docker images
   - Deploys to production VPS
   - Sends notification

3. **Monitor deployment:**
   - GitHub Actions UI: `https://github.com/<user>/<repo>/actions`
   - VPS logs: `ssh deploy@vps "docker-compose -f /opt/maicivy/docker-compose.yml logs -f"`

### Manual Deployment

**Using GitHub Actions:**
```bash
# Go to GitHub Actions → Deploy to Production → Run workflow
```

**Using deployment script:**
```bash
# SSH to VPS
ssh deploy@your-vps-ip

# Run deployment script
cd /opt/maicivy
./scripts/deploy.sh
```

**Dry-run (test without deploying):**
```bash
./scripts/deploy.sh --dry-run
```

**Backup only:**
```bash
./scripts/deploy.sh --backup
```

---

## Rollback Procedure

### Automatic Rollback

Failed deployments trigger automatic rollback:

```bash
# If health check fails after deployment:
1. Stop new containers
2. Restore previous containers from backup
3. Verify health check passes
4. Send failure notification
```

### Manual Rollback

**Using rollback script:**

```bash
# SSH to VPS
ssh deploy@your-vps-ip

# List available backups
cd /opt/maicivy
./scripts/rollback.sh

# Rollback to specific timestamp
./scripts/rollback.sh 20231208_143000

# Rollback to latest backup
./scripts/rollback.sh
```

**Using Docker Compose:**

```bash
# SSH to VPS
cd /opt/maicivy

# Stop current containers
docker-compose down

# Pull previous image tag
docker pull ghcr.io/username/maicivy-backend:<previous-sha>
docker pull ghcr.io/username/maicivy-frontend:<previous-sha>

# Update docker-compose.yml to use specific tags
# Then restart
docker-compose up -d
```

### Database Rollback

⚠️ **CAUTION**: Database rollbacks require restoring from backup

```bash
# Stop backend to prevent writes
docker-compose stop backend

# Restore PostgreSQL backup
docker exec maicivy-postgres psql -U postgres -d maicivy < /backups/postgres_20231208_140000.sql

# Restart services
docker-compose up -d
```

---

## Secrets Management

### Required GitHub Secrets

Configure in: `Settings → Secrets and variables → Actions → New repository secret`

| Secret Name | Description | Example |
|------------|-------------|---------|
| `VPS_HOST` | VPS IP or domain | `192.168.1.100` |
| `VPS_USER` | SSH user | `deploy` |
| `VPS_SSH_KEY` | SSH private key | `-----BEGIN...` |
| `CLAUDE_API_KEY` | Anthropic API key | `sk-ant-api03-...` |
| `OPENAI_API_KEY` | OpenAI API key | `sk-...` |
| `JWT_SECRET` | JWT signing secret | (generated) |
| `POSTGRES_PASSWORD` | Database password | (generated) |
| `REDIS_PASSWORD` | Redis password | (generated) |
| `GRAFANA_PASSWORD` | Grafana admin pwd | (generated) |
| `DISCORD_WEBHOOK_ID` | Discord webhook | (optional) |
| `DISCORD_WEBHOOK_TOKEN` | Discord token | (optional) |

### Setup Script

Use the provided helper script:

```bash
cd scripts
./setup-secrets.sh
```

Interactive menu:
1. Interactive secrets setup
2. List required secrets
3. List current secrets
4. Verify secrets configuration
5. Show SSH key setup instructions

### SSH Key Setup

```bash
# 1. Generate SSH key
ssh-keygen -t ed25519 -C "github-actions-deploy" -f ~/.ssh/maicivy_deploy
# No passphrase (for automation)

# 2. Copy public key to VPS
ssh-copy-id -i ~/.ssh/maicivy_deploy.pub deploy@your-vps-ip

# 3. Test connection
ssh -i ~/.ssh/maicivy_deploy deploy@your-vps-ip

# 4. Add private key to GitHub Secrets
cat ~/.ssh/maicivy_deploy
# Copy entire output (including BEGIN/END lines) to VPS_SSH_KEY secret
```

### VPS User Setup

```bash
# SSH as root
ssh root@your-vps-ip

# Create deploy user
useradd -m -s /bin/bash deploy
usermod -aG docker deploy

# Create deployment directory
mkdir -p /opt/maicivy
chown deploy:deploy /opt/maicivy

# Allow docker-compose without password
echo "deploy ALL=(ALL) NOPASSWD: /usr/bin/docker-compose" >> /etc/sudoers.d/deploy
```

### Environment Variables on VPS

```bash
# SSH to VPS as deploy user
ssh deploy@your-vps-ip

# Create .env file
cd /opt/maicivy
nano .env
```

Copy from `.env.production.example` and fill in secrets.

**⚠️ NEVER commit .env to git!**

---

## Monitoring & Health Checks

### Health Check Endpoints

**Backend:**
```bash
curl http://localhost:8080/health

# Expected response:
# {"status":"ok","services":{"database":"up","redis":"up"}}
```

**Frontend:**
```bash
curl http://localhost:3000

# Expected: 200 OK (HTML response)
```

**Nginx:**
```bash
curl http://localhost:80/health

# Expected: healthy
```

### Health Check Script

```bash
# SSH to VPS
cd /opt/maicivy

# One-time check
./scripts/health-check.sh

# Continuous monitoring
./scripts/health-check.sh --continuous --interval 30

# With statistics
./scripts/health-check.sh --stats
```

### Container Health

```bash
# Check container status
docker ps

# Check health status
docker inspect --format='{{.State.Health.Status}}' maicivy-backend

# View logs
docker-compose logs -f backend
docker-compose logs -f frontend
docker-compose logs --tail=100 nginx
```

### Prometheus Metrics

Access: `http://your-vps-ip:9090`

**Key metrics:**
- `http_requests_total` - Total HTTP requests
- `http_request_duration_seconds` - Request latency
- `up` - Service availability

### Grafana Dashboards

Access: `http://your-vps-ip:3001`

**Default credentials:**
- User: `admin`
- Password: (set in `GRAFANA_PASSWORD`)

**Dashboards:**
- System metrics (CPU, memory, disk)
- Application metrics (requests, latency)
- Database metrics (connections, queries)

---

## Troubleshooting

### Deployment Failures

#### Issue: CI tests failing

```bash
# Check GitHub Actions logs
# Go to: Actions → Failed workflow → View logs

# Common causes:
- Linting errors (fix with `npm run lint --fix`)
- Test failures (run `npm test` locally)
- Type errors (run `npm run type-check`)
```

#### Issue: Docker build failing

```bash
# Build locally to debug
docker build -f backend/Dockerfile backend/

# Common causes:
- Missing dependencies (update go.mod)
- Build args not set (check workflow YAML)
- Dockerfile syntax error (use hadolint)
```

#### Issue: SSH connection failed

```bash
# Test SSH connection
ssh -i ~/.ssh/maicivy_deploy deploy@your-vps-ip

# Common causes:
- Wrong SSH key in GitHub Secrets
- VPS firewall blocking port 22
- deploy user doesn't exist
- SSH key permissions (should be 600)
```

#### Issue: Health check failed

```bash
# SSH to VPS
ssh deploy@your-vps-ip

# Check container logs
docker-compose logs backend
docker-compose logs postgres

# Common causes:
- Database connection failed (check DATABASE_URL)
- Redis connection failed (check REDIS_URL)
- Backend crashed on startup (check logs)
- Port already in use (check `netstat -tulpn`)
```

### Rollback Failures

#### Issue: Rollback script not found

```bash
# Make scripts executable
chmod +x /opt/maicivy/scripts/*.sh
```

#### Issue: No backup available

```bash
# Check backups directory
ls -lh /opt/maicivy/backups/

# If empty, redeploy from known good commit:
git checkout <last-good-commit>
git push origin main --force
```

### Performance Issues

#### Issue: Slow deployment

```bash
# Optimize Docker build cache
# Add to workflow:
cache-from: type=gha
cache-to: type=gha,mode=max

# Pre-pull base images on VPS
docker pull golang:1.21-alpine
docker pull node:20-alpine
docker pull nginx:1.25-alpine
```

#### Issue: Out of disk space

```bash
# Check disk usage
df -h

# Cleanup Docker
docker system prune -af --volumes

# Cleanup old backups
find /opt/maicivy/backups -mtime +28 -delete
```

### Security Issues

#### Issue: Secrets exposed in logs

```bash
# Check GitHub Actions logs for secrets
# GitHub automatically masks secrets in logs

# If secrets exposed:
1. Rotate immediately
2. Update GitHub Secrets
3. Redeploy
```

#### Issue: Container running as root

```bash
# Check user in container
docker exec maicivy-backend whoami
# Should output: app (not root)

# Fix in Dockerfile:
USER app
```

---

## Best Practices

### 1. Git Workflow

```bash
# Feature development
git checkout -b feature/my-feature
git commit -m "feat: add feature"
git push origin feature/my-feature

# Create PR to main
# Wait for CI to pass
# Merge to main → auto-deploy
```

### 2. Commit Messages

Follow conventional commits:

```
feat: add new feature
fix: resolve bug
docs: update documentation
refactor: improve code structure
test: add tests
chore: update dependencies
```

### 3. Environment Variables

- **Development**: Use `.env.local`
- **Staging**: Use `.env.staging`
- **Production**: Use `.env` (never commit!)

### 4. Secrets Rotation

Rotate secrets regularly:

```bash
# Generate new secrets
openssl rand -hex 32

# Update GitHub Secrets
# Update VPS .env
# Redeploy
```

### 5. Monitoring

- Check Grafana daily
- Review error logs weekly
- Test health checks after deployment
- Verify backups weekly

### 6. Testing

Before pushing to main:

```bash
# Backend
cd backend
go test ./...
go vet ./...

# Frontend
cd frontend
npm test
npm run lint
npm run type-check

# Build test
docker-compose build
```

---

## Quick Reference

### Common Commands

```bash
# Deploy
./scripts/deploy.sh

# Rollback
./scripts/rollback.sh

# Health check
./scripts/health-check.sh

# Build images
./scripts/build-images.sh --tag v1.0.0 --push

# View logs
docker-compose logs -f

# Restart service
docker-compose restart backend

# Stop all
docker-compose down

# Start all
docker-compose up -d
```

### Useful Links

- GitHub Actions: `https://github.com/<user>/<repo>/actions`
- Grafana: `http://<vps-ip>:3001`
- Prometheus: `http://<vps-ip>:9090`
- Application: `https://maicivy.yourdomain.com`

---

**Last Updated:** 2025-12-08
**Version:** 1.0
**Author:** Alexi
