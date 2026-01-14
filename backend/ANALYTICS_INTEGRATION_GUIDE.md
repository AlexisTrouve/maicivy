# Guide d'Intégration Analytics Backend

Ce guide explique comment intégrer le système d'analytics dans votre application maicivy existante.

## Prérequis

- Phase 1 (Foundation) complétée ✅
- Phase 2 (CV) complétée ✅
- Phase 3 (Letters) complétée ✅
- PostgreSQL + Redis opérationnels
- Modèle `AnalyticsEvent` existant dans la base

## Étape 1: Installer les dépendances Go

```bash
cd backend

# WebSocket support
go get github.com/gofiber/contrib/websocket

# Prometheus metrics
go get github.com/prometheus/client_golang/prometheus
go get github.com/prometheus/client_golang/prometheus/promauto
go get github.com/prometheus/client_golang/prometheus/promhttp
```

## Étape 2: Vérifier les fichiers créés

Vérifiez que les fichiers suivants ont été créés :

```
backend/
├── internal/
│   ├── services/
│   │   ├── analytics.go           ✅
│   │   └── analytics_test.go      ✅
│   ├── api/
│   │   ├── analytics.go           ✅
│   │   └── analytics_test.go      ✅
│   ├── websocket/
│   │   ├── analytics.go           ✅
│   │   └── README.md              ✅
│   ├── middleware/
│   │   └── analytics.go           ✅
│   ├── metrics/
│   │   └── analytics.go           ✅
│   └── jobs/
│       └── analytics_cleanup.go   ✅
├── cmd/
│   └── main_with_analytics.go.example  ✅
├── ANALYTICS_IMPLEMENTATION_SUMMARY.md  ✅
└── ANALYTICS_INTEGRATION_GUIDE.md       ✅ (ce fichier)
```

## Étape 3: Intégrer dans main.go

### Option A: Remplacement complet (recommandé)

```bash
# Backup de l'ancien main.go
cp cmd/main.go cmd/main.go.backup

# Copier le nouveau main.go avec analytics
cp cmd/main_with_analytics.go.example cmd/main.go
```

### Option B: Intégration manuelle

Modifier `cmd/main.go` pour ajouter les imports nécessaires :

```go
import (
    // ... imports existants ...
    "context"
    "github.com/gofiber/fiber/v2/middleware/adaptor"
    "github.com/prometheus/client_golang/prometheus/promhttp"

    "maicivy/internal/jobs"
    "maicivy/internal/websocket"
)
```

Ajouter après l'initialisation des services existants (ligne ~83) :

```go
// 7. Initialiser services
cvService := services.NewCVService(db, redisClient)
analyticsService := services.NewAnalyticsService(db, redisClient)  // NOUVEAU
```

Ajouter le middleware analytics après le tracking (ligne ~76) :

```go
// 6. Tracking visiteurs
trackingMW := middleware.NewTracking(db, redisClient)
app.Use(trackingMW.Handler())

// NOUVEAU: Analytics middleware
analyticsMW := middleware.NewAnalytics(analyticsService)
app.Use(analyticsMW.Handler())

// 7. Rate limiting global
rateLimitMW := middleware.NewRateLimit(redisClient)
app.Use(rateLimitMW.Global())
```

Ajouter les handlers (ligne ~87) :

```go
// 8. Initialiser handlers
healthHandler := api.NewHealthHandler(db, redisClient)
cvHandler := api.NewCVHandler(cvService)
analyticsHandler := api.NewAnalyticsHandler(analyticsService)  // NOUVEAU

// NOUVEAU: WebSocket handler analytics
analyticsWSHandler := websocket.NewAnalyticsWSHandler(analyticsService, redisClient)
analyticsWSHandler.RegisterRoutes(app)
```

Ajouter les routes analytics (ligne ~103) :

```go
// Routes CV (Phase 2)
cvHandler.RegisterRoutes(app)

// NOUVEAU: Routes Analytics (Phase 4)
analyticsHandler.RegisterRoutes(app)
```

Ajouter l'endpoint Prometheus (après les routes health, ligne ~91) :

