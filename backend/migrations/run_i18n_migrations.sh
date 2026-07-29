#!/bin/bash
# Script to run i18n migrations on maicivy database
# Usage: ./run_i18n_migrations.sh
# Author: Alexi
# Date: 2026-01-12

set -e  # Exit on error

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Configuration (can be overridden with env vars)
DB_HOST="${DB_HOST:-localhost}"
DB_PORT="${DB_PORT:-5432}"
DB_NAME="${DB_NAME:-maicivydb}"
DB_USER="${DB_USER:-maicivyuser}"
DB_PASSWORD="${DB_PASSWORD:-maicivypass}"

# Migration files
MIGRATION_DIR="$(dirname "$0")"
MIGRATION_I18N_FIELDS="${MIGRATION_DIR}/add_i18n_fields.sql"
MIGRATION_I18N_SEED="${MIGRATION_DIR}/seed_data_i18n.sql"

echo -e "${YELLOW}========================================${NC}"
echo -e "${YELLOW}  maicivy i18n Migrations Runner${NC}"
echo -e "${YELLOW}========================================${NC}"
echo ""

# Check if psql is available
if ! command -v psql &> /dev/null; then
    echo -e "${RED}ERROR: psql command not found. Please install PostgreSQL client.${NC}"
    exit 1
fi

# Check if migration files exist
if [ ! -f "$MIGRATION_I18N_FIELDS" ]; then
    echo -e "${RED}ERROR: Migration file not found: $MIGRATION_I18N_FIELDS${NC}"
    exit 1
fi

if [ ! -f "$MIGRATION_I18N_SEED" ]; then
    echo -e "${RED}ERROR: Seed file not found: $MIGRATION_I18N_SEED${NC}"
    exit 1
fi

echo -e "${GREEN}✓${NC} Found migration files"
echo ""

# Function to run SQL file
run_sql_file() {
    local file=$1
    local description=$2

    echo -e "${YELLOW}Running: ${description}${NC}"
    echo "  File: $(basename $file)"

    PGPASSWORD="$DB_PASSWORD" psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -f "$file"

    if [ $? -eq 0 ]; then
        echo -e "${GREEN}✓${NC} Success: $description"
        echo ""
        return 0
    else
        echo -e "${RED}✗${NC} Failed: $description"
        echo ""
        return 1
    fi
}

# Step 1: Add i18n fields
echo -e "${YELLOW}Step 1/2: Adding i18n fields to database${NC}"
run_sql_file "$MIGRATION_I18N_FIELDS" "Add i18n columns"

# Step 2: Populate with English translations
echo -e "${YELLOW}Step 2/2: Populating English translations${NC}"
run_sql_file "$MIGRATION_I18N_SEED" "Seed English translations"

# Verify translations
echo -e "${YELLOW}Verifying translations...${NC}"

VERIFY_SQL="
SELECT
    'experiences' as table_name,
    COUNT(*) as total_records,
    COUNT(title_en) as translated_records,
    ROUND(100.0 * COUNT(title_en) / COUNT(*), 2) as percentage
FROM experiences
UNION ALL
SELECT
    'skills' as table_name,
    COUNT(*) as total_records,
    COUNT(name_en) as translated_records,
    ROUND(100.0 * COUNT(name_en) / COUNT(*), 2) as percentage
FROM skills
UNION ALL
SELECT
    'projects' as table_name,
    COUNT(*) as total_records,
    COUNT(title_en) as translated_records,
    ROUND(100.0 * COUNT(title_en) / COUNT(*), 2) as percentage
FROM projects;
"

PGPASSWORD="$DB_PASSWORD" psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -c "$VERIFY_SQL"

echo ""
echo -e "${GREEN}========================================${NC}"
echo -e "${GREEN}  Migrations completed successfully!${NC}"
echo -e "${GREEN}========================================${NC}"
echo ""
echo -e "Next steps:"
echo -e "  1. Restart your backend: ${YELLOW}docker-compose restart backend${NC}"
echo -e "  2. Clear Redis cache: ${YELLOW}docker exec maicivy-redis-1 redis-cli FLUSHDB${NC}"
echo -e "  3. Test API: ${YELLOW}curl http://localhost:8080/api/v1/cv?lang=en${NC}"
echo ""
echo -e "${GREEN}Documentation:${NC} /home/debian/maicivy/I18N_IMPLEMENTATION_RECAP.md"
echo ""
