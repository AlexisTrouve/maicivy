#!/bin/bash
# Script pour réinitialiser les analytics avec des données de demo

set -e

echo "🧹 Nettoyage complet..."

# 1. Nettoyer Redis (visiteurs + cache analytics)
echo "  - Nettoyage Redis..."
docker exec maicivy-redis redis-cli DEL analytics:realtime:visitors > /dev/null
# Supprimer tous les caches d'agrégation
docker exec maicivy-redis redis-cli --scan --pattern "analytics:stats:*" | xargs -r -n 100 docker exec -i maicivy-redis redis-cli DEL > /dev/null 2>&1 || true
docker exec maicivy-redis redis-cli --scan --pattern "analytics:visitors:unique:*" | xargs -r -n 100 docker exec -i maicivy-redis redis-cli DEL > /dev/null 2>&1 || true
docker exec maicivy-redis redis-cli --scan --pattern "analytics:themes:*" | xargs -r -n 100 docker exec -i maicivy-redis redis-cli DEL > /dev/null 2>&1 || true

# 2. Trouver les credentials PostgreSQL
DB_USER=$(grep "^DB_USER=" /home/debian/maicivy/.env 2>/dev/null | cut -d '=' -f2 || echo "maicivy")
DB_NAME=$(grep "^DB_NAME=" /home/debian/maicivy/.env 2>/dev/null | cut -d '=' -f2 || echo "maicivy")

# 3. Créer un script SQL pour tout reset
cat > /tmp/reset_analytics.sql <<'EOF'
-- Nettoyer TOUTES les données analytics (en respectant les foreign keys)
DELETE FROM generated_letters WHERE deleted_at IS NULL;
DELETE FROM analytics_events WHERE deleted_at IS NULL;
DELETE FROM visitors WHERE deleted_at IS NULL;

-- Créer 30 visiteurs fictifs
DO $$
DECLARE
  i INTEGER;
  visitor_uuid UUID;
  session_string TEXT;
  now_timestamp TIMESTAMP := NOW();
BEGIN
  FOR i IN 1..30 LOOP
    visitor_uuid := gen_random_uuid();
    session_string := 'demo-' || visitor_uuid::TEXT;

    INSERT INTO visitors (
      id,
      created_at,
      updated_at,
      session_id,
      ip_hash,
      user_agent,
      visit_count,
      profile_detected,
      first_visit,
      last_visit
    ) VALUES (
      visitor_uuid,
      now_timestamp - (random() * INTERVAL '4 minutes'),
      now_timestamp - (random() * INTERVAL '4 minutes'),
      session_string,
      encode(sha256(('demo-ip-' || i::TEXT)::bytea), 'hex'),
      'Mozilla/5.0 (Demo Visitor ' || i || ')',
      1,
      CASE
        WHEN i % 3 = 0 THEN 'professional'
        WHEN i % 5 = 0 THEN 'recruiter'
        ELSE ''
      END,
      now_timestamp - (random() * INTERVAL '4 minutes'),
      now_timestamp - (random() * INTERVAL '4 minutes')
    );
  END LOOP;
END $$;

-- Récupérer les UUIDs créés pour les utiliser dans les page views
CREATE TEMP TABLE demo_visitors AS
SELECT id FROM visitors WHERE session_id LIKE 'demo-%';

-- Créer 57 événements page_view en utilisant les visiteurs créés
DO $$
DECLARE
  i INTEGER;
  visitor_uuid UUID;
  event_timestamp TIMESTAMP;
  pages TEXT[] := ARRAY['/fr/cv', '/fr/letters', '/fr/analytics', '/fr/architecture', '/fr/', '/en/cv', '/en/letters'];
  random_page TEXT;
  visitor_ids UUID[];
BEGIN
  -- Récupérer tous les visitor_ids dans un array
  SELECT ARRAY_AGG(id) INTO visitor_ids FROM demo_visitors;

  FOR i IN 1..57 LOOP
    -- Choisir un visiteur aléatoire parmi ceux créés
    visitor_uuid := visitor_ids[1 + floor(random() * array_length(visitor_ids, 1))::int];

    -- Timestamp aléatoire dans les dernières 24 heures
    event_timestamp := NOW() - (random() * INTERVAL '24 hours');

    -- Choisir une page aléatoire
    random_page := pages[1 + floor(random() * array_length(pages, 1))::int];

    -- Insérer l'événement
    INSERT INTO analytics_events (
      id,
      created_at,
      updated_at,
      visitor_id,
      event_type,
      event_data,
      page_url,
      referrer,
      session_duration
    ) VALUES (
      gen_random_uuid(),
      event_timestamp,
      event_timestamp,
      visitor_uuid,
      'page_view',
      jsonb_build_object('path', random_page, 'method', 'GET'),
      random_page,
      CASE WHEN random() > 0.7 THEN 'https://www.google.com/' ELSE '' END,
      floor(random() * 300)::int
    );
  END LOOP;
END $$;

-- Retourner les UUIDs des visiteurs créés (pour Redis)
SELECT id::TEXT FROM demo_visitors;
EOF

echo "  - Nettoyage PostgreSQL et création des données..."
VISITOR_UUIDS=$(docker exec -i maicivy-postgres psql -U "$DB_USER" -d "$DB_NAME" -t -A < /tmp/reset_analytics.sql | tail -30)

echo ""
echo "👥 Ajout des 30 visiteurs dans Redis..."

NOW=$(date +%s)
COUNT=0

while IFS= read -r VISITOR_ID; do
  if [ ! -z "$VISITOR_ID" ]; then
    # Timestamp aléatoire entre maintenant et il y a 4 minutes
    OFFSET=$((RANDOM % 240))
    TIMESTAMP=$((NOW - OFFSET))

    docker exec maicivy-redis redis-cli ZADD analytics:realtime:visitors $TIMESTAMP "$VISITOR_ID" > /dev/null
    COUNT=$((COUNT + 1))
  fi
done <<< "$VISITOR_UUIDS"

echo "✅ $COUNT visiteurs actifs créés dans Redis"

# Cleanup
rm /tmp/reset_analytics.sql

echo ""
echo "📊 Vérification..."
VISITOR_COUNT=$(docker exec maicivy-redis redis-cli ZCARD analytics:realtime:visitors)
echo "  - Visiteurs actifs dans Redis: $VISITOR_COUNT"

# Vérifier dans PostgreSQL
echo ""
echo "  - Statistiques PostgreSQL:"
docker exec -i maicivy-postgres psql -U "$DB_USER" -d "$DB_NAME" -c "
SELECT
  (SELECT COUNT(*) FROM visitors WHERE session_id LIKE 'demo-%') as demo_visitors,
  (SELECT COUNT(*) FROM analytics_events WHERE event_type = 'page_view' AND deleted_at IS NULL) as total_pageviews,
  (SELECT COUNT(*) FROM analytics_events WHERE event_type = 'page_view' AND created_at >= CURRENT_DATE AND deleted_at IS NULL) as today_pageviews;
"

echo ""
echo "✅ TERMINÉ!"
echo ""
echo "🔍 Vérification via API:"
echo "curl http://localhost:8081/api/v1/analytics/stats?period=day"
