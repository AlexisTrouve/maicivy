-- Performance optimization indexes for maicivy database
-- Created: 2025-12-08
-- Purpose: Speed up frequent queries by adding strategic indexes

-- ==============================================================================
-- EXPERIENCES TABLE INDEXES
-- ==============================================================================

-- Index for filtering by category (backend, frontend, etc.)
CREATE INDEX IF NOT EXISTS idx_experiences_category ON experiences(category);

-- GIN index for searching in tags array
CREATE INDEX IF NOT EXISTS idx_experiences_tags ON experiences USING GIN(tags);

-- Index for date range queries (sorting by dates)
CREATE INDEX IF NOT EXISTS idx_experiences_dates ON experiences(start_date DESC, end_date DESC);

-- Index for active experiences (end_date IS NULL means currently working)
CREATE INDEX IF NOT EXISTS idx_experiences_active ON experiences(end_date) WHERE end_date IS NULL;

-- Composite index for category + dates (most common query pattern)
CREATE INDEX IF NOT EXISTS idx_experiences_category_dates ON experiences(category, start_date DESC);

-- ==============================================================================
-- SKILLS TABLE INDEXES
-- ==============================================================================

-- Index for filtering by category
CREATE INDEX IF NOT EXISTS idx_skills_category ON skills(category);

-- GIN index for tags array
CREATE INDEX IF NOT EXISTS idx_skills_tags ON skills USING GIN(tags);

-- Index for filtering by skill level
CREATE INDEX IF NOT EXISTS idx_skills_level ON skills(level);

-- Index for years of experience (for ranking)
CREATE INDEX IF NOT EXISTS idx_skills_years ON skills(years_experience DESC);

-- Composite index for category + level
CREATE INDEX IF NOT EXISTS idx_skills_category_level ON skills(category, level DESC);

-- ==============================================================================
-- PROJECTS TABLE INDEXES
-- ==============================================================================

-- Index for filtering by category
CREATE INDEX IF NOT EXISTS idx_projects_category ON projects(category);

-- GIN index for tags array
CREATE INDEX IF NOT EXISTS idx_projects_tags ON projects USING GIN(tags);

-- Index for featured projects
CREATE INDEX IF NOT EXISTS idx_projects_featured ON projects(featured) WHERE featured = true;

-- GIN index for technologies array
CREATE INDEX IF NOT EXISTS idx_projects_technologies ON projects USING GIN(technologies);

-- Index for created date (newest first)
CREATE INDEX IF NOT EXISTS idx_projects_created ON projects(created_at DESC);

-- ==============================================================================
-- GENERATED_LETTERS TABLE INDEXES
-- ==============================================================================

-- Index for looking up letters by company name
CREATE INDEX IF NOT EXISTS idx_letters_company ON generated_letters(company_name);

-- Index for visitor's letter history
CREATE INDEX IF NOT EXISTS idx_letters_visitor ON generated_letters(visitor_id);

-- Index for creation date (for analytics)
CREATE INDEX IF NOT EXISTS idx_letters_created_at ON generated_letters(created_at DESC);

-- Index for letter type
CREATE INDEX IF NOT EXISTS idx_letters_type ON generated_letters(letter_type);

-- Composite index for visitor + company lookup (check if already generated)
CREATE INDEX IF NOT EXISTS idx_letters_lookup ON generated_letters(visitor_id, company_name, letter_type);

-- Composite index for time-based analytics
CREATE INDEX IF NOT EXISTS idx_letters_date_type ON generated_letters(created_at DESC, letter_type);

-- ==============================================================================
-- ANALYTICS_EVENTS TABLE INDEXES
-- ==============================================================================

-- Index for visitor's event history
CREATE INDEX IF NOT EXISTS idx_analytics_events_visitor ON analytics_events(visitor_id);

-- Index for filtering by event type
CREATE INDEX IF NOT EXISTS idx_analytics_events_type ON analytics_events(event_type);

-- Index for time-based queries
CREATE INDEX IF NOT EXISTS idx_analytics_events_created ON analytics_events(created_at DESC);

-- Composite index for type + time range queries (most common pattern)
CREATE INDEX IF NOT EXISTS idx_analytics_timerange ON analytics_events(event_type, created_at DESC);

-- Composite index for visitor + time
CREATE INDEX IF NOT EXISTS idx_analytics_visitor_time ON analytics_events(visitor_id, created_at DESC);

