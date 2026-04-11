# Plan de Déploiement VPS - maicivy

## Vue d'ensemble

Ce document décrit le déploiement complet de maicivy sur un VPS personnel, incluant :
- Déploiement initial
- CI/CD automatisé
- Backups automatiques
- Monitoring

---

## 1. Prérequis VPS

### Specs minimales
- **OS:** Ubuntu 22.04 LTS
- **RAM:** 2GB minimum (4GB recommandé)
- **CPU:** 2 vCPU
- **Disque:** 20GB SSD
- **Ports ouverts:** 22 (SSH), 80 (HTTP), 443 (HTTPS)

### Logiciels à installer
```bash
# Update système
sudo apt update && sudo apt upgrade -y

# Docker
curl -fsSL https://get.docker.com | sh
sudo usermod -aG docker $USER

# Docker Compose
sudo apt install -y docker-compose

# Nginx (reverse proxy)
sudo apt install -y nginx

# Certbot (SSL)
sudo apt install -y certbot python3-certbot-nginx

# Outils utiles
sudo apt install -y git htop curl wget
```

---

## 2. Structure des fichiers sur le VPS

```
/opt/maicivy/
├── docker-compose.yml
├── docker-compose.prod.yml
├── .env                      # Variables d'environnement production
├── backend/
├── frontend/
├── nginx/
│   └── maicivy.conf
├── scripts/
│   ├── deploy.sh
│   ├── backup.sh
│   └── restore.sh
├── backups/                  # Backups locaux (optionnel)
└── monitoring/
    ├── prometheus/
    │   └── prometheus.yml
    └── grafana/
        └── dashboards/
```

---

## 3. Déploiement Initial

### Étape 1: Clone du repo
```bash
sudo mkdir -p /opt/maicivy
sudo chown $USER:$USER /opt/maicivy
cd /opt/maicivy
git clone https://git.etheryale.com/StillHammer/maicivy.git .
```

### Étape 2: Configuration environnement
```bash
# Copier et éditer le .env
cp backend/.env.example backend/.env
nano backend/.env
```

Variables à configurer :
```env
# Base de données
DB_HOST=postgres
DB_PORT=5432
DB_USER=maicivy
DB_PASSWORD=<MOT_DE_PASSE_SECURISE>
DB_NAME=maicivy_db

# Redis
REDIS_HOST=redis
REDIS_PORT=6379

# API Keys
ANTHROPIC_API_KEY=<TA_CLE_ANTHROPIC>
OPENAI_API_KEY=<TA_CLE_OPENAI>

# Production
APP_ENV=production
APP_URL=https://ton-domaine.com
```

### Étape 3: Lancement
```bash
docker-compose -f docker-compose.yml -f docker-compose.prod.yml up -d --build
```

### Étape 4: Configuration Nginx
```bash
sudo nano /etc/nginx/sites-available/maicivy
sudo ln -s /etc/nginx/sites-available/maicivy /etc/nginx/sites-enabled/
sudo nginx -t
sudo systemctl reload nginx
```

### Étape 5: SSL avec Let's Encrypt
```bash
sudo certbot --nginx -d ton-domaine.com
```

---

## 4. CI/CD avec GitHub Actions

### Workflow: `.github/workflows/deploy.yml`

```yaml
name: Deploy to VPS

on:
  push:
    branches: [main]
  workflow_dispatch:

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Setup Go
        uses: actions/setup-go@v5
        with:
          go-version: '1.21'

      - name: Run backend tests
        run: |
          cd backend
          go test ./... -v

      - name: Setup Node
        uses: actions/setup-node@v4
        with:
          node-version: '20'

      - name: Run frontend tests
        run: |
          cd frontend
          npm ci
          npm test -- --passWithNoTests

  deploy:
    needs: test
    runs-on: ubuntu-latest
    if: github.ref == 'refs/heads/main'

    steps:
      - name: Deploy to VPS
        uses: appleboy/ssh-action@v1.0.3
        with:
          host: ${{ secrets.VPS_HOST }}
          username: ${{ secrets.VPS_USER }}
          key: ${{ secrets.VPS_SSH_KEY }}
          script: |
            cd /opt/maicivy
            git pull origin main
            docker-compose -f docker-compose.yml -f docker-compose.prod.yml up -d --build
            docker system prune -f
```

### Secrets GitHub à configurer
- `VPS_HOST`: IP ou domaine du VPS
- `VPS_USER`: Utilisateur SSH
- `VPS_SSH_KEY`: Clé SSH privée

---

## 5. Backups Automatiques

### Script: `scripts/backup.sh`

