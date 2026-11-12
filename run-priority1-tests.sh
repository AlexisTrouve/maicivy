#!/bin/bash
# Script de validation des tests Priority 1
# Usage: ./run-priority1-tests.sh

set -e

echo "========================================"
echo "   Tests Frontend Priority 1 - maicivy"
echo "========================================"
echo ""

cd frontend

echo "📦 Installation des dépendances (si nécessaire)..."
if [ ! -d "node_modules" ]; then
  npm install
else
  echo "✓ node_modules existe déjà"
fi
echo ""

echo "🧪 TESTS CV COMPONENTS"
echo "----------------------------------------"
echo "1️⃣  CVThemeSelector..."
npm run test -- components/cv/__tests__/CVThemeSelector.test.tsx --silent --run || echo "❌ FAILED"
echo ""

echo "2️⃣  ExperienceTimeline..."
npm run test -- components/cv/__tests__/ExperienceTimeline.test.tsx --silent --run || echo "❌ FAILED"
echo ""

echo "3️⃣  SkillsCloud..."
npm run test -- components/cv/__tests__/SkillsCloud.test.tsx --silent --run || echo "❌ FAILED"
echo ""

echo "🧪 TESTS LETTERS COMPONENTS"
echo "----------------------------------------"
echo "4️⃣  LetterGenerator..."
npm run test -- components/letters/__tests__/LetterGenerator.test.tsx --silent --run || echo "❌ FAILED"
echo ""

echo "5️⃣  LetterPreview..."
npm run test -- components/letters/__tests__/LetterPreview.test.tsx --silent --run || echo "❌ FAILED"
echo ""

echo "6️⃣  AccessGate..."
npm run test -- components/letters/__tests__/AccessGate.test.tsx --silent --run || echo "❌ FAILED"
echo ""

echo "🧪 TESTS ANALYTICS COMPONENTS"
echo "----------------------------------------"
echo "7️⃣  RealtimeVisitors..."
npm run test -- components/analytics/__tests__/RealtimeVisitors.test.tsx --silent --run || echo "❌ FAILED"
echo ""

echo "8️⃣  ThemeStats..."
npm run test -- components/analytics/__tests__/ThemeStats.test.tsx --silent --run || echo "❌ FAILED"
echo ""

echo "========================================"
echo "📊 COVERAGE GLOBAL"
echo "========================================"
npm run test -- --coverage components/cv components/letters components/analytics --run || echo "❌ Coverage FAILED"
echo ""

echo "========================================"
echo "✅ TESTS PRIORITY 1 TERMINÉS"
echo "========================================"
echo ""
echo "Fichiers de tests:"
echo "  - CV: 3 fichiers (6 + 12 + 11 = 29 tests)"
echo "  - Letters: 3 fichiers (13 + 13 + 15 = 41 tests)"
echo "  - Analytics: 2 fichiers (12 + 13 = 25 tests)"
echo "  - TOTAL: 8 fichiers, 95 tests"
echo ""
echo "Voir TESTING_VALIDATION.md pour plus de détails."
