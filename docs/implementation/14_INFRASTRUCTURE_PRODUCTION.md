# 14. INFRASTRUCTURE_PRODUCTION

## 📋 Métadonnées

- **Phase:** 6
- **Priorité:** 🔴 CRITIQUE
- **Complexité:** ⭐⭐⭐⭐ (4/5)
- **Prérequis:** Tous modules fonctionnels (01-13)
- **Temps estimé:** 3-5 jours
- **Status:** 🔲 À faire

---

## 🎯 Objectif

Mettre en place l'infrastructure de production complète sur VPS OVH avec Nginx comme reverse proxy, SSL Let's Encrypt, et système de monitoring avec Prometheus et Grafana accessible publiquement. Incluant également la gestion des logs, health checks, et stratégie de backup pour assurer la fiabilité et l'observabilité du système en production.

---

## 🏗️ Architecture

### Vue d'Ensemble

```
Internet (HTTPS)
    │
    ▼
┌─────────────────────────────────────────┐
│          VPS OVH (Ubuntu)               │
│                                         │
│  ┌─────────────────────────────────┐   │
│  │         Nginx                   │   │
│  │  - Reverse Proxy                │   │
│  │  - SSL Termination (Let's Enc.) │   │
│  │  - Compression gzip/brotli      │   │
│  │  - Rate Limiting                │   │
│  │  - Static Caching               │   │
│  └──────────┬──────────────────────┘   │
│             │                           │
│  ┌──────────┴──────────────────────┐   │
│  │   Docker Compose Network        │   │
│  │                                 │   │
│  │  ┌──────────┐  ┌──────────┐    │   │
│  │  │ Frontend │  │ Backend  │    │   │
│  │  │ (Next.js)│  │   (Go)   │    │   │
│  │  │  :3000   │  │  :8080   │    │   │
│  │  └──────────┘  └────┬─────┘    │   │
│  │                     │          │   │
│  │  ┌──────────┐  ┌───┴──────┐   │   │
│  │  │PostgreSQL│  │  Redis   │   │   │
│  │  │  :5432   │  │  :6379   │   │   │
│  │  └──────────┘  └──────────┘   │   │
│  │                                │   │
│  │  ┌──────────┐  ┌──────────┐   │   │
│  │  │Prometheus│  │ Grafana  │   │   │
│  │  │  :9090   │  │  :3001   │   │   │
│  │  └──────────┘  └──────────┘   │   │
│  └─────────────────────────────────┘  │
└─────────────────────────────────────────┘
```

### Design Decisions

**1. Nginx comme Reverse Proxy:**
- Terminaison SSL centralisée
- Meilleure gestion du caching static files
- Rate limiting au niveau infrastructure
- Configuration security headers

**2. Prometheus + Grafana:**
- Prometheus: collecte métriques temps réel
- Grafana: dashboard public en readonly
- Rétention 15 jours (suffisant pour analytics)

**3. Let's Encrypt:**
- Certificats SSL gratuits et auto-renouvelables
- Certbot pour automatisation
- Wildcard certificates si sous-domaines multiples

**4. Logging JSON:**
- Structured logging pour parsing facile
- Stdout/stderr capturés par Docker
- Optionnel: Loki pour centralisation

**5. Backups PostgreSQL:**
- pg_dump quotidien via cron
- Rétention 30 jours
- Stockage local + optionnel S3

---

## 📦 Dépendances

### Packages Système (VPS)

```bash
# Mise à jour système
sudo apt update && sudo apt upgrade -y

# Docker
sudo apt install docker.io docker-compose -y

# Nginx
sudo apt install nginx -y

# Certbot (Let's Encrypt)
sudo apt install certbot python3-certbot-nginx -y

# Utilitaires
sudo apt install curl wget git htop -y
```

### Services Externes

- **VPS OVH** : serveur avec au minimum 2GB RAM, 2 vCPU, 40GB SSD
- **Domaine** : nom de domaine configuré (DNS A record vers IP VPS)
- **Let's Encrypt** : certificats SSL gratuits

---

## 🔨 Implémentation

### Étape 1: Configuration Nginx - Reverse Proxy

**Description:** Configurer Nginx comme reverse proxy avec SSL, compression et headers de sécurité.

**Fichier:** `docker/nginx/nginx.conf`

**Code:**

