#!/bin/bash

# Frontend profiling script using Lighthouse CI
# Profiles Core Web Vitals and performance metrics

set -e

echo "======================================"
echo "Frontend Performance Profiling"
echo "======================================"
echo ""

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Configuration
FRONTEND_URL="${FRONTEND_URL:-http://localhost:3001}"
OUTPUT_DIR="./profiling-results/frontend"
TIMESTAMP=$(date +%Y%m%d_%H%M%S)

# Create output directory
mkdir -p "${OUTPUT_DIR}"
echo -e "${GREEN}Output directory: ${OUTPUT_DIR}${NC}"
echo ""

# Check if Lighthouse is installed
check_lighthouse() {
    if command -v lighthouse &> /dev/null; then
        echo -e "${GREEN}✓ Lighthouse is installed${NC}"
        return 0
    else
        echo -e "${YELLOW}⚠ Lighthouse not found, attempting to install...${NC}"
        npm install -g @lhci/cli lighthouse
        if [ $? -eq 0 ]; then
            echo -e "${GREEN}✓ Lighthouse installed successfully${NC}"
            return 0
        else
            echo -e "${RED}✗ Failed to install Lighthouse${NC}"
            echo "Install manually: npm install -g @lhci/cli lighthouse"
            exit 1
        fi
    fi
}

# Check if frontend is running
check_frontend() {
    echo -e "${YELLOW}Checking if frontend is running...${NC}"
    if curl -s "${FRONTEND_URL}" > /dev/null; then
        echo -e "${GREEN}✓ Frontend is running at ${FRONTEND_URL}${NC}"
        return 0
    else
        echo -e "${RED}✗ Frontend is not running${NC}"
        echo "Start frontend with: cd frontend && npm run dev"
        exit 1
    fi
}

# Run Lighthouse audit
run_lighthouse() {
    local url=$1
    local output_name=$2

    echo -e "${YELLOW}Running Lighthouse audit for ${url}...${NC}"

    lighthouse "${url}" \
        --output=html \
        --output=json \
        --output-path="${OUTPUT_DIR}/${output_name}-${TIMESTAMP}" \
        --chrome-flags="--headless --no-sandbox" \
        --only-categories=performance,accessibility,best-practices,seo \
        --quiet

    echo -e "${GREEN}✓ Lighthouse audit completed: ${output_name}${NC}"
}

# Run bundle analyzer
run_bundle_analyzer() {
    echo -e "${YELLOW}Analyzing bundle size...${NC}"

    cd frontend

    # Build with analyzer
    ANALYZE=true npm run build 2>&1 | tee "${OUTPUT_DIR}/bundle-analysis-${TIMESTAMP}.txt"

    cd ..

    echo -e "${GREEN}✓ Bundle analysis completed${NC}"
}

