#!/bin/bash

# ============================================================================
# Secrets Rotation Script for maicivy
# ============================================================================
# This script helps rotate sensitive secrets safely
# Run this quarterly or when secrets are potentially compromised
# ============================================================================

set -e  # Exit on error

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# ============================================================================
# Configuration
# ============================================================================

ENV_FILE="${1:-.env}"
BACKUP_DIR="./backups/secrets"
TIMESTAMP=$(date +%Y%m%d_%H%M%S)

# ============================================================================
# Helper Functions
# ============================================================================

log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

confirm() {
    read -p "$1 (y/N): " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        return 1
    fi
    return 0
}

# Generate random string with OpenSSL
generate_secret() {
    local length=$1
    openssl rand -base64 "$length" | tr -d '\n'
}

# Generate random hex string
generate_hex() {
    local length=$1
    openssl rand -hex "$length" | tr -d '\n'
}

# Backup current .env file
backup_env() {
    if [ ! -f "$ENV_FILE" ]; then
        log_error "$ENV_FILE not found!"
        exit 1
    fi

    mkdir -p "$BACKUP_DIR"
    local backup_file="${BACKUP_DIR}/env_backup_${TIMESTAMP}"

    cp "$ENV_FILE" "$backup_file"
    log_info "Backed up $ENV_FILE to $backup_file"

    # Encrypt backup
    if command -v gpg &> /dev/null; then
        if confirm "Encrypt backup with GPG?"; then
            gpg --symmetric --cipher-algo AES256 "$backup_file"
            rm "$backup_file"
            log_info "Backup encrypted to ${backup_file}.gpg"
        fi
    fi
}

# Update secret in .env file
update_secret() {
    local key=$1
    local value=$2

    if grep -q "^${key}=" "$ENV_FILE"; then
        # Update existing key
        sed -i.tmp "s|^${key}=.*|${key}=${value}|" "$ENV_FILE"
        rm "${ENV_FILE}.tmp"
        log_info "Updated $key"
    else
        # Add new key
        echo "${key}=${value}" >> "$ENV_FILE"
        log_info "Added $key"
    fi
}

# ============================================================================
# Rotation Functions
# ============================================================================

rotate_jwt_secret() {
    log_info "Rotating JWT secret..."

    if ! confirm "This will invalidate all existing JWT tokens. Continue?"; then
        log_warn "Skipped JWT secret rotation"
        return
    fi

    local new_secret=$(generate_secret 64)
    update_secret "JWT_SECRET" "$new_secret"

    log_warn "Action required: Users will need to re-authenticate"
}

rotate_session_secret() {
    log_info "Rotating session secret..."

    if ! confirm "This will invalidate all existing sessions. Continue?"; then
        log_warn "Skipped session secret rotation"
        return
    fi

    local new_secret=$(generate_secret 64)
    update_secret "SESSION_SECRET" "$new_secret"

    log_warn "Action required: All users will be logged out"
}

rotate_db_password() {
    log_info "Rotating database password..."

    if ! confirm "This requires updating PostgreSQL password. Continue?"; then
        log_warn "Skipped database password rotation"
        return
    fi

    local new_password=$(generate_secret 32)

    # Update .env
    update_secret "DB_PASSWORD" "$new_password"

    log_warn "Action required: Update PostgreSQL password with:"
    echo ""
    echo "  docker-compose exec postgres psql -U maicivy -d postgres"
    echo "  ALTER USER maicivy WITH PASSWORD '$new_password';"
    echo ""
}

rotate_redis_password() {
    log_info "Rotating Redis password..."

    if ! confirm "This requires updating Redis configuration. Continue?"; then
        log_warn "Skipped Redis password rotation"
        return
    fi

    local new_password=$(generate_hex 32)

    # Update .env
    update_secret "REDIS_PASSWORD" "$new_password"

    log_warn "Action required: Update Redis configuration:"
    echo ""
    echo "  1. Edit redis.conf: requirepass $new_password"
    echo "  2. Restart Redis container"
    echo ""
}

rotate_rate_limit_secret() {
    log_info "Rotating rate limit secret..."

    local new_secret=$(generate_secret 32)
    update_secret "RATE_LIMIT_SECRET" "$new_secret"
}

rotate_all() {
    log_info "=== Rotating ALL secrets ==="
    echo ""

    rotate_jwt_secret
    echo ""
    rotate_session_secret
    echo ""
    rotate_rate_limit_secret
    echo ""

    if confirm "Also rotate database password?"; then
        rotate_db_password
        echo ""
    fi

    if confirm "Also rotate Redis password?"; then
        rotate_redis_password
        echo ""
    fi

    log_info "=== Rotation complete ==="
}

# ============================================================================
# Validation
# ============================================================================