```nginx
# Configuration Nginx Production

# Limite de connexions globale
limit_conn_zone $binary_remote_addr zone=addr:10m;
limit_req_zone $binary_remote_addr zone=ip_limit:10m rate=100r/m;

# Upstream backends
upstream backend_server {
    server backend:8080;
    keepalive 32;
}

upstream frontend_server {
    server frontend:3000;
    keepalive 32;
}

upstream grafana_server {
    server grafana:3001;
}

# Redirect HTTP to HTTPS
server {
    listen 80;
    listen [::]:80;
    server_name maicivy.com www.maicivy.com;

    location /.well-known/acme-challenge/ {
        root /var/www/certbot;
    }

    location / {
        return 301 https://$server_name$request_uri;
    }
}

# Main HTTPS Server
server {
    listen 443 ssl http2;
    listen [::]:443 ssl http2;
    server_name maicivy.com www.maicivy.com;

    # SSL Configuration
    ssl_certificate /etc/letsencrypt/live/maicivy.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/maicivy.com/privkey.pem;
    ssl_protocols TLSv1.2 TLSv1.3;
    ssl_ciphers HIGH:!aNULL:!MD5;
    ssl_prefer_server_ciphers on;
    ssl_session_cache shared:SSL:10m;
    ssl_session_timeout 10m;

    # Security Headers
    add_header Strict-Transport-Security "max-age=31536000; includeSubDomains" always;
    add_header X-Frame-Options "DENY" always;
    add_header X-Content-Type-Options "nosniff" always;
    add_header X-XSS-Protection "1; mode=block" always;
    add_header Referrer-Policy "strict-origin-when-cross-origin" always;
    add_header Content-Security-Policy "default-src 'self'; script-src 'self' 'unsafe-inline' 'unsafe-eval'; style-src 'self' 'unsafe-inline'; img-src 'self' data: https:; font-src 'self' data:; connect-src 'self' https://api.anthropic.com https://api.openai.com;" always;

    # Compression
    gzip on;
    gzip_vary on;
    gzip_min_length 1024;
    gzip_types text/plain text/css text/xml text/javascript application/json application/javascript application/xml+rss application/rss+xml font/truetype font/opentype application/vnd.ms-fontobject image/svg+xml;
    gzip_comp_level 6;

    # Brotli (si module disponible)
    # brotli on;
    # brotli_comp_level 6;
    # brotli_types text/plain text/css application/json application/javascript application/xml+rss text/xml;

    # Rate Limiting
    limit_req zone=ip_limit burst=20 nodelay;
    limit_conn addr 10;

    # Client body size (pour uploads PDF)
    client_max_body_size 10M;

    # Timeouts
    proxy_connect_timeout 60s;
    proxy_send_timeout 60s;
    proxy_read_timeout 60s;

    # Logs
    access_log /var/log/nginx/access.log combined;
    error_log /var/log/nginx/error.log warn;

    # API Backend
    location /api/ {
        proxy_pass http://backend_server;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection 'upgrade';
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_cache_bypass $http_upgrade;

        # Pas de cache pour les APIs
        add_header Cache-Control "no-store, no-cache, must-revalidate, proxy-revalidate, max-age=0";
    }

    # WebSocket pour Analytics
    location /ws/ {
        proxy_pass http://backend_server;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_read_timeout 86400;
    }

    # Health Check Backend
    location /health {
        proxy_pass http://backend_server/health;
        access_log off;
    }

    # Frontend Next.js
    location / {
        proxy_pass http://frontend_server;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection 'upgrade';
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_cache_bypass $http_upgrade;
    }

    # Static files caching
    location ~* \.(jpg|jpeg|png|gif|ico|css|js|svg|woff|woff2|ttf|eot)$ {
        proxy_pass http://frontend_server;
        expires 1y;
        add_header Cache-Control "public, immutable";
        access_log off;
    }
}

# Grafana Dashboard Public
server {
    listen 443 ssl http2;
    listen [::]:443 ssl http2;
    server_name analytics.maicivy.com;

    # SSL (même certificat ou wildcard)
    ssl_certificate /etc/letsencrypt/live/maicivy.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/maicivy.com/privkey.pem;
    ssl_protocols TLSv1.2 TLSv1.3;
    ssl_ciphers HIGH:!aNULL:!MD5;

    # Security Headers
    add_header Strict-Transport-Security "max-age=31536000; includeSubDomains" always;
    add_header X-Frame-Options "SAMEORIGIN" always;
    add_header X-Content-Type-Options "nosniff" always;

    location / {
        proxy_pass http://grafana_server;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
```

**Explications:**

- **SSL/TLS:** Terminaison SSL avec Let's Encrypt, protocoles sécurisés uniquement
- **Security Headers:** CSP, HSTS, X-Frame-Options pour protection contre attaques
- **Compression:** gzip activé pour réduire bande passante (textes, JS, CSS)
- **Rate Limiting:** 100 req/min par IP, protection DDoS basique
- **Caching:** Static files cachés 1 an, APIs jamais cachées
- **WebSocket:** Support pour analytics temps réel
- **Grafana:** Sous-domaine dédié pour dashboard public

---

### Étape 2: Docker Compose Production

**Description:** Configuration Docker Compose incluant Nginx, monitoring et tous les services.

**Fichier:** `docker/docker-compose.prod.yml`

**Code:**