# Extract Core Web Vitals from Lighthouse report
extract_web_vitals() {
    local report_file="${OUTPUT_DIR}/$1-${TIMESTAMP}.report.json"

    if [ -f "${report_file}" ]; then
        echo ""
        echo "Core Web Vitals from ${1}:"
        echo "----------------------------------------"

        # Extract metrics using jq if available
        if command -v jq &> /dev/null; then
            local lcp=$(jq -r '.audits["largest-contentful-paint"].displayValue' "${report_file}")
            local fid=$(jq -r '.audits["max-potential-fid"].displayValue' "${report_file}")
            local cls=$(jq -r '.audits["cumulative-layout-shift"].displayValue' "${report_file}")
            local fcp=$(jq -r '.audits["first-contentful-paint"].displayValue' "${report_file}")
            local tti=$(jq -r '.audits["interactive"].displayValue' "${report_file}")
            local tbt=$(jq -r '.audits["total-blocking-time"].displayValue' "${report_file}")
            local score=$(jq -r '.categories.performance.score' "${report_file}")

            # Convert score to percentage
            score=$(echo "$score * 100" | bc)

            echo "Performance Score: ${score}%"
            echo "LCP (Largest Contentful Paint): ${lcp}"
            echo "FID (First Input Delay): ${fid}"
            echo "CLS (Cumulative Layout Shift): ${cls}"
            echo "FCP (First Contentful Paint): ${fcp}"
            echo "TTI (Time to Interactive): ${tti}"
            echo "TBT (Total Blocking Time): ${tbt}"

            # Check thresholds
            echo ""
            echo "Thresholds Check:"

            # LCP threshold: < 2.5s (good), < 4s (needs improvement), >= 4s (poor)
            lcp_value=$(echo "${lcp}" | grep -oP '\d+\.\d+' | head -1)
            if [ -n "$lcp_value" ]; then
                if (( $(echo "$lcp_value < 2.5" | bc -l) )); then
                    echo -e "  LCP: ${GREEN}✓ Good${NC} (< 2.5s)"
                elif (( $(echo "$lcp_value < 4.0" | bc -l) )); then
                    echo -e "  LCP: ${YELLOW}⚠ Needs Improvement${NC} (< 4s)"
                else
                    echo -e "  LCP: ${RED}✗ Poor${NC} (>= 4s)"
                fi
            fi

            # CLS threshold: < 0.1 (good), < 0.25 (needs improvement), >= 0.25 (poor)
            cls_value=$(echo "${cls}" | grep -oP '\d+\.\d+' | head -1)
            if [ -n "$cls_value" ]; then
                if (( $(echo "$cls_value < 0.1" | bc -l) )); then
                    echo -e "  CLS: ${GREEN}✓ Good${NC} (< 0.1)"
                elif (( $(echo "$cls_value < 0.25" | bc -l) )); then
                    echo -e "  CLS: ${YELLOW}⚠ Needs Improvement${NC} (< 0.25)"
                else
                    echo -e "  CLS: ${RED}✗ Poor${NC} (>= 0.25)"
                fi
            fi

            # Performance score threshold
            if (( $(echo "$score >= 90" | bc -l) )); then
                echo -e "  Performance Score: ${GREEN}✓ Good${NC} (>= 90)"
            elif (( $(echo "$score >= 50" | bc -l) )); then
                echo -e "  Performance Score: ${YELLOW}⚠ Needs Improvement${NC} (>= 50)"
            else
                echo -e "  Performance Score: ${RED}✗ Poor${NC} (< 50)"
            fi
        else
            echo "Install jq to see detailed metrics: brew install jq"
        fi

        echo ""
    fi
}

# Check dependencies
check_lighthouse
check_frontend

# Run audits for different pages
echo ""
echo "======================================"
echo "Running Lighthouse Audits"
echo "======================================"
echo ""

# 1. Homepage
run_lighthouse "${FRONTEND_URL}/" "homepage"

# 2. CV page
run_lighthouse "${FRONTEND_URL}/cv" "cv-page"

# 3. Letters page
run_lighthouse "${FRONTEND_URL}/letters" "letters-page"

# 4. Analytics page
run_lighthouse "${FRONTEND_URL}/analytics" "analytics-page"

# Extract Web Vitals
echo ""
echo "======================================"
echo "Core Web Vitals Summary"
echo "======================================"

extract_web_vitals "homepage"
extract_web_vitals "cv-page"
extract_web_vitals "letters-page"
extract_web_vitals "analytics-page"

# Bundle analysis (optional, requires build)
if [ "${ANALYZE_BUNDLE}" = "true" ]; then
    echo ""
    echo "======================================"
    echo "Bundle Size Analysis"
    echo "======================================"
    run_bundle_analyzer
fi

# Generate summary report
echo ""
echo "======================================"
echo "Summary"
echo "======================================"
echo ""
echo "Reports saved to: ${OUTPUT_DIR}"
echo ""
echo "HTML Reports:"
find "${OUTPUT_DIR}" -name "*.report.html" -type f -exec basename {} \;
echo ""
echo "To view reports:"
echo "  open ${OUTPUT_DIR}/homepage-${TIMESTAMP}.report.html"
echo ""
echo "Lighthouse CI (continuous monitoring):"
echo "  npm install -g @lhci/cli"
echo "  lhci autorun --config=lighthouserc.json"
echo ""
echo -e "${GREEN}Frontend profiling completed successfully!${NC}"
echo ""

# Usage instructions
echo "======================================"
echo "Usage"
echo "======================================"
echo ""
echo "Basic usage:"
echo "  ./scripts/profile-frontend.sh"
echo ""
echo "With bundle analysis:"
echo "  ANALYZE_BUNDLE=true ./scripts/profile-frontend.sh"
echo ""
echo "Custom URL:"
echo "  FRONTEND_URL=https://your-site.com ./scripts/profile-frontend.sh"
echo ""
echo "Lighthouse options:"
echo "  lighthouse ${FRONTEND_URL} --view  # Open report immediately"
echo "  lighthouse ${FRONTEND_URL} --preset=desktop  # Desktop audit"
echo "  lighthouse ${FRONTEND_URL} --throttling.cpuSlowdownMultiplier=4  # Slow CPU"
echo ""
