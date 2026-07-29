# Architecture des Middlewares

## Vue d'Ensemble

```
┌─────────────────────────────────────────────────────────────────┐
│                        CLIENT REQUEST                            │
└────────────────────────────────┬────────────────────────────────┘
                                 │
                                 ▼
                    ┌────────────────────────┐
                    │   1. CORS Middleware   │
                    │  - Allow Origins       │
                    │  - Credentials         │
                    │  - Preflight Cache     │
                    └───────────┬────────────┘
                                │
                                ▼
                    ┌────────────────────────┐
                    │ 2. Recovery Middleware │
                    │  - Catch Panics        │
                    │  - Stack Trace Log     │
                    │  - Return 500 Error    │
                    └───────────┬────────────┘
                                │
                                ▼
                    ┌────────────────────────┐
                    │ 3. RequestID Middleware│
                    │  - Generate UUID       │
                    │  - c.Locals("requestid")│
                    │  - X-Request-ID Header │
                    └───────────┬────────────┘
                                │
                                ▼
                    ┌────────────────────────┐
                    │  4. Logger Middleware  │
                    │  - Structured JSON     │
                    │  - Request/Response    │
                    │  - Duration, Status    │
                    └───────────┬────────────┘
                                │
                                ▼
                    ┌────────────────────────┐
                    │ 5. Compression Middleware│
                    │  - Gzip Responses      │
                    │  - Best Speed Level    │
                    └───────────┬────────────┘
                                │
                                ▼
         ┌──────────────────────────────────────────┐
         │       6. Tracking Middleware             │
         │  - Session Cookie (UUID)                 │
         │  - Redis Visit Counter ──────► Redis    │
         │  - Profile Detection                     │
         │  - PostgreSQL Async ─────────► PostgreSQL│
         │                                           │
         │  Expose:                                  │
         │  - c.Locals("session_id")                │
         │  - c.Locals("visit_count")               │
         │  - c.Locals("profile_detected")          │
         └──────────────────┬───────────────────────┘
                            │
                            ▼
         ┌──────────────────────────────────────────┐
         │    7. Rate Limiting Global Middleware    │
         │  - 100 req/min per IP                    │
         │  - Redis Counter ─────────────► Redis   │
         │  - X-RateLimit-* Headers                 │
         │  - 429 if exceeded                       │
         └──────────────────┬───────────────────────┘
                            │
                            ▼
              ┌─────────────────────────┐
              │   ROUTE GROUP: /api/v1  │
              └────────────┬────────────┘
                           │
         ┌─────────────────┴─────────────────┐
         │                                    │
         ▼                                    ▼
  ┌─────────────┐               ┌────────────────────────┐
  │  CV Routes  │               │   Letters Routes       │
  │  (Phase 2)  │               │     (Phase 3)          │
  └─────────────┘               └───────────┬────────────┘
                                            │
                                            ▼
                               ┌────────────────────────┐
                               │ 8. Rate Limiting AI    │
                               │  - 5 gen/day/session   │
                               │  - 2min cooldown       │
                               │  - Redis Counters ──► Redis
                               │  - 429 if exceeded     │
                               └───────────┬────────────┘
                                           │
                                           ▼
                                  ┌────────────────┐
                                  │  HANDLER CODE  │
                                  │  - Generate AI │
                                  │  - Access Gate │
                                  └────────────────┘
```

---

## Flux de Données

### 1. Nouvelle Visite

```
Client (no cookie)
      │
      ▼
Tracking Middleware
      │
      ├─► Generate session_id (UUID)
      ├─► Set cookie (HTTPOnly, Secure, SameSite=Lax)
      ├─► Redis: INCR visitor:{session_id}:count → 1
      ├─► Redis: EXPIRE visitor:{session_id}:count 30d
      ├─► Detect profile (User-Agent)
      ├─► c.Locals("session_id", uuid)
      ├─► c.Locals("visit_count", 1)
      ├─► c.Locals("profile_detected", "professional")
      └─► Goroutine: Save visitor to PostgreSQL
            │
            ├─► Hash IP (SHA-256)
            ├─► INSERT/UPDATE visitors table
            └─► (async, non-blocking)
```

### 2. Visite Récurrente

```
Client (with cookie)
      │
      ▼
Tracking Middleware
      │
      ├─► Parse cookie → session_id
      ├─► Redis: INCR visitor:{session_id}:count → 2+
      ├─► Redis: GET visitor:{session_id}:profile
      ├─► c.Locals("session_id", uuid)
      ├─► c.Locals("visit_count", 2+)
      ├─► c.Locals("profile_detected", "recruiter")
      └─► Goroutine: UPDATE visitor in PostgreSQL
```