```yaml
version: '3.8'

services:
  # PostgreSQL Database
  postgres:
    image: postgres:15-alpine
    container_name: maicivy_postgres
    restart: unless-stopped
    environment:
      POSTGRES_DB: ${POSTGRES_DB}
      POSTGRES_USER: ${POSTGRES_USER}
      POSTGRES_PASSWORD: ${POSTGRES_PASSWORD}
    volumes:
      - postgres_data:/var/lib/postgresql/data
      - ./backups:/backups
    ports:
      - "127.0.0.1:5432:5432"
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U ${POSTGRES_USER} -d ${POSTGRES_DB}"]
      interval: 10s
      timeout: 5s
      retries: 5
    networks:
      - maicivy_network

  # Redis Cache
  redis:
    image: redis:7-alpine
    container_name: maicivy_redis
    restart: unless-stopped
    command: redis-server --appendonly yes --requirepass ${REDIS_PASSWORD}
    volumes:
      - redis_data:/data
    ports:
      - "127.0.0.1:6379:6379"
    healthcheck:
      test: ["CMD", "redis-cli", "--raw", "incr", "ping"]
      interval: 10s
      timeout: 5s
      retries: 5
    networks:
      - maicivy_network

  # Backend Go API
  backend:
    image: ${DOCKER_REGISTRY}/maicivy-backend:latest
    container_name: maicivy_backend
    restart: unless-stopped
    environment:
      - ENV=production
      - PORT=8080
      - DATABASE_URL=postgres://${POSTGRES_USER}:${POSTGRES_PASSWORD}@postgres:5432/${POSTGRES_DB}?sslmode=disable
      - REDIS_URL=redis://:${REDIS_PASSWORD}@redis:6379/0
      - CLAUDE_API_KEY=${CLAUDE_API_KEY}
      - OPENAI_API_KEY=${OPENAI_API_KEY}
      - FRONTEND_URL=${FRONTEND_URL}
    depends_on:
      postgres:
        condition: service_healthy
      redis:
        condition: service_healthy
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:8080/health"]
      interval: 30s
      timeout: 10s
      retries: 3
    networks:
      - maicivy_network

  # Frontend Next.js
  frontend:
    image: ${DOCKER_REGISTRY}/maicivy-frontend:latest
    container_name: maicivy_frontend
    restart: unless-stopped
    environment:
      - NODE_ENV=production
      - NEXT_PUBLIC_API_URL=${NEXT_PUBLIC_API_URL}
      - NEXT_PUBLIC_WS_URL=${NEXT_PUBLIC_WS_URL}
    depends_on:
      - backend
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:3000"]
      interval: 30s
      timeout: 10s
      retries: 3
    networks:
      - maicivy_network

  # Nginx Reverse Proxy
  nginx:
    image: nginx:alpine
    container_name: maicivy_nginx
    restart: unless-stopped
    ports:
      - "80:80"
      - "443:443"
    volumes:
      - ./nginx/nginx.conf:/etc/nginx/nginx.conf:ro
      - ./nginx/conf.d:/etc/nginx/conf.d:ro
      - /etc/letsencrypt:/etc/letsencrypt:ro
      - /var/www/certbot:/var/www/certbot:ro
      - nginx_logs:/var/log/nginx
    depends_on:
      - frontend
      - backend
      - grafana
    networks:
      - maicivy_network

  # Prometheus Monitoring
  prometheus:
    image: prom/prometheus:latest
    container_name: maicivy_prometheus
    restart: unless-stopped
    command:
      - '--config.file=/etc/prometheus/prometheus.yml'
      - '--storage.tsdb.path=/prometheus'
      - '--storage.tsdb.retention.time=15d'
      - '--web.console.libraries=/usr/share/prometheus/console_libraries'
      - '--web.console.templates=/usr/share/prometheus/consoles'
    volumes:
      - ./prometheus/prometheus.yml:/etc/prometheus/prometheus.yml:ro
      - ./prometheus/alerts.yml:/etc/prometheus/alerts.yml:ro
      - prometheus_data:/prometheus
    ports:
      - "127.0.0.1:9090:9090"
    networks:
      - maicivy_network

  # Grafana Dashboard
  grafana:
    image: grafana/grafana:latest
    container_name: maicivy_grafana
    restart: unless-stopped
    environment:
      - GF_SECURITY_ADMIN_USER=${GRAFANA_ADMIN_USER}
      - GF_SECURITY_ADMIN_PASSWORD=${GRAFANA_ADMIN_PASSWORD}
      - GF_SERVER_ROOT_URL=https://analytics.maicivy.com
      - GF_SERVER_HTTP_PORT=3001
      - GF_AUTH_ANONYMOUS_ENABLED=true
      - GF_AUTH_ANONYMOUS_ORG_ROLE=Viewer
      - GF_USERS_ALLOW_SIGN_UP=false
    volumes:
      - grafana_data:/var/lib/grafana
      - ./grafana/provisioning:/etc/grafana/provisioning:ro
      - ./grafana/dashboards:/var/lib/grafana/dashboards:ro
    depends_on:
      - prometheus
    networks:
      - maicivy_network

volumes:
  postgres_data:
    driver: local
  redis_data:
    driver: local
  prometheus_data:
    driver: local
  grafana_data:
    driver: local
  nginx_logs:
    driver: local

networks:
  maicivy_network:
    driver: bridge
```

**Explications:**

- **Health Checks:** Tous les services critiques ont des health checks
- **Restart Policy:** `unless-stopped` pour auto-restart après crash
- **Volumes:** Données persistées (PostgreSQL, Redis, Prometheus, Grafana)
- **Networks:** Réseau bridge isolé pour communication inter-services
- **Environment:** Variables d'environnement injectées via `.env`
- **Ports:** Exposition minimale (seulement 80/443 publics, reste 127.0.0.1)

---

### Étape 3: Configuration Prometheus

**Description:** Setup Prometheus pour scraper métriques des services.

**Fichier:** `monitoring/prometheus/prometheus.yml`

**Code:**

```yaml
# Prometheus Configuration

global:
  scrape_interval: 15s
  evaluation_interval: 15s
  external_labels:
    cluster: 'maicivy-production'
    environment: 'production'

# Alerting (optionnel)
alerting:
  alertmanagers:
    - static_configs:
        - targets: []
          # - alertmanager:9093

# Load rules (optionnel)
rule_files:
  - "alerts.yml"

# Scrape configurations
scrape_configs:
  # Prometheus self-monitoring
  - job_name: 'prometheus'
    static_configs:
      - targets: ['localhost:9090']

  # Backend Go API
  - job_name: 'backend'
    metrics_path: '/metrics'
    static_configs:
      - targets: ['backend:8080']
        labels:
          service: 'backend'
          app: 'maicivy'

  # PostgreSQL Exporter (si installé)
  - job_name: 'postgres'
    static_configs:
      - targets: ['postgres-exporter:9187']
        labels:
          service: 'postgres'
    # Note: nécessite postgres_exporter container

  # Redis Exporter (si installé)
  - job_name: 'redis'
    static_configs:
      - targets: ['redis-exporter:9121']
        labels:
          service: 'redis'
    # Note: nécessite redis_exporter container

  # Node Exporter (métriques système VPS)
  - job_name: 'node'
    static_configs:
      - targets: ['node-exporter:9100']
        labels:
          service: 'node'
    # Note: nécessite node_exporter container

  # Nginx Exporter (optionnel)
  - job_name: 'nginx'
    static_configs:
      - targets: ['nginx-exporter:9113']
        labels:
          service: 'nginx'
```

