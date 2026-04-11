# 🌍 Internationalization (i18n) - Complete Implementation Summary

## ✅ Status: FULLY IMPLEMENTED

All components of the maicivy platform now support French and English:

- ✅ Frontend UI (pages, components, forms, validation)
- ✅ Backend API (CV, letters, analytics)
- ✅ Database content (experiences, skills, projects)
- ✅ AI letter generation (prompts in FR/EN)

---

## 📋 What's Translated

### 1. Frontend UI (Next.js)

**Files:**
- `frontend/messages/fr.json` - French translations
- `frontend/messages/en.json` - English translations

**Pages:**
- `/` → Home page
- `/cv` → CV page
- `/letters` → Letter generator
- `/analytics` → Analytics dashboard
- `/architecture` → Architecture page

**Usage:**
```tsx
const t = useTranslations('cv');
const locale = useLocale(); // 'fr' or 'en'

<h1>{t('title')}</h1>
```

### 2. Backend Data (PostgreSQL)

**Tables with i18n:**
- `experiences` - Professional experiences
- `skills` - Technical skills
- `projects` - Portfolio projects

**Fields pattern:**
```sql
-- French (default)
title, description, catchphrase, functional_description, technical_description

-- English
title_en, description_en, catchphrase_en, functional_description_en, technical_description_en
```

**API Usage:**
```bash
# French
GET /api/v1/cv?theme=fullstack&lang=fr

# English
GET /api/v1/cv?theme=fullstack&lang=en
```

### 3. AI Letter Generation

**Backend prompts:**
- `BuildMotivationPrompt(company, "fr")` → French cover letter
- `BuildMotivationPrompt(company, "en")` → English cover letter
- `BuildAntiMotivationPrompt(company, "fr")` → French anti-motivation
- `BuildAntiMotivationPrompt(company, "en")` → English anti-motivation

**Frontend integration:**
```tsx
const locale = useLocale();
const response = await lettersApi.generateAndWait({
  company_name: data.companyName,
  lang: locale // 'fr' or 'en'
});
```

---

## 🚀 URL Structure

### Language Detection

The platform uses URL-based locale detection:

```
https://maicivy.etheryale.com/         → French (default)
https://maicivy.etheryale.com/fr/cv    → French (explicit)
https://maicivy.etheryale.com/en/cv    → English
```

### Language Switcher

The header includes a 🇬🇧/🇫🇷 button that switches between:
- `/fr/[page]` ↔️ `/en/[page]`

Implemented in: `frontend/components/shared/LanguageSwitcher.tsx`

---

## 📁 Key Files

### Frontend

```
frontend/
├── app/[locale]/              # Locale-based routing
│   ├── layout.tsx            # ✅ Dynamic metadata
│   ├── page.tsx              # ✅ Home page
│   ├── cv/page.tsx           # ✅ CV with lang parameter
│   ├── letters/page.tsx      # ✅ Letters with lang
│   └── analytics/page.tsx    # ✅ Analytics
├── components/
│   ├── letters/
│   │   └── LetterGenerator.tsx  # ✅ Sends locale to API
│   └── shared/
│       └── LanguageSwitcher.tsx # ✅ Language toggle
├── messages/
│   ├── fr.json               # ✅ French translations
│   └── en.json               # ✅ English translations
├── i18n/
│   ├── config.ts             # ✅ Locales config
│   └── navigation.ts         # ✅ Localized routing
└── middleware.ts             # ✅ Locale detection
```

### Backend

```
backend/
├── internal/
│   ├── models/
│   │   ├── experience.go     # ✅ *_en fields
│   │   ├── skill.go          # ✅ *_en fields
│   │   └── project.go        # ✅ *_en fields
│   ├── services/
│   │   ├── localization.go   # ✅ LocalizeExperience/Skill/Project
│   │   ├── cv_service.go     # ✅ Uses lang parameter
│   │   ├── prompts.go        # ✅ FR/EN prompt builders
│   │   └── letter_generator.go # ✅ Uses lang
│   └── api/
│       ├── cv.go             # ✅ ?lang=fr|en
│       └── letters.go        # ✅ Accepts lang in body
└── migrations/
    ├── translate_experiences_en.sql   # ✅ Translation script
    ├── run_translate_experiences.sh   # ✅ Apply script
    ├── check_translations.sh          # ✅ Verify translations
    └── README_i18n.md                 # ✅ Documentation
```

---

## 🛠️ How to Add New Translations

### 1. Frontend (UI Text)

Edit translation files:

```bash
# French
vim frontend/messages/fr.json

# English
vim frontend/messages/en.json
```

Add your key:
```json
{
  "mySection": {
    "myKey": "Texte français" // fr.json
    "myKey": "English text"   // en.json
  }
}
```

Use in component:
```tsx
const t = useTranslations('mySection');
<p>{t('myKey')}</p>
```

### 2. Backend (Database Content)

For experiences/skills/projects:

```sql
-- Add new experience with both languages
INSERT INTO experiences (
    id, title, title_en, description, description_en,
    company, start_date, category
) VALUES (
    gen_random_uuid(),
    'Développeur Backend',
    'Backend Developer',
    'Description française...',
    'English description...',
    'TechCorp',
    '2024-01-01',
    'backend'
);

-- Or update existing
UPDATE experiences
SET
    title_en = 'Backend Developer',
    description_en = 'English description...'
WHERE id = 'uuid-here';
```

