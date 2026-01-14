#!/bin/bash

# Script d'arrêt complet pour maicivy
# Usage: ./STOP.sh ou bash STOP.sh

echo "🛑 Arrêt de maicivy..."
echo ""

# 1. Arrêt Backend
echo "1️⃣  Arrêt Backend (Go)..."
pkill -f "go run cmd/main.go" 2>/dev/null && echo "✅ Backend arrêté" || echo "ℹ️  Backend pas en cours"
lsof -ti:8080 | xargs kill -9 2>/dev/null

# 2. Arrêt Frontend
echo ""
echo "2️⃣  Arrêt Frontend (Next.js)..."
pkill -f "next dev" 2>/dev/null && echo "✅ Frontend arrêté" || echo "ℹ️  Frontend pas en cours"
lsof -ti:3000 | xargs kill -9 2>/dev/null

# 3. Vérification
sleep 2
echo ""
echo "3️⃣  Vérification..."
if lsof -ti:8080 > /dev/null 2>&1 || lsof -ti:3000 > /dev/null 2>&1; then
    echo "⚠️  Ports encore utilisés - nettoyage forcé..."
    lsof -ti:8080 | xargs kill -9 2>/dev/null
    lsof -ti:3000 | xargs kill -9 2>/dev/null
    sleep 1
fi

echo "✅ Tous les services maicivy sont arrêtés"
echo ""
echo "ℹ️  PostgreSQL et Redis restent actifs (services systèmes)"
echo "   Pour les arrêter aussi:"
echo "   • sudo systemctl stop postgresql"
echo "   • sudo systemctl stop redis-server (ou redis-cli shutdown)"
