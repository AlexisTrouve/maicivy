# Database Migrations

Ce dossier contient les migrations SQL pour le projet maicivy.

## Structure

- `000001_init_schema.up.sql` - Migration initiale : création de toutes les tables
- `000001_init_schema.down.sql` - Rollback de la migration initiale

## Utilisation

### Via le script migrate.go

```bash
# Appliquer toutes les migrations
cd backend
go run scripts/migrate.go up

# Rollback toutes les migrations
go run scripts/migrate.go down

# Voir la version actuelle
go run scripts/migrate.go version

# Forcer une version spécifique (en cas d'erreur)
go run scripts/migrate.go force 1
```

### Via golang-migrate CLI

```bash
# Installer l'outil
go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest

# Appliquer migrations
migrate -path ./migrations -database "postgres://user:pass@localhost:5432/maicivy?sslmode=disable" up

# Rollback
migrate -path ./migrations -database "postgres://user:pass@localhost:5432/maicivy?sslmode=disable" down
```

### Via GORM AutoMigrate

L'alternative est d'utiliser GORM AutoMigrate :

```bash
# Dans votre code backend
cd backend
# La fonction RunAutoMigrations() dans internal/database/migrations.go
# sera appelée automatiquement au démarrage de l'application
```

## Tables Créées

1. **experiences** - Expériences professionnelles
   - UUID primary key
   - Technologies (PostgreSQL array)
   - Soft deletes

2. **skills** - Compétences techniques
   - UUID primary key
   - Niveaux (beginner, intermediate, advanced, expert)
   - Soft deletes

3. **projects** - Projets réalisés
   - UUID primary key
   - Intégration GitHub (stars, forks)
   - Soft deletes

4. **visitors** - Tracking visiteurs
   - UUID primary key
   - Session tracking
   - Détection de profil
   - Soft deletes

5. **generated_letters** - Lettres IA générées
   - UUID primary key
   - Foreign key vers visitors
   - Métadonnées IA (model, tokens, temps)
   - Soft deletes

6. **analytics_events** - Événements analytics
   - UUID primary key
   - Foreign key vers visitors
   - JSONB pour données flexibles
   - Soft deletes

## Index Créés

Les index suivants sont créés pour optimiser les performances :

- **experiences**: category, deleted_at
- **skills**: category, level, deleted_at
- **projects**: category, featured, deleted_at
- **visitors**: session_id (unique), ip_hash, profile_detected, deleted_at
- **generated_letters**: visitor_id, letter_type, created_at, deleted_at
- **analytics_events**: visitor_id, event_type, created_at, deleted_at, composite (event_type, created_at)

## Triggers

Trigger `update_updated_at_column()` sur toutes les tables pour mettre à jour automatiquement le champ `updated_at`.

## Notes

- Toutes les tables utilisent des UUIDs v4 comme clés primaires
- Extension `uuid-ossp` requise pour `gen_random_uuid()`
- Timezone UTC par défaut
- Soft deletes activés via le champ `deleted_at`
- Foreign keys avec `ON DELETE CASCADE` pour intégrité référentielle
- JSONB utilisé pour données flexibles (company_info, event_data)
- PostgreSQL arrays natifs pour technologies et tags

## Prochaines Migrations

Pour créer une nouvelle migration :

```bash
# Via golang-migrate
migrate create -ext sql -dir ./migrations -seq description_de_la_migration

# Cela crée deux fichiers :
# - XXXXXX_description_de_la_migration.up.sql
# - XXXXXX_description_de_la_migration.down.sql
```

Toujours tester le rollback (`down`) avant de commiter !