validate_secrets() {
    log_info "Validating secrets in $ENV_FILE..."

    local errors=0

    # Check for placeholder values
    if grep -q "xxxxxxxxxxxxx" "$ENV_FILE"; then
        log_error "Found placeholder API keys (xxxxxxxxxxxxx)"
        errors=$((errors + 1))
    fi

    if grep -q "change-this" "$ENV_FILE"; then
        log_error "Found default secrets (change-this-*)"
        errors=$((errors + 1))
    fi

    # Check secret lengths
    local jwt_secret=$(grep "^JWT_SECRET=" "$ENV_FILE" | cut -d'=' -f2)
    if [ ${#jwt_secret} -lt 32 ]; then
        log_error "JWT_SECRET is too short (minimum 32 characters)"
        errors=$((errors + 1))
    fi

    local session_secret=$(grep "^SESSION_SECRET=" "$ENV_FILE" | cut -d'=' -f2)
    if [ ${#session_secret} -lt 32 ]; then
        log_error "SESSION_SECRET is too short (minimum 32 characters)"
        errors=$((errors + 1))
    fi

    # Check for weak passwords
    local db_password=$(grep "^DB_PASSWORD=" "$ENV_FILE" | cut -d'=' -f2)
    if [[ "$db_password" == "password" ]] || [[ "$db_password" == "dev-password"* ]]; then
        log_error "Database password is weak"
        errors=$((errors + 1))
    fi

    if [ $errors -eq 0 ]; then
        log_info "All secrets appear valid"
        return 0
    else
        log_error "Found $errors validation errors"
        return 1
    fi
}

# ============================================================================
# Generate new secrets from scratch
# ============================================================================

generate_all_secrets() {
    log_info "Generating new secrets..."

    echo ""
    echo "# ============================================================================"
    echo "# Generated Secrets - $(date)"
    echo "# ============================================================================"
    echo ""
    echo "# JWT & Authentication"
    echo "JWT_SECRET=$(generate_secret 64)"
    echo "SESSION_SECRET=$(generate_secret 64)"
    echo "RATE_LIMIT_SECRET=$(generate_secret 32)"
    echo ""
    echo "# Database"
    echo "DB_PASSWORD=$(generate_secret 32)"
    echo ""
    echo "# Redis"
    echo "REDIS_PASSWORD=$(generate_hex 32)"
    echo ""
    echo "# ============================================================================"
    echo "# IMPORTANT: Save these secrets securely!"
    echo "# ============================================================================"
}

# ============================================================================
# Check secrets strength
# ============================================================================

check_strength() {
    log_info "Checking secrets strength..."

    # JWT Secret
    local jwt_secret=$(grep "^JWT_SECRET=" "$ENV_FILE" | cut -d'=' -f2)
    local jwt_length=${#jwt_secret}

    if [ $jwt_length -lt 32 ]; then
        log_error "JWT_SECRET: WEAK (length: $jwt_length, minimum: 32)"
    elif [ $jwt_length -lt 64 ]; then
        log_warn "JWT_SECRET: MODERATE (length: $jwt_length, recommended: 64)"
    else
        log_info "JWT_SECRET: STRONG (length: $jwt_length)"
    fi

    # Session Secret
    local session_secret=$(grep "^SESSION_SECRET=" "$ENV_FILE" | cut -d'=' -f2)
    local session_length=${#session_secret}

    if [ $session_length -lt 32 ]; then
        log_error "SESSION_SECRET: WEAK (length: $session_length, minimum: 32)"
    elif [ $session_length -lt 64 ]; then
        log_warn "SESSION_SECRET: MODERATE (length: $session_length, recommended: 64)"
    else
        log_info "SESSION_SECRET: STRONG (length: $session_length)"
    fi

    # Database Password
    local db_password=$(grep "^DB_PASSWORD=" "$ENV_FILE" | cut -d'=' -f2)
    local db_length=${#db_password}

    if [ $db_length -lt 16 ]; then
        log_error "DB_PASSWORD: WEAK (length: $db_length, minimum: 16)"
    elif [ $db_length -lt 24 ]; then
        log_warn "DB_PASSWORD: MODERATE (length: $db_length, recommended: 24)"
    else
        log_info "DB_PASSWORD: STRONG (length: $db_length)"
    fi
}

# ============================================================================
# Main Menu
# ============================================================================

show_menu() {
    echo ""
    echo "========================================="
    echo "     maicivy Secrets Rotation Tool"
    echo "========================================="
    echo ""
    echo "1. Rotate JWT secret"
    echo "2. Rotate session secret"
    echo "3. Rotate rate limit secret"
    echo "4. Rotate database password"
    echo "5. Rotate Redis password"
    echo "6. Rotate ALL secrets"
    echo "7. Validate secrets"
    echo "8. Check secrets strength"
    echo "9. Generate new secrets (output only)"
    echo "0. Exit"
    echo ""
    read -p "Select option: " -n 1 -r choice
    echo ""

    case $choice in
        1) rotate_jwt_secret ;;
        2) rotate_session_secret ;;
        3) rotate_rate_limit_secret ;;
        4) rotate_db_password ;;
        5) rotate_redis_password ;;
        6) rotate_all ;;
        7) validate_secrets ;;
        8) check_strength ;;
        9) generate_all_secrets ;;
        0) exit 0 ;;
        *) log_error "Invalid option" ;;
    esac

    show_menu
}

# ============================================================================
# Main Script
# ============================================================================

echo ""
echo "========================================="
echo "   maicivy Secrets Rotation Tool"
echo "========================================="
echo ""
echo "Environment file: $ENV_FILE"
echo ""

# Check if .env file exists
if [ ! -f "$ENV_FILE" ]; then
    log_error "$ENV_FILE not found!"
    echo ""
    echo "Create one from .env.example:"
    echo "  cp .env.example $ENV_FILE"
    exit 1
fi

# Create backup before any changes
backup_env

# Parse command line arguments
case "${2:-menu}" in
    jwt)
        rotate_jwt_secret
        ;;
    session)
        rotate_session_secret
        ;;
    db)
        rotate_db_password
        ;;
    redis)
        rotate_redis_password
        ;;
    all)
        rotate_all
        ;;
    validate)
        validate_secrets
        ;;
    check)
        check_strength
        ;;
    generate)
        generate_all_secrets
        ;;
    *)
        show_menu
        ;;
esac

echo ""
log_info "Done!"
echo ""