### 3. Génération Lettre IA (Phase 3)

```
Client (3+ visits)
      │
      ▼
Rate Limiting AI Middleware
      │
      ├─► Get session_id from c.Locals()
      ├─► Redis: EXISTS ratelimit:ai:cooldown:{session_id}
      │     ├─► If exists → 429 "Cooldown active"
      │     └─► If not exists → Continue
      │
      ├─► Redis: GET ratelimit:ai:daily:{session_id}
      │     ├─► If >= 5 → 429 "Daily limit reached"
      │     └─► If < 5 → Continue
      │
      ├─► Redis: INCR ratelimit:ai:daily:{session_id}
      ├─► Redis: EXPIRE ratelimit:ai:daily:{session_id} 24h
      ├─► Redis: SET ratelimit:ai:cooldown:{session_id} "1" 2min
      ├─► Set X-RateLimit-AI-* headers
      └─► c.Next() → Handler
            │
            ▼
      Handler: Generate Letter
            │
            ├─► Check visit_count >= 3 (Access Gate)
            ├─► Call Claude API
            └─► Return letter
```

---

## Dépendances entre Middlewares

### CRITIQUES (ordre non-négociable)

1. **CORS AVANT TOUT**
   - Sinon: OPTIONS preflight échoue
   - Impact: Frontend ne peut pas faire de requêtes

2. **Recovery AVANT handlers**
   - Sinon: Panics crashent le serveur
   - Impact: Downtime complet

3. **RequestID AVANT Logger**
   - Sinon: Logs sans request ID
   - Impact: Pas de traçabilité

4. **Logger APRÈS RequestID**
   - Sinon: c.Locals("requestid") = nil
   - Impact: Panic dans Logger

5. **Tracking AVANT RateLimiting AI**
   - Sinon: session_id non disponible
   - Impact: Rate limiting AI échoue (401)

### OPTIONNELLES (ordre flexible)

- Compression peut être avant ou après Tracking
- RateLimiting Global peut être avant ou après Tracking

---

## Stockage Redis

### Clés Utilisées

```
# Tracking
visitor:{session_id}:count          TTL: 30 jours
visitor:{session_id}:profile        TTL: 30 jours

# Rate Limiting Global
ratelimit:global:{ip}               TTL: 1 minute

# Rate Limiting IA
ratelimit:ai:daily:{session_id}     TTL: 24 heures
ratelimit:ai:cooldown:{session_id}  TTL: 2 minutes
```

### Exemple Redis Dump

```redis
127.0.0.1:6379> KEYS visitor:*
1) "visitor:550e8400-e29b-41d4-a716-446655440000:count"
2) "visitor:550e8400-e29b-41d4-a716-446655440000:profile"

127.0.0.1:6379> GET visitor:550e8400-e29b-41d4-a716-446655440000:count
"3"

127.0.0.1:6379> TTL visitor:550e8400-e29b-41d4-a716-446655440000:count
(integer) 2591999  # ~30 jours

127.0.0.1:6379> KEYS ratelimit:*
1) "ratelimit:global:192.168.1.1"
2) "ratelimit:ai:daily:550e8400-e29b-41d4-a716-446655440000"
3) "ratelimit:ai:cooldown:550e8400-e29b-41d4-a716-446655440000"
```

---

## Stockage PostgreSQL

### Table: visitors

```sql
CREATE TABLE visitors (
    id SERIAL PRIMARY KEY,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    deleted_at TIMESTAMP,

    session_id VARCHAR(255) UNIQUE NOT NULL,
    ip_hash VARCHAR(64) NOT NULL,  -- SHA-256

    user_agent TEXT,
    browser VARCHAR(100),
    os VARCHAR(100),
    device VARCHAR(50),

    visit_count INT DEFAULT 1,
    first_visit TIMESTAMP NOT NULL,
    last_visit TIMESTAMP NOT NULL,

    profile_detected VARCHAR(50) DEFAULT 'unknown',
    company_name VARCHAR(255),
    linkedin_url VARCHAR(500),

    country VARCHAR(100),
    city VARCHAR(100),
    timezone VARCHAR(50)
);

CREATE INDEX idx_visitors_session_id ON visitors(session_id);
CREATE INDEX idx_visitors_ip_hash ON visitors(ip_hash);
CREATE INDEX idx_visitors_profile ON visitors(profile_detected);
```

### Exemple Query

```sql
-- Visiteur avec 3+ visites
SELECT * FROM visitors WHERE visit_count >= 3;

-- Profils cibles
SELECT * FROM visitors WHERE profile_detected IN ('recruiter', 'tech_lead', 'cto');

-- Statistiques
SELECT
    profile_detected,
    COUNT(*) as count,
    AVG(visit_count) as avg_visits
FROM visitors
GROUP BY profile_detected
ORDER BY count DESC;
```

