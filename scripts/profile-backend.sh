#!/bin/bash

# Backend profiling script using Go pprof
# Requires backend server running on localhost:6060 (pprof server)

set -e

echo "======================================"
echo "Backend Performance Profiling"
echo "======================================"
echo ""

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Configuration
PPROF_URL="http://localhost:6060"
OUTPUT_DIR="./profiling-results/backend"
DURATION=30

# Check if pprof server is running
check_pprof_server() {
    echo -e "${YELLOW}Checking if pprof server is running...${NC}"
    if curl -s "${PPROF_URL}/debug/pprof/" > /dev/null; then
        echo -e "${GREEN}✓ pprof server is running${NC}"
        return 0
    else
        echo -e "${RED}✗ pprof server is not running${NC}"
        echo "Make sure backend is running with pprof enabled (port 6060)"
        echo "Add to main.go:"
        echo '  import _ "net/http/pprof"'
        echo '  go http.ListenAndServe("localhost:6060", nil)'
        exit 1
    fi
}

# Create output directory
mkdir -p "${OUTPUT_DIR}"
echo -e "${GREEN}Output directory: ${OUTPUT_DIR}${NC}"
echo ""

# 1. CPU Profiling
echo -e "${YELLOW}[1/5] Starting CPU profiling (${DURATION}s)...${NC}"
go tool pprof -http=:8080 "${PPROF_URL}/debug/pprof/profile?seconds=${DURATION}" &
PPROF_PID=$!
echo "CPU profile will open in browser at http://localhost:8080"
echo "Wait ${DURATION} seconds..."
sleep $((DURATION + 5))
kill $PPROF_PID 2>/dev/null || true
echo -e "${GREEN}✓ CPU profiling completed${NC}"
echo ""

# 2. Memory Profiling (Heap)
echo -e "${YELLOW}[2/5] Memory profiling (heap)...${NC}"
go tool pprof -http=:8081 "${PPROF_URL}/debug/pprof/heap" &
PPROF_PID=$!
echo "Memory profile will open in browser at http://localhost:8081"
sleep 5
kill $PPROF_PID 2>/dev/null || true
echo -e "${GREEN}✓ Memory profiling completed${NC}"
echo ""

# 3. Goroutine Profiling
echo -e "${YELLOW}[3/5] Goroutine profiling...${NC}"
go tool pprof -http=:8082 "${PPROF_URL}/debug/pprof/goroutine" &
PPROF_PID=$!
echo "Goroutine profile will open in browser at http://localhost:8082"
sleep 5
kill $PPROF_PID 2>/dev/null || true
echo -e "${GREEN}✓ Goroutine profiling completed${NC}"
echo ""

# 4. Allocation Profiling
echo -e "${YELLOW}[4/5] Allocation profiling...${NC}"
go tool pprof -http=:8083 "${PPROF_URL}/debug/pprof/allocs" &
PPROF_PID=$!
echo "Allocation profile will open in browser at http://localhost:8083"
sleep 5
kill $PPROF_PID 2>/dev/null || true
echo -e "${GREEN}✓ Allocation profiling completed${NC}"
echo ""

# 5. Block Profiling (mutex contention)
echo -e "${YELLOW}[5/5] Block profiling...${NC}"
go tool pprof -http=:8084 "${PPROF_URL}/debug/pprof/block" &
PPROF_PID=$!
echo "Block profile will open in browser at http://localhost:8084"
sleep 5
kill $PPROF_PID 2>/dev/null || true
echo -e "${GREEN}✓ Block profiling completed${NC}"
echo ""

# Generate text reports
echo -e "${YELLOW}Generating text reports...${NC}"

# CPU profile text report
curl -s "${PPROF_URL}/debug/pprof/profile?seconds=10" -o "${OUTPUT_DIR}/cpu.prof"
go tool pprof -text "${OUTPUT_DIR}/cpu.prof" > "${OUTPUT_DIR}/cpu-top.txt" 2>/dev/null || true

# Heap profile text report
curl -s "${PPROF_URL}/debug/pprof/heap" -o "${OUTPUT_DIR}/heap.prof"
go tool pprof -text "${OUTPUT_DIR}/heap.prof" > "${OUTPUT_DIR}/heap-top.txt" 2>/dev/null || true

# Goroutine profile text report
curl -s "${PPROF_URL}/debug/pprof/goroutine" -o "${OUTPUT_DIR}/goroutine.prof"
go tool pprof -text "${OUTPUT_DIR}/goroutine.prof" > "${OUTPUT_DIR}/goroutine-top.txt" 2>/dev/null || true

echo -e "${GREEN}✓ Text reports generated${NC}"
echo ""

# Display summary
echo "======================================"
echo "Profiling Summary"
echo "======================================"
echo ""
echo "Profiles saved to: ${OUTPUT_DIR}"
echo ""
echo "Available profiles:"
echo "  - cpu.prof: CPU usage by function"
echo "  - heap.prof: Memory allocations"
echo "  - goroutine.prof: Goroutine stacks"
echo ""
echo "Text reports:"
echo "  - cpu-top.txt: Top CPU consumers"
echo "  - heap-top.txt: Top memory allocations"
echo "  - goroutine-top.txt: Active goroutines"
echo ""
echo "To view profiles interactively:"
echo "  go tool pprof -http=:8080 ${OUTPUT_DIR}/cpu.prof"
echo "  go tool pprof -http=:8081 ${OUTPUT_DIR}/heap.prof"
echo ""
echo "Common pprof commands:"
echo "  top10       - Show top 10 entries"
echo "  list func   - Show source code for function"
echo "  web         - Generate graph visualization"
echo "  svg > out.svg - Save graph as SVG"
echo ""

# Quick analysis
echo "======================================"
echo "Quick Analysis"
echo "======================================"

# Show top CPU consumers
if [ -f "${OUTPUT_DIR}/cpu-top.txt" ]; then
    echo ""
    echo "Top 5 CPU consumers:"
    head -n 10 "${OUTPUT_DIR}/cpu-top.txt" | tail -n 5
fi

# Show top memory allocations
if [ -f "${OUTPUT_DIR}/heap-top.txt" ]; then
    echo ""
    echo "Top 5 memory allocations:"
    head -n 10 "${OUTPUT_DIR}/heap-top.txt" | tail -n 5
fi

# Check goroutine count
GOROUTINE_COUNT=$(curl -s "${PPROF_URL}/debug/pprof/goroutine?debug=1" | grep -c "^goroutine" || echo "unknown")
echo ""
echo "Active goroutines: ${GOROUTINE_COUNT}"

if [ "$GOROUTINE_COUNT" != "unknown" ] && [ "$GOROUTINE_COUNT" -gt 1000 ]; then
    echo -e "${RED}⚠ Warning: High goroutine count (possible leak)${NC}"
fi

echo ""
echo -e "${GREEN}Profiling completed successfully!${NC}"
echo ""

# Check pprof server first
check_pprof_server

# Usage instructions
echo "======================================"
echo "Usage"
echo "======================================"
echo ""
echo "Run this script while backend is running:"
echo "  ./scripts/profile-backend.sh"
echo ""
echo "Options:"
echo "  DURATION=60 ./scripts/profile-backend.sh  # 60 second CPU profile"
echo ""
echo "Flamegraph generation:"
echo "  go-torch -u ${PPROF_URL} -t 30 > flamegraph.svg"
echo "  (requires go-torch: go get github.com/uber/go-torch)"
echo ""
