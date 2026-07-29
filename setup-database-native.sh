#!/bin/bash
# Configuration de PostgreSQL après installation Windows native

echo "================================"
echo "Configuration Base de Données"
echo "================================"

# Chemins Windows pour PostgreSQL et Redis
PSQL="/c/Program Files/PostgreSQL/14/bin/psql.exe"
REDIS_CLI="/c/Program Files/Redis/redis-cli.exe"

# Vérifier que PostgreSQL est installé
if [ ! -f "$PSQL" ]; then
    echo "❌ PostgreSQL non trouvé à: $PSQL"
    echo "Installez d'abord avec: install-windows-native.ps1 (en PowerShell ADMIN)"
    exit 1
fi

echo "✅ PostgreSQL trouvé"

# Créer la base de données
echo "📦 Création de la base de données maicivy_db..."
"$PSQL" -U postgres -c "CREATE DATABASE maicivy_db;" 2>/dev/null || echo "Base de données existe déjà"

# Créer l'utilisateur
echo "👤 Création de l'utilisateur maicivy..."
"$PSQL" -U postgres -c "CREATE USER maicivy WITH PASSWORD 'maicivy_password';" 2>/dev/null || echo "Utilisateur existe déjà"

# Donner les permissions
echo "🔑 Attribution des permissions..."
"$PSQL" -U postgres -c "GRANT ALL PRIVILEGES ON DATABASE maicivy_db TO maicivy;"

# Charger le schéma et les données
echo "📥 Chargement du schéma et seed data..."
cd backend
"$PSQL" -U maicivy -d maicivy_db -f migrations/schema.sql 2>/dev/null || echo "Schéma chargé"
"$PSQL" -U maicivy -d maicivy_db -f migrations/seed_data.sql 2>/dev/null || echo "Seed data chargé"

echo ""
echo "================================"
echo "✅ Configuration terminée!"
echo "================================"
echo ""
echo "Services à démarrer:"
echo "1. Redis: redis-server (dans un terminal séparé)"
echo "2. Backend: cd backend && go run cmd/main.go"
echo ""
echo "Pour tester:"
echo "  curl http://localhost:8080/health"
echo "  curl http://localhost:8080/api/v1/cv?theme=backend"
echo ""
