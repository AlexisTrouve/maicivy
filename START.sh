#!/bin/bash

# Script de démarrage complet pour maicivy
# Usage: ./START.sh ou bash START.sh

# Déterminer le répertoire du script
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR" || exit 1

echo "🚀 Démarrage de maicivy depuis $SCRIPT_DIR..."
echo ""

# 1. Vérifier et démarrer PostgreSQL
echo "1️⃣  Démarrage PostgreSQL..."
if pg_isready -h localhost -p 5432 > /dev/null 2>&1; then
    echo "✅ PostgreSQL déjà démarré"
else
    echo "⏳ Démarrage de PostgreSQL..."
    sudo systemctl start postgresql || sudo service postgresql start
    sleep 2
fi

# 2. Vérifier et démarrer Redis
echo ""
echo "2️⃣  Démarrage Redis..."
if redis-cli ping > /dev/null 2>&1; then
    echo "✅ Redis déjà démarré"
else
    echo "⏳ Tentative de démarrage Redis (peut échouer si déjà actif)..."
    redis-server --daemonize yes 2>/dev/null || echo "ℹ️  Redis déjà en cours ou erreur (vérifier avec redis-cli ping)"
    sleep 1
fi

# 3. Démarrer le Backend (Go)
echo ""
echo "3️⃣  Démarrage Backend (Go + Fiber)..."
if [ ! -d "$SCRIPT_DIR/backend" ]; then
    echo "❌ Erreur: dossier backend/ non trouvé dans $SCRIPT_DIR"
    exit 1
fi
cd "$SCRIPT_DIR/backend" || { echo "❌ Impossible d'accéder à backend/"; exit 1; }
if lsof -ti:8080 > /dev/null 2>&1; then
    echo "⚠️  Port 8080 déjà utilisé - arrêt du processus..."
    lsof -ti:8080 | xargs kill -9 2>/dev/null
    sleep 2
fi
nohup go run cmd/main.go > /tmp/maicivy-backend.log 2>&1 &
BACKEND_PID=$!
echo "⏳ Backend démarré (PID: $BACKEND_PID) - attente de l'initialisation..."
sleep 8

# Vérifier que le backend a démarré
if ps -p $BACKEND_PID > /dev/null 2>&1; then
    echo "✅ Backend opérationnel sur http://localhost:8080"
    echo "   📊 Logs: tail -f /tmp/maicivy-backend.log"
else
    echo "❌ Erreur démarrage backend - vérifier les logs: tail -f /tmp/maicivy-backend.log"
fi

# 4. Démarrer le Frontend (Next.js)
echo ""
echo "4️⃣  Démarrage Frontend (Next.js 14)..."
if [ ! -d "$SCRIPT_DIR/frontend" ]; then
    echo "❌ Erreur: dossier frontend/ non trouvé dans $SCRIPT_DIR"
    exit 1
fi
cd "$SCRIPT_DIR/frontend" || { echo "❌ Impossible d'accéder à frontend/"; exit 1; }
if lsof -ti:3000 > /dev/null 2>&1; then
    echo "⚠️  Port 3000 déjà utilisé - arrêt du processus..."
    lsof -ti:3000 | xargs kill -9 2>/dev/null
    sleep 2
fi
nohup npm run dev > /tmp/maicivy-frontend.log 2>&1 &
FRONTEND_PID=$!
echo "⏳ Frontend démarré (PID: $FRONTEND_PID) - compilation initiale..."
sleep 15

# Vérifier que le frontend a démarré
if ps -p $FRONTEND_PID > /dev/null 2>&1; then
    echo "✅ Frontend opérationnel sur http://localhost:3000"
    echo "   📊 Logs: tail -f /tmp/maicivy-frontend.log"
else
    echo "❌ Erreur démarrage frontend - vérifier les logs: tail -f /tmp/maicivy-frontend.log"
fi

# 5. Résumé
echo ""
echo "=================================================="
echo "✨ maicivy est prêt !"
echo "=================================================="
echo ""
echo "🌐 Frontend (UI):    http://localhost:3000"
echo "🔧 Backend (API):    http://localhost:8080"
echo ""
echo "📄 Pages disponibles:"
echo "   • Accueil:        http://localhost:3000/"
echo "   • CV Dynamique:   http://localhost:3000/cv"
echo "   • Lettres IA:     http://localhost:3000/letters"
echo "   • Analytics:      http://localhost:3000/analytics"
echo ""
echo "📊 Logs en temps réel:"
echo "   • Backend:  tail -f /tmp/maicivy-backend.log"
echo "   • Frontend: tail -f /tmp/maicivy-frontend.log"
echo ""
echo "🛑 Pour arrêter tout:"
echo "   • kill $BACKEND_PID $FRONTEND_PID"
echo "   • Ou: pkill -f 'go run cmd/main.go' && pkill -f 'next dev'"
echo "=================================================="