```go
app.Get("/health", healthHandler.Health)
app.Get("/health/deep", healthHandler.HealthDeep)

// NOUVEAU: Prometheus metrics endpoint
app.Get("/metrics", adaptor.HTTPHandler(promhttp.Handler()))
```

Ajouter le cleanup job (avant le graceful shutdown, ligne ~111) :

```go
// NOUVEAU: Démarrer cleanup job
ctx, cancel := context.WithCancel(context.Background())
defer cancel()

cleanupJob := jobs.NewAnalyticsCleanupJob(analyticsService, 90)
go cleanupJob.Start(ctx)

// 10. Graceful shutdown
go func() {
    addr := fmt.Sprintf("%s:%s", cfg.ServerHost, cfg.ServerPort)
    // ...
```

Modifier le graceful shutdown (ligne ~125) :

```go
// 11. Attendre signal de shutdown
quit := make(chan os.Signal, 1)
signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
<-quit

log.Info().Msg("Shutting down server...")

// NOUVEAU: Arrêter cleanup job
cancel()

// NOUVEAU: Fermer WebSocket handler
if err := analyticsWSHandler.Close(); err != nil {
    log.Error().Err(err).Msg("Failed to close WebSocket handler")
}

// Shutdown serveur
if err := app.ShutdownWithTimeout(30 * time.Second); err != nil {
    log.Error().Err(err).Msg("Server forced to shutdown")
}
```

## Étape 4: Vérifier la compilation

```bash
cd backend
go mod tidy
go build -o maicivy ./cmd/main.go
```

Si erreurs de compilation :
- Vérifier tous les imports
- `go mod tidy` pour nettoyer les dépendances
- Vérifier que tous les fichiers sont bien créés

## Étape 5: Tester l'application

### Démarrer le serveur

```bash
cd backend
./maicivy
# Ou: go run cmd/main.go
```

Vérifier les logs :
```
[INFO] Starting analytics cleanup scheduler
[INFO] Started listening to Redis Pub/Sub for analytics events
[INFO] Started WebSocket broadcaster
[INFO] Started WebSocket heartbeat
[INFO] Starting server addr=:8080 environment=development
```

### Tester les endpoints

```bash
# Health check
curl http://localhost:8080/health

# Stats temps réel
curl http://localhost:8080/api/v1/analytics/realtime

# Stats par période
curl http://localhost:8080/api/v1/analytics/stats?period=day

# Top thèmes
curl http://localhost:8080/api/v1/analytics/themes?limit=5

# Prometheus metrics
curl http://localhost:8080/metrics | grep maicivy
```

### Tester WebSocket

```bash
# Installer wscat si nécessaire
npm install -g wscat

# Se connecter
wscat -c ws://localhost:8080/ws/analytics

# Vous devriez recevoir:
< {"type":"initial_stats","data":{...}}
< {"type":"heartbeat","data":{...}}
```

### Tester le tracking automatique

```bash
# Visiter une page (générer un pageview)
curl http://localhost:8080/cv?theme=backend

# Vérifier que l'événement est tracké
curl http://localhost:8080/api/v1/analytics/timeline?limit=1
```

## Étape 6: Lancer les tests

```bash
cd backend

# Tests service analytics
go test ./internal/services -v -run TestAnalyticsService

# Tests API analytics
go test ./internal/api -v -run TestAnalyticsAPI

# Coverage global
go test ./... -coverprofile=coverage.out
go tool cover -html=coverage.out
```

Objectifs de coverage :
- `analytics.go` : > 80%
- `analytics_test.go` : 100%

## Étape 7: Vérifier Redis

```bash
# Se connecter à Redis
redis-cli

# Vérifier les clés créées
127.0.0.1:6379> KEYS analytics:*

# Exemples de clés attendues:
# - analytics:stats:day:2025-12-08:total_events
# - analytics:visitors:unique:day:2025-12-08
# - analytics:themes:top
# - analytics:realtime:visitors

# Vérifier HyperLogLog
127.0.0.1:6379> PFCOUNT analytics:visitors:unique:day:2025-12-08

# Vérifier Sorted Set (top thèmes)
127.0.0.1:6379> ZREVRANGE analytics:themes:top 0 4 WITHSCORES

# Vérifier Set (visiteurs actifs)
127.0.0.1:6379> SMEMBERS analytics:realtime:visitors

# Vérifier Pub/Sub
127.0.0.1:6379> PUBSUB CHANNELS
```

