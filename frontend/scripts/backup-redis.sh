#!/bin/bash

# Script de backup Redis

set -e

CONTAINER_NAME="maicivy_redis"
BACKUP_DIR="/opt/maicivy/backups/redis"
RETENTION_DAYS=7

mkdir -p "$BACKUP_DIR"

TIMESTAMP=$(date +"%Y%m%d_%H%M%S")
BACKUP_FILE="$BACKUP_DIR/redis_backup_$TIMESTAMP.rdb"

echo "$(date): Starting Redis backup..."

# Trigger BGSAVE
docker exec $CONTAINER_NAME redis-cli -a ${REDIS_PASSWORD} BGSAVE

# Attendre fin du save
sleep 5

# Copier le fichier RDB
docker cp $CONTAINER_NAME:/data/dump.rdb "$BACKUP_FILE"

if [ $? -eq 0 ]; then
    echo "$(date): Redis backup successful: $BACKUP_FILE"

    # Cleanup old backups
    find "$BACKUP_DIR" -name "redis_backup_*.rdb" -mtime +$RETENTION_DAYS -delete
else
    echo "$(date): Redis backup FAILED!" >&2
    exit 1
fi
