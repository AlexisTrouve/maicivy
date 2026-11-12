#!/bin/bash

# Implementation Validation Script
# Vérifie que tous les fichiers Phase 3 (Letters) sont bien créés

echo "========================================="
echo "Phase 3 - Letters Implementation Validation"
echo "========================================="
echo ""

# Colors
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Counters
PASSED=0
FAILED=0

# Function to check file exists
check_file() {
  if [ -f "$1" ]; then
    echo -e "${GREEN}✓${NC} $1"
    ((PASSED++))
  else
    echo -e "${RED}✗${NC} $1 (MISSING)"
    ((FAILED++))
  fi
}

echo "Checking Components..."
check_file "components/letters/AccessGate.tsx"
check_file "components/letters/LetterGenerator.tsx"
check_file "components/letters/LetterPreview.tsx"
check_file "components/letters/index.ts"
check_file "components/letters/README.md"

echo ""
echo "Checking Hooks..."
check_file "hooks/useVisitCount.ts"

echo ""
echo "Checking Pages..."
check_file "app/letters/page.tsx"

echo ""
echo "Checking Types & API..."
check_file "lib/types.ts"
check_file "lib/api.ts"

echo ""
echo "Checking Documentation..."
check_file "LETTERS_FRONTEND_IMPLEMENTATION_SUMMARY.md"
check_file "API_REQUIREMENTS_LETTERS.md"
check_file "PHASE3_COMPLETION_REPORT.md"

echo ""
echo "========================================="
echo "Results:"
echo "========================================="
echo -e "${GREEN}Passed: $PASSED${NC}"
echo -e "${RED}Failed: $FAILED${NC}"
echo ""

if [ $FAILED -eq 0 ]; then
  echo -e "${GREEN}✓ All files present!${NC}"
  echo ""
  echo "Next steps:"
  echo "1. npm run type-check"
  echo "2. npm run lint"
  echo "3. npm run dev (test /letters route)"
  exit 0
else
  echo -e "${RED}✗ Some files are missing!${NC}"
  exit 1
fi