-- Partial index for specific event types (for heatmap)
CREATE INDEX IF NOT EXISTS idx_analytics_clicks ON analytics_events(event_type, event_data)
WHERE event_type = 'click';

-- ==============================================================================
-- VISITORS TABLE INDEXES
-- ==============================================================================

-- Index for session lookup (most frequent operation)
CREATE INDEX IF NOT EXISTS idx_visitors_session ON visitors(session_id);

-- Index for profile detection filtering
CREATE INDEX IF NOT EXISTS idx_visitors_profile ON visitors(profile_detected);

-- Index for last visit date (for cleanup/analytics)
CREATE INDEX IF NOT EXISTS idx_visitors_last_visit ON visitors(last_visit DESC);

-- Index for first visit date
CREATE INDEX IF NOT EXISTS idx_visitors_first_visit ON visitors(first_visit DESC);

-- Index for visit count (for access gate logic)
CREATE INDEX IF NOT EXISTS idx_visitors_visit_count ON visitors(visit_count);

-- Composite index for profile + last visit
CREATE INDEX IF NOT EXISTS idx_visitors_profile_visit ON visitors(profile_detected, last_visit DESC);

-- Partial index for high-value profiles (recruiters, etc.)
CREATE INDEX IF NOT EXISTS idx_visitors_high_value ON visitors(profile_detected, visit_count)
WHERE profile_detected IS NOT NULL;

-- ==============================================================================
-- STATISTICS & VERIFICATION
-- ==============================================================================

-- Query to check index sizes
-- SELECT
--     schemaname,
--     tablename,
--     indexname,
--     pg_size_pretty(pg_relation_size(indexrelid)) AS size
-- FROM pg_stat_user_indexes
-- ORDER BY pg_relation_size(indexrelid) DESC;

-- Query to check index usage
-- SELECT
--     schemaname,
--     tablename,
--     indexname,
--     idx_scan AS times_used,
--     idx_tup_read AS tuples_read,
--     idx_tup_fetch AS tuples_fetched
-- FROM pg_stat_user_indexes
-- WHERE schemaname = 'public'
-- ORDER BY idx_scan DESC;

-- Query to find unused indexes (idx_scan = 0)
-- SELECT
--     schemaname,
--     tablename,
--     indexname,
--     pg_size_pretty(pg_relation_size(indexrelid)) AS size
-- FROM pg_stat_user_indexes
-- WHERE idx_scan = 0
--   AND indexrelid NOT IN (
--       SELECT indexrelid FROM pg_index WHERE indisprimary OR indisunique
--   )
-- ORDER BY pg_relation_size(indexrelid) DESC;

-- ==============================================================================
-- NOTES
-- ==============================================================================

-- 1. GIN indexes are used for array columns (tags, technologies) for fast containment queries
--    Example: WHERE tags @> ARRAY['golang', 'backend']
--
-- 2. Partial indexes (WHERE clause) are used when we frequently query a subset of data
--    This reduces index size and improves performance
--
-- 3. Composite indexes follow the rule: equality filters first, then range/sort filters
--    Example: (category, created_at DESC) is optimized for WHERE category = 'X' ORDER BY created_at
--
-- 4. DESC in index definition helps with ORDER BY DESC queries (avoid sorting)
--
-- 5. All indexes use IF NOT EXISTS to make migration idempotent (can run multiple times safely)
--
-- 6. Monitor index usage with pg_stat_user_indexes to identify unused indexes for removal
--
-- 7. Index maintenance: PostgreSQL auto-vacuums but can run REINDEX manually if needed:
--    REINDEX TABLE experiences;
--
-- 8. For large tables (>100K rows), consider CONCURRENTLY option to avoid locking:
--    CREATE INDEX CONCURRENTLY idx_name ON table(column);

-- ==============================================================================
-- PERFORMANCE IMPACT
-- ==============================================================================

-- Expected improvements:
-- - CV queries (filtered by theme/category): 10-50x faster
-- - Letter lookup (visitor + company): 20-100x faster
-- - Analytics time-range queries: 5-20x faster
-- - Tag searches: 50-200x faster (GIN indexes)
-- - Session lookups: 10-30x faster
--
-- Trade-offs:
-- - INSERT/UPDATE/DELETE slightly slower (5-10%)
-- - Additional storage: ~10-30% of table size per index
-- - Maintenance overhead: VACUUM, REINDEX occasionally needed
