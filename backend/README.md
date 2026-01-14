# Maicivy Backend

Backend API server pour le projet maicivy - CV interactif avec génération de lettres par IA.

## Stack Technique

- **Language:** Go 1.22+
- **Framework:** Fiber v2
- **ORM:** GORM
- **Base de données:** PostgreSQL
- **Cache:** Redis
- **Logger:** zerolog

## Structure du Projet

```
backend/
├── cmd/
│   └── main.go                 # Point d'entrée
├── internal/
│   ├── config/                 # Configuration
│   ├── database/               # Connexions DB (PostgreSQL, Redis)
│   ├── api/                    # Handlers HTTP
│   └── utils/                  # Utilitaires (errors, etc.)
├── pkg/
│   └── logger/                 # Logger structuré
├── go.mod
├── go.sum
├── Dockerfile
└── .env.example
```

## Installation

### Prérequis

- Go 1.22+
- Docker et Docker Compose (pour dev local)

### Développement Local

1. **Cloner le repository et naviguer dans le dossier backend:**

```bash
cd backend
```

2. **Copier le fichier d'environnement:**

```bash
cp .env.example .env
```

3. **Installer les dépendances:**

```bash
go mod download
```

4. **Lancer les services (PostgreSQL + Redis) avec Docker Compose:**

Depuis la racine du projet:

```bash
docker-compose up -d postgres redis
```

5. **Lancer l'application:**

```bash
go run cmd/main.go
```

Le serveur démarre sur `http://localhost:8080`

## Endpoints Disponibles

### Health Checks

- **GET /health** - Health check rapide (API seulement)
- **GET /health/deep** - Health check complet (API + PostgreSQL + Redis)

### API v1

- **GET /api/v1/** - Information sur l'API

## Tests

### Tests Unitaires (rapides)

```bash
go test -v -short ./...
```

### Tous les Tests (incluant integration)

```bash
go test -v ./...
```

### Coverage

```bash
go test -v -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

### Linting

```bash
go fmt ./...
go vet ./...
```

## Build

### Build Local

```bash
go build -o main ./cmd/main.go
./main
```

### Build Docker

```bash
docker build -t maicivy-backend .
docker run -p 8080:8080 --env-file .env maicivy-backend
```

## Variables d'Environnement

Voir `.env.example` pour la liste complète des variables.

Variables critiques:
- `DB_HOST` - Hôte PostgreSQL
- `DB_PASSWORD` - Mot de passe PostgreSQL
- `REDIS_HOST` - Hôte Redis
- `SERVER_PORT` - Port du serveur (défaut: 8080)
- `ENVIRONMENT` - Environnement (development/production)

## Logging

Le logger utilise zerolog avec deux modes:

- **Development:** Logs colorés dans la console (niveau DEBUG)
- **Production:** Logs JSON structurés (niveau INFO)

## Architecture

### Middlewares Globaux

1. **Recover** - Récupération des panics
2. **RequestID** - ID unique par requête
3. **Compress** - Compression gzip/brotli
4. **CORS** - Configuration CORS
5. **Logger** - Logging de chaque requête HTTP

### Error Handling

Utilisation de `AppError` pour les erreurs typées avec codes HTTP appropriés.

Exemple:
```go
return utils.NewBadRequestError("Invalid input")
```

### Database Connection

- **PostgreSQL:** Pool de connexions configuré (10 idle, 100 max)
- **Redis:** Client avec timeout et pool configurés

## Prochaines Étapes

Phase 2:
- Implémentation du schema de base de données (models)
- Middlewares custom (tracking, rate limiting)
- API CV dynamique

Phase 3:
- Services IA (Claude, GPT-4)
- Génération de lettres de motivation

## Documentation

Pour plus de détails, voir:
- [PROJECT_SPEC.md](../docs/PROJECT_SPEC.md)
- [02_BACKEND_FOUNDATION.md](../docs/implementation/02_BACKEND_FOUNDATION.md)
