#!/bin/bash

# ============================================
# maicivy - Setup GitHub Secrets Helper
# ============================================
# Usage: ./scripts/setup-secrets.sh
# ============================================

set -e  # Exit on error

# Colors
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# ============================================
# Functions
# ============================================

log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

print_header() {
    echo ""
    echo "============================================"
    echo "  maicivy - GitHub Secrets Setup Helper"
    echo "============================================"
    echo ""
}

check_gh_cli() {
    if ! command -v gh &> /dev/null; then
        log_error "GitHub CLI (gh) is not installed!"
        echo ""
        echo "Install it from: https://cli.github.com/"
        echo ""
        echo "macOS:   brew install gh"
        echo "Ubuntu:  sudo apt install gh"
        echo "Windows: winget install GitHub.cli"
        exit 1
    fi

    # Check if authenticated
    if ! gh auth status &> /dev/null; then
        log_error "GitHub CLI is not authenticated!"
        echo ""
        echo "Run: gh auth login"
        exit 1
    fi

    log_success "GitHub CLI is installed and authenticated"
}

list_required_secrets() {
    log_info "Required GitHub Secrets for maicivy:"
    echo ""
    echo "1. VPS_HOST                - VPS IP or domain"
    echo "2. VPS_USER                - SSH user (e.g., deploy)"
    echo "3. VPS_SSH_KEY             - Private SSH key for deployment"
    echo "4. CLAUDE_API_KEY          - Anthropic Claude API key"
    echo "5. OPENAI_API_KEY          - OpenAI API key"
    echo "6. JWT_SECRET              - JWT secret for authentication"
    echo "7. POSTGRES_PASSWORD       - PostgreSQL database password"
    echo "8. REDIS_PASSWORD          - Redis password"
    echo "9. GRAFANA_PASSWORD        - Grafana admin password"
    echo "10. DISCORD_WEBHOOK_ID     - Discord webhook ID (optional)"
    echo "11. DISCORD_WEBHOOK_TOKEN  - Discord webhook token (optional)"
    echo ""
}

generate_random_secret() {
    openssl rand -hex 32
}

set_secret() {
    local secret_name=$1
    local secret_value=$2

    echo "$secret_value" | gh secret set "$secret_name"
    log_success "Secret $secret_name set successfully"
}

prompt_secret() {
    local secret_name=$1
    local description=$2
    local allow_empty=${3:-false}

    echo ""
    log_info "Setting: $secret_name"
    echo "Description: $description"
    echo ""

    local secret_value=""

    # Special handling for SSH key (multiline)
    if [[ "$secret_name" == *"SSH_KEY"* ]]; then
        echo "Paste the SSH private key (press Ctrl+D when done):"
        secret_value=$(cat)
    else
        read -p "Enter value (or press Enter to skip): " secret_value
    fi

    if [ -z "$secret_value" ] && [ "$allow_empty" = false ]; then
        log_warning "Skipped $secret_name"
        return 1
    fi

    set_secret "$secret_name" "$secret_value"
    return 0
}

