#!/bin/bash

# Script de restauration Redis

set -e

if [ -z "$1" ]; then
    echo "Usage: $0 <backup_file.rdb>"
    exit 1
fi

BACKUP_FILE=$1
CONTAINER_NAME="maicivy_redis"

if [ ! -f "$BACKUP_FILE" ]; then
    echo "Error: Backup file not found: $BACKUP_FILE"
    exit 1
fi

echo "WARNING: This will OVERWRITE the current Redis data!"
read -p "Are you sure? (yes/no): " CONFIRM

if [ "$CONFIRM" != "yes" ]; then
    echo "Restore cancelled."
    exit 0
fi

echo "$(date): Starting Redis restore from $BACKUP_FILE..."

# Stop Redis
docker-compose -f /opt/maicivy/docker/docker-compose.prod.yml stop redis

# Copy backup file
docker cp "$BACKUP_FILE" $CONTAINER_NAME:/data/dump.rdb

# Start Redis
docker-compose -f /opt/maicivy/docker/docker-compose.prod.yml start redis

if [ $? -eq 0 ]; then
    echo "$(date): Restore successful!"
else
    echo "$(date): Restore FAILED!" >&2
    exit 1
fi
