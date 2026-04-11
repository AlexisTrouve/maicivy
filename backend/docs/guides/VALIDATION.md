# Backend Validation Guide

Ce guide explique comment valider que le backend foundation est correctement implémenté.

## Prérequis

- Go 1.22+ installé
- Docker et Docker Compose installés
- Les services PostgreSQL et Redis sont démarrés

## Étape 1: Vérifier la Structure des Fichiers

Vérifiez que tous les fichiers Go sont présents:

```bash
cd backend

# Vérifier les fichiers principaux
ls cmd/main.go
ls internal/config/config.go
ls internal/database/postgres.go
ls internal/database/redis.go
ls internal/api/health.go
ls internal/utils/errors.go
ls pkg/logger/logger.go

# Vérifier les tests
ls internal/config/config_test.go
ls internal/utils/errors_test.go
ls internal/database/postgres_test.go
```

## Étape 2: Installer les Dépendances

```bash
cd backend
go mod download
```

Vous devriez voir les packages suivants être téléchargés:
- github.com/gofiber/fiber/v2
- gorm.io/gorm
- gorm.io/driver/postgres
- github.com/redis/go-redis/v9
- github.com/rs/zerolog
- github.com/joho/godotenv
- github.com/go-playground/validator/v10

## Étape 3: Vérifier la Compilation

Le code doit compiler sans erreur:

```bash
cd backend
go build ./cmd/main.go
```

Si la compilation réussit, un binaire `main` (ou `main.exe` sur Windows) sera créé.

## Étape 4: Lancer les Tests Unitaires

Les tests unitaires ne nécessitent PAS de base de données:

```bash
cd backend
go test -v -short ./...
```

**Résultats attendus:**
```
=== RUN   TestLoad
--- PASS: TestLoad (0.00s)
=== RUN   TestGetEnv
--- PASS: TestGetEnv (0.00s)
=== RUN   TestAppError
--- PASS: TestAppError (0.00s)
=== RUN   TestErrorConstructors
--- PASS: TestErrorConstructors (0.00s)
PASS
ok      github.com/yourusername/maicivy/internal/config    0.XXXs
ok      github.com/yourusername/maicivy/internal/utils     0.XXXs
```

## Étape 5: Démarrer PostgreSQL et Redis

Depuis la racine du projet:

```bash
docker-compose up -d postgres redis
```

Vérifiez que les containers sont actifs:

```bash
docker-compose ps
```

**Résultats attendus:**
```
NAME                COMMAND                  SERVICE             STATUS
maicivy-postgres    "docker-entrypoint.s…"   postgres            Up
maicivy-redis       "docker-entrypoint.s…"   redis               Up
```

## Étape 6: Créer le Fichier .env

```bash
cd backend
cp .env.example .env
```

Éditez `.env` et assurez-vous que les valeurs correspondent à votre docker-compose:

```env
# PostgreSQL
DB_HOST=localhost
DB_PORT=5432
DB_USER=maicivy
DB_PASSWORD=maicivy_secure_password_2024
DB_NAME=maicivy
DB_SSL_MODE=disable

# Redis
REDIS_HOST=localhost
REDIS_PORT=6379
REDIS_PASSWORD=
REDIS_DB=0
```

**Note:** Utilisez `localhost` car vous lancez le backend en dehors de Docker.

## Étape 7: Lancer le Backend

```bash
cd backend
go run cmd/main.go
```

**Résultats attendus (logs colorés en dev):**

```
11:30AM INF Logger initialized environment=development
11:30AM INF PostgreSQL connected successfully database=maicivy host=localhost
11:30AM INF Redis connected successfully addr=localhost:6379 db=0
11:30AM INF Starting server addr=0.0.0.0:8080 environment=development
```

Si vous voyez ces logs, le backend démarre correctement ! ✅

## Étape 8: Tester les Endpoints

Dans un autre terminal, testez les endpoints:

### Health Check Simple

```bash
curl http://localhost:8080/health
```

**Résultat attendu:**
```json
{
  "status": "ok",
  "services": {
    "api": "up"
  }
}
```

### Health Check Complet

```bash
curl http://localhost:8080/health/deep
```

**Résultat attendu:**
```json
{
  "status": "ok",
  "services": {
    "api": "up",
    "postgres": "up",
    "redis": "up"
  }
}
```

### API Info

```bash
curl http://localhost:8080/api/v1/
```

**Résultat attendu:**
```json
{
  "message": "maicivy API v1",
  "version": "1.0.0"
}
```

## Étape 9: Vérifier les Logs

Dans le terminal où le backend tourne, vous devriez voir les logs de requêtes:

```
11:31AM INF HTTP request duration_ms=0.123 method=GET path=/health request_id=abc123... status=200
11:31AM INF HTTP request duration_ms=1.234 method=GET path=/health/deep request_id=def456... status=200
```

## Étape 10: Tester le Graceful Shutdown