## Étape 8: Vérifier PostgreSQL

```bash
# Se connecter à PostgreSQL
psql -U maicivy_user -d maicivy_db

# Vérifier la table analytics_events
maicivy_db=# SELECT COUNT(*) FROM analytics_events;

# Vérifier les événements récents
maicivy_db=# SELECT event_type, COUNT(*)
             FROM analytics_events
             WHERE created_at > NOW() - INTERVAL '1 hour'
             GROUP BY event_type;

# Vérifier les indexes
maicivy_db=# \d analytics_events
```

## Étape 9: Monitoring Prometheus (optionnel)

Si vous avez Prometheus installé :

```yaml
# prometheus.yml
scrape_configs:
  - job_name: 'maicivy'
    static_configs:
      - targets: ['localhost:8080']
    metrics_path: '/metrics'
    scrape_interval: 15s
```

Restart Prometheus et vérifier :
```bash
# Ouvrir Prometheus UI
open http://localhost:9090

# Query example
maicivy_current_visitors
```

## Troubleshooting

### Erreur: "visitor_id not found in context"

**Cause:** Le middleware tracking n'est pas exécuté avant analytics

**Solution:** Vérifier l'ordre des middlewares dans main.go
```go
app.Use(trackingMW.Handler())      // DOIT être avant
app.Use(analyticsMW.Handler())     // analytics
```

### Erreur: "Failed to connect to Redis"

**Cause:** Redis n'est pas démarré ou mauvaise config

**Solution:**
```bash
# Vérifier Redis
redis-cli ping
# Devrait retourner: PONG

# Ou démarrer Redis
docker-compose up -d redis
```

### WebSocket: "Connection refused"

**Cause:** Serveur pas démarré ou port incorrect

**Solution:**
```bash
# Vérifier que le serveur écoute
lsof -i :8080

# Vérifier logs serveur
grep "Starting server" logs/app.log
```

### Cleanup job ne s'exécute pas

**Cause:** Context annulé trop tôt

**Solution:** Vérifier que `cancel()` n'est appelé que dans le shutdown

### Tests échouent: "database locked"

**Cause:** SQLite in-memory utilisé par tests, conflit de transactions

**Solution:** Utiliser `t.Parallel()` avec précaution ou setup DB séparées

## Validation Finale

Checklist complète :

- [ ] Serveur démarre sans erreur
- [ ] Logs montrent "Started listening to Redis Pub/Sub"
- [ ] Logs montrent "Started WebSocket broadcaster"
- [ ] GET `/health` retourne 200
- [ ] GET `/metrics` expose métriques Prometheus
- [ ] GET `/api/v1/analytics/realtime` retourne JSON
- [ ] WebSocket `/ws/analytics` se connecte
- [ ] WebSocket reçoit messages heartbeat
- [ ] Redis contient clés `analytics:*`
- [ ] PostgreSQL contient événements dans `analytics_events`
- [ ] Tests passent : `go test ./... -v`
- [ ] Coverage > 80% : `go test ./... -cover`

Si toutes les cases sont cochées : **Félicitations, l'intégration est réussie !** ✅

## Prochaines Étapes

1. **Intégrer le frontend** (Phase 4 - Doc 12)
   - Créer dashboard analytics React
   - Connecter WebSocket pour temps réel
   - Afficher graphiques Chart.js

2. **Setup Grafana** (Phase 6)
   - Importer dashboards Prometheus
   - Configurer alerting

3. **Load testing**
   - Tester avec k6
   - Optimiser si nécessaire

4. **Deploy en production**
   - Vérifier variables d'environnement
   - Activer HTTPS
   - Backup Redis

## Support

En cas de problème, consulter :
- `ANALYTICS_IMPLEMENTATION_SUMMARY.md` - Documentation complète
- `docs/implementation/11_BACKEND_ANALYTICS.md` - Spécifications
- Logs serveur : `tail -f logs/app.log`
- Redis logs : `redis-cli MONITOR`

---

**Version:** 1.0
**Date:** 2025-12-08
**Auteur:** Agent IA (Claude)