```bash
#!/bin/bash
set -e

# Configuration
BACKUP_DIR="/opt/maicivy/backups"
RETENTION_DAYS=7
DATE=$(date +%Y%m%d_%H%M%S)
S3_BUCKET="s3://ton-bucket/maicivy-backups"  # Optionnel

# Créer dossier backup
mkdir -p $BACKUP_DIR

# Backup PostgreSQL
echo "Backing up PostgreSQL..."
docker exec maicivy-postgres pg_dump -U maicivy maicivy_db | gzip > $BACKUP_DIR/postgres_$DATE.sql.gz

# Backup Redis
echo "Backing up Redis..."
docker exec maicivy-redis redis-cli BGSAVE
sleep 2
docker cp maicivy-redis:/data/dump.rdb $BACKUP_DIR/redis_$DATE.rdb

# Cleanup vieux backups
echo "Cleaning old backups..."
find $BACKUP_DIR -type f -mtime +$RETENTION_DAYS -delete

# Upload vers S3 (optionnel)
# aws s3 cp $BACKUP_DIR/postgres_$DATE.sql.gz $S3_BUCKET/
# aws s3 cp $BACKUP_DIR/redis_$DATE.rdb $S3_BUCKET/

echo "Backup completed: $DATE"
```

### Script: `scripts/restore.sh`

```bash
#!/bin/bash
set -e

if [ -z "$1" ]; then
    echo "Usage: ./restore.sh <backup_date>"
    echo "Example: ./restore.sh 20250104_120000"
    exit 1
fi

BACKUP_DIR="/opt/maicivy/backups"
DATE=$1

# Restore PostgreSQL
echo "Restoring PostgreSQL..."
gunzip -c $BACKUP_DIR/postgres_$DATE.sql.gz | docker exec -i maicivy-postgres psql -U maicivy maicivy_db

# Restore Redis
echo "Restoring Redis..."
docker cp $BACKUP_DIR/redis_$DATE.rdb maicivy-redis:/data/dump.rdb
docker exec maicivy-redis redis-cli SHUTDOWN NOSAVE
# Redis redémarrera automatiquement via Docker

echo "Restore completed!"
```

### Cron job pour backups automatiques
```bash
# Ajouter au crontab (crontab -e)
0 3 * * * /opt/maicivy/scripts/backup.sh >> /var/log/maicivy-backup.log 2>&1
```

---

## 6. Monitoring

### Structure monitoring
```
monitoring/
├── prometheus/
│   └── prometheus.yml
├── grafana/
│   └── provisioning/
│       ├── dashboards/
│       │   └── maicivy.json
│       └── datasources/
│           └── prometheus.yml
└── docker-compose.monitoring.yml
```

### `monitoring/prometheus/prometheus.yml`

```yaml
global:
  scrape_interval: 15s
  evaluation_interval: 15s

scrape_configs:
  - job_name: 'prometheus'
    static_configs:
      - targets: ['localhost:9090']

  - job_name: 'maicivy-backend'
    static_configs:
      - targets: ['backend:8080']
    metrics_path: /metrics

  - job_name: 'postgres'
    static_configs:
      - targets: ['postgres-exporter:9187']

  - job_name: 'redis'
    static_configs:
      - targets: ['redis-exporter:9121']

  - job_name: 'node'
    static_configs:
      - targets: ['node-exporter:9100']
```

### `monitoring/docker-compose.monitoring.yml`

```yaml
version: '3.8'

services:
  prometheus:
    image: prom/prometheus:latest
    container_name: maicivy-prometheus
    volumes:
      - ./prometheus/prometheus.yml:/etc/prometheus/prometheus.yml
      - prometheus_data:/prometheus
    command:
      - '--config.file=/etc/prometheus/prometheus.yml'
      - '--storage.tsdb.path=/prometheus'
      - '--storage.tsdb.retention.time=30d'
    ports:
      - "9090:9090"
    networks:
      - maicivy-network
    restart: unless-stopped

  grafana:
    image: grafana/grafana:latest
    container_name: maicivy-grafana
    volumes:
      - grafana_data:/var/lib/grafana
      - ./grafana/provisioning:/etc/grafana/provisioning
    environment:
      - GF_SECURITY_ADMIN_PASSWORD=${GRAFANA_PASSWORD:-admin}
      - GF_USERS_ALLOW_SIGN_UP=false
    ports:
      - "3001:3000"
    networks:
      - maicivy-network
    restart: unless-stopped

  node-exporter:
    image: prom/node-exporter:latest
    container_name: maicivy-node-exporter
    volumes:
      - /proc:/host/proc:ro
      - /sys:/host/sys:ro
      - /:/rootfs:ro
    command:
      - '--path.procfs=/host/proc'
      - '--path.sysfs=/host/sys'
      - '--collector.filesystem.mount-points-exclude=^/(sys|proc|dev|host|etc)($$|/)'
    networks:
      - maicivy-network
    restart: unless-stopped

  postgres-exporter:
    image: prometheuscommunity/postgres-exporter:latest
    container_name: maicivy-postgres-exporter
    environment:
      - DATA_SOURCE_NAME=postgresql://maicivy:${DB_PASSWORD}@postgres:5432/maicivy_db?sslmode=disable
    networks:
      - maicivy-network
    restart: unless-stopped

  redis-exporter:
    image: oliver006/redis_exporter:latest
    container_name: maicivy-redis-exporter
    environment:
      - REDIS_ADDR=redis://redis:6379
    networks:
      - maicivy-network
    restart: unless-stopped

volumes:
  prometheus_data:
  grafana_data:

networks:
  maicivy-network:
    external: true
```

