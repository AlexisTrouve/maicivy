# Quick Start - Production Deployment

Guide rapide pour déployer maicivy en production sur VPS OVH.

---

## Prérequis

- **VPS OVH** (ou autre): 2GB RAM, 2 vCPU, 40GB SSD
- **Domaine**: maicivy.com (remplacer par le vôtre)
- **DNS configuré**: A records vers IP VPS
- **API Keys**: Anthropic (Claude) et OpenAI (GPT-4)

---

## Étape 1: Setup Initial VPS

### 1.1 Se Connecter au VPS

```bash
ssh root@YOUR_VPS_IP
```

### 1.2 Cloner Projet

```bash
cd /opt
git clone https://github.com/YOUR_USERNAME/maicivy.git
cd maicivy
```

### 1.3 Setup Infrastructure

```bash
# Rendre script exécutable
chmod +x scripts/setup-infrastructure.sh

# Exécuter setup (installe Docker, Nginx, Certbot, etc.)
sudo DOMAIN=maicivy.com EMAIL=your@email.com ./scripts/setup-infrastructure.sh
```

Ce script va:
- Installer Docker + Docker Compose
- Installer Nginx
- Installer Certbot
- Configurer firewall UFW
- Obtenir certificats SSL Let's Encrypt
- Créer user maicivy
- Configurer cron jobs

**Durée:** ~5-10 minutes

---

## Étape 2: Configuration

### 2.1 Créer Fichier .env

```bash
cd /opt/maicivy
cp .env.production.example .env
nano .env
```

### 2.2 Configurer Secrets

```env
# PostgreSQL
POSTGRES_PASSWORD=GENERATE_STRONG_PASSWORD_HERE

# Redis
REDIS_PASSWORD=GENERATE_STRONG_PASSWORD_HERE

# API Keys
CLAUDE_API_KEY=sk-ant-api03-xxxxx
OPENAI_API_KEY=sk-xxxxx

# Grafana
GRAFANA_ADMIN_PASSWORD=GENERATE_STRONG_PASSWORD_HERE

# URLs (remplacer maicivy.com par votre domaine)
NEXT_PUBLIC_API_URL=https://maicivy.com/api
NEXT_PUBLIC_WS_URL=wss://maicivy.com/ws
FRONTEND_URL=https://maicivy.com
```

**Génération passwords forts:**

```bash
# Méthode 1
openssl rand -base64 32

# Méthode 2
head /dev/urandom | tr -dc A-Za-z0-9 | head -c 32
```

---

## Étape 3: Build Images Docker

### Option A: Build Local

```bash
# Backend
cd backend
docker build -t maicivy-backend:latest .

# Frontend
cd ../frontend
docker build -t maicivy-frontend:latest .
```

### Option B: Pull depuis Registry (recommandé)

```bash
# Configurer registry dans .env
DOCKER_REGISTRY=ghcr.io/YOUR_USERNAME

# Pull images
docker pull ghcr.io/YOUR_USERNAME/maicivy-backend:latest
docker pull ghcr.io/YOUR_USERNAME/maicivy-frontend:latest
```

---

## Étape 4: Déploiement

### 4.1 Déployer Services

```bash
cd /opt/maicivy
./scripts/deploy.sh
```

Ce script va:
1. Pull latest images
2. Backup database (si existe)
3. Stop services
4. Start services
5. Health checks
6. Cleanup old images

**Durée:** ~2-3 minutes

### 4.2 Vérifier Logs

```bash
# Tous services
docker-compose -f docker/docker-compose.prod.yml logs -f

# Service spécifique
docker logs -f maicivy_backend
docker logs -f maicivy_frontend
```

### 4.3 Health Check

```bash
./scripts/health-check.sh
```

**Attendu:**
```
Checking Frontend... OK
Checking Backend Health... OK
Checking Backend Deep Health... OK
Checking Grafana... OK

All services are healthy!
```

---

## Étape 5: Vérification Production

### 5.1 Tester Frontend

```bash
curl -I https://maicivy.com
```

**Attendu:** HTTP 200 OK

### 5.2 Tester API

```bash
curl https://maicivy.com/api/health
```

**Attendu:**
```json
{
  "status": "ok",
  "timestamp": 1704729600,
  "service": "maicivy-backend"
}
```

### 5.3 Tester Grafana

Ouvrir navigateur: https://analytics.maicivy.com

**Attendu:** Dashboard public visible

### 5.4 Vérifier SSL

```bash
curl -I https://maicivy.com | grep "HTTP"
curl -I https://maicivy.com | grep "Strict-Transport-Security"
```

**Attendu:**
- HTTP/2 200
- Strict-Transport-Security header présent

---

## Étape 6: Monitoring

### 6.1 Monitoring Temps Réel

```bash
./scripts/monitor-services.sh
```

Affiche:
- Status tous services
- CPU/Memory usage
- Disk usage
- Recent errors

### 6.2 Prometheus

```bash
# Local seulement
curl http://localhost:9090/api/v1/targets | jq
```

### 6.3 Grafana

Browser: https://analytics.maicivy.com

Dashboard affiche:
- Visiteurs temps réel
- Request rate (RPS)
- Response time (P50, P95)
- Error rate
- CPU/Memory/Disk usage

---

## Étape 7: Backup & Maintenance

### 7.1 Test Backup Manuel

```bash
./scripts/backup-postgres.sh
./scripts/backup-redis.sh

# Vérifier backups créés
ls -lh /opt/maicivy/backups/
```

### 7.2 Test Restore (ATTENTION: environnement test seulement!)

