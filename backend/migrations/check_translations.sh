#!/bin/bash
# ============================================================================
# Check translation status for experiences, projects, and skills
# ============================================================================

set -e

echo "🌍 Translation Status Report"
echo "============================"
echo ""

# Check if Docker container is running
if ! docker ps | grep -q maicivy-postgres; then
    echo "❌ Error: PostgreSQL container 'maicivy-postgres' is not running"
    exit 1
fi

# Experiences
echo "📋 EXPERIENCES"
echo "-------------"
docker exec -i maicivy-postgres psql -U maicivy -d maicivy -c "
SELECT
    COUNT(*) as total,
    COUNT(CASE WHEN title_en IS NOT NULL AND title_en != '' THEN 1 END) as translated,
    COUNT(CASE WHEN title_en IS NULL OR title_en = '' THEN 1 END) as missing
FROM experiences;
" -t

echo ""
echo "Details:"
docker exec -i maicivy-postgres psql -U maicivy -d maicivy -c "
SELECT
    company,
    LEFT(title, 50) as title,
    CASE
        WHEN title_en IS NOT NULL AND title_en != '' THEN '✓ YES'
        ELSE '✗ NO'
    END as english
FROM experiences
ORDER BY start_date DESC;
"

echo ""
echo "📊 SKILLS"
echo "--------"
docker exec -i maicivy-postgres psql -U maicivy -d maicivy -c "
SELECT
    COUNT(*) as total,
    COUNT(CASE WHEN name_en IS NOT NULL AND name_en != '' THEN 1 END) as translated,
    COUNT(CASE WHEN name_en IS NULL OR name_en = '' THEN 1 END) as missing
FROM skills;
" -t

echo ""
echo "📁 PROJECTS"
echo "----------"
docker exec -i maicivy-postgres psql -U maicivy -d maicivy -c "
SELECT
    COUNT(*) as total,
    COUNT(CASE WHEN title_en IS NOT NULL AND title_en != '' THEN 1 END) as translated,
    COUNT(CASE WHEN title_en IS NULL OR title_en = '' THEN 1 END) as missing
FROM projects;
" -t

echo ""
echo "🔍 MISSING TRANSLATIONS"
echo "----------------------"

# Experiences missing translations
MISSING_EXP=$(docker exec -i maicivy-postgres psql -U maicivy -d maicivy -t -c "
SELECT COUNT(*) FROM experiences WHERE title_en IS NULL OR title_en = '';
")

if [ "$MISSING_EXP" -gt 0 ]; then
    echo "⚠️  $MISSING_EXP experience(s) need English translation:"
    docker exec -i maicivy-postgres psql -U maicivy -d maicivy -c "
    SELECT
        id,
        company,
        LEFT(title, 60) as title
    FROM experiences
    WHERE title_en IS NULL OR title_en = ''
    ORDER BY start_date DESC;
    "
else
    echo "✅ All experiences have English translations"
fi

echo ""
echo "💡 TIP: To add translations, edit and run:"
echo "   backend/migrations/translate_experiences_en.sql"