---

## 7. Nginx Configuration

### `/etc/nginx/sites-available/maicivy`

```nginx
# Redirect HTTP to HTTPS
server {
    listen 80;
    server_name ton-domaine.com;
    return 301 https://$server_name$request_uri;
}

# Main HTTPS server
server {
    listen 443 ssl http2;
    server_name ton-domaine.com;

    # SSL (géré par Certbot)
    ssl_certificate /etc/letsencrypt/live/ton-domaine.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/ton-domaine.com/privkey.pem;
    include /etc/letsencrypt/options-ssl-nginx.conf;
    ssl_dhparam /etc/letsencrypt/ssl-dhparams.pem;

    # Security headers
    add_header X-Frame-Options "SAMEORIGIN" always;
    add_header X-Content-Type-Options "nosniff" always;
    add_header X-XSS-Protection "1; mode=block" always;
    add_header Referrer-Policy "strict-origin-when-cross-origin" always;

    # Gzip
    gzip on;
    gzip_vary on;
    gzip_min_length 1024;
    gzip_types text/plain text/css text/xml text/javascript application/javascript application/json application/xml;

    # Frontend (Next.js)
    location / {
        proxy_pass http://localhost:3000;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection 'upgrade';
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_cache_bypass $http_upgrade;
    }

    # Backend API
    location /api/ {
        proxy_pass http://localhost:8080;
        proxy_http_version 1.1;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }

    # Health check
    location /health {
        proxy_pass http://localhost:8080/health;
    }

    # WebSocket pour analytics temps réel
    location /ws/ {
        proxy_pass http://localhost:8080;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
        proxy_set_header Host $host;
        proxy_read_timeout 86400;
    }

    # Grafana (optionnel, protégé)
    location /grafana/ {
        proxy_pass http://localhost:3001/;
        proxy_set_header Host $host;
    }
}
```

---

## 8. Script de Déploiement Complet

### `scripts/deploy.sh`

```bash
#!/bin/bash
set -e

echo "=== Déploiement maicivy ==="
cd /opt/maicivy

# Pull latest changes
echo "Pulling latest changes..."
git pull origin main

# Build and restart containers
echo "Building and starting containers..."
docker-compose -f docker-compose.yml -f docker-compose.prod.yml up -d --build

# Cleanup
echo "Cleaning up..."
docker system prune -f

# Health check
echo "Checking health..."
sleep 10
curl -f http://localhost:8080/health || exit 1

echo "=== Deployment successful! ==="
```

---

## 9. Checklist de Déploiement

### Premier déploiement
- [ ] VPS provisionné avec Ubuntu 22.04
- [ ] Docker et Docker Compose installés
- [ ] Nginx installé
- [ ] Repo cloné dans `/opt/maicivy`
- [ ] `.env` configuré avec vraies valeurs
- [ ] `docker-compose up -d --build` exécuté
- [ ] Nginx configuré
- [ ] SSL avec Certbot
- [ ] DNS pointant vers le VPS

### CI/CD
- [ ] Secrets GitHub configurés (VPS_HOST, VPS_USER, VPS_SSH_KEY)
- [ ] Clé SSH du runner GitHub ajoutée aux authorized_keys du VPS
- [ ] Workflow `.github/workflows/deploy.yml` créé
- [ ] Test du déploiement automatique

### Backups
- [ ] Script `backup.sh` créé et testé
- [ ] Script `restore.sh` créé et testé
- [ ] Cron job configuré
- [ ] Test de restauration effectué

### Monitoring (optionnel)
- [ ] Prometheus configuré
- [ ] Grafana configuré
- [ ] Dashboards importés
- [ ] Alertes configurées

---

## 10. Commandes Utiles

```bash
# Voir les logs
docker-compose logs -f

# Voir les logs d'un service
docker-compose logs -f backend

# Restart un service
docker-compose restart backend

# Rebuild et restart
docker-compose up -d --build backend

# Entrer dans un container
docker exec -it maicivy-backend sh

# Status des containers
docker-compose ps

# Utilisation ressources
docker stats

# Backup manuel
./scripts/backup.sh

# Voir les backups
ls -la /opt/maicivy/backups/
```

---

**Document créé le:** 2025-01-04
**Auteur:** Claude + Alexi