**Fichier:** `monitoring/prometheus/alerts.yml`

**Code:**

```yaml
# Alerting Rules (optionnel)

groups:
  - name: maicivy_alerts
    interval: 30s
    rules:
      # Backend Down
      - alert: BackendDown
        expr: up{job="backend"} == 0
        for: 2m
        labels:
          severity: critical
        annotations:
          summary: "Backend API is down"
          description: "Backend service has been down for more than 2 minutes."

      # High Error Rate
      - alert: HighErrorRate
        expr: rate(http_requests_total{status=~"5.."}[5m]) > 0.05
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "High error rate detected"
          description: "More than 5% of requests are returning 5xx errors."

      # Database Connection Issues
      - alert: PostgresDown
        expr: up{job="postgres"} == 0
        for: 1m
        labels:
          severity: critical
        annotations:
          summary: "PostgreSQL is down"
          description: "PostgreSQL database is unreachable."

      # Redis Down
      - alert: RedisDown
        expr: up{job="redis"} == 0
        for: 1m
        labels:
          severity: warning
        annotations:
          summary: "Redis is down"
          description: "Redis cache is unreachable."

      # Disk Space Low
      - alert: DiskSpaceLow
        expr: node_filesystem_avail_bytes{mountpoint="/"} / node_filesystem_size_bytes{mountpoint="/"} < 0.1
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "Disk space low"
          description: "Less than 10% disk space remaining."

      # High Memory Usage
      - alert: HighMemoryUsage
        expr: (1 - (node_memory_MemAvailable_bytes / node_memory_MemTotal_bytes)) > 0.9
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "High memory usage"
          description: "Memory usage above 90%."
```

**Explications:**

- **Scrape Interval:** 15s pour métriques temps réel
- **Retention:** 15 jours (configuré dans docker-compose)
- **Jobs:** Backend, PostgreSQL, Redis, Node (système)
- **Alerts:** Règles critiques (services down, erreurs, ressources)

---

### Étape 4: Configuration Grafana

**Description:** Provisioning automatique des dashboards et datasources Grafana.

**Fichier:** `monitoring/grafana/provisioning/datasources/prometheus.yml`

**Code:**

```yaml
apiVersion: 1

datasources:
  - name: Prometheus
    type: prometheus
    access: proxy
    url: http://prometheus:9090
    isDefault: true
    editable: false
    jsonData:
      timeInterval: "15s"
```

**Fichier:** `monitoring/grafana/provisioning/dashboards/dashboard.yml`

**Code:**

```yaml
apiVersion: 1

providers:
  - name: 'maicivy-dashboards'
    orgId: 1
    folder: ''
    type: file
    disableDeletion: false
    updateIntervalSeconds: 10
    allowUiUpdates: true
    options:
      path: /var/lib/grafana/dashboards
      foldersFromFilesStructure: true
```

**Fichier:** `monitoring/grafana/dashboards/maicivy_overview.json`

**Code (structure de base):**

```json
{
  "dashboard": {
    "title": "maicivy - Production Overview",
    "tags": ["maicivy", "production"],
    "timezone": "browser",
    "panels": [
      {
        "id": 1,
        "title": "Visiteurs en Temps Réel",
        "type": "stat",
        "targets": [
          {
            "expr": "analytics_realtime_visitors",
            "legendFormat": "Visiteurs actuels"
          }
        ],
        "fieldConfig": {
          "defaults": {
            "unit": "none",
            "thresholds": {
              "mode": "absolute",
              "steps": [
                {"value": 0, "color": "green"},
                {"value": 10, "color": "yellow"},
                {"value": 50, "color": "red"}
              ]
            }
          }
        }
      },
      {
        "id": 2,
        "title": "Request Rate (RPS)",
        "type": "graph",
        "targets": [
          {
            "expr": "rate(http_requests_total[5m])",
            "legendFormat": "{{method}} {{path}}"
          }
        ]
      },
      {
        "id": 3,
        "title": "Response Time (P95)",
        "type": "graph",
        "targets": [
          {
            "expr": "histogram_quantile(0.95, rate(http_request_duration_seconds_bucket[5m]))",
            "legendFormat": "P95"
          }
        ],
        "fieldConfig": {
          "defaults": {
            "unit": "s"
          }
        }
      },
      {
        "id": 4,
        "title": "Error Rate",
        "type": "graph",
        "targets": [
          {
            "expr": "rate(http_requests_total{status=~\"5..\"}[5m])",
            "legendFormat": "5xx errors"
          }
        ]
      },
      {
        "id": 5,
        "title": "Database Connections",
        "type": "graph",
        "targets": [
          {
            "expr": "pg_stat_database_numbackends",
            "legendFormat": "Active connections"
          }
        ]
      },
      {
        "id": 6,
        "title": "Redis Memory Usage",
        "type": "graph",
        "targets": [
          {
            "expr": "redis_memory_used_bytes",
            "legendFormat": "Used memory"
          }
        ],
        "fieldConfig": {
          "defaults": {
            "unit": "bytes"
          }
        }
      }
    ],
    "refresh": "5s",
    "time": {
      "from": "now-1h",
      "to": "now"
    }
  }
}
```

**Explications:**

