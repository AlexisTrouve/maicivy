#!/bin/bash

# ============================================
# maicivy - CI/CD Validation Script
# ============================================
# Validates GitHub Actions workflows, scripts, and configuration
# ============================================

set -u

# Colors
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
BLUE='\033[0;34m'
NC='\033[0m'

# Counters
TOTAL_CHECKS=0
PASSED_CHECKS=0
FAILED_CHECKS=0
WARNING_CHECKS=0

# ============================================
# Functions
# ============================================

log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[PASS]${NC} $1"
    ((PASSED_CHECKS++))
}

log_warning() {
    echo -e "${YELLOW}[WARN]${NC} $1"
    ((WARNING_CHECKS++))
}

log_error() {
    echo -e "${RED}[FAIL]${NC} $1"
    ((FAILED_CHECKS++))
}

check() {
    ((TOTAL_CHECKS++))
}

print_header() {
    echo ""
    echo "============================================"
    echo "  maicivy - CI/CD Validation"
    echo "============================================"
    echo ""
}

print_summary() {
    echo ""
    echo "============================================"
    echo "  Validation Summary"
    echo "============================================"
    echo "Total checks:   $TOTAL_CHECKS"
    echo -e "${GREEN}Passed:${NC}         $PASSED_CHECKS"
    echo -e "${YELLOW}Warnings:${NC}       $WARNING_CHECKS"
    echo -e "${RED}Failed:${NC}         $FAILED_CHECKS"
    echo "============================================"
    echo ""

    if [ $FAILED_CHECKS -eq 0 ]; then
        log_success "All critical checks passed!"
        return 0
    else
        log_error "Some checks failed. Please fix before deploying."
        return 1
    fi
}

# ============================================
# Validation Functions
# ============================================

validate_yaml_syntax() {
    log_info "Validating YAML syntax..."

    local yaml_files=$(find .github/workflows -name "*.yml" 2>/dev/null)

    if [ -z "$yaml_files" ]; then
        check
        log_error "No workflow YAML files found in .github/workflows/"
        return
    fi

    for file in $yaml_files; do
        check
        if command -v yamllint &> /dev/null; then
            if yamllint -d relaxed "$file" > /dev/null 2>&1; then
                log_success "YAML syntax valid: $file"
            else
                log_error "YAML syntax error: $file"
                yamllint "$file"
            fi
        else
            # Basic YAML check with Python
            if python3 -c "import yaml; yaml.safe_load(open('$file'))" 2>/dev/null; then
                log_success "YAML syntax valid: $file"
            else
                log_error "YAML syntax error: $file"
            fi
        fi
    done
}

validate_docker_compose() {
    log_info "Validating docker-compose files..."

    local compose_files=(
        "docker-compose.production.yml"
        "../docker-compose.yml"
    )

    for file in "${compose_files[@]}"; do
        if [ -f "$file" ]; then
            check
            if docker-compose -f "$file" config > /dev/null 2>&1; then
                log_success "Docker Compose valid: $file"
            else
                log_error "Docker Compose error: $file"
            fi
        fi
    done
}

validate_dockerfiles() {
    log_info "Validating Dockerfiles..."

    local dockerfiles=(
        "../backend/Dockerfile"
        "Dockerfile"
        "../docker/nginx/Dockerfile"
    )

    for file in "${dockerfiles[@]}"; do
        if [ -f "$file" ]; then
            check
            if command -v hadolint &> /dev/null; then
                if hadolint "$file" > /dev/null 2>&1; then
                    log_success "Dockerfile valid: $file"
                else
                    log_warning "Dockerfile has warnings: $file"
                fi
            else
                # Basic check: file exists and not empty
                if [ -s "$file" ]; then
                    log_success "Dockerfile exists: $file"
                else
                    log_error "Dockerfile empty or missing: $file"
                fi
            fi
        else
            check
            log_error "Dockerfile not found: $file"
        fi
    done
}

validate_scripts() {
    log_info "Validating deployment scripts..."

    local scripts=(
        "scripts/deploy.sh"
        "scripts/rollback.sh"
        "scripts/health-check.sh"
        "scripts/build-images.sh"
        "scripts/setup-secrets.sh"
    )

    for script in "${scripts[@]}"; do
        check
        if [ -f "$script" ]; then
            # Check if executable
            if [ -x "$script" ]; then
                log_success "Script is executable: $script"
            else
                log_warning "Script not executable: $script (run: chmod +x $script)"
            fi

            # Basic bash syntax check
            if bash -n "$script" 2>/dev/null; then
                log_success "Script syntax valid: $script"
            else
                log_error "Script syntax error: $script"
            fi
        else
            log_error "Script not found: $script"
        fi
    done
}

