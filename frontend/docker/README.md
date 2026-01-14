# Docker Configuration - maicivy

## Structure

```
docker/
├── docker-compose.prod.yml    # Production stack complet
├── nginx/
│   └── nginx.conf             # Configuration Nginx reverse proxy
└── README.md                  # Ce fichier
```

## Services

### Production Stack (docker-compose.prod.yml)

- **postgres**: PostgreSQL 15 database
- **redis**: Redis 7 cache & session store
- **backend**: Go/Fiber API
- **frontend**: Next.js 14 app
- **nginx**: Reverse proxy, SSL termination
- **prometheus**: Metrics collection
- **grafana**: Public analytics dashboard
- **node-exporter**: System metrics

## Usage

### Development

Use `docker-compose.yml` in project root (if exists).

### Production

```bash
# Pull images
docker-compose -f docker/docker-compose.prod.yml pull

# Start services
docker-compose -f docker/docker-compose.prod.yml up -d

# View logs
docker-compose -f docker/docker-compose.prod.yml logs -f

# Stop services
docker-compose -f docker/docker-compose.prod.yml down

# Restart specific service
docker-compose -f docker/docker-compose.prod.yml restart backend
```

## Configuration

### Environment Variables

Copy `.env.production.example` to `.env` and configure:

- `POSTGRES_PASSWORD`: Strong password for PostgreSQL
- `REDIS_PASSWORD`: Strong password for Redis
- `CLAUDE_API_KEY`: Anthropic API key
- `OPENAI_API_KEY`: OpenAI API key
- `GRAFANA_ADMIN_PASSWORD`: Grafana admin password
- `DOCKER_REGISTRY`: Your Docker registry (optional)

### Nginx

Edit `nginx/nginx.conf` to:

- Change domain names (replace `maicivy.com`)
- Adjust rate limiting
- Modify security headers
- Configure caching

### SSL/TLS

Certificates expected at:
```
/etc/letsencrypt/live/maicivy.com/fullchain.pem
/etc/letsencrypt/live/maicivy.com/privkey.pem
```

Obtain with:
```bash
certbot certonly --standalone -d maicivy.com -d www.maicivy.com -d analytics.maicivy.com
```

## Health Checks

All services have health checks:

- **postgres**: `pg_isready`
- **redis**: `redis-cli ping`
- **backend**: `curl http://localhost:8080/health`
- **frontend**: `curl http://localhost:3000`

## Volumes

Persistent data:

- `postgres_data`: Database files
- `redis_data`: Redis persistence
- `prometheus_data`: Metrics history (15 days)
- `grafana_data`: Dashboards and config
- `nginx_logs`: Access and error logs

## Networks

All services on `maicivy_network` bridge network.

## Ports

**Public (exposed):**
- 80: HTTP (redirects to HTTPS)
- 443: HTTPS

**Local only (127.0.0.1):**
- 5432: PostgreSQL
- 6379: Redis
- 9090: Prometheus
- 9100: Node Exporter

**Internal only:**
- 3000: Frontend (accessed via Nginx)
- 3001: Grafana (accessed via Nginx)
- 8080: Backend (accessed via Nginx)

## Troubleshooting

### Service won't start

```bash
# Check logs
docker logs maicivy_backend

# Check health
docker inspect maicivy_backend | jq '.[0].State.Health'

# Restart
docker-compose restart backend
```

### Database connection issues

```bash
# Test PostgreSQL
docker exec maicivy_postgres pg_isready -U maicivy

# Check password in .env
cat .env | grep POSTGRES_PASSWORD
```

### SSL errors

```bash
# Check certificates
ls -la /etc/letsencrypt/live/maicivy.com/

# Test Nginx config
docker exec maicivy_nginx nginx -t

# Reload Nginx
docker exec maicivy_nginx nginx -s reload
```

## Monitoring

- **Prometheus**: http://localhost:9090
- **Grafana**: https://analytics.maicivy.com
- **Backend Metrics**: https://maicivy.com/metrics

## Maintenance

See `/scripts/` for maintenance scripts:

- `backup-postgres.sh`: Database backup
- `backup-redis.sh`: Redis backup
- `restore-postgres.sh`: Database restore
- `deploy.sh`: Full deployment
- `health-check.sh`: Health verification

## Documentation

Full guide: `/INFRASTRUCTURE_PRODUCTION_GUIDE.md`