- **Datasource:** Prometheus auto-configuré
- **Dashboards:** Provisionnés automatiquement au démarrage
- **Panels:** Visiteurs temps réel, RPS, latence, erreurs, DB, Redis
- **Refresh:** 5s pour temps réel
- **Public Access:** Anonyme en mode Viewer (docker-compose)

---

### Étape 5: SSL/TLS avec Let's Encrypt

**Description:** Configuration SSL automatique avec certbot.

**Commandes:**

```bash
# Installation certbot (déjà fait apt install)
# Obtenir certificat initial
sudo certbot --nginx -d maicivy.com -d www.maicivy.com -d analytics.maicivy.com

# Ou en mode standalone (si Nginx pas encore configuré)
sudo certbot certonly --standalone -d maicivy.com -d www.maicivy.com -d analytics.maicivy.com

# Test auto-renewal
sudo certbot renew --dry-run

# Setup cron pour auto-renewal (déjà fait par défaut certbot)
# Vérifier: sudo systemctl status certbot.timer
```

**Fichier:** `scripts/renew-ssl.sh`

**Code:**

```bash
#!/bin/bash

# Script de renouvellement SSL
# Exécuté par cron tous les jours

set -e

echo "$(date): Checking SSL certificates renewal..."

# Renouveler les certificats si nécessaire
certbot renew --quiet --nginx

# Recharger Nginx si certificats renouvelés
if [ $? -eq 0 ]; then
    echo "$(date): SSL certificates renewed, reloading Nginx..."
    docker exec maicivy_nginx nginx -s reload
else
    echo "$(date): No renewal needed."
fi
```

**Cron job:**

```bash
# Ajouter au crontab (sudo crontab -e)
0 3 * * * /path/to/scripts/renew-ssl.sh >> /var/log/ssl-renew.log 2>&1
```

**Explications:**

- **Certbot:** Obtient et renouvelle certificats Let's Encrypt
- **Auto-renewal:** Script cron journalier (3h du matin)
- **Nginx reload:** Recharge config après renouvellement
- **Wildcard (optionnel):** Nécessite DNS-01 challenge (plus complexe)

---

### Étape 6: Health Checks

**Description:** Endpoints et scripts de vérification santé des services.

**Backend Health Check (déjà implémenté dans 02_BACKEND_FOUNDATION):**

```go
// GET /health - Shallow check
func HealthHandler(c *fiber.Ctx) error {
    return c.JSON(fiber.Map{
        "status": "ok",
        "timestamp": time.Now().Unix(),
        "service": "maicivy-backend",
    })
}

// GET /health/deep - Deep check (DB, Redis)
func HealthDeepHandler(c *fiber.Ctx) error {
    ctx := context.Background()

    // Check PostgreSQL
    dbErr := database.DB.Exec("SELECT 1").Error

    // Check Redis
    redisErr := database.Redis.Ping(ctx).Err()

    status := "ok"
    statusCode := 200

    if dbErr != nil || redisErr != nil {
        status = "degraded"
        statusCode = 503
    }

    return c.Status(statusCode).JSON(fiber.Map{
        "status": status,
        "checks": fiber.Map{
            "database": dbErr == nil,
            "redis": redisErr == nil,
        },
        "timestamp": time.Now().Unix(),
    })
}
```

**Script de Monitoring externe:**

**Fichier:** `scripts/health-check.sh`

**Code:**

```bash
#!/bin/bash

# Script de vérification santé des services
# Peut être utilisé par monitoring externe (UptimeRobot, etc.)

set -e

API_URL="https://maicivy.com"
TIMEOUT=10

# Couleurs
RED='\033[0;31m'
GREEN='\033[0;32m'
NC='\033[0m' # No Color

# Fonction de check
check_service() {
    local service_name=$1
    local url=$2

    echo -n "Checking $service_name... "

    if curl -f -s -o /dev/null -w "%{http_code}" --max-time $TIMEOUT "$url" | grep -q "200"; then
        echo -e "${GREEN}OK${NC}"
        return 0
    else
        echo -e "${RED}FAIL${NC}"
        return 1
    fi
}

# Checks
FAILED=0

check_service "Frontend" "$API_URL" || FAILED=$((FAILED+1))
check_service "Backend Health" "$API_URL/health" || FAILED=$((FAILED+1))
check_service "Backend Deep Health" "$API_URL/health/deep" || FAILED=$((FAILED+1))
check_service "Grafana" "https://analytics.maicivy.com" || FAILED=$((FAILED+1))

# Résultat final
echo ""
if [ $FAILED -eq 0 ]; then
    echo -e "${GREEN}All services are healthy!${NC}"
    exit 0
else
    echo -e "${RED}$FAILED service(s) failed health check!${NC}"
    exit 1
fi
```

**Explications:**

- **Shallow Health:** Vérification rapide (service up)
- **Deep Health:** Vérification dépendances (DB, Redis)
- **Script externe:** Monitoring automatisé ou manuel
- **Kubernetes-style:** Liveness + Readiness probes

---

### Étape 7: Logging Structuré

**Description:** Configuration logging JSON pour parsing et analyse facile.

**Backend Logger (déjà configuré dans 02_BACKEND_FOUNDATION avec zerolog):**

```go
// Middleware logging HTTP
func LoggerMiddleware(c *fiber.Ctx) error {
    start := time.Now()

    // Process request
    err := c.Next()

    // Log après traitement
    logger.Info().
        Str("method", c.Method()).
        Str("path", c.Path()).
        Int("status", c.Response().StatusCode()).
        Dur("duration", time.Since(start)).
        Str("ip", c.IP()).
        Str("user_agent", c.Get("User-Agent")).
        Msg("HTTP Request")

    return err
}
```

**Configuration Docker Logging:**

**Fichier:** `docker/docker-compose.prod.yml` (ajout section logging)

