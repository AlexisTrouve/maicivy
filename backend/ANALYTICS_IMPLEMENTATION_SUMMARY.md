# Analytics Backend - Implementation Summary

**Date:** 2025-12-08
**Phase:** Phase 4 - Analytics
**Document:** 11_BACKEND_ANALYTICS.md

## Vue d'Ensemble

Ce document résume l'implémentation complète du système d'analytics backend pour le projet maicivy. Le système permet de :
- Collecter et agréger les événements utilisateurs en temps réel
- Fournir des statistiques via API REST et WebSocket
- Exposer des métriques Prometheus pour monitoring
- Gérer la rétention des données avec cleanup automatique

---

## Architecture

### Diagramme de Flow de Données

```
┌─────────────┐
│   Visiteur  │
└──────┬──────┘
       │ HTTP Request
       ▼
┌─────────────────────────────────────┐
│   Middleware Analytics               │
│   - Auto-track pageviews            │
│   - Mark visitor active             │
└──────────┬──────────────────────────┘
           │
           ▼
    ┌──────────────────┐
    │ Analytics Service │
    └──────┬───────────┘
           │
     ┌─────┴─────┐
     ▼           ▼
┌─────────┐  ┌──────────┐
│  Redis  │  │PostgreSQL│
│         │  │          │
│ • HLL   │  │ • Events │
│ • ZSets │  │ • Raw    │
│ • Sets  │  │ • 90d    │
│ • TTL   │  │          │
└────┬────┘  └─────┬────┘
     │             │
     │             ▼
     │      ┌────────────────┐
     │      │ Cleanup Job    │
     │      │ (2AM daily)    │
     │      └────────────────┘
     │
     ▼
┌────────────────┐
│  Redis Pub/Sub │
└────┬───────────┘
     │
     ▼
┌────────────────┐       ┌──────────────┐
│  WebSocket     │       │  Prometheus  │
│  /ws/analytics │◄──────┤  /metrics    │
└────┬───────────┘       └──────────────┘
     │
     ▼
┌─────────────┐
│  Dashboard  │
│   Frontend  │
└─────────────┘
```

### Composants Principaux

1. **AnalyticsService** (`internal/services/analytics.go`)
   - Service central de gestion analytics
   - Collecte événements → PostgreSQL
   - Agrégations → Redis (HyperLogLog, Sorted Sets)
   - Publication temps réel → Redis Pub/Sub

