#!/bin/bash

# Script de renouvellement SSL
# Exécuté par cron tous les jours

set -e

echo "$(date): Checking SSL certificates renewal..."

# Renouveler les certificats si nécessaire
certbot renew --quiet --nginx

# Recharger Nginx si certificats renouvelés
if [ $? -eq 0 ]; then
    echo "$(date): SSL certificates renewed, reloading Nginx..."
    docker exec maicivy_nginx nginx -s reload
else
    echo "$(date): No renewal needed."
fi