```yaml
services:
  backend:
    # ... (config existante)
    logging:
      driver: "json-file"
      options:
        max-size: "10m"
        max-file: "3"
        labels: "service,app"
```

**Visualisation logs:**

```bash
# Logs en temps réel
docker-compose -f docker/docker-compose.prod.yml logs -f backend

# Logs avec filtering
docker-compose logs -f backend | grep ERROR

# Logs JSON parsés (jq)
docker logs maicivy_backend --tail 100 | jq '.level, .msg, .duration'
```

**Optionnel: Loki pour centralisation:**

```yaml
# Ajouter à docker-compose.prod.yml
  loki:
    image: grafana/loki:latest
    ports:
      - "127.0.0.1:3100:3100"
    volumes:
      - ./loki/loki-config.yml:/etc/loki/local-config.yaml
      - loki_data:/loki

  promtail:
    image: grafana/promtail:latest
    volumes:
      - /var/lib/docker/containers:/var/lib/docker/containers:ro
      - ./loki/promtail-config.yml:/etc/promtail/config.yml
    command: -config.file=/etc/promtail/config.yml
```

**Explications:**

- **JSON Logs:** Structured logging pour parsing automatisé
- **Rotation:** Max 10MB par fichier, 3 fichiers gardés
- **Docker Logs:** Logs capturés par Docker daemon
- **Loki (optionnel):** Centralisation logs multi-services (comme Elasticsearch mais lightweight)

---

### Étape 8: Backup PostgreSQL

**Description:** Stratégie de backup automatisée avec pg_dump.

**Fichier:** `scripts/backup-postgres.sh`

**Code:**

```bash
#!/bin/bash

# Script de backup PostgreSQL
# Exécuté quotidiennement par cron

set -e

# Configuration
CONTAINER_NAME="maicivy_postgres"
POSTGRES_USER="${POSTGRES_USER:-maicivy}"
POSTGRES_DB="${POSTGRES_DB:-maicivy}"
BACKUP_DIR="/opt/maicivy/backups"
RETENTION_DAYS=30

# Créer répertoire backup si inexistant
mkdir -p "$BACKUP_DIR"

# Nom du fichier avec timestamp
TIMESTAMP=$(date +"%Y%m%d_%H%M%S")
BACKUP_FILE="$BACKUP_DIR/maicivy_backup_$TIMESTAMP.sql.gz"

echo "$(date): Starting PostgreSQL backup..."

# Backup avec pg_dump via Docker
docker exec -t $CONTAINER_NAME pg_dump -U $POSTGRES_USER $POSTGRES_DB | gzip > "$BACKUP_FILE"

if [ $? -eq 0 ]; then
    echo "$(date): Backup successful: $BACKUP_FILE"

    # Taille du backup
    SIZE=$(du -h "$BACKUP_FILE" | cut -f1)
    echo "$(date): Backup size: $SIZE"

    # Suppression des backups anciens (> RETENTION_DAYS)
    find "$BACKUP_DIR" -name "maicivy_backup_*.sql.gz" -mtime +$RETENTION_DAYS -delete
    echo "$(date): Old backups cleaned (retention: $RETENTION_DAYS days)"
else
    echo "$(date): Backup FAILED!" >&2
    exit 1
fi

# Optionnel: Upload vers S3
# aws s3 cp "$BACKUP_FILE" s3://maicivy-backups/postgres/
```

**Script de Restore:**

**Fichier:** `scripts/restore-postgres.sh`

**Code:**

```bash
#!/bin/bash

# Script de restauration PostgreSQL

set -e

if [ -z "$1" ]; then
    echo "Usage: $0 <backup_file.sql.gz>"
    exit 1
fi

BACKUP_FILE=$1
CONTAINER_NAME="maicivy_postgres"
POSTGRES_USER="${POSTGRES_USER:-maicivy}"
POSTGRES_DB="${POSTGRES_DB:-maicivy}"

if [ ! -f "$BACKUP_FILE" ]; then
    echo "Error: Backup file not found: $BACKUP_FILE"
    exit 1
fi

echo "WARNING: This will OVERWRITE the current database!"
read -p "Are you sure? (yes/no): " CONFIRM

if [ "$CONFIRM" != "yes" ]; then
    echo "Restore cancelled."
    exit 0
fi

echo "$(date): Starting PostgreSQL restore from $BACKUP_FILE..."

# Drop et recréer la DB
docker exec -t $CONTAINER_NAME psql -U $POSTGRES_USER -c "DROP DATABASE IF EXISTS $POSTGRES_DB;"
docker exec -t $CONTAINER_NAME psql -U $POSTGRES_USER -c "CREATE DATABASE $POSTGRES_DB;"

# Restore
gunzip -c "$BACKUP_FILE" | docker exec -i $CONTAINER_NAME psql -U $POSTGRES_USER -d $POSTGRES_DB

if [ $? -eq 0 ]; then
    echo "$(date): Restore successful!"
else
    echo "$(date): Restore FAILED!" >&2
    exit 1
fi
```

**Cron job:**

```bash
# Ajouter au crontab (sudo crontab -e)
0 2 * * * /opt/maicivy/scripts/backup-postgres.sh >> /var/log/maicivy-backup.log 2>&1
```

**Explications:**

- **pg_dump:** Backup complet de la base de données
- **Compression:** gzip pour réduire espace disque
- **Rétention:** 30 jours (configurable)
- **Restore:** Script interactif avec confirmation
- **Cron:** Backup quotidien à 2h du matin
- **Optionnel:** Upload S3 pour backup off-site

---

### Étape 9: Backup Redis