interactive_setup() {
    log_info "Starting interactive secrets setup..."
    echo ""
    log_warning "This will set secrets for the current repository."
    echo ""
    read -p "Continue? (yes/no): " confirm

    if [ "$confirm" != "yes" ]; then
        log_info "Setup cancelled"
        exit 0
    fi

    # VPS Configuration
    echo ""
    log_info "=== VPS Configuration ==="
    prompt_secret "VPS_HOST" "VPS IP address or domain (e.g., 192.168.1.100 or maicivy.example.com)"
    prompt_secret "VPS_USER" "SSH user for deployment (e.g., deploy)"
    prompt_secret "VPS_SSH_KEY" "SSH private key for deployment"

    # Application Secrets
    echo ""
    log_info "=== Application Secrets ==="
    prompt_secret "CLAUDE_API_KEY" "Anthropic Claude API key"
    prompt_secret "OPENAI_API_KEY" "OpenAI API key (optional)" true

    # Generate JWT secret
    echo ""
    log_info "Generating JWT secret..."
    local jwt_secret=$(generate_random_secret)
    set_secret "JWT_SECRET" "$jwt_secret"

    # Database Passwords
    echo ""
    log_info "=== Database Configuration ==="
    echo ""
    log_info "Generating PostgreSQL password..."
    local postgres_password=$(generate_random_secret)
    set_secret "POSTGRES_PASSWORD" "$postgres_password"

    echo ""
    log_info "Generating Redis password..."
    local redis_password=$(generate_random_secret)
    set_secret "REDIS_PASSWORD" "$redis_password"

    # Monitoring
    echo ""
    log_info "=== Monitoring Configuration ==="
    prompt_secret "GRAFANA_PASSWORD" "Grafana admin password"

    # Notifications (optional)
    echo ""
    log_info "=== Notifications (Optional) ==="
    prompt_secret "DISCORD_WEBHOOK_ID" "Discord webhook ID (optional)" true
    prompt_secret "DISCORD_WEBHOOK_TOKEN" "Discord webhook token (optional)" true

    echo ""
    log_success "✅ All secrets configured!"
}

list_current_secrets() {
    log_info "Current GitHub Secrets:"
    echo ""
    gh secret list
    echo ""
}

verify_secrets() {
    log_info "Verifying required secrets..."
    echo ""

    local missing_secrets=()
    local required=(
        "VPS_HOST"
        "VPS_USER"
        "VPS_SSH_KEY"
        "CLAUDE_API_KEY"
        "JWT_SECRET"
        "POSTGRES_PASSWORD"
        "REDIS_PASSWORD"
        "GRAFANA_PASSWORD"
    )

    for secret in "${required[@]}"; do
        if gh secret list | grep -q "^$secret"; then
            log_success "$secret is set"
        else
            log_error "$secret is missing"
            missing_secrets+=("$secret")
        fi
    done

    echo ""

    if [ ${#missing_secrets[@]} -eq 0 ]; then
        log_success "✅ All required secrets are configured!"
        return 0
    else
        log_error "Missing secrets:"
        for secret in "${missing_secrets[@]}"; do
            echo "  - $secret"
        done
        return 1
    fi
}

show_ssh_key_instructions() {
    echo ""
    log_info "=== SSH Key Setup Instructions ==="
    echo ""
    echo "1. Generate SSH key pair:"
    echo "   ssh-keygen -t ed25519 -C 'github-actions-deploy' -f ~/.ssh/maicivy_deploy"
    echo ""
    echo "2. Copy public key to VPS:"
    echo "   ssh-copy-id -i ~/.ssh/maicivy_deploy.pub deploy@your-vps-ip"
    echo ""
    echo "3. Test connection:"
    echo "   ssh -i ~/.ssh/maicivy_deploy deploy@your-vps-ip"
    echo ""
    echo "4. Add private key to GitHub Secrets:"
    echo "   cat ~/.ssh/maicivy_deploy"
    echo "   Then paste the entire output (including BEGIN/END lines) as VPS_SSH_KEY"
    echo ""
}

# ============================================
# Main Menu
# ============================================

show_menu() {
    echo ""
    echo "What would you like to do?"
    echo ""
    echo "1. Interactive secrets setup"
    echo "2. List required secrets"
    echo "3. List current secrets"
    echo "4. Verify secrets configuration"
    echo "5. Show SSH key setup instructions"
    echo "6. Exit"
    echo ""
    read -p "Enter your choice (1-6): " choice

    case $choice in
        1)
            interactive_setup
            ;;
        2)
            list_required_secrets
            ;;
        3)
            list_current_secrets
            ;;
        4)
            verify_secrets
            ;;
        5)
            show_ssh_key_instructions
            ;;
        6)
            log_info "Exiting"
            exit 0
            ;;
        *)
            log_error "Invalid choice"
            exit 1
            ;;
    esac
}

# ============================================
# Main
# ============================================

main() {
    print_header
    check_gh_cli
    show_menu
}

# ============================================
# Execute
# ============================================
main "$@"
