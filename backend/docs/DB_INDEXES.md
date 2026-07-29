# Database Indexes - Performance Optimization

This document explains the indexes created for the maicivy database and their justification.

## Overview

Indexes are critical for database performance. They allow PostgreSQL to quickly locate rows without scanning the entire table. However, they come with trade-offs:

**Benefits:**
- Dramatically faster SELECT queries (10-200x improvement)
- Efficient sorting and filtering
- Fast JOIN operations

**Costs:**
- Slower INSERT/UPDATE/DELETE operations (5-10% overhead)
- Additional storage space (~10-30% per index)
- Maintenance overhead (VACUUM, REINDEX)

## Index Strategy

Our indexing strategy follows these principles:

1. **Index frequently queried columns** - WHERE, JOIN, ORDER BY clauses
2. **Use composite indexes** for multi-column queries
3. **GIN indexes for array columns** - Fast containment queries
4. **Partial indexes** for subset queries - Smaller, faster indexes
5. **Avoid over-indexing** - Monitor usage, drop unused indexes

## Indexes by Table

### Experiences Table

| Index Name | Columns | Type | Justification |
|------------|---------|------|---------------|
| `idx_experiences_category` | category | B-tree | Filter CV by theme (backend, frontend, etc.) |
| `idx_experiences_tags` | tags | GIN | Search experiences by technology tags |
| `idx_experiences_dates` | start_date DESC, end_date DESC | B-tree | Sort by date, timeline display |
| `idx_experiences_active` | end_date (WHERE NULL) | Partial | Find current positions quickly |
| `idx_experiences_category_dates` | category, start_date DESC | Composite | Most common query: theme + sorted by date |

**Query patterns optimized:**
```sql
-- Theme filtering
SELECT * FROM experiences WHERE category = 'backend' ORDER BY start_date DESC;

-- Tag search
SELECT * FROM experiences WHERE tags @> ARRAY['golang', 'docker'];

-- Current positions
SELECT * FROM experiences WHERE end_date IS NULL;
```

**Expected improvement:** 10-50x faster for CV queries

---

### Skills Table

| Index Name | Columns | Type | Justification |
|------------|---------|------|---------------|
| `idx_skills_category` | category | B-tree | Group skills by type (languages, tools, etc.) |
| `idx_skills_tags` | tags | GIN | Search skills by related technologies |
| `idx_skills_level` | level | B-tree | Filter by proficiency level |
| `idx_skills_years` | years_experience DESC | B-tree | Sort by experience (expert skills first) |
| `idx_skills_category_level` | category, level DESC | Composite | Category + expertise ranking |

**Query patterns optimized:**
```sql
-- Category filtering
SELECT * FROM skills WHERE category = 'languages' ORDER BY level DESC;

-- Experience-based sorting
SELECT * FROM skills ORDER BY years_experience DESC LIMIT 10;
```

**Expected improvement:** 5-20x faster for skill queries

---

### Projects Table

| Index Name | Columns | Type | Justification |
|------------|---------|------|---------------|
| `idx_projects_category` | category | B-tree | Filter projects by type (web, mobile, etc.) |
| `idx_projects_tags` | tags | GIN | Search by project characteristics |
| `idx_projects_featured` | featured (WHERE true) | Partial | Quick access to featured projects |
| `idx_projects_technologies` | technologies | GIN | Search by tech stack |
| `idx_projects_created` | created_at DESC | B-tree | Newest projects first |

**Query patterns optimized:**
```sql
-- Featured projects
SELECT * FROM projects WHERE featured = true ORDER BY created_at DESC;

-- Technology search
SELECT * FROM projects WHERE technologies @> ARRAY['react', 'typescript'];
```

**Expected improvement:** 10-30x faster for project queries

---

### Generated Letters Table

| Index Name | Columns | Type | Justification |
|------------|---------|------|---------------|
| `idx_letters_company` | company_name | B-tree | Search letters by company |
| `idx_letters_visitor` | visitor_id | B-tree | Visitor's letter history |
| `idx_letters_created_at` | created_at DESC | B-tree | Time-based analytics |
| `idx_letters_type` | letter_type | B-tree | Filter by motivation/anti-motivation |
| `idx_letters_lookup` | visitor_id, company_name, letter_type | Composite | Check if letter already exists (avoid duplicates) |
| `idx_letters_date_type` | created_at DESC, letter_type | Composite | Analytics: letters per day by type |