**Description:** Backup des données Redis (RDB snapshots).

**Configuration Redis (déjà dans docker-compose):**

```yaml
redis:
  command: redis-server --appendonly yes --requirepass ${REDIS_PASSWORD}
  volumes:
    - redis_data:/data
```

**Script backup Redis:**

**Fichier:** `scripts/backup-redis.sh`

**Code:**

```bash
#!/bin/bash

# Script de backup Redis

set -e

CONTAINER_NAME="maicivy_redis"
BACKUP_DIR="/opt/maicivy/backups/redis"
RETENTION_DAYS=7

mkdir -p "$BACKUP_DIR"

TIMESTAMP=$(date +"%Y%m%d_%H%M%S")
BACKUP_FILE="$BACKUP_DIR/redis_backup_$TIMESTAMP.rdb"

echo "$(date): Starting Redis backup..."

# Trigger BGSAVE
docker exec $CONTAINER_NAME redis-cli -a ${REDIS_PASSWORD} BGSAVE

# Attendre fin du save
sleep 5

# Copier le fichier RDB
docker cp $CONTAINER_NAME:/data/dump.rdb "$BACKUP_FILE"

if [ $? -eq 0 ]; then
    echo "$(date): Redis backup successful: $BACKUP_FILE"

    # Cleanup old backups
    find "$BACKUP_DIR" -name "redis_backup_*.rdb" -mtime +$RETENTION_DAYS -delete
else
    echo "$(date): Redis backup FAILED!" >&2
    exit 1
fi
```

**Explications:**

- **AOF + RDB:** Redis configuré avec persistence (appendonly + snapshots)
- **BGSAVE:** Background save (non-bloquant)
- **Rétention:** 7 jours (Redis moins critique que PostgreSQL)
- **Restore:** Copier le fichier .rdb dans /data et restart Redis

---

### Étape 10: Déploiement Initial

**Description:** Procédure de déploiement en production.

**Script:** `scripts/deploy.sh`

**Code:**

```bash
#!/bin/bash

# Script de déploiement production

set -e

echo "========================================="
echo "  maicivy - Production Deployment"
echo "========================================="

# Variables
PROJECT_DIR="/opt/maicivy"
ENV_FILE="$PROJECT_DIR/.env"

# 1. Vérifier environnement
if [ ! -f "$ENV_FILE" ]; then
    echo "Error: .env file not found at $ENV_FILE"
    exit 1
fi

# 2. Pull latest images
echo "Pulling latest Docker images..."
cd "$PROJECT_DIR"
docker-compose -f docker/docker-compose.prod.yml pull

# 3. Stop services gracefully
echo "Stopping services..."
docker-compose -f docker/docker-compose.prod.yml down

# 4. Backup database avant déploiement
echo "Creating pre-deployment backup..."
/opt/maicivy/scripts/backup-postgres.sh

# 5. Start services
echo "Starting services..."
docker-compose -f docker/docker-compose.prod.yml up -d

# 6. Wait for services to be healthy
echo "Waiting for services to be healthy..."
sleep 10

# 7. Health check
echo "Running health checks..."
/opt/maicivy/scripts/health-check.sh

if [ $? -eq 0 ]; then
    echo "========================================="
    echo "  Deployment successful!"
    echo "========================================="
else
    echo "========================================="
    echo "  Deployment FAILED - Health checks failed!"
    echo "  Consider rollback"
    echo "========================================="
    exit 1
fi

# 8. Cleanup old images
echo "Cleaning up old Docker images..."
docker image prune -f

echo "Deployment completed at $(date)"
```

**Fichier `.env` template:**

**Fichier:** `.env.production.example`

**Code:**

```bash
# Environment Configuration - Production

# General
ENV=production

# Database PostgreSQL
POSTGRES_DB=maicivy
POSTGRES_USER=maicivy
POSTGRES_PASSWORD=CHANGE_ME_STRONG_PASSWORD_HERE

# Redis
REDIS_PASSWORD=CHANGE_ME_STRONG_PASSWORD_HERE

# API Keys
CLAUDE_API_KEY=sk-ant-xxxxx
OPENAI_API_KEY=sk-xxxxx

# Frontend
NEXT_PUBLIC_API_URL=https://maicivy.com/api
NEXT_PUBLIC_WS_URL=wss://maicivy.com/ws
FRONTEND_URL=https://maicivy.com

# Grafana
GRAFANA_ADMIN_USER=admin
GRAFANA_ADMIN_PASSWORD=CHANGE_ME_STRONG_PASSWORD_HERE

# Docker Registry (optionnel)
DOCKER_REGISTRY=ghcr.io/username
```

**Explications:**

- **Deployment:** Script automatisé de déploiement
- **Backup:** Backup pré-déploiement (safety net)
- **Health Checks:** Validation post-déploiement
- **Rollback:** Si health checks fail, possibilité de rollback
- **Cleanup:** Suppression images Docker obsolètes

---

## 🧪 Tests

### Test 1: Vérification SSL

```bash
# Test SSL configuration
openssl s_client -connect maicivy.com:443 -servername maicivy.com

# Test SSL grade (externe)
# https://www.ssllabs.com/ssltest/analyze.html?d=maicivy.com

# Test HTTPS redirect
curl -I http://maicivy.com
# Doit retourner 301 vers https://
```

### Test 2: Vérification Compression

```bash
# Test gzip compression
curl -H "Accept-Encoding: gzip" -I https://maicivy.com

# Doit contenir: Content-Encoding: gzip
```

### Test 3: Vérification Security Headers

```bash
# Test headers de sécurité
curl -I https://maicivy.com

# Doit contenir:
# Strict-Transport-Security: max-age=31536000
# X-Frame-Options: DENY
# X-Content-Type-Options: nosniff
```

