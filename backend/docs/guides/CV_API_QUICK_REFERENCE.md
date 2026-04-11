# CV API - Quick Reference

## 🚀 Quick Start

### Endpoints Disponibles

```bash
# CV adaptatif (thème backend)
GET /api/v1/cv?theme=backend

# Liste des thèmes
GET /api/v1/cv/themes

# Toutes les expériences
GET /api/v1/experiences

# Toutes les compétences
GET /api/v1/skills

# Tous les projets
GET /api/v1/projects

# Export PDF
GET /api/v1/cv/export?theme=fullstack&format=pdf
```

---

## 📋 Thèmes Disponibles

| Theme ID | Nom | Description |
|----------|-----|-------------|
| `backend` | Backend Developer | Go, APIs, bases de données |
| `cpp` | C++ Developer | C++, systèmes bas niveau |
| `artistique` | Creative & Artistic | Design, 3D, visualisation |
| `fullstack` | Full-Stack Developer | Frontend + Backend |
| `devops` | DevOps Engineer | Infrastructure, CI/CD |

---

## 🧪 Tests Rapides

```bash
# Test endpoint CV
curl http://localhost:8080/api/v1/cv?theme=backend | jq

# Test thèmes
curl http://localhost:8080/api/v1/cv/themes | jq

# Télécharger PDF
curl http://localhost:8080/api/v1/cv/export?theme=devops -o cv.pdf

# Vérifier cache Redis
redis-cli KEYS "cv:theme:*"
redis-cli TTL "cv:theme:backend"
```

---

## 🔄 Cache Management

```bash
# Invalider cache (à implémenter endpoint admin)
# Pour l'instant: manual via Redis CLI

redis-cli DEL "cv:theme:backend"
redis-cli DEL "cv:theme:fullstack"

# Ou invalider tous les thèmes
redis-cli KEYS "cv:theme:*" | xargs redis-cli DEL
```

---

## 🏗️ Structure Code

```
backend/
├── internal/
│   ├── config/
│   │   └── themes.go           # Configuration des 5 thèmes
│   ├── services/
│   │   ├── cv_scoring.go       # Algorithme de scoring
│   │   ├── cv_service.go       # Service métier principal
│   │   └── pdf_service.go      # Génération PDF
│   └── api/
│       └── cv.go               # Handlers HTTP
├── templates/
│   └── cv/
│       └── cv_base.html        # Template PDF
└── cmd/
    └── main.go                 # Intégration routes
```

---

## 🎯 Scoring Algorithm

```
Experience Score =
  (Tags Match × Weight) +
  (Tech Match × Weight × 0.8) +
  (Category Match ? 0.5 : 0)

Skill Score =
  (Name Match × Weight) +
  (Tags Match × Weight × 0.7) +
  (Level Bonus: 0.1-0.3) +
  (Years Bonus: 0.1-0.2)

Project Score =
  (Tech Match × Weight) +
  (Featured ? 0.3 : 0) +
  (Category Match ? 0.4 : 0)
```

---

## 📊 Example Response

```json
{
  "theme": {
    "id": "backend",
    "name": "Backend Developer",
    "description": "Focus sur développement backend..."
  },
  "experiences": [
    {
      "id": 1,
      "title": "Senior Backend Dev",
      "company": "TechCorp",
      "technologies": ["go", "postgresql"],
      "tags": ["backend", "api"]
    }
  ],
  "skills": [...],
  "projects": [...],
  "generated_at": "2025-12-08T12:00:00Z"
}
```

---

## 🐛 Troubleshooting

### PDF Generation Fails

**Erreur:** `chromedp failed: exec: "chrome": executable file not found`

**Solution:**
```bash
# Install Chrome/Chromium
apt-get install chromium-browser

# Ou dans Dockerfile
RUN apt-get update && apt-get install -y chromium
```

### Cache Not Working

**Vérifier Redis connexion:**
```bash
redis-cli PING
# Should return: PONG

# Check keys
redis-cli KEYS "*"
```

### Theme Not Found

**Erreur:** `theme not found: xxx`

**Cause:** ThemeID invalide

**Solution:** Utiliser un des 5 thèmes: `backend`, `cpp`, `artistique`, `fullstack`, `devops`

---

## 🔧 Configuration

### Ajouter un Nouveau Thème

**Fichier:** `internal/config/themes.go`

```go
"ai": {
    ID:          "ai",
    Name:        "AI Developer",
    Description: "Focus sur IA et ML",
    TagWeights: map[string]float64{
        "ai":         1.0,
        "ml":         1.0,
        "python":     0.9,
        "tensorflow": 0.9,
    },
},
```

### Modifier les Poids

Éditer `TagWeights` dans `themes.go`:
- `1.0` = Très important
- `0.5` = Moyennement important
- `0.0` = Non pertinent

---

## 📝 Next Steps

1. **Frontend Integration (Phase 2):**
   - Créer composant `CVThemeSelector`
   - Fetcher `/api/v1/cv?theme=X`
   - Afficher résultats adaptés

2. **AI Letters (Phase 3):**
   - Intégrer Claude/GPT
   - Créer endpoint `/api/v1/letters/generate`

3. **Tests E2E (Phase 6):**
   - Testcontainers
   - Coverage >80%

---

**Documentation complète:** `CV_API_IMPLEMENTATION_SUMMARY.md`