---

## Headers HTTP

### Request Headers (Client → Server)

```http
GET /api/v1/cv HTTP/1.1
Host: localhost:8080
User-Agent: Mozilla/5.0 (Windows NT 10.0; Win64; x64)
Cookie: maicivy_session=550e8400-e29b-41d4-a716-446655440000
Origin: http://localhost:3000
X-Request-ID: 7f3e6a1c-2b4d-4e8f-9c0a-1a2b3c4d5e6f  (optionnel, proxy)
```

### Response Headers (Server → Client)

```http
HTTP/1.1 200 OK
X-Request-ID: 7f3e6a1c-2b4d-4e8f-9c0a-1a2b3c4d5e6f
X-RateLimit-Limit: 100
X-RateLimit-Remaining: 95
X-RateLimit-AI-Limit: 5
X-RateLimit-AI-Remaining: 3
Access-Control-Allow-Origin: http://localhost:3000
Access-Control-Allow-Credentials: true
Set-Cookie: maicivy_session=550e8400-...; HttpOnly; Secure; SameSite=Lax
Content-Type: application/json
```

### Rate Limit Exceeded (429)

```http
HTTP/1.1 429 Too Many Requests
X-RateLimit-Limit: 100
X-RateLimit-Remaining: 0
Retry-After: 60
Content-Type: application/json

{
  "error": "Too many requests",
  "message": "Rate limit exceeded. Max 100 requests per minute.",
  "retry_after": 60
}
```

### AI Cooldown (429)

```http
HTTP/1.1 429 Too Many Requests
X-RateLimit-AI-Limit: 5
X-RateLimit-AI-Remaining: 4
Content-Type: application/json

{
  "error": "Cooldown active",
  "message": "Please wait before generating another letter",
  "retry_after": 95
}
```

---

## Logs Structurés (zerolog)

### Format JSON

```json
{
  "level": "info",
  "request_id": "7f3e6a1c-2b4d-4e8f-9c0a-1a2b3c4d5e6f",
  "method": "GET",
  "path": "/api/v1/cv",
  "ip": "192.168.1.1",
  "status": 200,
  "duration_ms": 45,
  "user_agent": "Mozilla/5.0 (Windows NT 10.0; Win64; x64)",
  "size": 1234,
  "message": "HTTP request",
  "time": "2025-12-08T14:30:45Z"
}
```

### Panic Recovery Log

```json
{
  "level": "error",
  "request_id": "7f3e6a1c-2b4d-4e8f-9c0a-1a2b3c4d5e6f",
  "path": "/api/v1/letters/generate",
  "method": "POST",
  "panic": "runtime error: invalid memory address or nil pointer dereference",
  "stack": "goroutine 1 [running]:\nruntime/debug.Stack()\n\t/usr/local/go/src/runtime/debug/stack.go:24 +0x65\n...",
  "message": "Panic recovered",
  "time": "2025-12-08T14:30:45Z"
}
```

---

## Métriques (Phase 6 - Prometheus)

### Counters

```prometheus
# Nombre total de requêtes
http_requests_total{method="GET", path="/api/v1/cv", status="200"} 1234

# Rejets rate limiting
rate_limit_rejections_total{type="global"} 45
rate_limit_rejections_total{type="ai"} 12
```

### Histograms

```prometheus
# Durée des requêtes
http_request_duration_seconds_bucket{method="GET", path="/api/v1/cv", le="0.005"} 890
http_request_duration_seconds_bucket{method="GET", path="/api/v1/cv", le="0.01"} 1200
http_request_duration_seconds_bucket{method="GET", path="/api/v1/cv", le="0.05"} 1230
http_request_duration_seconds_sum{method="GET", path="/api/v1/cv"} 12.34
http_request_duration_seconds_count{method="GET", path="/api/v1/cv"} 1234
```

### Gauges

```prometheus
# Sessions actives (30 derniers jours)
visitor_sessions_active 456

# Profils cibles détectés
visitor_target_profiles{profile="recruiter"} 23
visitor_target_profiles{profile="tech_lead"} 12
visitor_target_profiles{profile="cto"} 5
```

---

## Cas d'Usage

### Cas 1: Premier Visiteur Recruteur

