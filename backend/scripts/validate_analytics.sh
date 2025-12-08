#!/bin/bash

# Script de validation rapide de l'implémentation analytics
# Usage: ./scripts/validate_analytics.sh

set -e

BACKEND_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$BACKEND_DIR"

echo "======================================"
echo "  Analytics Implementation Validator"
echo "======================================"
echo ""

# Colors
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

SUCCESS=0
FAILED=0

check_file() {
    if [ -f "$1" ]; then
        echo -e "${GREEN}✓${NC} $1"
        SUCCESS=$((SUCCESS + 1))
    else
        echo -e "${RED}✗${NC} $1 - MISSING"
        FAILED=$((FAILED + 1))
    fi
}

check_dir() {
    if [ -d "$1" ]; then
        echo -e "${GREEN}✓${NC} $1/"
        SUCCESS=$((SUCCESS + 1))
    else
        echo -e "${RED}✗${NC} $1/ - MISSING"
        FAILED=$((FAILED + 1))
    fi
}

echo "1. Checking directory structure..."
echo "-----------------------------------"
check_dir "internal/services"
check_dir "internal/api"
check_dir "internal/websocket"
check_dir "internal/middleware"
check_dir "internal/metrics"
check_dir "internal/jobs"
echo ""

echo "2. Checking service files..."
echo "----------------------------"
check_file "internal/services/analytics.go"
check_file "internal/services/analytics_test.go"
echo ""

echo "3. Checking API files..."
echo "------------------------"
check_file "internal/api/analytics.go"
check_file "internal/api/analytics_test.go"
echo ""

echo "4. Checking WebSocket files..."
echo "------------------------------"
check_file "internal/websocket/analytics.go"
check_file "internal/websocket/README.md"
echo ""

echo "5. Checking middleware files..."
echo "--------------------------------"
check_file "internal/middleware/analytics.go"
echo ""

echo "6. Checking metrics files..."
echo "----------------------------"
check_file "internal/metrics/analytics.go"
echo ""

echo "7. Checking jobs files..."
echo "-------------------------"
check_file "internal/jobs/analytics_cleanup.go"
echo ""

echo "8. Checking documentation..."
echo "----------------------------"
check_file "ANALYTICS_IMPLEMENTATION_SUMMARY.md"
check_file "ANALYTICS_INTEGRATION_GUIDE.md"
check_file "ANALYTICS_DELIVERABLES.md"
check_file "cmd/main_with_analytics.go.example"
echo ""

echo "9. Checking Go dependencies..."
echo "------------------------------"
if grep -q "github.com/gofiber/contrib/websocket" go.mod 2>/dev/null; then
    echo -e "${GREEN}✓${NC} gofiber/websocket dependency"
    SUCCESS=$((SUCCESS + 1))
else
    echo -e "${YELLOW}⚠${NC} gofiber/websocket dependency - NOT FOUND (run: go get github.com/gofiber/contrib/websocket)"
fi

if grep -q "github.com/prometheus/client_golang" go.mod 2>/dev/null; then
    echo -e "${GREEN}✓${NC} prometheus/client_golang dependency"
    SUCCESS=$((SUCCESS + 1))
else
    echo -e "${YELLOW}⚠${NC} prometheus/client_golang dependency - NOT FOUND (run: go get github.com/prometheus/client_golang/prometheus)"
fi
echo ""

echo "10. Checking code compilation..."
echo "--------------------------------"
if go build -o /dev/null ./... 2>/dev/null; then
    echo -e "${GREEN}✓${NC} Code compiles successfully"
    SUCCESS=$((SUCCESS + 1))
else
    echo -e "${RED}✗${NC} Compilation failed"
    FAILED=$((FAILED + 1))
    echo "  Run 'go build ./...' to see errors"
fi
echo ""

echo "11. Running tests..."
echo "-------------------"
if go test ./internal/services -run TestAnalyticsService -v > /dev/null 2>&1; then
    echo -e "${GREEN}✓${NC} Service tests pass"
    SUCCESS=$((SUCCESS + 1))
else
    echo -e "${RED}✗${NC} Service tests failed"
    FAILED=$((FAILED + 1))
    echo "  Run 'go test ./internal/services -v' to see errors"
fi

if go test ./internal/api -run TestAnalyticsAPI -v > /dev/null 2>&1; then
    echo -e "${GREEN}✓${NC} API tests pass"
    SUCCESS=$((SUCCESS + 1))
else
    echo -e "${RED}✗${NC} API tests failed"
    FAILED=$((FAILED + 1))
    echo "  Run 'go test ./internal/api -v' to see errors"
fi
echo ""

echo "12. Checking test coverage..."
echo "-----------------------------"
COVERAGE=$(go test ./internal/services -coverprofile=/tmp/coverage.out 2>/dev/null | grep coverage | awk '{print $2}' | sed 's/%//')
if [ ! -z "$COVERAGE" ]; then
    if (( $(echo "$COVERAGE >= 80" | bc -l) )); then
        echo -e "${GREEN}✓${NC} Service coverage: ${COVERAGE}% (>= 80%)"
        SUCCESS=$((SUCCESS + 1))
    else
        echo -e "${YELLOW}⚠${NC} Service coverage: ${COVERAGE}% (< 80%)"
    fi
else
    echo -e "${YELLOW}⚠${NC} Could not measure coverage"
fi
echo ""

echo "======================================"
echo "  Summary"
echo "======================================"
echo ""
echo -e "Success: ${GREEN}${SUCCESS}${NC}"
echo -e "Failed:  ${RED}${FAILED}${NC}"
echo ""

if [ $FAILED -eq 0 ]; then
    echo -e "${GREEN}✓ All checks passed!${NC}"
    echo ""
    echo "Next steps:"
    echo "  1. Copy main_with_analytics.go.example to main.go"
    echo "  2. Build: go build -o maicivy ./cmd/main.go"
    echo "  3. Run: ./maicivy"
    echo "  4. Test: curl http://localhost:8080/api/v1/analytics/realtime"
    echo ""
    exit 0
else
    echo -e "${RED}✗ Some checks failed${NC}"
    echo ""
    echo "Please fix the issues above before proceeding."
    echo ""
    exit 1
fi
