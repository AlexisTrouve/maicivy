# Database Quickstart Guide

Guide rapide pour configurer et utiliser la base de données maicivy.

## Prérequis

1. PostgreSQL 15+ installé et en cours d'exécution
2. Go 1.22+ installé
3. Fichier `.env` configuré

## Configuration .env

Créer un fichier `.env` à la racine du backend :

```bash
# Database PostgreSQL
DB_HOST=localhost
DB_PORT=5432
DB_USER=maicivy
DB_PASSWORD=your_password
DB_NAME=maicivy
DB_SSLMODE=disable

# Database Redis
REDIS_HOST=localhost
REDIS_PORT=6379
REDIS_PASSWORD=
REDIS_DB=0

# Server
SERVER_HOST=0.0.0.0
SERVER_PORT=8080
ENVIRONMENT=development
```

## Option 1 : Docker Compose (Recommandé)

```bash
# Démarrer PostgreSQL et Redis
docker-compose up -d postgres redis

# Attendre que PostgreSQL soit prêt
sleep 5

# La base de données est prête !
```

## Option 2 : Installation Locale PostgreSQL

```bash
# Créer la base de données
createdb maicivy

# Ou via psql
psql -U postgres
CREATE DATABASE maicivy;
\q
```

## Appliquer les Migrations

### Méthode 1 : GORM AutoMigrate (Simple)

Les migrations sont automatiques au démarrage de l'application :

```bash
cd backend
go run cmd/main.go
```

La fonction `RunAutoMigrations()` créera toutes les tables automatiquement.

### Méthode 2 : Migrations SQL Manuelles (Contrôle Fin)

```bash
# Installer golang-migrate
go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest

# Appliquer les migrations
cd backend
go run scripts/migrate.go up

# Vérifier la version
go run scripts/migrate.go version

# Rollback si nécessaire
go run scripts/migrate.go down
```

## Peupler avec des Données de Test

```bash
cd backend
go run scripts/seed.go
```

Cela créera :
- 3 expériences professionnelles
- 10 compétences techniques
- 3 projets

## Vérifier que Tout Fonctionne

### Via psql

```bash
psql -U maicivy -d maicivy

# Lister les tables
\dt

# Vérifier les données
SELECT * FROM experiences;
SELECT * FROM skills;
SELECT * FROM projects;

# Quitter
\q
```

### Via l'API

```bash
# Démarrer le serveur
cd backend
go run cmd/main.go

# Dans un autre terminal, tester le health check
curl http://localhost:8080/health/deep
```

Réponse attendue :

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

## Structure des Tables

6 tables principales créées :

1. **experiences** - Expériences professionnelles
2. **skills** - Compétences techniques
3. **projects** - Projets réalisés
4. **visitors** - Tracking visiteurs
5. **generated_letters** - Lettres IA générées
6. **analytics_events** - Événements analytics

Voir `migrations/ERD.md` pour le diagramme complet.

## Commandes Utiles

### Développement

```bash
# Réinitialiser la base de données
go run scripts/migrate.go down
go run scripts/migrate.go up
go run scripts/seed.go

# Tests unitaires models
go test ./internal/models -v

# Tests integration database
go test ./internal/database -v

# Coverage
go test ./... -coverprofile=coverage.out
go tool cover -html=coverage.out
```

### Production

```bash
# Backup base de données
pg_dump -U maicivy maicivy > backup.sql

# Restore base de données
psql -U maicivy -d maicivy < backup.sql

# Appliquer migrations en prod
go run scripts/migrate.go up
```

## Troubleshooting

### Erreur : "database does not exist"

```bash
createdb maicivy
# ou
psql -U postgres -c "CREATE DATABASE maicivy;"
```

### Erreur : "relation already exists"

La table existe déjà. Soit :

1. Utiliser les migrations sans recréer :
```bash
go run scripts/migrate.go version
```

2. Ou réinitialiser complètement :
```bash
go run scripts/migrate.go down
go run scripts/migrate.go up
```

### Erreur : "password authentication failed"

Vérifier le `.env` :

```bash
DB_USER=maicivy
DB_PASSWORD=your_correct_password
```

### Erreur : "could not connect to server"

PostgreSQL n'est pas démarré :

```bash
# Docker
docker-compose up -d postgres

# Ou système
sudo service postgresql start
```

### Erreur : "pq: SSL is not enabled on the server"

Changer dans `.env` :

```bash
DB_SSLMODE=disable
```

## Prochaines Étapes

Une fois la base de données configurée :

1. Implémenter les endpoints CV API (Phase 2)
2. Créer les services IA (Phase 3)
3. Ajouter les analytics (Phase 4)

Voir `docs/IMPLEMENTATION_PLAN.md` pour la roadmap complète.

## Ressources

- [GORM Documentation](https://gorm.io/docs/)
- [PostgreSQL Documentation](https://www.postgresql.org/docs/)
- [golang-migrate](https://github.com/golang-migrate/migrate)
- [Models README](internal/models/README.md)
- [Migrations README](migrations/README.md)
- [ERD Diagram](migrations/ERD.md)