2. **AnalyticsHandler** (`internal/api/analytics.go`)
   - API REST pour récupérer statistiques
   - 6 endpoints publics (pas d'auth requise)

3. **AnalyticsWSHandler** (`internal/websocket/analytics.go`)
   - WebSocket pour broadcast temps réel
   - Heartbeat toutes les 5 secondes
   - Support multi-clients via goroutines

4. **Analytics Middleware** (`internal/middleware/analytics.go`)
   - Auto-tracking pageviews (non API/WS)
   - Marque visiteurs actifs
   - Métriques Prometheus

5. **Prometheus Metrics** (`internal/metrics/analytics.go`)
   - Compteurs, Gauges, Histogrammes
   - Exposition sur `/metrics`

6. **Cleanup Job** (`internal/jobs/analytics_cleanup.go`)
   - Exécution quotidienne à 2h du matin
   - Suppression événements > 90 jours

---

## API REST Endpoints

### Base URL
```
/api/v1/analytics
```

### 1. GET `/realtime`
**Description:** Statistiques temps réel
**Response:**
```json
{
  "success": true,
  "data": {
    "current_visitors": 12,
    "unique_today": 145,
    "total_events": 2340,
    "letters_today": 23,
    "timestamp": 1733683200
  }
}
```

### 2. GET `/stats?period=day|week|month`
**Description:** Statistiques agrégées par période
**Query Params:**
- `period` (required): `day`, `week`, ou `month`

**Response:**
```json
{
  "success": true,
  "data": {
    "period": "day",
    "period_key": "2025-12-08",
    "total_events": 2340,
    "letters_generated": 23,
    "unique_visitors": 145,
    "conversion_rate": 0.1586
  }
}
```

### 3. GET `/themes?limit=5`
**Description:** Top thèmes CV consultés
**Query Params:**
- `limit` (optional, default: 5, max: 20)

**Response:**
```json
{
  "success": true,
  "data": [
    { "theme": "backend", "views": 523 },
    { "theme": "fullstack", "views": 312 },
    { "theme": "devops", "views": 198 }
  ]
}
```

### 4. GET `/letters?period=day|week|month`
**Description:** Statistiques lettres générées
**Response:**
```json
{
  "success": true,
  "data": {
    "period": "day",
    "total": 23,
    "motivation": 15,
    "anti_motivation": 8,
    "unique_visitors": 145,
    "conversion_rate": 0.1586
  }
}
```

### 5. GET `/timeline?limit=50&offset=0`
**Description:** Timeline événements récents
**Query Params:**
- `limit` (optional, default: 50, max: 100)
- `offset` (optional, default: 0)

**Response:**
```json
{
  "success": true,
  "data": [
    {
      "id": "uuid",
      "visitor_id": "uuid",
      "event_type": "page_view",
      "event_data": "{\"path\": \"/cv\"}",
      "page_url": "/cv?theme=backend",
      "created_at": "2025-12-08T12:34:56Z"
    }
  ],
  "meta": {
    "limit": 50,
    "offset": 0,
    "count": 42
  }
}
```

### 6. GET `/heatmap?page_url=/cv&hours=24`
**Description:** Données heatmap interactions
**Query Params:**
- `page_url` (optional): Filtrer par page
- `hours` (optional, default: 24, max: 168)

**Response:**
```json
{
  "success": true,
  "data": [
    { "x": 450, "y": 200, "count": 23 },
    { "x": 320, "y": 450, "count": 18 }
  ],
  "meta": {
    "page_url": "/cv",
    "hours": 24,
    "count": 2
  }
}
```

### 7. POST `/event`
**Description:** Track événement custom
**Body:**
```json
{
  "event_type": "button_click",
  "event_data": {
    "button": "cta",
    "x": 450,
    "y": 200
  },
  "page_url": "/cv"
}
```

**Response:**
```json
{
  "success": true,
  "message": "Event tracked successfully"
}
```

---

## WebSocket Protocol

### Endpoint
```
ws://localhost:8080/ws/analytics
```

### Messages envoyés par le serveur

#### 1. Initial Stats (à la connexion)
```json
{
  "type": "initial_stats",
  "data": {
    "current_visitors": 12,
    "unique_today": 145,
    "total_events": 2340,
    "letters_today": 23,
    "timestamp": 1733683200
  }
}
```

#### 2. Heartbeat (toutes les 5 secondes)
```json
{
  "type": "heartbeat",
  "data": {
    "current_visitors": 13,
    "unique_today": 146,
    "total_events": 2345,
    "letters_today": 23,
    "timestamp": 1733683205
  }
}
```

#### 3. Realtime Event (via Pub/Sub)
```json
{
  "type": "page_view",
  "visitor_id": "uuid",
  "timestamp": "2025-12-08T12:34:56Z",
  "page_url": "/cv?theme=backend",
  "data": {
    "theme": "backend",
    "path": "/cv"
  }
}
```

### Messages envoyés par le client

#### 1. Refresh Stats
```json
{
  "type": "refresh_stats"
}
```

#### 2. Ping
```json
{
  "type": "ping"
}
```
**Response:**
```json
{
  "type": "pong",
  "time": 1733683200
}
```

---

## Métriques Prometheus

### Endpoint
```
GET /metrics
```

### Métriques Custom

#### Counters
```prometheus
# Visiteurs total
maicivy_visitors_total 1543

# Lettres générées par type
maicivy_letters_generated_total{type="motivation"} 234
maicivy_letters_generated_total{type="anti_motivation"} 89

# Événements par type
maicivy_events_total{event_type="page_view"} 8234
maicivy_events_total{event_type="button_click"} 523

# Page views par path
maicivy_page_views_total{path="/cv"} 4512
maicivy_page_views_total{path="/letters"} 1234
```

#### Gauges
```prometheus
# Visiteurs actuels (5 dernières minutes)
maicivy_current_visitors 12

# Connexions WebSocket actives
maicivy_websocket_connections 3

# Taux de conversion
maicivy_conversion_rate 0.296

# Vues par thème CV
maicivy_cv_theme_views{theme="backend"} 523
maicivy_cv_theme_views{theme="fullstack"} 312
```

#### Histograms
```prometheus
# Durée requêtes analytics
maicivy_analytics_request_duration_seconds_bucket{endpoint="/api/v1/analytics/realtime",method="GET",status="200",le="0.05"} 1543
maicivy_analytics_request_duration_seconds_sum{endpoint="/api/v1/analytics/realtime",method="GET",status="200"} 23.45
maicivy_analytics_request_duration_seconds_count{endpoint="/api/v1/analytics/realtime",method="GET",status="200"} 1543

# Durée opérations Redis
maicivy_redis_operation_duration_seconds_bucket{operation="pfcount",le="0.01"} 234

# Durée requêtes PostgreSQL
maicivy_database_query_duration_seconds_bucket{query_type="select_events",le="0.1"} 456
```

---

## Structures Redis

### Keys Pattern

#### HyperLogLog (visiteurs uniques)
```
analytics:visitors:unique:day:2025-12-08       # Visiteurs uniques aujourd'hui
analytics:visitors:unique:week:2025-W49        # Visiteurs uniques cette semaine
analytics:visitors:unique:month:2025-12        # Visiteurs uniques ce mois
```

#### Compteurs (stats agrégées)
```
analytics:stats:day:2025-12-08:total_events         # Total événements aujourd'hui
analytics:stats:day:2025-12-08:letters_generated    # Lettres générées aujourd'hui
analytics:stats:week:2025-W49:total_events          # Total semaine
analytics:stats:month:2025-12:total_events          # Total mois
```

#### Sorted Sets (classements)
```
analytics:themes:top   # ZSet: {member: "backend", score: 523}
```

#### Sets (temps réel)
```
analytics:realtime:visitors   # Set des visiteurs actifs (TTL 5min)
```

#### Pub/Sub
```
analytics:realtime   # Channel pour broadcast événements
```

### TTL par clé
- `day:*` → 90 jours
- `week:*` → 365 jours
- `month:*` → 365 jours
- `realtime:visitors` → 5 minutes (auto-refresh)

---

## PostgreSQL

### Table: analytics_events

```sql
CREATE TABLE analytics_events (
    id UUID PRIMARY KEY,
    visitor_id UUID NOT NULL REFERENCES visitors(id),
    event_type VARCHAR(50) NOT NULL,
    event_data JSONB,
    page_url VARCHAR(500),
    referrer VARCHAR(500),
    session_duration INTEGER DEFAULT 0,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMP
);

-- Indexes
CREATE INDEX idx_analytics_events_visitor_id ON analytics_events(visitor_id);
CREATE INDEX idx_analytics_events_event_type ON analytics_events(event_type);
CREATE INDEX idx_analytics_events_created_at ON analytics_events(created_at DESC);
CREATE INDEX idx_analytics_events_type_date ON analytics_events(event_type, created_at DESC);
```

### Rétention
- Événements conservés **90 jours**
- Cleanup automatique via job quotidien à 2h du matin
- Soft delete (deleted_at) non utilisé pour analytics

---

## Exemples de Requêtes

### cURL - Récupérer stats temps réel
```bash
curl http://localhost:8080/api/v1/analytics/realtime
```

### cURL - Top thèmes
```bash
curl "http://localhost:8080/api/v1/analytics/themes?limit=10"
```

### cURL - Stats par période
```bash
curl "http://localhost:8080/api/v1/analytics/stats?period=week"
```

### cURL - Track événement custom
```bash
curl -X POST http://localhost:8080/api/v1/analytics/event \
  -H "Content-Type: application/json" \
  -d '{
    "event_type": "button_click",
    "event_data": {"button": "download_cv", "x": 450, "y": 200},
    "page_url": "/cv"
  }'
```

### WebSocket - Connexion (JavaScript)
```javascript
const ws = new WebSocket('ws://localhost:8080/ws/analytics');

ws.onopen = () => {
  console.log('Connected to analytics WebSocket');
};

ws.onmessage = (event) => {
  const data = JSON.parse(event.data);
  console.log('Analytics update:', data);

  if (data.type === 'heartbeat') {
    updateDashboard(data.data);
  }
};

// Demander refresh manuel
ws.send(JSON.stringify({ type: 'refresh_stats' }));
```

### Prometheus - Query visiteurs actuels
```promql
# Visiteurs actuels
maicivy_current_visitors

# Taux de conversion moyen (1h)
avg_over_time(maicivy_conversion_rate[1h])

# Requêtes analytics P95 latency
histogram_quantile(0.95,
  rate(maicivy_analytics_request_duration_seconds_bucket[5m])
)

# Événements par seconde
rate(maicivy_events_total[1m])
```

---

## Grafana Dashboard Queries (Bonus)

### Panel: Visiteurs Actuels
```promql
maicivy_current_visitors
```

### Panel: Taux de Conversion
```promql
maicivy_conversion_rate * 100
```

### Panel: Événements par Type (Rate 5min)
```promql
sum by (event_type) (rate(maicivy_events_total[5m]))
```

### Panel: Latence API P50/P95/P99
```promql
histogram_quantile(0.50, sum(rate(maicivy_analytics_request_duration_seconds_bucket[5m])) by (le))
histogram_quantile(0.95, sum(rate(maicivy_analytics_request_duration_seconds_bucket[5m])) by (le))
histogram_quantile(0.99, sum(rate(maicivy_analytics_request_duration_seconds_bucket[5m])) by (le))
```

### Panel: Top 5 Thèmes CV
```promql
topk(5, maicivy_cv_theme_views)
```

### Panel: Lettres Générées (Stacked)
```promql
sum by (type) (rate(maicivy_letters_generated_total[1h]))
```

---

## Commandes de Validation

### Tests unitaires
```bash
cd backend
go test ./internal/services -v -cover -run TestAnalyticsService
go test ./internal/api -v -cover -run TestAnalyticsAPI
```

### Coverage global
```bash
go test ./... -coverprofile=coverage.out
go tool cover -html=coverage.out
```

### Tester WebSocket (wscat)
```bash
npm install -g wscat
wscat -c ws://localhost:8080/ws/analytics
```

### Tester Prometheus metrics
```bash
curl http://localhost:8080/metrics | grep maicivy
```

### Load testing (k6)
```javascript
// analytics_load_test.js
import http from 'k6/http';
import { check } from 'k6';

export let options = {
  vus: 50,
  duration: '30s',
};

export default function() {
  let res = http.get('http://localhost:8080/api/v1/analytics/realtime');
  check(res, {
    'status is 200': (r) => r.status === 200,
    'response time < 100ms': (r) => r.timings.duration < 100,
  });
}
```

```bash
k6 run analytics_load_test.js
```

---

## Performance Objectives

### Latence
- `GET /analytics/*` : **< 100ms** (grâce au cache Redis)
- WebSocket broadcast : **< 50ms**
- Cleanup job cycle : **< 5s** (pour 1M événements)

### Throughput
- **1000 événements/min** supportés
- **50 WebSocket clients** simultanés
- Prometheus scrape : **< 200ms**

### Ressources
- **Redis memory** : ~10MB pour 1M visiteurs uniques (HyperLogLog)
- **PostgreSQL disk** : ~500MB pour 10M événements (90 jours)
- **CPU** : < 5% idle, < 30% sous charge

---

## Sécurité & Privacy

### RGPD Compliance
- **Pas d'IP en clair** : hash SHA-256 via middleware tracking
- **Pas de PII** : analytics_events ne stocke jamais noms/emails
- **Anonymisation** : visitor_id = UUID opaque
- **Rétention** : 90 jours max, cleanup automatique

### Rate Limiting
- Endpoint POST `/event` : **100 req/min** par IP
- WebSocket connexions : **50 max simultanées**

### Validation
- Query params validés (period, limit, hours)
- Body JSON validé (event_type required)
- Visitor session vérifié (middleware)

---

## Troubleshooting

### WebSocket ne se connecte pas
1. Vérifier que le serveur est démarré
2. Vérifier logs : `WebSocket client connected`
3. Tester avec wscat : `wscat -c ws://localhost:8080/ws/analytics`
4. Vérifier firewall/proxy (WebSocket = upgrade HTTP)

### Redis mémoire pleine
1. Vérifier TTL sur keys : `redis-cli TTL analytics:stats:day:2025-12-08:total_events`
2. Forcer cleanup : `redis-cli FLUSHDB` (DANGER: perte data)
3. Augmenter maxmemory-policy : `allkeys-lru`

### PostgreSQL slow queries
1. Vérifier indexes : `EXPLAIN ANALYZE SELECT * FROM analytics_events WHERE created_at > NOW() - INTERVAL '7 days'`
2. Vacuum : `VACUUM ANALYZE analytics_events`
3. Partitioning (si > 100M events)

### Cleanup job ne s'exécute pas
1. Vérifier logs : `Starting analytics cleanup scheduler`
2. Tester manuellement : appeler `cleanupJob.RunOnce(ctx)`
3. Vérifier timezone serveur (job à 2h locale)

---

## Prochaines Étapes (Post-Phase 4)

### Phase 5 - Features Avancées
- [ ] Segmentation visiteurs (profils détectés)
- [ ] Funnels de conversion (CV → Letters → Download)
- [ ] A/B testing framework
- [ ] Cohort analysis (rétention visiteurs)

### Phase 6 - Production
- [ ] Setup Prometheus + Grafana
- [ ] Alerting (Alertmanager)
- [ ] Backup Redis (RDB/AOF)
- [ ] Database partitioning (si volume élevé)
- [ ] CDN pour WebSocket (Cloudflare)

---

## Ressources

### Documentation
- [Redis HyperLogLog](https://redis.io/docs/data-types/probabilistic/hyperloglogs/)
- [Redis Pub/Sub](https://redis.io/docs/interact/pubsub/)
- [Fiber WebSocket](https://github.com/gofiber/contrib/tree/main/websocket)
- [Prometheus Client Go](https://github.com/prometheus/client_golang)

### Articles
- [Real-time Analytics at Scale (Netflix)](https://netflixtechblog.com/scaling-time-series-data-storage-part-i-ec2b6d44ba39)
- [HyperLogLog in Practice](https://research.neustar.biz/2012/10/25/sketch-of-the-day-hyperloglog-cornerstone-of-a-big-data-infrastructure/)

---

**Implémentation complétée le:** 2025-12-08
**Auteur:** Agent IA (Claude)
**Status:** ✅ Ready for integration