Then clear cache:
```bash
docker exec -i maicivy-redis redis-cli FLUSHDB
```

### 3. AI Prompts (Letter Generation)

Already handled automatically! The `PromptBuilder` class in `backend/internal/services/prompts.go` has:
- `buildMotivationPromptFR()` - French prompt
- `buildMotivationPromptEN()` - English prompt
- `buildAntiMotivationPromptFR()` - French humor
- `buildAntiMotivationPromptEN()` - English humor (adapted)

No changes needed unless modifying prompt structure.

---

## 🧪 Testing

### 1. Verify Frontend

```bash
# Build (check for errors)
cd frontend && npm run build

# Check translation files are valid JSON
cat messages/fr.json | python3 -m json.tool > /dev/null
cat messages/en.json | python3 -m json.tool > /dev/null
```

### 2. Verify Backend Translations

```bash
# Run check script
./backend/migrations/check_translations.sh

# Should show all items translated
```

### 3. Test in Browser

**French:**
- https://maicivy.etheryale.com/cv
- https://maicivy.etheryale.com/letters

**English:**
- https://maicivy.etheryale.com/en/cv
- https://maicivy.etheryale.com/en/letters

### 4. Test API Directly

```bash
# CV in English
curl "https://maicivy.etheryale.com/api/v1/cv?theme=fullstack&lang=en"

# CV in French
curl "https://maicivy.etheryale.com/api/v1/cv?theme=fullstack&lang=fr"
```

### 5. Test Letter Generation

1. Go to `/en/letters` (English mode)
2. Generate a letter for a company
3. Verify the letter is in English
4. Go to `/letters` (French mode)
5. Generate another letter
6. Verify it's in French

---

## 🔧 Troubleshooting

### Issue: English page shows French text

**Symptoms:**
- URL is `/en/...` but content is French

**Causes & Solutions:**

1. **Frontend not rebuilt**
   ```bash
   cd frontend && npm run build
   docker compose up -d --build frontend
   ```

2. **Missing English translations in DB**
   ```bash
   ./backend/migrations/check_translations.sh
   # If missing, add them via SQL
   ```

3. **Cache not cleared**
   ```bash
   docker exec -i maicivy-redis redis-cli FLUSHDB
   ```

4. **Next.js params as Promise issue**
   - Already fixed in all page files
   - Verify `params instanceof Promise ? await params : params`

### Issue: Letter generated in wrong language

**Check:**

1. Frontend sends correct locale:
   ```tsx
   // In LetterGenerator.tsx
   const locale = useLocale(); // Should be 'en' or 'fr'
   ```

2. Backend receives lang parameter:
   ```go
   // In letters.go
   lang := req.Lang // Should be "en" or "fr"
   ```

3. Prompt builder uses correct language:
   ```go
   // In prompts.go
   if lang == "en" {
       return pb.buildMotivationPromptEN(company)
   }
   ```

### Issue: Next.js 404 on `/en/...`

**Cause:** Middleware not routing locales

**Fix:**
```typescript
// middleware.ts should have:
export const config = {
  matcher: ['/((?!api|_next|_vercel|3d-demo|api-test|.*\\..*).*)']
};
```

---

## 📊 Translation Coverage

Current status (as of 2026-01-14):

| Component | French | English | Status |
|-----------|--------|---------|--------|
| Frontend UI | ✅ 100% | ✅ 100% | Complete |
| Experiences (4) | ✅ 4/4 | ✅ 4/4 | Complete |
| Skills (17) | ✅ 17/17 | ✅ 17/17 | Complete |
| Projects (6) | ✅ 6/6 | ✅ 6/6 | Complete |
| AI Prompts | ✅ Yes | ✅ Yes | Complete |

**Total: 100% translated** 🎉

---

## 🔮 Future Enhancements

Potential improvements:

1. **Browser Language Detection**
   - Auto-detect `navigator.language`
   - Redirect `/` → `/en/` or `/fr/` based on browser

2. **More Languages**
   - Add Spanish: `messages/es.json`
   - Add German: `messages/de.json`
   - Update `i18n/config.ts` locales

3. **Translation Management**
   - Use translation platform (Lokalise, Phrase)
   - Export/import JSON files
   - Translation memory

4. **SEO**
   - Add `<link rel="alternate" hreflang="en" href="..." />`
   - Localized sitemaps
   - Localized metadata

---

## 📚 Resources

- [Next.js i18n Routing](https://next-intl-docs.vercel.app/docs/routing)
- [next-intl Documentation](https://next-intl-docs.vercel.app/)
- [PostgreSQL Text Search](https://www.postgresql.org/docs/current/textsearch.html)
- [Go text/template](https://pkg.go.dev/text/template)

---

## ✅ Checklist for New Features

When adding new features, ensure i18n:

- [ ] Add keys to `messages/fr.json`
- [ ] Add keys to `messages/en.json`
- [ ] Use `useTranslations()` in components
- [ ] Test on `/en/...` URL
- [ ] If database content, add `*_en` fields
- [ ] Update this documentation
- [ ] Test letter generation in both languages

---

**Last Updated:** 2026-01-14
**Status:** ✅ Production Ready
**Version:** 1.0.0
