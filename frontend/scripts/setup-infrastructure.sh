#!/bin/bash

# Script de setup initial de l'infrastructure production
# À exécuter sur le VPS OVH après première connexion

set -e

echo "========================================="
echo "  maicivy - Infrastructure Setup"
echo "========================================="

# Couleurs
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

# Variables
PROJECT_DIR="/opt/maicivy"
DOMAIN="${DOMAIN:-maicivy.com}"
EMAIL="${EMAIL:-admin@maicivy.com}"

# Fonction de log
log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Vérifier si root
if [ "$EUID" -ne 0 ]; then
    log_error "Please run as root (sudo)"
    exit 1
fi

log_info "Starting infrastructure setup..."

# 1. Update système
log_info "Updating system packages..."
apt update && apt upgrade -y

# 2. Install Docker
log_info "Installing Docker..."
if ! command -v docker &> /dev/null; then
    apt install -y docker.io docker-compose
    systemctl enable docker
    systemctl start docker
    log_info "Docker installed successfully"
else
    log_warn "Docker already installed"
fi

# 3. Install Nginx
log_info "Installing Nginx..."
if ! command -v nginx &> /dev/null; then
    apt install -y nginx
    systemctl enable nginx
    systemctl start nginx
    log_info "Nginx installed successfully"
else
    log_warn "Nginx already installed"
fi

# 4. Install Certbot
log_info "Installing Certbot..."
if ! command -v certbot &> /dev/null; then
    apt install -y certbot python3-certbot-nginx
    log_info "Certbot installed successfully"
else
    log_warn "Certbot already installed"
fi

# 5. Install utilities
log_info "Installing utilities..."
apt install -y curl wget git htop jq

# 6. Configure firewall UFW
log_info "Configuring firewall..."
if command -v ufw &> /dev/null; then
    ufw --force enable
    ufw allow 22/tcp   # SSH
    ufw allow 80/tcp   # HTTP
    ufw allow 443/tcp  # HTTPS
    ufw status
    log_info "Firewall configured"
else
    apt install -y ufw
    ufw --force enable
    ufw allow 22/tcp
    ufw allow 80/tcp
    ufw allow 443/tcp
fi

# 7. Create project directory
log_info "Creating project directory..."
mkdir -p $PROJECT_DIR
mkdir -p $PROJECT_DIR/backups
mkdir -p $PROJECT_DIR/backups/redis

# 8. Create maicivy user (optionnel)
if ! id -u maicivy &> /dev/null; then
    log_info "Creating maicivy user..."
    useradd -m -s /bin/bash maicivy
    usermod -aG docker maicivy
    usermod -aG sudo maicivy
    log_info "User maicivy created (set password manually: passwd maicivy)"
else
    log_warn "User maicivy already exists"
fi

# 9. Obtenir certificats SSL
log_info "Obtaining SSL certificates..."
if [ ! -d "/etc/letsencrypt/live/$DOMAIN" ]; then
    log_warn "Stopping Nginx temporarily for certificate generation..."
    systemctl stop nginx

    certbot certonly --standalone \
        -d $DOMAIN \
        -d www.$DOMAIN \
        -d analytics.$DOMAIN \
        --non-interactive \
        --agree-tos \
        --email $EMAIL

    if [ $? -eq 0 ]; then
        log_info "SSL certificates obtained successfully"
    else
        log_error "Failed to obtain SSL certificates"
        log_warn "You can retry manually later: certbot certonly --standalone -d $DOMAIN"
    fi

    systemctl start nginx
else
    log_warn "SSL certificates already exist for $DOMAIN"
fi

# 10. Setup cron jobs
log_info "Setting up cron jobs..."
(crontab -l 2>/dev/null || true; cat <<EOF
# maicivy - SSL renewal (daily at 3 AM)
0 3 * * * $PROJECT_DIR/scripts/renew-ssl.sh >> /var/log/ssl-renew.log 2>&1

# maicivy - PostgreSQL backup (daily at 2 AM)
0 2 * * * $PROJECT_DIR/scripts/backup-postgres.sh >> /var/log/maicivy-backup.log 2>&1

# maicivy - Redis backup (daily at 2:30 AM)
30 2 * * * $PROJECT_DIR/scripts/backup-redis.sh >> /var/log/maicivy-backup.log 2>&1
EOF
) | crontab -

log_info "Cron jobs configured"

# 11. Configure systemd service
if [ -f "$PROJECT_DIR/scripts/systemd/maicivy.service" ]; then
    log_info "Installing systemd service..."
    cp $PROJECT_DIR/scripts/systemd/maicivy.service /etc/systemd/system/
    systemctl daemon-reload
    systemctl enable maicivy
    log_info "Systemd service installed and enabled"
else
    log_warn "Systemd service file not found at $PROJECT_DIR/scripts/systemd/maicivy.service"
fi

# 12. Set permissions
log_info "Setting permissions..."
chmod +x $PROJECT_DIR/scripts/*.sh
chown -R root:docker $PROJECT_DIR

# 13. Create .env template if not exists
if [ ! -f "$PROJECT_DIR/.env" ]; then
    log_warn ".env file not found!"
    if [ -f "$PROJECT_DIR/.env.production.example" ]; then
        log_info "Copying .env.production.example to .env"
        cp $PROJECT_DIR/.env.production.example $PROJECT_DIR/.env
        log_warn "IMPORTANT: Edit $PROJECT_DIR/.env and configure all secrets!"
    fi
fi

# Summary
echo ""
echo "========================================="
echo "  Infrastructure Setup Complete!"
echo "========================================="
echo ""
log_info "Summary:"
echo "  - Docker installed and running"
echo "  - Nginx installed and running"
echo "  - Certbot installed"
echo "  - SSL certificates obtained (if domain configured)"
echo "  - Firewall configured (UFW)"
echo "  - Cron jobs configured"
echo "  - Systemd service installed"
echo ""
log_warn "Next steps:"
echo "  1. Edit .env file: nano $PROJECT_DIR/.env"
echo "  2. Configure secrets (passwords, API keys)"
echo "  3. Deploy application: $PROJECT_DIR/scripts/deploy.sh"
echo "  4. Verify health: $PROJECT_DIR/scripts/health-check.sh"
echo ""
echo "========================================="
