# Internationalization (i18n) for Experiences

## Overview

Experiences support French (default) and English translations through dedicated `*_en` fields in the database.

## Database Schema

The `experiences` table includes the following i18n fields:

```sql
-- French fields (default)
title                       VARCHAR(255)  -- "Développeur Full-Stack"
description                 TEXT          -- Main description
catchphrase                 VARCHAR(200)  -- Short tagline
functional_description      TEXT          -- What the role achieved
technical_description       TEXT          -- How it was built

-- English translations
title_en                    VARCHAR(255)  -- "Full-Stack Developer"
description_en              TEXT
catchphrase_en              VARCHAR(200)
functional_description_en   TEXT
technical_description_en    TEXT
```

## How It Works

### Backend (Automatic)

The backend automatically serves the correct language based on the `lang` query parameter:

```bash
# French (default)
GET /api/v1/cv?theme=fullstack&lang=fr

# English
GET /api/v1/cv?theme=fullstack&lang=en
```

The `LocalizationHelper` service (in `backend/internal/services/localization.go`) automatically:
1. Detects the requested language
2. Returns `*_en` fields if `lang=en` and they are not empty
3. Falls back to French fields if English translations are missing

### Frontend (Automatic)

The frontend automatically detects the user's locale from the URL:
- `/en/cv` → Requests English (`lang=en`)
- `/cv` or `/fr/cv` → Requests French (`lang=fr`)

## Adding New Experiences

### Option 1: SQL Insert (Bilingual)

```sql
INSERT INTO experiences (
    id, title, description, catchphrase,
    title_en, description_en, catchphrase_en,
    company, start_date, category
) VALUES (
    gen_random_uuid(),
    'Développeur Backend Go',
    'Description en français...',
    'Phrase d''accroche',
    'Backend Go Developer',
    'English description...',
    'Catchphrase',
    'TechCorp',
    '2024-01-01',
    'backend'
);
```

### Option 2: Update Existing Experience

```sql
UPDATE experiences
SET
    title_en = 'Backend Go Developer',
    description_en = 'English description...',
    catchphrase_en = 'Catchphrase',
    functional_description_en = 'What was achieved...',
    technical_description_en = 'Technical implementation details...'
WHERE id = 'uuid-here';
```

## Translation Guidelines

### 1. Title Translation
- Keep technical terms in English (Go, React, DevOps)
- Translate job titles naturally:
  - ❌ "Developer Full-Stack"
  - ✅ "Full-Stack Developer"

### 2. Description Translation
- Professional tone, active voice
- Preserve technical accuracy
- Keep metrics and numbers identical
- Example:
  ```
  FR: "Réduction de 70% de la latence API"
  EN: "70% API latency reduction"
  ```

### 3. Catchphrase Translation
- Keep it concise (under 200 chars)
- Focus on impact and technologies
- Example:
  ```
  FR: "APIs haute performance et architecture microservices"
  EN: "High-performance APIs and microservices architecture"
  ```

### 4. Technical Description
- Keep technology names unchanged
- Translate methodologies and concepts
- List technologies at the end:
  ```
  EN: "Built RESTful APIs using Go and Fiber framework.
       Implemented CQRS pattern for data consistency.
       Technologies: Go, Fiber, PostgreSQL, Redis, Docker."
  ```

## Testing Translations

### 1. Via API

```bash
# Test French
curl "http://localhost:8081/api/v1/cv?theme=fullstack&lang=fr"

# Test English
curl "http://localhost:8081/api/v1/cv?theme=fullstack&lang=en"
```

### 2. Via Frontend

```bash
# French
http://localhost:3002/cv

# English
http://localhost:3002/en/cv
```

### 3. Verify Database

```sql
-- Check which experiences have English translations
SELECT
    company,
    title,
    CASE
        WHEN title_en IS NOT NULL AND title_en != '' THEN '✓'
        ELSE '✗'
    END as has_english
FROM experiences
ORDER BY start_date DESC;
```

## Cache Management

The CV API caches responses by theme AND language. When updating translations:

```bash
# Clear all CV cache entries
docker exec -i maicivy-redis redis-cli KEYS "cv:theme:*" | \
    xargs docker exec -i maicivy-redis redis-cli DEL

# Or restart Redis
docker compose restart redis
```

## Common Issues

### Issue: English page shows French text

**Causes:**
1. English fields are empty in database
2. Cache hasn't been invalidated

**Solution:**
```bash
# 1. Check if translations exist
docker exec -i maicivy-postgres psql -U maicivy -d maicivy -c \
  "SELECT id, title, title_en FROM experiences WHERE title_en IS NULL;"

# 2. Clear cache
docker exec -i maicivy-redis redis-cli FLUSHDB
```

### Issue: New experience not showing translations

**Cause:** Forgot to set `*_en` fields

**Solution:** Run an UPDATE query to add translations (see examples above)

## Translation Scripts

### Generate Translation Template

Run this to generate a SQL template for missing translations:

```bash
docker exec -i maicivy-postgres psql -U maicivy -d maicivy -c "
SELECT
    'UPDATE experiences SET' ||
    E'\n    title_en = \'' || title || '\',' ||
    E'\n    description_en = \'' || description || '\',' ||
    E'\n    catchphrase_en = \'\',' ||
    E'\nWHERE id = \'' || id || '\';' ||
    E'\n'
FROM experiences
WHERE title_en IS NULL OR title_en = '';
"
```

Then manually translate and run the generated SQL.

## Future: Projects and Skills

The same i18n pattern applies to:
- **Projects**: `title_en`, `description_en`, `catchphrase_en`, etc.
- **Skills**: `name_en`, `description_en`

Use the same approach for translations.
