#!/bin/bash

# Script de backup PostgreSQL
# Exécuté quotidiennement par cron

set -e

# Configuration
CONTAINER_NAME="maicivy_postgres"
POSTGRES_USER="${POSTGRES_USER:-maicivy}"
POSTGRES_DB="${POSTGRES_DB:-maicivy}"
BACKUP_DIR="/opt/maicivy/backups"
RETENTION_DAYS=30

# Créer répertoire backup si inexistant
mkdir -p "$BACKUP_DIR"

# Nom du fichier avec timestamp
TIMESTAMP=$(date +"%Y%m%d_%H%M%S")
BACKUP_FILE="$BACKUP_DIR/maicivy_backup_$TIMESTAMP.sql.gz"

echo "$(date): Starting PostgreSQL backup..."

# Backup avec pg_dump via Docker
docker exec -t $CONTAINER_NAME pg_dump -U $POSTGRES_USER $POSTGRES_DB | gzip > "$BACKUP_FILE"

if [ $? -eq 0 ]; then
    echo "$(date): Backup successful: $BACKUP_FILE"

    # Taille du backup
    SIZE=$(du -h "$BACKUP_FILE" | cut -f1)
    echo "$(date): Backup size: $SIZE"

    # Suppression des backups anciens (> RETENTION_DAYS)
    find "$BACKUP_DIR" -name "maicivy_backup_*.sql.gz" -mtime +$RETENTION_DAYS -delete
    echo "$(date): Old backups cleaned (retention: $RETENTION_DAYS days)"
else
    echo "$(date): Backup FAILED!" >&2
    exit 1
fi

# Optionnel: Upload vers S3
# aws s3 cp "$BACKUP_FILE" s3://maicivy-backups/postgres/
