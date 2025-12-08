# Quick Start - Backend

Guide rapide pour démarrer le backend maicivy en moins de 5 minutes.

## 🚀 Démarrage Rapide

### 1. Installer les dépendances

```bash
cd backend
go mod download
```

### 2. Démarrer les services

Depuis la racine du projet:

```bash
docker-compose up -d postgres redis
```

### 3. Configurer l'environnement

```bash
cd backend
cp .env.example .env
# Éditez .env si nécessaire (valeurs par défaut OK pour dev local)
```

### 4. Lancer le backend

```bash
go run cmd/main.go
```

### 5. Tester

```bash
curl http://localhost:8080/health
```

**C'est tout !** 🎉

---

## 📋 Commandes Utiles

### Développement

```bash
# Lancer avec hot reload
make dev

# Ou sans Makefile
go run cmd/main.go
```

### Tests

```bash
# Tests rapides (unitaires seulement)
make test-short

# Tous les tests
make test

# Coverage
make test-coverage
```

### Build

```bash
# Build binaire
make build

# Build Docker
make docker-build
```

### Linting

```bash
# Formater et vérifier
make lint

# Juste formater
make fmt

# Juste vérifier
make vet
```

---

## 🌐 Endpoints

Une fois démarré sur http://localhost:8080:

| Endpoint | Description |
|----------|-------------|
| `GET /health` | Health check rapide |
| `GET /health/deep` | Health check complet (DB + Redis) |
| `GET /api/v1/` | Info API |

---

## 🐛 Problèmes Courants

### Port 8080 déjà utilisé

```bash
# Changer le port dans .env
echo "SERVER_PORT=8081" >> .env
```

### DB connection failed

```bash
# Vérifier que les services tournent
docker-compose ps

# Redémarrer si nécessaire
docker-compose restart postgres redis
```

### Module path incorrect

Si vous voyez des erreurs d'import, éditez `go.mod` ligne 1:

```go
module github.com/VOTRE_USERNAME/maicivy
```

---

## 📚 Documentation Complète

- **Guide de validation:** `backend/VALIDATION.md`
- **Documentation backend:** `backend/README.md`
- **Implémentation:** `docs/implementation/02_BACKEND_FOUNDATION.md`

---

**Prêt pour le développement !** 🚀