**Query patterns optimized:**
```sql
-- Check if letter already generated
SELECT * FROM generated_letters
WHERE visitor_id = 'xxx' AND company_name = 'Google' AND letter_type = 'motivation';

-- Visitor's history
SELECT * FROM generated_letters WHERE visitor_id = 'xxx' ORDER BY created_at DESC;

-- Daily stats
SELECT letter_type, COUNT(*) FROM generated_letters
WHERE created_at >= '2025-01-01'
GROUP BY letter_type;
```

**Expected improvement:** 20-100x faster for letter lookups (critical for deduplication)

---

### Analytics Events Table

| Index Name | Columns | Type | Justification |
|------------|---------|------|---------------|
| `idx_analytics_events_visitor` | visitor_id | B-tree | Visitor's activity timeline |
| `idx_analytics_events_type` | event_type | B-tree | Filter by event type (click, scroll, etc.) |
| `idx_analytics_events_created` | created_at DESC | B-tree | Time-based queries |
| `idx_analytics_timerange` | event_type, created_at DESC | Composite | Most common: event type + time range |
| `idx_analytics_visitor_time` | visitor_id, created_at DESC | Composite | Visitor activity timeline |
| `idx_analytics_clicks` | event_type, event_data (WHERE click) | Partial | Heatmap generation (only click events) |

**Query patterns optimized:**
```sql
-- Event type stats over time
SELECT DATE(created_at), COUNT(*) FROM analytics_events
WHERE event_type = 'page_view' AND created_at >= NOW() - INTERVAL '7 days'
GROUP BY DATE(created_at);

-- Heatmap data (clicks only)
SELECT event_data FROM analytics_events WHERE event_type = 'click';
```

**Expected improvement:** 5-20x faster for analytics queries

---

### Visitors Table

| Index Name | Columns | Type | Justification |
|------------|---------|------|---------------|
| `idx_visitors_session` | session_id | B-tree | **Most frequent query** - session lookup |
| `idx_visitors_profile` | profile_detected | B-tree | Filter by profile type (recruiter, etc.) |
| `idx_visitors_last_visit` | last_visit DESC | B-tree | Recent visitors, cleanup old sessions |
| `idx_visitors_first_visit` | first_visit DESC | B-tree | New vs returning analysis |
| `idx_visitors_visit_count` | visit_count | B-tree | Access gate logic (>= 3 visits) |
| `idx_visitors_profile_visit` | profile_detected, last_visit DESC | Composite | High-value visitors (recruiters) activity |
| `idx_visitors_high_value` | profile_detected, visit_count (WHERE NOT NULL) | Partial | Fast access to detected profiles only |

**Query patterns optimized:**
```sql
-- Session lookup (every request!)
SELECT * FROM visitors WHERE session_id = 'xxx';

-- Access gate check
SELECT visit_count FROM visitors WHERE session_id = 'xxx';

-- High-value visitors
SELECT * FROM visitors
WHERE profile_detected IN ('recruiter', 'tech_lead')
ORDER BY last_visit DESC;
```

**Expected improvement:** 10-30x faster for session lookups (critical for every request)

---

## Index Types Explained

### B-tree Indexes (Default)
- **Use case:** Equality and range queries
- **Examples:** `WHERE id = 5`, `WHERE date > '2025-01-01'`, `ORDER BY date`
- **Most common type** - Default PostgreSQL index

### GIN Indexes (Generalized Inverted Index)
- **Use case:** Array, JSONB, full-text search
- **Examples:** `WHERE tags @> ARRAY['golang']`, `WHERE data ? 'key'`
- **Larger but much faster** for containment queries

### Partial Indexes
- **Use case:** Queries with constant WHERE clause
- **Examples:** `WHERE featured = true`, `WHERE deleted_at IS NULL`
- **Benefits:** Smaller index, faster queries on subset

### Composite Indexes
- **Use case:** Multi-column queries
- **Examples:** `WHERE category = 'X' ORDER BY date`
- **Rule:** Equality filters first, then range/sort filters

---

## Monitoring Index Performance

### Check Index Usage

```sql
-- Most used indexes
SELECT
    schemaname,
    tablename,
    indexname,
    idx_scan AS times_used,
    idx_tup_read AS tuples_read
FROM pg_stat_user_indexes
WHERE schemaname = 'public'
ORDER BY idx_scan DESC
LIMIT 20;
```