```bash
# NE PAS FAIRE EN PRODUCTION
./scripts/restore-postgres.sh /opt/maicivy/backups/maicivy_backup_YYYYMMDD.sql.gz
```

### 7.3 Vérifier Cron Jobs

```bash
crontab -l
```

**Attendu:**
```
0 3 * * * /opt/maicivy/scripts/renew-ssl.sh
0 2 * * * /opt/maicivy/scripts/backup-postgres.sh
30 2 * * * /opt/maicivy/scripts/backup-redis.sh
```

### 7.4 Vérifier Systemd Service

```bash
systemctl status maicivy
```

**Attendu:** active (exited)

---

## Étape 8: Tests Sécurité

### 8.1 SSL Grade

Visiter: https://www.ssllabs.com/ssltest/analyze.html?d=maicivy.com

**Cible:** Grade A ou A+

### 8.2 Security Headers

Visiter: https://securityheaders.com/?q=maicivy.com

**Cible:** Grade A

### 8.3 Test Rate Limiting

```bash
# Envoyer 150 requêtes rapides
for i in {1..150}; do
    curl -s -o /dev/null -w "%{http_code}\n" https://maicivy.com/api/health
done
```

**Attendu:** Les dernières requêtes retournent 429 (Too Many Requests)

---

## Dépannage Rapide

### Service ne démarre pas

```bash
# Voir logs
docker logs maicivy_backend

# Causes communes:
# - DB password incorrect (.env)
# - Redis password incorrect (.env)
# - API keys manquantes

# Redémarrer
docker-compose -f docker/docker-compose.prod.yml restart backend
```

### SSL Errors

```bash
# Renouveler certificats
certbot renew --force-renewal
docker exec maicivy_nginx nginx -s reload
```

### Nginx Errors

```bash
# Test config
nginx -t

# Voir logs
tail -f /var/log/nginx/error.log
```

### Database Connection Issues

```bash
# Test connection
docker exec maicivy_postgres pg_isready -U maicivy

# Vérifier password
cat .env | grep POSTGRES_PASSWORD

# Redémarrer
docker-compose restart postgres
```

---

## Commandes Utiles

### Logs

```bash
# Backend logs
docker logs -f maicivy_backend

# Frontend logs
docker logs -f maicivy_frontend

# Nginx logs
tail -f /var/log/nginx/access.log
```

### Restart Services

```bash
# Restart service spécifique
docker-compose restart backend

# Restart tous services
docker-compose -f docker/docker-compose.prod.yml restart
```

### Update Application

```bash
# Pull latest code
cd /opt/maicivy
git pull origin main

# Redéployer
./scripts/deploy.sh
```

### Cleanup

```bash
# Remove old images
docker image prune -f

# Remove old volumes
docker volume prune -f
```

---

## Monitoring Externe (Optionnel)

### UptimeRobot

1. Créer compte: https://uptimerobot.com
2. Ajouter monitors:
   - URL: https://maicivy.com
   - Interval: 5 minutes
   - Alert: Email/Slack
3. Ajouter monitor API:
   - URL: https://maicivy.com/health
   - Keyword: "ok"

### Cloudflare (Optionnel)

1. Créer compte Cloudflare
2. Ajouter domaine maicivy.com
3. Changer nameservers chez registrar
4. Activer proxy (orange cloud)
5. Activer:
   - Auto minify (JS, CSS, HTML)
   - Brotli compression
   - Always Use HTTPS
   - Auto HTTPS Rewrites

**Avantages:**
- DDoS protection gratuite
- Global CDN
- Analytics
- Cache automatique

---

## Checklist Post-Déploiement

- [ ] Tous services running (monitor-services.sh)
- [ ] Health checks passent
- [ ] SSL certificats valides (ssllabs.com)
- [ ] Security headers OK (securityheaders.com)
- [ ] Grafana dashboard accessible
- [ ] Backups automatiques testés
- [ ] Cron jobs actifs
- [ ] Systemd service actif
- [ ] Monitoring externe configuré (UptimeRobot)
- [ ] Team informée des URLs et credentials
- [ ] Documentation accessible

---

## URLs à Noter

- **Frontend:** https://maicivy.com
- **API:** https://maicivy.com/api
- **Health:** https://maicivy.com/health
- **Grafana Admin:** https://analytics.maicivy.com (login avec GRAFANA_ADMIN_USER/PASSWORD)
- **Grafana Public:** https://analytics.maicivy.com (accès anonyme en lecture)

---

## Support

### Documentation Complète

- `INFRASTRUCTURE_PRODUCTION_GUIDE.md` - Guide détaillé
- `INFRASTRUCTURE_VALIDATION.md` - Tests et validation
- `docker/README.md` - Documentation Docker

### Contact

- Developer: [YOUR_EMAIL]
- VPS Support: support.ovh.com

---

## Prochaines Étapes

1. **Performance:**
   - Tuner PostgreSQL (shared_buffers, work_mem)
   - Configurer Redis maxmemory
   - Activer Nginx caching pour APIs statiques

2. **Monitoring:**
   - Setup Alertmanager (Slack/Email)
   - Activer Loki pour logs centralisés
   - Créer dashboards custom Grafana

3. **Sécurité:**
   - Audit régulier (OWASP)
   - Penetration testing
   - Security scanning automatique

4. **Scaling:**
   - Load balancing si > 1000 users
   - Database read replicas
   - CDN pour assets statiques

---

**Durée totale déploiement:** ~30-45 minutes

**Prêt pour production!** 🚀

---

**Date:** 2025-12-08
**Version:** 1.0
