#!/bin/bash

# Script de restauration PostgreSQL

set -e

if [ -z "$1" ]; then
    echo "Usage: $0 <backup_file.sql.gz>"
    exit 1
fi

BACKUP_FILE=$1
CONTAINER_NAME="maicivy_postgres"
POSTGRES_USER="${POSTGRES_USER:-maicivy}"
POSTGRES_DB="${POSTGRES_DB:-maicivy}"

if [ ! -f "$BACKUP_FILE" ]; then
    echo "Error: Backup file not found: $BACKUP_FILE"
    exit 1
fi

echo "WARNING: This will OVERWRITE the current database!"
read -p "Are you sure? (yes/no): " CONFIRM

if [ "$CONFIRM" != "yes" ]; then
    echo "Restore cancelled."
    exit 0
fi

echo "$(date): Starting PostgreSQL restore from $BACKUP_FILE..."

# Drop et recréer la DB
docker exec -t $CONTAINER_NAME psql -U $POSTGRES_USER -c "DROP DATABASE IF EXISTS $POSTGRES_DB;"
docker exec -t $CONTAINER_NAME psql -U $POSTGRES_USER -c "CREATE DATABASE $POSTGRES_DB;"

# Restore
gunzip -c "$BACKUP_FILE" | docker exec -i $CONTAINER_NAME psql -U $POSTGRES_USER -d $POSTGRES_DB

if [ $? -eq 0 ]; then
    echo "$(date): Restore successful!"
else
    echo "$(date): Restore FAILED!" >&2
    exit 1
fi