### Find Unused Indexes

```sql
-- Indexes never used (candidates for removal)
SELECT
    schemaname,
    tablename,
    indexname,
    pg_size_pretty(pg_relation_size(indexrelid)) AS size
FROM pg_stat_user_indexes
WHERE idx_scan = 0
  AND indexrelid NOT IN (
      SELECT indexrelid FROM pg_index WHERE indisprimary OR indisunique
  )
ORDER BY pg_relation_size(indexrelid) DESC;
```

### Check Index Sizes

```sql
-- Index storage usage
SELECT
    schemaname,
    tablename,
    indexname,
    pg_size_pretty(pg_relation_size(indexrelid)) AS size
FROM pg_stat_user_indexes
ORDER BY pg_relation_size(indexrelid) DESC
LIMIT 20;
```

---

## Maintenance

### VACUUM

PostgreSQL auto-vacuums, but manual VACUUM can help after large deletions:

```sql
VACUUM ANALYZE experiences;
VACUUM ANALYZE analytics_events;
```

### REINDEX

Rebuild indexes if they become bloated:

```sql
-- Single table
REINDEX TABLE experiences;

-- All tables
REINDEX DATABASE maicivy;

-- Concurrent (no locking, for production)
REINDEX INDEX CONCURRENTLY idx_experiences_category;
```

### Statistics

Update table statistics for better query planning:

```sql
ANALYZE experiences;
ANALYZE VERBOSE; -- All tables with details
```

---

## Performance Testing

### Before/After Comparison

```sql
-- Enable timing
\timing on

-- Test query before index
EXPLAIN ANALYZE SELECT * FROM experiences WHERE category = 'backend';
-- Note: Seq Scan on experiences (cost=X rows=Y time=Z)

-- Create index
CREATE INDEX idx_experiences_category ON experiences(category);

-- Test query after index
EXPLAIN ANALYZE SELECT * FROM experiences WHERE category = 'backend';
-- Note: Index Scan using idx_experiences_category (cost=X rows=Y time=Z)
```

### Expected Results

| Table | Query Type | Without Index | With Index | Improvement |
|-------|------------|---------------|------------|-------------|
| experiences | Category filter | 50ms | 2ms | 25x faster |
| generated_letters | Duplicate check | 200ms | 2ms | 100x faster |
| analytics_events | Time range | 100ms | 5ms | 20x faster |
| visitors | Session lookup | 30ms | 1ms | 30x faster |
| skills | Tag search | 80ms | 3ms | 27x faster |

---

## Best Practices

1. **Don't over-index** - Each index has a cost
2. **Monitor usage** - Drop unused indexes after 30 days
3. **Test before production** - Use EXPLAIN ANALYZE
4. **Update statistics** - Run ANALYZE regularly
5. **Use CONCURRENTLY** - For production index creation
6. **Composite order matters** - Equality first, range/sort second
7. **Partial indexes** - For subset queries (WHERE clause constants)
8. **GIN for arrays** - Much faster than B-tree for containment

---

## Troubleshooting

### Query Not Using Index

**Problem:** Query still slow despite index

**Solutions:**
1. Check query plan: `EXPLAIN ANALYZE your_query;`
2. Ensure column types match (no implicit casts)
3. Update statistics: `ANALYZE table_name;`
4. Check for functional indexes (e.g., `LOWER(column)`)
5. Verify WHERE clause matches index columns

### Index Bloat

**Problem:** Index size growing unexpectedly

**Solutions:**
1. Run VACUUM: `VACUUM FULL table_name;`
2. Reindex: `REINDEX TABLE table_name;`
3. Check for excessive UPDATE/DELETE operations
4. Consider fillfactor setting

### Slow INSERT/UPDATE

**Problem:** Writes are slow

**Solutions:**
1. Review index count (> 5 indexes per table = investigate)
2. Drop unused indexes
3. Use partial indexes to reduce size
4. Consider batching inserts

---

## References

- [PostgreSQL Index Documentation](https://www.postgresql.org/docs/current/indexes.html)
- [EXPLAIN Documentation](https://www.postgresql.org/docs/current/sql-explain.html)
- [GIN Indexes Guide](https://www.postgresql.org/docs/current/gin.html)
- [Index Maintenance Best Practices](https://www.postgresql.org/docs/current/maintenance.html)

---

**Last Updated:** 2025-12-08
**Author:** Alexi
**Version:** 1.0