### Test 4: Vérification Rate Limiting

```bash
# Test rate limiting (100 req/min)
for i in {1..150}; do
    curl -s -o /dev/null -w "%{http_code}\n" https://maicivy.com/api/health
done

# Les dernières requêtes doivent retourner 429 (Too Many Requests)
```

### Test 5: Vérification Monitoring

```bash
# Test Prometheus
curl http://localhost:9090/metrics

# Test Grafana public
curl https://analytics.maicivy.com

# Test métriques backend
curl https://maicivy.com/metrics
```

### Test 6: Vérification Backups

```bash
# Test backup PostgreSQL
./scripts/backup-postgres.sh

# Vérifier fichier créé
ls -lh /opt/maicivy/backups/

# Test restore (sur environnement de test!)
./scripts/restore-postgres.sh /opt/maicivy/backups/maicivy_backup_YYYYMMDD_HHMMSS.sql.gz
```

---

## ⚠️ Points d'Attention

- ⚠️ **Secrets:** Ne JAMAIS commit `.env` ou fichiers contenant secrets. Utiliser `.env.example` template.

- ⚠️ **SSL Renewal:** Vérifier que certbot timer est actif: `sudo systemctl status certbot.timer`. Let's Encrypt expire après 90 jours.

- ⚠️ **Firewall:** Configurer UFW pour bloquer tous les ports sauf 80, 443, et SSH:
  ```bash
  sudo ufw allow 22/tcp
  sudo ufw allow 80/tcp
  sudo ufw allow 443/tcp
  sudo ufw enable
  ```

- ⚠️ **Backup Testing:** Tester régulièrement la restauration des backups. Un backup non testé est inutile.

- ⚠️ **Disk Space:** Surveiller l'espace disque (logs, backups, Prometheus data). Configurer alertes.

- ⚠️ **Rate Limiting:** Ajuster les limites selon le trafic réel. Trop strict = faux positifs, trop laxiste = abus possibles.

- ⚠️ **Nginx Buffer Size:** Si uploads PDF volumineux, augmenter `client_max_body_size`.

- ⚠️ **PostgreSQL Connections:** Limiter le nombre de connexions simultanées dans `postgresql.conf` (max_connections).

- ⚠️ **Redis Memory:** Configurer `maxmemory` et `maxmemory-policy` pour éviter OOM:
  ```bash
  redis-cli CONFIG SET maxmemory 256mb
  redis-cli CONFIG SET maxmemory-policy allkeys-lru
  ```

- 💡 **Monitoring External:** Utiliser service externe (UptimeRobot, Pingdom) pour monitoring uptime et alertes.

- 💡 **CDN:** Considérer Cloudflare (gratuit) pour:
  - Protection DDoS
  - Caching global
  - SSL automatique
  - Analytics

- 💡 **Database Tuning:** Optimiser PostgreSQL selon charge (shared_buffers, work_mem, effective_cache_size).

- 💡 **Nginx Caching:** Considérer proxy_cache pour APIs peu changeantes (ex: liste thèmes CV).

---

## 📚 Ressources

### Documentation Officielle

- [Nginx Configuration Guide](https://nginx.org/en/docs/http/ngx_http_core_module.html)
- [Let's Encrypt Documentation](https://letsencrypt.org/docs/)
- [Prometheus Configuration](https://prometheus.io/docs/prometheus/latest/configuration/configuration/)
- [Grafana Provisioning](https://grafana.com/docs/grafana/latest/administration/provisioning/)
- [Docker Compose Production](https://docs.docker.com/compose/production/)

### Tutoriels

- [SSL Best Practices](https://github.com/ssllabs/research/wiki/SSL-and-TLS-Deployment-Best-Practices)
- [Nginx Security Headers](https://securityheaders.com/)
- [PostgreSQL Backup & Recovery](https://www.postgresql.org/docs/current/backup.html)

### Outils

- [SSL Labs Server Test](https://www.ssllabs.com/ssltest/)
- [Security Headers Scanner](https://securityheaders.com/)
- [UptimeRobot](https://uptimerobot.com/) - Monitoring externe gratuit
- [Cloudflare](https://www.cloudflare.com/) - CDN + protection DDoS

---

## ✅ Checklist de Complétion

- [ ] VPS OVH provisionné et accessible via SSH
- [ ] Domaine configuré (DNS A record vers IP VPS)
- [ ] Docker et Docker Compose installés
- [ ] Nginx installé et configuré
- [ ] Certificats SSL Let's Encrypt obtenus
- [ ] Auto-renewal SSL configuré (certbot timer)
- [ ] Firewall UFW configuré (ports 22, 80, 443)
- [ ] Docker Compose production déployé
- [ ] Prometheus configuré et scraping métriques
- [ ] Grafana configuré avec dashboards
- [ ] Dashboard Grafana accessible publiquement en readonly
- [ ] Health checks testés (shallow + deep)
- [ ] Logging JSON configuré et fonctionnel
- [ ] Script backup PostgreSQL testé
- [ ] Script restore PostgreSQL testé
- [ ] Cron jobs configurés (SSL renewal, backups)
- [ ] Security headers testés (SSL Labs, securityheaders.com)
- [ ] Compression gzip/brotli vérifiée
- [ ] Rate limiting testé
- [ ] Monitoring externe configuré (UptimeRobot ou équivalent)
- [ ] Documentation déploiement mise à jour
- [ ] Runbook incidents créé (procédures rollback, restore, etc.)

---

**Dernière mise à jour:** 2025-12-08
**Auteur:** Alexi
