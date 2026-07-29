#!/bin/bash
# ============================================================================
# Script to apply English translations for experiences
# ============================================================================

set -e  # Exit on error

echo "🌍 Applying English translations for experiences..."
echo ""

# Check if Docker container is running
if ! docker ps | grep -q maicivy-postgres; then
    echo "❌ Error: PostgreSQL container 'maicivy-postgres' is not running"
    echo "   Start it with: docker compose up -d postgres"
    exit 1
fi

# Apply migration
echo "📝 Executing SQL migration..."
docker exec -i maicivy-postgres psql -U maicivy -d maicivy < "$(dirname "$0")/translate_experiences_en.sql"

echo ""
echo "✅ English translations applied successfully!"
echo ""
echo "📊 Verification:"
docker exec -i maicivy-postgres psql -U maicivy -d maicivy -c "
SELECT
    company,
    LEFT(title, 40) as title,
    CASE
        WHEN title_en IS NOT NULL AND title_en != '' THEN '✓ YES'
        ELSE '✗ NO'
    END as has_english
FROM experiences
ORDER BY start_date DESC;
"

echo ""
echo "🔄 Invalidating CV cache..."
# Optional: Clear Redis cache for CV to pick up new translations
docker exec -i maicivy-redis redis-cli KEYS "cv:theme:*" | xargs -r docker exec -i maicivy-redis redis-cli DEL 2>/dev/null || true

echo "✅ Done! Test at: https://maicivy.etheryale.com/en/cv"
