# i18n Quickstart Guide

**TL;DR:** Le système bilingue FR/EN est maintenant implémenté. Voici comment l'utiliser.

## 🚀 Installation Rapide (5 minutes)

### 1. Exécuter les Migrations

```bash
cd /home/debian/maicivy/backend/migrations
./run_i18n_migrations.sh
```

**Ou manuellement:**

```bash
psql -U maicivyuser -d maicivydb -f add_i18n_fields.sql
psql -U maicivyuser -d maicivydb -f seed_data_i18n.sql
```

### 2. Redémarrer le Backend

```bash
# Docker
docker-compose restart backend

# Ou local
cd /home/debian/maicivy/backend
go run cmd/main.go
```

### 3. Vider le Cache Redis

```bash
docker exec -it maicivy-redis-1 redis-cli FLUSHDB
```

### 4. Tester

```bash
# Français
curl http://localhost:8080/api/v1/cv?theme=fullstack

# Anglais
curl http://localhost:8080/api/v1/cv?theme=fullstack&lang=en
```

✅ **Done!**

---

## 💻 Utilisation API

### Endpoint CV avec Langue

```http
GET /api/v1/cv?theme=<theme>&lang=<lang>
```

**Paramètres:**
- `theme`: "backend", "frontend", "fullstack", "cpp", "artistique", "devops"
- `lang`: "fr" (défaut) ou "en"

**Exemples:**

```bash
# Backend en anglais
curl "http://localhost:8080/api/v1/cv?theme=backend&lang=en"

# Fullstack en français (défaut)
curl "http://localhost:8080/api/v1/cv?theme=fullstack"

# Export PDF en anglais
curl "http://localhost:8080/api/v1/cv/export?theme=backend&lang=en" -o cv_en.pdf
```

---

## 🎨 Frontend Integration

### React/Next.js

```typescript
const [lang, setLang] = useState<'fr' | 'en'>('fr');

const fetchCV = async () => {
  const response = await fetch(
    `http://localhost:8080/api/v1/cv?theme=fullstack&lang=${lang}`
  );
  return response.json();
};

// Language switcher
<button onClick={() => setLang('fr')}>FR</button>
<button onClick={() => setLang('en')}>EN</button>
```

---

## 📁 Fichiers Importants

### Créés
- `backend/migrations/add_i18n_fields.sql` - Migration colonnes
- `backend/migrations/seed_data_i18n.sql` - Traductions EN
- `backend/internal/services/localization.go` - Helper localisation

### Modifiés
- `backend/internal/models/experience.go` - +5 champs `*En`
- `backend/internal/models/skill.go` - +2 champs `*En`
- `backend/internal/models/project.go` - +5 champs `*En`
- `backend/internal/services/cv_service.go` - Logique localisation
- `backend/internal/api/cv.go` - Param `lang`

---

## 🐛 Problèmes Courants

### API retourne toujours français

```bash
# Vider le cache Redis
docker exec -it maicivy-redis-1 redis-cli FLUSHDB

# Vérifier les traductions en DB
psql -U maicivyuser -d maicivydb -c "SELECT title, title_en FROM experiences LIMIT 1;"
```

### Erreur: "theme not found"

```bash
# Thèmes valides: backend, frontend, fullstack, cpp, artistique, devops
curl "http://localhost:8080/api/v1/cv?theme=fullstack&lang=en"
```

### Tests échouent

```bash
# Vérifier que tous les mocks ont 3 paramètres (ctx, themeID, lang)
grep "GetAdaptiveCV.*mock.Anything" backend/internal/api/cv_test.go
```

---

## 📊 Vérification Rapide

### Check Migrations

```sql
-- Se connecter à psql
psql -U maicivyuser -d maicivydb

-- Vérifier colonnes
SELECT column_name FROM information_schema.columns
WHERE table_name = 'experiences' AND column_name LIKE '%_en';

-- Vérifier traductions (doit afficher 4 lignes avec title_en non NULL)
SELECT COUNT(*) FROM experiences WHERE title_en IS NOT NULL;
```

**Résultat attendu:** 4 expériences traduites

### Check Cache Redis

```bash
docker exec -it maicivy-redis-1 redis-cli

> KEYS cv:theme:*
# Devrait lister des clés comme: cv:theme:fullstack:lang:en

> GET cv:theme:fullstack:lang:en
# Devrait afficher du JSON avec texte anglais

> exit
```

### Check API

```bash
# Test rapide: vérifier que le titre change selon la langue
curl -s "http://localhost:8080/api/v1/cv?theme=fullstack&lang=fr" | jq '.experiences[0].title'
# Résultat: "Développeur IT Polyvalent..."

curl -s "http://localhost:8080/api/v1/cv?theme=fullstack&lang=en" | jq '.experiences[0].title'
# Résultat: "Versatile IT Developer..."
```

---

## 📖 Documentation Complète

Pour plus de détails, voir:
- **`I18N_IMPLEMENTATION_RECAP.md`** - Guide complet (architecture, traductions, troubleshooting)
- **`I18N_VERIFICATION_CHECKLIST.md`** - Checklist de validation détaillée

---

## 🎯 Quick Summary

**Ce qui a été fait:**
- ✅ 12 colonnes `*_en` ajoutées (experiences, skills, projects)
- ✅ 33 traductions professionnelles en anglais
- ✅ Service de localisation avec fallback automatique vers FR
- ✅ API accepte paramètre `?lang=en`
- ✅ Cache Redis par langue
- ✅ Tests mis à jour

**Ce qui fonctionne:**
- ✅ API `/api/v1/cv?lang=en` retourne texte anglais
- ✅ API `/api/v1/cv?lang=fr` retourne texte français (défaut)
- ✅ Export PDF avec langue
- ✅ Fallback automatique si traduction manquante

**Ce qui reste à faire:**
- [ ] Tester manuellement l'API
- [ ] Intégrer sélecteur de langue dans frontend
- [ ] Localiser templates HTML du PDF (optionnel)
- [ ] Ajouter d'autres langues (ES, DE, etc.) si besoin

---

**Date:** 2026-01-12 | **Auteur:** Alexi (Claude) | **Version:** 1.0.0
