# Middlewares Backend

Ce package contient tous les middlewares HTTP de l'application maicivy.

## Architecture

Les middlewares sont appliqués dans cet ordre précis:

```
Request → CORS → Recovery → RequestID → Logger → Compression → Tracking → RateLimiting → Handler
            ↓                                                      ↓            ↓
         Security                                            Redis        Redis
                                                            (visits)   (limits)
                                                               ↓
                                                         PostgreSQL
                                                          (visitors)
```

## Middlewares

### 1. CORS (`cors.go`)

Configure la politique CORS pour autoriser le frontend.

**Configuration:**
- Origins: Via variable d'environnement `ALLOWED_ORIGINS`
- Credentials: Activé (cookies supportés)
- Headers exposés: X-Request-ID, X-RateLimit-*

**Usage:**
```go
app.Use(middleware.CORS(cfg.AllowedOrigins))
```

### 2. Recovery (`recovery.go`)

Récupère les panics pour éviter le crash du serveur.

**Fonctionnalités:**
- Capture les panics
- Log stack trace complète
- Retourne erreur 500 propre au client

**Usage:**
```go
app.Use(middleware.Recovery())
```

### 3. Request ID (`requestid.go`)

Génère un ID unique (UUID) pour chaque requête.

**Fonctionnalités:**
- ID unique par requête
- Ajouté dans header `X-Request-ID`
- Disponible via `c.Locals("requestid")`
- Utilisé pour traçabilité dans logs

**Usage:**
```go
app.Use(middleware.RequestID())
```

### 4. Logger (`logger.go`)

Log structuré (JSON) de chaque requête HTTP.

**Informations loggées:**
- Request ID
- Méthode, path, IP
- Status code
- Durée de traitement
- User-Agent
- Taille de la réponse

**Niveaux de log:**
- Info: Status 2xx
- Warn: Status 4xx
- Error: Status 5xx ou erreur

**Usage:**
```go
app.Use(middleware.Logger())
```

### 5. Tracking (`tracking.go`)

Tracking des visiteurs avec détection de profil.

**Fonctionnalités:**
- Cookie session (30 jours, HTTPOnly, Secure)
- Compteur de visites (Redis)
- Détection de profil (User-Agent analysis)
- Stockage visiteur (PostgreSQL async)
- Hash IP (RGPD compliant)

**Profils détectés:**
- `linkedin_bot`: Bot LinkedIn
- `recruiter`: Patterns recruteur (LinkedIn Sales Navigator, etc.)
- `professional`: Desktop professionnel (pas mobile)

**Données exposées:**
- `c.Locals("session_id")`: UUID de session
- `c.Locals("visit_count")`: Nombre de visites
- `c.Locals("profile_detected")`: Profil détecté

**Usage:**
```go
trackingMW := middleware.NewTracking(db, redisClient)
app.Use(trackingMW.Handler())
```

### 6. Rate Limiting (`ratelimit.go`)

Rate limiting basé Redis avec règles spécifiques pour l'IA.

**Rate Limiting Global:**
- 100 requêtes par minute par IP
- Headers: X-RateLimit-Limit, X-RateLimit-Remaining
- Erreur 429 si limite dépassée

**Rate Limiting IA:**
- 5 générations par jour par session
- Cooldown de 2 minutes entre générations
- Headers: X-RateLimit-AI-Limit, X-RateLimit-AI-Remaining
- Appliqué uniquement sur routes `/api/v1/letters`

**Usage:**
```go
rateLimitMW := middleware.NewRateLimit(redisClient)

// Global (toutes les routes)
app.Use(rateLimitMW.Global())

// IA (routes spécifiques)
lettersGroup := app.Group("/api/v1/letters")
lettersGroup.Use(rateLimitMW.AI())
```

## Configuration

Variables d'environnement:

```bash
# CORS
ALLOWED_ORIGINS=http://localhost:3000,https://maicivy.com

# Cookie (en production, Secure=true nécessite HTTPS)
COOKIE_SECURE=true

# Rate limiting (optionnel - valeurs par défaut OK)
RATE_LIMIT_GLOBAL=100
RATE_LIMIT_AI_DAILY=5
```

## Tests

