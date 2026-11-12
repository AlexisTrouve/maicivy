# Infrastructure Production Guide - maicivy

## Table des Matières

- [Architecture](#architecture)
- [Déploiement Initial](#déploiement-initial)
- [Configuration Nginx](#configuration-nginx)
- [SSL/TLS Configuration](#ssltls-configuration)
- [Monitoring](#monitoring)
- [Backup & Restore](#backup--restore)
- [Maintenance](#maintenance)
- [Troubleshooting](#troubleshooting)
- [Scaling](#scaling)

---

## Architecture

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
│  │  - SSL Termination              │   │
│  │  - Rate Limiting                │   │
│  └──────────┬──────────────────────┘   │
│             │                           │
│  ┌──────────┴──────────────────────┐   │
│  │   Docker Compose Network        │   │
│  │                                 │   │
│  │  Frontend ─── Backend           │   │
│  │     │           │               │   │
│  │  PostgreSQL ─ Redis             │   │
│  │                                 │   │
│  │  Prometheus ─ Grafana           │   │
│  └─────────────────────────────────┘  │
└─────────────────────────────────────────┘
```

### Services

- **Nginx**: Reverse proxy, SSL termination, rate limiting
- **Frontend**: Next.js 14 (port 3000)
- **Backend**: Go/Fiber API (port 8080)
- **PostgreSQL**: Base de données principale (port 5432)
- **Redis**: Cache et session store (port 6379)
- **Prometheus**: Collecte métriques (port 9090)
- **Grafana**: Dashboard public (port 3001)
- **Node Exporter**: Métriques système (port 9100)

---

## Déploiement Initial

### Prérequis

**VPS:**
- Ubuntu 20.04+ ou Debian 11+
- 2GB RAM minimum (4GB recommandé)
- 2 vCPU minimum
- 40GB SSD minimum
- IP publique fixe

**Domaine:**
- Nom de domaine configuré
- DNS A record: `maicivy.com` → IP VPS
- DNS A record: `analytics.maicivy.com` → IP VPS

### 1. Setup Serveur

```bash
# Se connecter au VPS
ssh root@YOUR_VPS_IP

# Mise à jour système
apt update && apt upgrade -y

# Installation Docker
apt install docker.io docker-compose -y
systemctl enable docker
systemctl start docker

# Installation Nginx
apt install nginx -y
systemctl enable nginx

# Installation Certbot
apt install certbot python3-certbot-nginx -y

# Installation utilitaires
apt install curl wget git htop -y

# Création utilisateur maicivy (optionnel)
adduser maicivy
usermod -aG docker maicivy
usermod -aG sudo maicivy
```

### 2. Configuration Firewall

```bash
# Configuration UFW
ufw allow 22/tcp    # SSH
ufw allow 80/tcp    # HTTP
ufw allow 443/tcp   # HTTPS
ufw enable

# Vérifier status
ufw status
```

### 3. Cloner Projet

```bash
# Se placer dans /opt
cd /opt

# Cloner repository
git clone https://github.com/YOUR_USERNAME/maicivy.git
cd maicivy

# Ou upload via rsync/scp
rsync -avz --exclude 'node_modules' --exclude '.git' \
  ~/maicivy/ root@YOUR_VPS_IP:/opt/maicivy/
```

### 4. Configuration Environment

```bash
# Copier template
cp .env.production.example .env

# Éditer .env
nano .env

# Générer passwords forts:
# - POSTGRES_PASSWORD
# - REDIS_PASSWORD
# - GRAFANA_ADMIN_PASSWORD

# Ajouter API keys:
# - CLAUDE_API_KEY
# - OPENAI_API_KEY

# Remplacer maicivy.com par votre domaine
```

### 5. Setup SSL/TLS

```bash
# Obtenir certificats Let's Encrypt
certbot certonly --standalone \
  -d maicivy.com \
  -d www.maicivy.com \
  -d analytics.maicivy.com \
  --non-interactive \
  --agree-tos \
  --email YOUR_EMAIL

# Vérifier certificats
ls -la /etc/letsencrypt/live/maicivy.com/

# Test auto-renewal
certbot renew --dry-run
```

### 6. Déploiement

```bash
# Rendre scripts exécutables
chmod +x scripts/*.sh

# Pull images Docker
cd /opt/maicivy
docker-compose -f docker/docker-compose.prod.yml pull

# Démarrer services
docker-compose -f docker/docker-compose.prod.yml up -d

# Vérifier logs
docker-compose -f docker/docker-compose.prod.yml logs -f

# Vérifier health
./scripts/health-check.sh
```

### 7. Configuration Systemd (Auto-start)

```bash
# Copier service file
cp scripts/systemd/maicivy.service /etc/systemd/system/

# Recharger systemd
systemctl daemon-reload

# Activer service
systemctl enable maicivy

# Tester
systemctl start maicivy
systemctl status maicivy
```

### 8. Configuration Cron Jobs

```bash
# Éditer crontab root
crontab -e

# Ajouter:
# SSL renewal (tous les jours à 3h)
0 3 * * * /opt/maicivy/scripts/renew-ssl.sh >> /var/log/ssl-renew.log 2>&1

# Backup PostgreSQL (tous les jours à 2h)
0 2 * * * /opt/maicivy/scripts/backup-postgres.sh >> /var/log/maicivy-backup.log 2>&1

# Backup Redis (tous les jours à 2h30)
30 2 * * * /opt/maicivy/scripts/backup-redis.sh >> /var/log/maicivy-backup.log 2>&1
```

---

## Configuration Nginx

### Structure Fichiers

```
/opt/maicivy/docker/nginx/
├── nginx.conf           # Configuration principale
└── conf.d/              # Configurations additionnelles (optionnel)
```

### Test Configuration

```bash
# Test config Nginx
nginx -t

# Reload Nginx
nginx -s reload

# Ou via systemctl
systemctl reload nginx
```

### Personnalisation

**Rate Limiting:**

Éditer `nginx.conf`, section `limit_req_zone`:

```nginx
# 100 req/min par IP
limit_req_zone $binary_remote_addr zone=ip_limit:10m rate=100r/m;

# Ajuster selon besoin (ex: 200r/m pour plus de tolérance)
```

**Upload Size:**

```nginx
# Pour uploads PDF plus volumineux
client_max_body_size 20M;
```

---

## SSL/TLS Configuration

### Renouvellement Automatique

Let's Encrypt certificates expirent après 90 jours.

**Vérifier timer certbot:**

```bash
systemctl status certbot.timer
systemctl list-timers | grep certbot
```

**Test manuel renewal:**

```bash
certbot renew --dry-run
```

**Forcer renewal:**

```bash
certbot renew --force-renewal
docker exec maicivy_nginx nginx -s reload
```

### Wildcard Certificates (Optionnel)

```bash
# Nécessite DNS-01 challenge (plus complexe)
certbot certonly --manual --preferred-challenges dns \
  -d maicivy.com -d '*.maicivy.com'
```

---

## Monitoring

### Prometheus

**Accès:** `http://localhost:9090` (local seulement)

**Vérifier targets:**

```bash
curl http://localhost:9090/api/v1/targets | jq
```

**Métriques disponibles:**

- Backend: `http://backend:8080/metrics`
- Node Exporter: `http://node-exporter:9100/metrics`

### Grafana

**Accès public:** `https://analytics.maicivy.com`

**Accès admin:** `https://analytics.maicivy.com` + login avec credentials `.env`

**Ajout dashboards:**

1. Placer fichier `.json` dans `monitoring/grafana/dashboards/`
2. Redémarrer Grafana: `docker-compose restart grafana`

**Import dashboards communautaires:**

- Node Exporter Full: Dashboard ID `1860`
- Docker Container: Dashboard ID `893`

### Alerting (Optionnel)

**Setup Alertmanager:**

```yaml
# Ajouter à docker-compose.prod.yml
  alertmanager:
    image: prom/alertmanager:latest
    volumes:
      - ./prometheus/alertmanager.yml:/etc/alertmanager/config.yml
    ports:
      - "127.0.0.1:9093:9093"
```

**Configuration Slack/Email:**

```yaml
# prometheus/alertmanager.yml
route:
  receiver: 'slack'
receivers:
  - name: 'slack'
    slack_configs:
      - api_url: 'YOUR_SLACK_WEBHOOK'
        channel: '#alerts'
```

---

## Backup & Restore

### PostgreSQL

**Backup manuel:**

```bash
./scripts/backup-postgres.sh
```

**Restore:**

```bash
./scripts/restore-postgres.sh /opt/maicivy/backups/maicivy_backup_20250108_020000.sql.gz
```

**Vérifier backups:**

```bash
ls -lh /opt/maicivy/backups/
```

**Upload S3 (optionnel):**

```bash
# Installer AWS CLI
apt install awscli

# Configurer
aws configure

# Modifier backup-postgres.sh (ligne finale):
aws s3 cp "$BACKUP_FILE" s3://YOUR_BUCKET/postgres/
```

### Redis

**Backup manuel:**

```bash
./scripts/backup-redis.sh
```

**Restore:**

```bash
./scripts/restore-redis.sh /opt/maicivy/backups/redis/redis_backup_20250108_023000.rdb
```

### Stratégie Backup

- **PostgreSQL:** Quotidien, rétention 30 jours
- **Redis:** Quotidien, rétention 7 jours
- **Test restore:** Mensuel sur environnement de test

---

## Maintenance

### Logs

**View logs temps réel:**

```bash
# Tous services
docker-compose -f docker/docker-compose.prod.yml logs -f

# Service spécifique
docker-compose logs -f backend
docker-compose logs -f frontend

# Dernières 100 lignes
docker logs --tail 100 maicivy_backend
```

**Logs Nginx:**

```bash
tail -f /var/log/nginx/access.log
tail -f /var/log/nginx/error.log
```

**Log rotation:**

Docker logs sont automatiquement rotés (config `max-size: 10m`, `max-file: 3`).

### Updates

**Update application:**

```bash
cd /opt/maicivy
git pull origin main
./scripts/deploy.sh
```

**Update Docker images:**

```bash
docker-compose -f docker/docker-compose.prod.yml pull
docker-compose -f docker/docker-compose.prod.yml up -d
```

**Update système:**

```bash
apt update && apt upgrade -y
reboot  # Si kernel upgrade
```

### Cleanup

**Docker cleanup:**

```bash
# Remove unused images
docker image prune -a -f

# Remove unused volumes
docker volume prune -f

# Remove unused containers
docker container prune -f
```

**Disk space:**

```bash
# Vérifier usage
df -h
du -sh /var/lib/docker

# Nettoyer logs anciens
find /var/log -type f -name "*.log" -mtime +30 -delete
```

---

## Troubleshooting

### Service Down

**Backend ne démarre pas:**

```bash
# Check logs
docker logs maicivy_backend

# Causes communes:
# - DB connection fail (vérifier POSTGRES_PASSWORD)
# - Redis connection fail (vérifier REDIS_PASSWORD)
# - Missing API keys

# Tester connexions
docker exec maicivy_backend ping postgres
docker exec maicivy_backend ping redis
```

**Frontend ne démarre pas:**

```bash
docker logs maicivy_frontend

# Causes communes:
# - Build failure
# - Env vars manquantes (NEXT_PUBLIC_API_URL)
```

### SSL Issues

**Certificats expirés:**

```bash
certbot renew --force-renewal
docker exec maicivy_nginx nginx -s reload
```

**Nginx ne démarre pas:**

```bash
# Test config
nginx -t

# Check logs
tail -f /var/log/nginx/error.log
```

### Database Issues

**PostgreSQL connection refused:**

```bash
# Check service
docker exec maicivy_postgres pg_isready

# Check connections
docker exec maicivy_postgres psql -U maicivy -c "\conninfo"

# Restart
docker-compose restart postgres
```

**Disk full:**

```bash
# Check space
df -h

# Cleanup old backups
find /opt/maicivy/backups -mtime +30 -delete

# Vacuum PostgreSQL
docker exec maicivy_postgres psql -U maicivy -d maicivy -c "VACUUM FULL;"
```

### Performance Issues

**High CPU:**

```bash
# Check processes
htop

# Check Docker stats
docker stats

# Identify heavy containers
docker stats --no-stream | sort -k3 -h
```

**High Memory:**

```bash
# Check memory
free -h

# Restart services
docker-compose restart
```

---

## Scaling

### Horizontal Scaling

**Load Balancer:**

```nginx
# nginx.conf
upstream backend_servers {
    server backend1:8080;
    server backend2:8080;
    server backend3:8080;
}
```

**Database Replication:**

- Primary-Replica PostgreSQL
- Read replicas pour queries intensives

### Vertical Scaling

**Upgrade VPS:**

- Augmenter RAM/CPU via panel OVH
- Redémarrer VPS

**Resource Limits:**

```yaml
# docker-compose.prod.yml
services:
  backend:
    deploy:
      resources:
        limits:
          cpus: '2.0'
          memory: 2G
        reservations:
          cpus: '1.0'
          memory: 1G
```

### Caching

**Nginx Proxy Cache:**

```nginx
# nginx.conf
proxy_cache_path /var/cache/nginx levels=1:2 keys_zone=api_cache:10m;

location /api/cv {
    proxy_cache api_cache;
    proxy_cache_valid 200 5m;
    proxy_pass http://backend_server;
}
```

**Redis Cache:**

- Déjà implémenté dans backend
- Augmenter `maxmemory` si nécessaire

### CDN

**Cloudflare (gratuit):**

1. Créer compte Cloudflare
2. Ajouter domaine
3. Changer nameservers chez registrar
4. Activer proxy (orange cloud)
5. Activer cache, minification, Brotli

**Avantages:**

- DDoS protection
- Global caching
- SSL automatique
- Analytics

---

## Monitoring Externe

### UptimeRobot

1. Créer compte: https://uptimerobot.com
2. Ajouter monitors:
   - `https://maicivy.com` (HTTP)
   - `https://maicivy.com/health` (HTTP - keyword "ok")
   - `https://analytics.maicivy.com` (HTTP)
3. Configurer alertes (email, Slack)

### Status Page

- Utiliser UptimeRobot Status Page (gratuit)
- Ou custom: `status.maicivy.com`

---

## Checklist Production

### Pre-Launch

- [ ] SSL certificats valides
- [ ] DNS configuré correctement
- [ ] Firewall actif (UFW)
- [ ] Backups automatiques configurés
- [ ] Monitoring actif (Prometheus + Grafana)
- [ ] Health checks passent
- [ ] Security headers testés (securityheaders.com)
- [ ] SSL grade A+ (ssllabs.com)
- [ ] Rate limiting testé
- [ ] Logs configurés et rotés
- [ ] Systemd service actif
- [ ] Cron jobs configurés
- [ ] Documentation à jour

### Post-Launch

- [ ] Monitoring externe actif (UptimeRobot)
- [ ] Alerting configuré
- [ ] Test restore backups effectué
- [ ] Runbook incidents créé
- [ ] Team formée sur procédures
- [ ] Status page publique

---

## Support

### Ressources

- **Documentation:** `/opt/maicivy/docs/`
- **Logs:** `/var/log/nginx/`, `docker logs`
- **Monitoring:** `https://analytics.maicivy.com`

### Contacts

- **Développeur:** [YOUR_EMAIL]
- **VPS Support:** support.ovh.com
- **Let's Encrypt:** https://community.letsencrypt.org

---

**Version:** 1.0
**Date:** 2025-12-08
**Auteur:** Alexi