```
1. Client arrive sur site (pas de cookie)
2. CORS: Autorise origin frontend
3. Recovery: Aucun panic
4. RequestID: Génère UUID "abc-123"
5. Logger: Log "GET / 200" avec request_id="abc-123"
6. Tracking:
   - Génère session_id "def-456"
   - Set cookie maicivy_session="def-456"
   - Redis: INCR visitor:def-456:count → 1
   - User-Agent: "LinkedIn Sales Navigator"
   - Détecte profile="recruiter"
   - c.Locals("visit_count", 1)
   - c.Locals("profile_detected", "recruiter")
   - Goroutine: INSERT visitor PostgreSQL
7. RateLimiting Global:
   - IP="1.2.3.4"
   - Redis: INCR ratelimit:global:1.2.3.4 → 1
   - Headers: X-RateLimit-Limit=100, X-RateLimit-Remaining=99
8. Handler: Retourne page CV
```

**Résultat:**
- Cookie créé
- Profil "recruiter" détecté
- Accès immédiat features IA (profil cible)

### Cas 2: Visiteur 3ème Visite

```
1. Client revient (cookie existant)
2-5. (CORS, Recovery, RequestID, Logger comme avant)
6. Tracking:
   - Parse cookie → session_id="def-456"
   - Redis: INCR visitor:def-456:count → 3
   - c.Locals("visit_count", 3)
   - Goroutine: UPDATE visitor PostgreSQL (visit_count=3)
7. RateLimiting Global: OK
8. Handler:
   - Check visit_count >= 3 → Access granted
   - Affiche bouton "Generate Letter"
```

**Résultat:**
- Access gate débloqué (3+ visites)
- Features IA disponibles

### Cas 3: Génération Lettre IA (6ème de la journée)

```
1. Client clique "Generate Letter" (cookie existant, 5 lettres déjà générées)
2-7. (Middlewares standards)
8. RateLimiting AI:
   - Get session_id="def-456"
   - Redis: GET ratelimit:ai:daily:def-456 → 5
   - 5 >= 5 → LIMIT REACHED
   - Return 429 "Daily limit reached"
```

**Résultat:**
- Erreur 429 avec message clair
- Client doit attendre 24h

### Cas 4: Génération Lettre IA (Cooldown Actif)

```
1. Client génère lettre, puis réessaie immédiatement
2-7. (Middlewares standards)
8. RateLimiting AI:
   - Get session_id="def-456"
   - Redis: EXISTS ratelimit:ai:cooldown:def-456 → YES
   - Redis: TTL ratelimit:ai:cooldown:def-456 → 95 secondes
   - Return 429 "Cooldown active, retry_after=95"
```

**Résultat:**
- Erreur 429 avec timer
- Frontend affiche "Attendez 1m35s"

---

## Sécurité

### Attaque DDoS

```
Attaquant: 200 req/sec depuis IP 1.2.3.4

Middleware RateLimiting Global:
- Requête 1-100: OK (1min)
- Requête 101+: 429 Too Many Requests
- Headers: Retry-After: 60

Résultat: Serveur protégé, seulement 100 req/min passent
```

### Attaque Spam IA

```
Attaquant: Génère 10 lettres/jour (même session)

Middleware RateLimiting AI:
- Génération 1-5: OK
- Génération 6: 429 "Daily limit reached"

Résultat: Coûts API Claude/GPT contrôlés (5 max/jour/session)
```

### XSS via Cookie

```
Attaquant: Injecte script dans cookie

Cookie: HttpOnly=true
→ JavaScript ne peut pas accéder au cookie
→ XSS échoue
```

### CSRF

```
Attaquant: Site malveillant fait requête cross-origin

Cookie: SameSite=Lax
CORS: AllowOrigins spécifiques (pas *)
→ Requête POST cross-origin bloquée
```

---

## Performance

### Benchmark (estimé)

```
BenchmarkCORS-8              50000000    0.1 ms/op
BenchmarkRecovery-8          50000000    0.1 ms/op
BenchmarkRequestID-8         30000000    0.2 ms/op
BenchmarkLogger-8            20000000    0.5 ms/op
BenchmarkTracking-8           5000000    2.0 ms/op
BenchmarkRateLimiting-8      10000000    1.0 ms/op

Total Middleware Overhead:   ~4 ms/request
```

### Optimisations Appliquées

1. **Redis Pipelining** (futur):
   ```go
   pipe := redis.Pipeline()
   pipe.Incr(ctx, visitCountKey)
   pipe.Get(ctx, profileKey)
   pipe.Exec(ctx)
   ```

2. **PostgreSQL Async**:
   ```go
   go tm.saveVisitor(...)  // Non-blocking
   ```

3. **Cookie Parsing Cache** (Fiber built-in):
   - Cookie parsé une fois par requête

4. **TTL Automatique Redis**:
   - Pas de cleanup manuel (Redis gère expiration)

---

**Version:** 1.0
**Date:** 2025-12-08
**Auteur:** Claude (Sonnet 4.5)