### Lancer les tests unitaires

```bash
# Tous les tests
go test -v ./internal/middleware/...

# Avec coverage
go test -v -cover ./internal/middleware/...

# Tests integration (nécessite Redis/PostgreSQL)
go test -v -tags=integration ./internal/middleware/...

# Benchmarks
go test -bench=. ./internal/middleware/...
```

### Tests disponibles

- `tracking_test.go`: Tests tracking visiteurs
  - Test nouveau visiteur (création cookie)
  - Test visiteur récurrent (incrémentation compteur)
  - Test détection profil LinkedIn

- `ratelimit_test.go`: Tests rate limiting
  - Test limite journalière IA (5 générations)
  - Test cooldown IA (2 minutes)
  - Test limite globale (100/min)

## Sécurité

### RGPD / Privacy

- **IP hashing**: Les IPs sont hashées (SHA-256) avant stockage
- **Pas de données sensibles**: Cookie contient uniquement UUID session
- **Anonymisation**: Pas de tracking cross-site

### Cookies

- **HTTPOnly**: Empêche accès JavaScript (XSS protection)
- **Secure**: HTTPS uniquement en production
- **SameSite=Lax**: Protection CSRF partielle

### Rate Limiting

- **DDoS protection**: Limite globale 100 req/min
- **Cost control**: Limite IA 5 générations/jour
- **Fail open**: Si Redis down, autoriser (ne pas bloquer tout le site)

## Performance

### Benchmarks

Les middlewares ajoutent ~2-5ms de latency:

- CORS: ~0.1ms
- Recovery: ~0.1ms (0 si pas de panic)
- Request ID: ~0.2ms (génération UUID)
- Logger: ~0.5ms (écriture log)
- Tracking: ~2ms (Redis + async PostgreSQL)
- Rate Limiting: ~1ms (Redis)

### Optimisations

- Redis pipelining pour grouper commandes
- PostgreSQL writes async (goroutine)
- Cache User-Agent parsing
- TTL automatique Redis (pas de cleanup manuel)

## Monitoring

### Métriques Prometheus (Phase 6)

Les middlewares exposent ces métriques:

```
# Compteurs
http_requests_total{method, path, status}
rate_limit_rejections_total{type="global|ai"}

# Histogrammes
http_request_duration_seconds{method, path}

# Gauges
visitor_sessions_active
```

### Logs structurés

Format JSON (compatible Loki/ELK):

```json
{
  "level": "info",
  "request_id": "550e8400-e29b-41d4-a716-446655440000",
  "method": "GET",
  "path": "/api/v1/cv",
  "ip": "192.168.1.1",
  "status": 200,
  "duration_ms": 45,
  "user_agent": "Mozilla/5.0...",
  "size": 1234,
  "message": "HTTP request"
}
```

## Troubleshooting

### Cookie non créé

- Vérifier `COOKIE_SECURE` = false en dev local (HTTP)
- Vérifier CORS origins autorisées

### Rate limit trop strict

- Augmenter `RATE_LIMIT_GLOBAL` en env var
- Pour IA: ajuster constantes dans `ratelimit.go`

### Redis down

- Middlewares en mode "fail open" (autorisent requêtes)
- Tracking continue en PostgreSQL (sans compteur visites)

### Profil non détecté

- Ajouter patterns dans `tracking.go` > `detectProfile()`
- Considérer intégration API Clearbit/IPinfo pour lookup IP→entreprise

## Roadmap

### Phase 2-3 (actuel)
- ✅ CORS
- ✅ Recovery
- ✅ Request ID
- ✅ Logger
- ✅ Tracking visiteurs
- ✅ Rate limiting global + IA

### Phase 6 (production)
- [ ] Métriques Prometheus
- [ ] Health checks détaillés
- [ ] Distributed tracing (OpenTelemetry)
- [ ] IP lookup enrichment (Clearbit/GeoIP)

## Liens Utiles

- [Fiber Middleware](https://docs.gofiber.io/api/middleware)
- [go-redis Documentation](https://redis.uptrace.dev/)
- [zerolog Logging](https://github.com/rs/zerolog)
- [Rate Limiting Strategies](https://redis.io/glossary/rate-limiting/)