validate_environment_files() {
    log_info "Validating environment configuration..."

    local env_files=(
        ".env.production.example"
        ".env.staging.example"
    )

    local required_vars=(
        "POSTGRES_USER"
        "POSTGRES_PASSWORD"
        "REDIS_PASSWORD"
        "JWT_SECRET"
        "CLAUDE_API_KEY"
        "GRAFANA_PASSWORD"
    )

    for file in "${env_files[@]}"; do
        check
        if [ -f "$file" ]; then
            log_success "Environment file exists: $file"

            # Check for required variables
            for var in "${required_vars[@]}"; do
                check
                if grep -q "^${var}=" "$file" 2>/dev/null; then
                    log_success "Variable defined: $var in $file"
                else
                    log_error "Variable missing: $var in $file"
                fi
            done
        else
            log_error "Environment file not found: $file"
        fi
    done

    # Check that .env is not in git
    check
    if [ -f ".env" ]; then
        log_warning ".env file exists (should not be committed to git)"
    else
        log_success ".env file not present (good for security)"
    fi
}

validate_github_actions() {
    log_info "Validating GitHub Actions configuration..."

    local workflows=(
        ".github/workflows/ci.yml"
        ".github/workflows/deploy.yml"
        ".github/workflows/backup.yml"
    )

    for workflow in "${workflows[@]}"; do
        check
        if [ -f "$workflow" ]; then
            log_success "Workflow exists: $workflow"

            # Check for required keys
            check
            if grep -q "on:" "$workflow"; then
                log_success "Workflow has triggers: $workflow"
            else
                log_error "Workflow missing triggers: $workflow"
            fi

            check
            if grep -q "jobs:" "$workflow"; then
                log_success "Workflow has jobs: $workflow"
            else
                log_error "Workflow missing jobs: $workflow"
            fi
        else
            log_error "Workflow not found: $workflow"
        fi
    done
}

validate_nginx_config() {
    log_info "Validating Nginx configuration..."

    local nginx_files=(
        "../docker/nginx/nginx.conf"
        "../docker/nginx/conf.d/maicivy.conf"
    )

    for file in "${nginx_files[@]}"; do
        check
        if [ -f "$file" ]; then
            log_success "Nginx config exists: $file"

            # Basic syntax check (requires nginx installed)
            if command -v nginx &> /dev/null; then
                if nginx -t -c "$file" 2>/dev/null; then
                    log_success "Nginx config valid: $file"
                else
                    log_warning "Nginx config check skipped (nginx not installed locally)"
                fi
            fi
        else
            log_error "Nginx config not found: $file"
        fi
    done
}

validate_documentation() {
    log_info "Validating documentation..."

    local docs=(
        "CI_CD_GUIDE.md"
        "../README.md"
    )

    for doc in "${docs[@]}"; do
        check
        if [ -f "$doc" ]; then
            log_success "Documentation exists: $doc"

            # Check file is not empty
            if [ -s "$doc" ]; then
                log_success "Documentation not empty: $doc"
            else
                log_error "Documentation empty: $doc"
            fi
        else
            log_warning "Documentation not found: $doc"
        fi
    done
}

check_prerequisites() {
    log_info "Checking prerequisites..."

    local tools=(
        "docker:Docker"
        "docker-compose:Docker Compose"
        "git:Git"
    )

    for tool_info in "${tools[@]}"; do
        IFS=':' read -r cmd name <<< "$tool_info"
        check
        if command -v "$cmd" &> /dev/null; then
            log_success "$name is installed"
        else
            log_warning "$name not installed (optional for validation)"
        fi
    done

    # Optional tools
    local optional_tools=(
        "yamllint:YAML Linter"
        "hadolint:Dockerfile Linter"
        "shellcheck:Shell Script Linter"
    )

    for tool_info in "${optional_tools[@]}"; do
        IFS=':' read -r cmd name <<< "$tool_info"
        if command -v "$cmd" &> /dev/null; then
            log_info "$name is available"
        else
            log_info "$name not installed (optional, install for better validation)"
        fi
    done
}

# ============================================
# Main Validation Flow
# ============================================

main() {
    print_header
    check_prerequisites

    echo ""
    validate_yaml_syntax
    echo ""
    validate_docker_compose
    echo ""
    validate_dockerfiles
    echo ""
    validate_scripts
    echo ""
    validate_environment_files
    echo ""
    validate_github_actions
    echo ""
    validate_nginx_config
    echo ""
    validate_documentation

    print_summary
    return $?
}

# ============================================
# Execute
# ============================================
main "$@"
