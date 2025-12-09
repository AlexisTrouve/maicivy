#!/bin/bash

# Script pour exécuter les tests backend Go
# Ce script vérifie Go, installe les dépendances et lance les tests

set -e

echo "========================================="
echo "Backend Go Tests Runner"
echo "========================================="

# Vérifier si Go est installé
if ! command -v go &> /dev/null; then
    echo "❌ Go n'est pas installé."
    echo "Veuillez installer Go depuis https://golang.org/dl/"
    echo "Version recommandée: Go 1.22+"
    exit 1
fi

echo "✅ Go version: $(go version)"

# Vérifier qu'on est dans le bon dossier
if [ ! -f "go.mod" ]; then
    echo "❌ go.mod non trouvé. Êtes-vous dans le dossier backend/?"
    exit 1
fi

echo "📦 Installation des dépendances..."
go mod download
go mod tidy

echo ""
echo "========================================="
echo "🧪 Exécution des tests..."
echo "========================================="
echo ""

# Exécuter les tests avec verbose et coverage
go test -v -race -cover -coverprofile=coverage.out ./... 2>&1 | tee test_results.log

# Extraire le coverage
echo ""
echo "========================================="
echo "📊 Coverage Summary"
echo "========================================="
go tool cover -func=coverage.out | tail -20

# Générer rapport HTML
echo ""
echo "Génération du rapport HTML..."
go tool cover -html=coverage.out -o coverage.html

echo ""
echo "========================================="
echo "✅ Tests terminés!"
echo "========================================="
echo "📄 Logs: test_results.log"
echo "📊 Coverage: coverage.html"
echo "Pour voir le coverage: open coverage.html (macOS) ou start coverage.html (Windows)"