Dans le terminal du backend, appuyez sur `Ctrl+C`:

**Résultats attendus:**
```
11:32AM INF Shutting down server...
11:32AM INF Server stopped
```

Le serveur doit s'arrêter proprement en quelques secondes.

## Étape 11: Lancer les Tests d'Integration (Optionnel)

Ces tests nécessitent PostgreSQL et Redis actifs:

```bash
cd backend
go test -v ./...
```

**Note:** Le test `TestConnectPostgres` peut échouer si les credentials ne correspondent pas. C'est normal, il sera amélioré en Phase 6 avec testcontainers.

## Étape 12: Tester avec Docker (Optionnel)

Build l'image Docker:

```bash
cd backend
docker build -t maicivy-backend:test .
```

Lancez le container:

```bash
docker run -p 8080:8080 \
  -e DB_HOST=host.docker.internal \
  -e REDIS_HOST=host.docker.internal \
  -e DB_PASSWORD=maicivy_secure_password_2024 \
  maicivy-backend:test
```

**Note:** Sur Linux, remplacez `host.docker.internal` par l'IP de votre machine.

Testez les endpoints comme à l'étape 8.

## ✅ Checklist de Validation

- [ ] Tous les fichiers Go sont présents
- [ ] `go mod download` réussit
- [ ] `go build ./cmd/main.go` compile sans erreur
- [ ] `go test -v -short ./...` passe tous les tests unitaires
- [ ] PostgreSQL et Redis sont démarrés (docker-compose)
- [ ] Fichier `.env` créé avec bonnes valeurs
- [ ] `go run cmd/main.go` démarre le serveur
- [ ] Logs PostgreSQL et Redis "connected successfully"
- [ ] `curl http://localhost:8080/health` retourne `{"status":"ok"}`
- [ ] `curl http://localhost:8080/health/deep` retourne tous les services "up"
- [ ] Logs HTTP apparaissent dans le terminal
- [ ] `Ctrl+C` arrête proprement le serveur
- [ ] Image Docker build avec succès (optionnel)

## 🐛 Troubleshooting

### Erreur: "Failed to connect to PostgreSQL"

**Symptôme:**
```
FATAL Failed to connect to PostgreSQL error="failed to connect to `host=localhost ...
```

**Solution:**
1. Vérifiez que PostgreSQL est démarré: `docker-compose ps`
2. Vérifiez le password dans `.env` correspond à `docker-compose.yml`
3. Vérifiez le port 5432 est bien exposé: `docker-compose port postgres 5432`

### Erreur: "Failed to connect to Redis"

**Symptôme:**
```
FATAL Failed to connect to Redis error="dial tcp ...
```

**Solution:**
1. Vérifiez que Redis est démarré: `docker-compose ps`
2. Vérifiez le port 6379 est bien exposé: `docker-compose port redis 6379`

### Erreur: "bind: address already in use"

**Symptôme:**
```
FATAL Failed to start server error="listen tcp :8080: bind: address already in use"
```

**Solution:**
1. Un autre processus utilise le port 8080
2. Trouvez le processus: `lsof -i :8080` (Mac/Linux) ou `netstat -ano | findstr :8080` (Windows)
3. Arrêtez-le ou changez le port dans `.env`: `SERVER_PORT=8081`

### Tests Integration Échouent

**Symptôme:**
```
--- FAIL: TestConnectPostgres (5.00s)
```

**Solution:**
C'est normal pour l'instant. Les tests integration seront améliorés en Phase 6 avec testcontainers. Utilisez `-short` pour skip ces tests:
```bash
go test -v -short ./...
```

### Module "github.com/yourusername/maicivy" Not Found

**Symptôme:**
```
go: cannot find main module, but found .git/config in ...
```

**Solution:**
Vous devez modifier le module path dans `go.mod` pour correspondre à votre repository:
```go
module github.com/VOTRE_USERNAME/maicivy
```

Puis cherchez/remplacez tous les imports dans les fichiers .go.

## 📊 Métriques de Succès

Une fois validé, votre backend devrait:

- ✅ Compiler en ~5 secondes
- ✅ Démarrer en <2 secondes
- ✅ Répondre aux health checks en <10ms
- ✅ Se connecter à PostgreSQL en <500ms
- ✅ Se connecter à Redis en <100ms
- ✅ Tous les tests unitaires passent en <1 seconde

## 🎯 Prochaine Étape

Une fois cette validation complète, vous êtes prêt pour:

**Sprint 1 - Vague 3: Database Schema**
- Document: `docs/implementation/03_DATABASE_SCHEMA.md`
- Créer les models GORM
- Créer les migrations
- Ajouter les seed data

---

**Besoin d'aide?** Consultez:
- `backend/README.md` - Documentation backend
- `docs/implementation/02_BACKEND_FOUNDATION.md` - Document d'implémentation
- `BACKEND_FOUNDATION_COMPLETE.md` - Rapport d'implémentation
