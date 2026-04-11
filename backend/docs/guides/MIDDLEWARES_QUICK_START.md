# Middlewares Backend - Quick Start

## ✅ Implémentation Complète

Document 04_BACKEND_MIDDLEWARES.md → **100% TERMINÉ**

## 📁 Fichiers Créés

### Middlewares (6 fichiers)
- `internal/middleware/cors.go` - Configuration CORS
- `internal/middleware/recovery.go` - Récupération panics
- `internal/middleware/requestid.go` - UUID unique par requête
- `internal/middleware/logger.go` - Logging structuré JSON
- `internal/middleware/tracking.go` - Tracking visiteurs + profil
- `internal/middleware/ratelimit.go` - Rate limiting global + IA

### Tests (3 fichiers)
- `internal/middleware/tracking_test.go`
- `internal/middleware/ratelimit_test.go`
- `internal/middleware/testing_helpers.go`

### Documentation (4 fichiers)
- `internal/middleware/README.md` - Guide complet (320 lignes)
- `internal/middleware/ARCHITECTURE.md` - Architecture technique (450 lignes)
- `MIDDLEWARES_IMPLEMENTATION_SUMMARY.md` - Summary (550 lignes)
- `MIDDLEWARES_CHECKLIST.md` - Checklist validation (430 lignes)

## 🚀 Démarrage Rapide

### 1. Installer Dépendances

```bash
cd backend

# Dépendances middlewares
go get github.com/google/uuid
go get github.com/mileusna/useragent

# Dépendances tests (optionnel)
go get github.com/stretchr/testify
go get gorm.io/driver/sqlite
```

### 2. Configuration

Créer `.env` (ou copier `.env.example`):

```bash
# CORS
ALLOWED_ORIGINS=http://localhost:3000

# Database
DB_HOST=localhost
DB_PORT=5432
DB_USER=maicivy
DB_PASSWORD=your_password
DB_NAME=maicivy

# Redis
REDIS_HOST=localhost
REDIS_PORT=6379
```

### 3. Démarrer Services

```bash
# Démarrer PostgreSQL + Redis
docker-compose up -d postgres redis

# Vérifier que services sont up
docker-compose ps
```

### 4. Compiler & Lancer

```bash
# Compiler
go build -o bin/maicivy ./cmd/main.go

# Lancer
./bin/maicivy
```

**Attendu:**
```
{"level":"info","addr":"0.0.0.0:8080","environment":"development","message":"Starting server"}
```

### 5. Tester

```bash
# Health check
curl http://localhost:8080/health
# → {"status":"ok"}

# Vérifier headers
curl -v http://localhost:8080/health 2>&1 | grep X-Request-ID
# → X-Request-ID: <uuid>

# Tester rate limiting (101 requêtes)
for i in {1..101}; do
    curl -s http://localhost:8080/health > /dev/null
    echo "Request $i"
done
# → Requête 101 devrait retourner 429 Too Many Requests

# Tester cookie tracking
curl -c cookies.txt http://localhost:8080/health
curl -b cookies.txt http://localhost:8080/health
cat cookies.txt | grep maicivy_session
# → Cookie présent
```

## ⚠️ Point d'Attention Dev Local

### Cookie Secure Flag

Si vous développez en HTTP local (pas HTTPS), le cookie ne sera pas créé car `Secure: true`.

**Solution:**

Modifier `internal/middleware/tracking.go` ligne 56:

```go
// Avant
Secure:   true,

// Après
Secure:   cfg.Environment == "production",
```

Puis recompiler:
```bash
go build -o bin/maicivy ./cmd/main.go
```

## 📊 Middlewares Actifs

### Ordre d'Exécution

1. **CORS** → Autorise frontend
2. **Recovery** → Capture panics
3. **RequestID** → UUID unique
4. **Logger** → Log structuré
5. **Compression** → Gzip responses
6. **Tracking** → Tracking visiteurs (cookie + Redis + PostgreSQL)
7. **RateLimiting Global** → 100 req/min par IP

### Rate Limiting IA (Phase 3)

Actuellement commenté, sera activé lors implémentation routes `/letters`:

```go
lettersGroup := apiV1.Group("/letters")
lettersGroup.Use(rateLimitMW.AI())  // 5 gen/jour, 2min cooldown
```

## 📖 Documentation Complète

- **Guide usage:** `internal/middleware/README.md`
- **Architecture:** `internal/middleware/ARCHITECTURE.md`
- **Summary:** `MIDDLEWARES_IMPLEMENTATION_SUMMARY.md`
- **Checklist:** `MIDDLEWARES_CHECKLIST.md`
- **Ce fichier:** Quick Start

## 🧪 Lancer Tests

```bash
# Tests unitaires
go test -v ./internal/middleware/...

# Avec coverage
go test -v -cover ./internal/middleware/...

# Tests integration (nécessite Redis + PostgreSQL)
docker-compose up -d postgres redis
go test -v -tags=integration ./internal/middleware/...
```

## 📈 Prochaines Étapes

### Sprint 2 (Phase 2)
- Implémenter doc 06 (BACKEND_CV_API)
- Décommenter routes CV

### Sprint 3 (Phase 3)
- Implémenter doc 08-09 (IA Services + Letters API)
- Activer rate limiting IA
- Access gate 3+ visites

## 🆘 Troubleshooting

### Erreur: "go: command not found"
→ Go n'est pas installé. Installer Go 1.21+

### Erreur: "Redis connection refused"
→ Démarrer Redis: `docker-compose up -d redis`

### Erreur: "PostgreSQL connection refused"
→ Démarrer PostgreSQL: `docker-compose up -d postgres`

### Cookie non créé en dev local
→ Voir section "Cookie Secure Flag" ci-dessus

### Rate limit trop strict
→ Modifier constantes dans `ratelimit.go`:
```go
const GlobalRateLimit = 1000  // Au lieu de 100
```

## ✅ Checklist Validation

- [ ] Dépendances Go installées
- [ ] Cookie Secure adapté pour dev local
- [ ] PostgreSQL + Redis running
- [ ] Compilation sans erreur
- [ ] Health check OK
- [ ] Headers X-Request-ID présents
- [ ] Rate limiting fonctionne (429 après 100 req)
- [ ] Cookie tracking créé

**Si toutes les cases cochées:** ✅ Prêt pour Phase 2!

---

**Version:** 1.0
**Date:** 2025-12-08
**Status:** ✅ COMPLÉTÉ
