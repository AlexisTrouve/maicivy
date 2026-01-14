# Séquençage du Développement - maicivy

**Version:** 1.0
**Date:** 2025-12-08
**Auteur:** Alexi

---

## 🎯 Objectif

Ce document définit le **séquençage optimal** du développement du projet **maicivy** en utilisant des **agents parallèles** pour maximiser la vitesse tout en respectant les dépendances techniques.

---

## 📊 Vue d'Ensemble

### Gain de Performance

| Métrique | Séquentiel | Parallélisé | Gain |
|----------|------------|-------------|------|
| **Durée totale** | 68-88 jours | **38-44 jours** | **-45%** |
| **Nombre d'agents max simultanés** | 1 | 6 | x6 |
| **Phases parallélisables** | 0/6 | 4/6 | 67% |

### Structure en 6 Sprints

```
Sprint 1 (8-12j)  → Phase 1 - MVP Foundation (parallélisation limitée)
Sprint 2 (5j)     → Phase 2 - CV Dynamique (parallélisation totale)
Sprint 3 (10j)    → Phase 3 - IA Lettres (parallélisation partielle)
Sprint 4 (5j)     → Phase 4 - Analytics (parallélisation totale)
Sprint 5 (5-7j)   → Phase 5 - Features Avancées (parallélisation maximale)
Sprint 6 (5j)     → Phase 6 - Production (parallélisation totale)
```

---

## ⚖️ Règles de Parallélisation

### ✅ Règle #1 : Isolation des Fichiers

**Principe :** Deux agents ne doivent JAMAIS modifier le même fichier.

```
✅ AUTORISÉ
Agent A → backend/internal/api/cv.go
Agent B → backend/internal/api/letters.go

❌ INTERDIT
Agent A → backend/cmd/main.go
Agent B → backend/cmd/main.go  ← CONFLIT !
```

### ✅ Règle #2 : Respect des Dépendances

**Principe :** Un agent ne peut démarrer que si ses prérequis sont terminés.

```
✅ CORRECT
Database (Doc 03) → attend Backend (Doc 02) ✓
Middlewares (Doc 04) → attend Database (Doc 03) ✓

❌ INCORRECT
Middlewares (Doc 04) ⟷ Database (Doc 03) en parallèle ✗
```

### ✅ Règle #3 : Contrat API Préalable

**Principe :** Backend et Frontend peuvent être parallèles SI le contrat API est défini.

```
1. Définir schemas OpenAPI (types request/response)
2. Lancer en parallèle :
   ├─ Backend implémente les endpoints
   └─ Frontend consomme avec mocks au début
```

### ✅ Règle #4 : Git Branching Strategy

**Principe :** Un agent = une branche Git.

```
main
├─ feature/01-infrastructure       (Agent 1)
├─ feature/02-backend-foundation   (Agent 2)
├─ feature/05-frontend-foundation  (Agent 3) ← parallèle à 02
└─ feature/03-database-schema      (Agent 4) ← après 02
```

**Merge :** Séquentiellement après validation de chaque agent.

---

## 🏃 Sprint 1 - MVP Foundation (8-12 jours)

**Objectif :** Infrastructure + Backend + Frontend foundational

### Architecture de Parallélisation

```
┌──────────────────────────────────────────────────┐
│  SPRINT 1 - SÉQUENÇAGE                           │
└──────────────────────────────────────────────────┘

Jour 1-2 │ [Agent Infra] Doc 01
         │
Jour 3-5 │ [Agent Backend] Doc 02  ⟷  [Agent Frontend] Doc 05
         │        ↓                        │
Jour 6-9 │ [Agent Database] Doc 03        │ (continue)
         │        ↓                        ↓
Jour 10-12│ [Agent Middlewares] Doc 04
```

### Vague 1 : Infrastructure (Jours 1-2)

**Agents à lancer :** 1

**Prérequis :** Aucun

**Commande :**

```bash
# Lancer l'agent Infrastructure
claude-agent start \
  --doc "docs/implementation/01_SETUP_INFRASTRUCTURE.md" \
  --branch "feature/01-infrastructure" \
  --description "Setup Docker Compose, PostgreSQL, Redis" \
  --deliverables "docker-compose.yml, .env.example, Dockerfiles"
```

**Livrables attendus :**
- `docker-compose.yml`
- `.env.example`
- `backend/Dockerfile`
- `frontend/Dockerfile`
- `docker/nginx/nginx.conf`
- `scripts/health-check.sh`

**Validation :**
```bash
docker-compose up -d
docker ps  # 4 services running (postgres, redis, backend, frontend)
```

---

### Vague 2 : Backend + Frontend en Parallèle (Jours 3-5)

**Agents à lancer :** 2 (parallèle)

**Prérequis :** Vague 1 terminée

**Commandes (lancer simultanément) :**

```bash
# Agent Backend (terminal 1)
claude-agent start \
  --doc "docs/implementation/02_BACKEND_FOUNDATION.md" \
  --branch "feature/02-backend-foundation" \
  --description "Fiber setup, GORM, Redis, Logger" \
  --deliverables "backend/cmd/main.go, backend/internal/{config,database}, go.mod"

# Agent Frontend (terminal 2 - EN PARALLÈLE)
claude-agent start \
  --doc "docs/implementation/05_FRONTEND_FOUNDATION.md" \
  --branch "feature/05-frontend-foundation" \
  --description "Next.js 14, Tailwind, API client, Layout" \
  --deliverables "frontend/app/, frontend/components/, frontend/lib/, package.json"
```

**Livrables attendus :**

**Backend :**
- `backend/cmd/main.go`
- `backend/internal/config/config.go`
- `backend/internal/database/{postgres.go, redis.go}`
- `backend/pkg/logger/logger.go`
- `go.mod` avec dépendances

**Frontend :**
- `frontend/app/layout.tsx`
- `frontend/app/page.tsx`
- `frontend/lib/api.ts`
- `frontend/components/ui/`
- `tailwind.config.ts`
- `package.json`

**Validation :**
```bash
# Backend
cd backend && go build ./cmd && ./cmd/main
curl http://localhost:8080/health  # {"status":"ok"}

# Frontend
cd frontend && npm run dev
curl http://localhost:3000  # Page loads
```

---

### Vague 3 : Database Schema (Jours 6-9)

**Agents à lancer :** 1

**Prérequis :** Backend Foundation (Vague 2 - Backend) terminé

**Commande :**

```bash
# Agent Database
claude-agent start \
  --doc "docs/implementation/03_DATABASE_SCHEMA.md" \
  --branch "feature/03-database-schema" \
  --description "Models GORM, Migrations, Seed data" \
  --deliverables "backend/internal/models/, backend/migrations/, seed.go"
```

**Livrables attendus :**
- `backend/internal/models/{experience.go, skill.go, project.go, visitor.go, letter.go, analytics.go}`
- `backend/migrations/000001_init_schema.up.sql`
- `backend/migrations/000001_init_schema.down.sql`
- `backend/scripts/seed.go`

**Validation :**
```bash
# Run migrations
migrate -path backend/migrations -database "postgres://..." up

# Verify tables
psql -U postgres -d maicivy -c "\dt"
# Should show 6+ tables

# Seed data
cd backend && go run scripts/seed.go
```

---

### Vague 4 : Middlewares (Jours 10-12)

**Agents à lancer :** 1

**Prérequis :** Database Schema (Vague 3) terminé

**Commande :**

```bash
# Agent Middlewares
claude-agent start \
  --doc "docs/implementation/04_BACKEND_MIDDLEWARES.md" \
  --branch "feature/04-middlewares" \
  --description "CORS, Tracking, Rate limiting, Logger" \
  --deliverables "backend/internal/middleware/{cors,tracking,ratelimit,logger,recovery}.go"
```

**Livrables attendus :**
- `backend/internal/middleware/cors.go`
- `backend/internal/middleware/tracking.go`
- `backend/internal/middleware/ratelimit.go`
- `backend/internal/middleware/logger.go`
- `backend/internal/middleware/recovery.go`

**Validation :**
```bash
# Test tracking
curl -c cookies.txt http://localhost:8080/health
# Check cookie "session_id" is set

# Test rate limiting
for i in {1..150}; do curl http://localhost:8080/health; done
# Should get 429 after 100 requests
```

---

### Checklist Sprint 1

- [ ] Docker Compose lance 4 services (postgres, redis, backend, frontend)
- [ ] Backend répond sur `/health`
- [ ] Frontend affiche la homepage
- [ ] Database a 6+ tables créées
- [ ] Migrations up/down fonctionnent
- [ ] Seed data insère des fixtures
- [ ] Middlewares CORS configuré
- [ ] Tracking visiteurs fonctionne (cookie + Redis)
- [ ] Rate limiting fonctionne (429 après limite)
- [ ] Tests unitaires passent (backend + frontend)
- [ ] Branche `main` mergée avec feature/01, 02, 03, 04, 05

---

## 🏃 Sprint 2 - CV Dynamique (5 jours)

**Objectif :** CV adaptatif backend + frontend

### Architecture de Parallélisation

```
┌──────────────────────────────────────────────────┐
│  SPRINT 2 - SÉQUENÇAGE                           │
└──────────────────────────────────────────────────┘

Jour 1-5 │ [Agent Backend CV] Doc 06  ⟷  [Agent Frontend CV] Doc 07
```

### Vague 1 : Backend CV API + Frontend CV en Parallèle (Jours 1-5)

**Agents à lancer :** 2 (parallèle total)

**Prérequis :** Sprint 1 terminé

**⚠️ AVANT de lancer : Définir le contrat API**

Créer `docs/api-contracts/cv.yaml` :

```yaml
# OpenAPI spec pour CV
/api/cv:
  get:
    parameters:
      - name: theme
        schema:
          type: string
          enum: [backend, cpp, artistique, fullstack, devops]
    responses:
      200:
        content:
          application/json:
            schema:
              type: object
              properties:
                experiences: array
                skills: array
                projects: array
```

**Commandes (lancer simultanément) :**

```bash
# Agent Backend CV (terminal 1)
claude-agent start \
  --doc "docs/implementation/06_BACKEND_CV_API.md" \
  --branch "feature/06-backend-cv-api" \
  --description "Algorithme scoring, Endpoints CV, Export PDF" \
  --deliverables "backend/internal/api/cv.go, backend/internal/services/cv_scoring.go" \
  --api-contract "docs/api-contracts/cv.yaml"

# Agent Frontend CV (terminal 2 - EN PARALLÈLE)
claude-agent start \
  --doc "docs/implementation/07_FRONTEND_CV_DYNAMIC.md" \
  --branch "feature/07-frontend-cv-dynamic" \
  --description "Page /cv, CVThemeSelector, Timeline, SkillsCloud, ProjectsGrid" \
  --deliverables "frontend/app/cv/page.tsx, frontend/components/cv/" \
  --api-contract "docs/api-contracts/cv.yaml"
```

**Livrables attendus :**

**Backend :**
- `backend/internal/api/cv.go`
- `backend/internal/services/cv_scoring.go`
- `backend/internal/services/pdf.go`

**Frontend :**
- `frontend/app/cv/page.tsx`
- `frontend/components/cv/CVThemeSelector.tsx`
- `frontend/components/cv/ExperienceTimeline.tsx`
- `frontend/components/cv/SkillsCloud.tsx`
- `frontend/components/cv/ProjectsGrid.tsx`

**Validation :**
```bash
# Backend
curl "http://localhost:8080/api/cv?theme=backend"
# Retourne JSON avec experiences/skills/projects filtrés

curl "http://localhost:8080/api/cv/export?theme=backend&format=pdf" -o cv.pdf
# PDF généré

# Frontend
open http://localhost:3000/cv?theme=backend
# Page affiche CV adapté avec animations
```

---

### Checklist Sprint 2

- [ ] Backend : 5 thèmes CV fonctionnent (backend, cpp, artistique, fullstack, devops)
- [ ] Backend : Algorithme de scoring fonctionne (tags matching)
- [ ] Backend : Export PDF génère un fichier valide
- [ ] Backend : Cache Redis fonctionne (TTL 1h)
- [ ] Frontend : CVThemeSelector affiche les 5 thèmes
- [ ] Frontend : ExperienceTimeline avec animations Framer Motion
- [ ] Frontend : SkillsCloud interactif (hover, tailles dynamiques)
- [ ] Frontend : ProjectsGrid affiche projets GitHub
- [ ] Frontend : Export PDF button télécharge le PDF
- [ ] Frontend : Responsive (mobile, tablet, desktop)
- [ ] Tests E2E : Navigation thèmes + export PDF
- [ ] Branche `main` mergée avec feature/06, 07

---

## 🏃 Sprint 3 - IA Lettres (10 jours)

**Objectif :** Génération lettres motivation + anti-motivation par IA

### Architecture de Parallélisation

```
┌──────────────────────────────────────────────────┐
│  SPRINT 3 - SÉQUENÇAGE                           │
└──────────────────────────────────────────────────┘

Jour 1-7  │ [Agent AI Services] Doc 08
          │         ↓
Jour 8-10 │ [Agent Letters API] Doc 09  ⟷  [Agent Frontend Letters] Doc 10
```

### Vague 1 : AI Services (Jours 1-7)

**Agents à lancer :** 1

**Prérequis :** Sprint 1 terminé

**Commande :**

```bash
# Agent AI Services
claude-agent start \
  --doc "docs/implementation/08_BACKEND_AI_SERVICES.md" \
  --branch "feature/08-ai-services" \
  --description "Service IA (Claude/GPT), Scraper, PDF lettres, Prompts" \
  --deliverables "backend/internal/services/{ai,scraper,pdf_letters}.go" \
  --env-required "ANTHROPIC_API_KEY, OPENAI_API_KEY"
```

**Livrables attendus :**
- `backend/internal/services/ai.go`
- `backend/internal/services/scraper.go`
- `backend/internal/services/pdf_letters.go`
- `backend/internal/services/prompts.go`
- `backend/templates/letter_motivation.html`
- `backend/templates/letter_antimotivation.html`

**Validation :**
```bash
# Test génération lettres (unit test avec mocks)
cd backend && go test ./internal/services -run TestGenerateLetters
# 2 lettres générées (motivation + anti-motivation)
```

---

### Vague 2 : Letters API + Frontend Letters en Parallèle (Jours 8-10)

**Agents à lancer :** 2 (parallèle)

**Prérequis :** AI Services (Vague 1) terminé

**⚠️ AVANT de lancer : Définir le contrat API**

Créer `docs/api-contracts/letters.yaml` :

```yaml
/api/letters/generate:
  post:
    requestBody:
      content:
        application/json:
          schema:
            type: object
            properties:
              company_name:
                type: string
    responses:
      202:
        content:
          application/json:
            schema:
              type: object
              properties:
                job_id: string
                status: string
```

**Commandes (lancer simultanément) :**

```bash
# Agent Letters API (terminal 1)
claude-agent start \
  --doc "docs/implementation/09_BACKEND_LETTERS_API.md" \
  --branch "feature/09-letters-api" \
  --description "POST /api/letters/generate, Queue, Rate limiting 5/jour" \
  --deliverables "backend/internal/api/letters.go, backend/internal/services/queue.go" \
  --api-contract "docs/api-contracts/letters.yaml"

# Agent Frontend Letters (terminal 2 - EN PARALLÈLE)
claude-agent start \
  --doc "docs/implementation/10_FRONTEND_LETTERS.md" \
  --branch "feature/10-frontend-letters" \
  --description "Page /letters, LetterGenerator, LetterPreview (dual), AccessGate" \
  --deliverables "frontend/app/letters/page.tsx, frontend/components/letters/" \
  --api-contract "docs/api-contracts/letters.yaml"
```

**Livrables attendus :**

**Backend :**
- `backend/internal/api/letters.go`
- `backend/internal/services/queue.go`
- `backend/internal/middleware/access_gate.go`
- `backend/internal/middleware/ai_ratelimit.go`

**Frontend :**
- `frontend/app/letters/page.tsx`
- `frontend/components/letters/LetterGenerator.tsx`
- `frontend/components/letters/LetterPreview.tsx`
- `frontend/components/letters/AccessGate.tsx`

**Validation :**
```bash
# Backend
# Test access gate (< 3 visites)
curl -c cookies.txt http://localhost:8080/api/letters/generate -X POST -d '{"company_name":"Google"}'
# 403 Forbidden (pas assez de visites)

# Simuler 3 visites
curl -b cookies.txt http://localhost:8080/health  # visite 1
curl -b cookies.txt http://localhost:8080/health  # visite 2
curl -b cookies.txt http://localhost:8080/health  # visite 3

# Retry génération
curl -b cookies.txt http://localhost:8080/api/letters/generate -X POST -d '{"company_name":"Google"}'
# 202 Accepted {"job_id":"..."}

# Frontend
open http://localhost:3000/letters
# Affiche teaser si < 3 visites
# Affiche formulaire après 3 visites
# Génère 2 lettres en parallèle (motivation + anti-motivation)
```

---

### Checklist Sprint 3

- [ ] Backend : Service IA Claude fonctionne (+ GPT-4 fallback)
- [ ] Backend : Service Scraper récupère infos entreprises
- [ ] Backend : Génération 2 lettres (motivation + anti-motivation)
- [ ] Backend : PDF lettres avec templates HTML
- [ ] Backend : Access gate fonctionne (3 visites OU profil détecté)
- [ ] Backend : Rate limiting 5/jour fonctionne
- [ ] Backend : Cooldown 2min entre générations
- [ ] Backend : Queue système asynchrone
- [ ] Frontend : AccessGate affiche teaser si < 3 visites
- [ ] Frontend : LetterGenerator avec validation Zod
- [ ] Frontend : LetterPreview dual (2 colonnes desktop, stack mobile)
- [ ] Frontend : Export PDF dual (2 lettres ensemble)
- [ ] Frontend : Loading states avec progress bar
- [ ] Frontend : Error handling (403, 429, 500)
- [ ] Tests E2E : Flow complet génération lettres
- [ ] Branche `main` mergée avec feature/08, 09, 10

---

## 🏃 Sprint 4 - Analytics (5 jours)

**Objectif :** Dashboard analytics temps réel

### Architecture de Parallélisation

```
┌──────────────────────────────────────────────────┐
│  SPRINT 4 - SÉQUENÇAGE                           │
└──────────────────────────────────────────────────┘

Jour 1-5 │ [Agent Analytics Backend] Doc 11  ⟷  [Agent Analytics Frontend] Doc 12
```

### Vague 1 : Backend Analytics + Frontend Dashboard en Parallèle (Jours 1-5)

**Agents à lancer :** 2 (parallèle total)

**Prérequis :** Sprint 1 terminé

**⚠️ AVANT de lancer : Définir le contrat API + WebSocket**

Créer `docs/api-contracts/analytics.yaml` :

```yaml
/api/analytics/realtime:
  get:
    responses:
      200:
        content:
          application/json:
            schema:
              type: object
              properties:
                current_visitors: integer

/ws/analytics:
  websocket:
    messages:
      - type: visitors_update
        payload:
          current_visitors: integer
```

**Commandes (lancer simultanément) :**

```bash
# Agent Analytics Backend (terminal 1)
claude-agent start \
  --doc "docs/implementation/11_BACKEND_ANALYTICS.md" \
  --branch "feature/11-analytics-backend" \
  --description "Service analytics, Endpoints, WebSocket, Prometheus" \
  --deliverables "backend/internal/api/analytics.go, backend/internal/websocket/analytics.go" \
  --api-contract "docs/api-contracts/analytics.yaml"

# Agent Analytics Frontend (terminal 2 - EN PARALLÈLE)
claude-agent start \
  --doc "docs/implementation/12_FRONTEND_ANALYTICS_DASHBOARD.md" \
  --branch "feature/12-analytics-frontend" \
  --description "Page /analytics, RealtimeVisitors, ThemeStats, Heatmap" \
  --deliverables "frontend/app/analytics/page.tsx, frontend/components/analytics/" \
  --api-contract "docs/api-contracts/analytics.yaml"
```

**Livrables attendus :**

**Backend :**
- `backend/internal/api/analytics.go`
- `backend/internal/services/analytics.go`
- `backend/internal/websocket/analytics.go`

**Frontend :**
- `frontend/app/analytics/page.tsx`
- `frontend/components/analytics/RealtimeVisitors.tsx`
- `frontend/components/analytics/ThemeStats.tsx`
- `frontend/components/analytics/Heatmap.tsx`
- `frontend/components/analytics/DateFilter.tsx`

**Validation :**
```bash
# Backend
curl http://localhost:8080/api/analytics/realtime
# {"current_visitors":5}

curl http://localhost:8080/api/analytics/stats?period=day
# {"total_visits":1234, "letters_generated":56, ...}

# WebSocket test
wscat -c ws://localhost:8080/ws/analytics
# Receives real-time updates

# Frontend
open http://localhost:3000/analytics
# Dashboard affiche métriques temps réel
# Charts Chart.js animés
# Heatmap interactive
```

---

### Checklist Sprint 4

- [ ] Backend : Endpoints analytics (realtime, stats, themes, letters)
- [ ] Backend : WebSocket /ws/analytics broadcast temps réel
- [ ] Backend : Agrégations Redis (HyperLogLog, Sorted Sets)
- [ ] Backend : Métriques Prometheus custom
- [ ] Backend : Data retention (90j événements, 1an agrégations)
- [ ] Frontend : RealtimeVisitors avec WebSocket + auto-reconnect
- [ ] Frontend : ThemeStats avec Chart.js (bar chart)
- [ ] Frontend : Heatmap avec gradient de chaleur
- [ ] Frontend : DateFilter avec plages configurables
- [ ] Frontend : Auto-refresh 30s (polling stats)
- [ ] Frontend : Responsive dashboard
- [ ] Tests E2E : Mise à jour temps réel fonctionne
- [ ] Branche `main` mergée avec feature/11, 12

---

## 🏃 Sprint 5 - Features Avancées (5-7 jours)

**Objectif :** Import GitHub, Timeline, Détection profils, 3D (optionnel)

### Architecture de Parallélisation

```
┌──────────────────────────────────────────────────┐
│  SPRINT 5 - SÉQUENÇAGE                           │
└──────────────────────────────────────────────────┘

Jour 1-7 │ [Agent GitHub] ⟷ [Agent Timeline] ⟷ [Agent Profiling] ⟷ [Agent 3D]
         │  (4 agents en parallèle total)
```

### Vague 1 : Toutes les Features en Parallèle (Jours 1-7)

**Agents à lancer :** 4 (parallèle total, features indépendantes)

**Prérequis :** Sprint 2 terminé (pour Timeline et GitHub import dans CV)

**Commandes (lancer simultanément) :**

```bash
# Agent GitHub Import (terminal 1)
claude-agent start \
  --doc "docs/implementation/13_FEATURES_ADVANCED.md" \
  --section "Import Automatique GitHub" \
  --branch "feature/13-github-import" \
  --description "OAuth GitHub, API sync, Cron job" \
  --deliverables "backend/internal/services/github.go, frontend/components/github/"

# Agent Timeline Interactive (terminal 2 - EN PARALLÈLE)
claude-agent start \
  --doc "docs/implementation/13_FEATURES_ADVANCED.md" \
  --section "Timeline Interactive" \
  --branch "feature/13-timeline" \
  --description "Timeline avec animations, filtrage" \
  --deliverables "backend/internal/api/timeline.go, frontend/components/timeline/"

# Agent Détection Profils (terminal 3 - EN PARALLÈLE)
claude-agent start \
  --doc "docs/implementation/13_FEATURES_ADVANCED.md" \
  --section "Détection Profils Avancée" \
  --branch "feature/13-profile-detection" \
  --description "User-Agent parsing, IP lookup, Clearbit API" \
  --deliverables "backend/internal/services/profile_detection.go"

# Agent 3D (terminal 4 - EN PARALLÈLE, OPTIONNEL)
claude-agent start \
  --doc "docs/implementation/13_FEATURES_ADVANCED.md" \
  --section "Effets 3D Optionnels" \
  --branch "feature/13-3d-effects" \
  --description "Three.js, Avatar 3D, Skills graph 3D" \
  --deliverables "frontend/components/3d/"
```

**Livrables attendus :**

**GitHub Import :**
- `backend/internal/services/github.go`
- `backend/internal/api/github.go`
- `frontend/components/github/RepoCard.tsx`

**Timeline :**
- `backend/internal/api/timeline.go`
- `frontend/components/timeline/Timeline.tsx`

**Profiling :**
- `backend/internal/services/profile_detection.go`
- `backend/internal/middleware/profile_detector.go`

**3D (optionnel) :**
- `frontend/components/3d/Avatar3D.tsx`
- `frontend/components/3d/SkillsGraph3D.tsx`

**Validation :**
```bash
# GitHub Import
curl http://localhost:8080/api/github/repos
# Liste repos importés

# Timeline
open http://localhost:3000/cv?view=timeline
# Timeline interactive avec filtres

# Profile Detection
curl -A "LinkedInBot" http://localhost:8080/health
# Détecté comme recruteur

# 3D
open http://localhost:3000
# Avatar 3D interactif visible
```

---

### Checklist Sprint 5

**GitHub Import :**
- [ ] OAuth GitHub fonctionnel
- [ ] Sync API GitHub (repos publics)
- [ ] Cron job quotidien de sync
- [ ] Affichage repos avec stars, languages

**Timeline :**
- [ ] Endpoint backend données timeline
- [ ] Timeline avec Framer Motion
- [ ] Filtrage par catégorie
- [ ] Modal détails événement
- [ ] Responsive

**Profiling :**
- [ ] Détection User-Agent (recruteurs, bots)
- [ ] IP lookup (Clearbit API)
- [ ] Confidence scoring
- [ ] Dashboard profils détectés

**3D (optionnel) :**
- [ ] Three.js setup
- [ ] Avatar 3D avec rotations
- [ ] Visualisation 3D compétences
- [ ] Performance acceptable mobile

- [ ] Branche `main` mergée avec feature/13-*

---

## 🏃 Sprint 6 - Production & Qualité (5 jours)

**Objectif :** Infrastructure prod, CI/CD, Tests, Sécurité, Performance

### Architecture de Parallélisation

```
┌──────────────────────────────────────────────────┐
│  SPRINT 6 - SÉQUENÇAGE                           │
└──────────────────────────────────────────────────┘

Jour 1-5 │ [Infra Prod] ⟷ [CI/CD] ⟷ [Tests] ⟷ [Security] ⟷ [Perf] ⟷ [API Ref]
         │  (6 agents en parallèle total)
```

### Vague 1 : Toute la Production en Parallèle (Jours 1-5)

**Agents à lancer :** 6 (parallèle total)

**Prérequis :** Tous les sprints 1-5 terminés

**Commandes (lancer simultanément) :**

```bash
# Agent Infrastructure Production (terminal 1)
claude-agent start \
  --doc "docs/implementation/14_INFRASTRUCTURE_PRODUCTION.md" \
  --branch "feature/14-infra-prod" \
  --description "Nginx, Prometheus, Grafana, Backups, SSL" \
  --deliverables "docker/nginx/, monitoring/{prometheus,grafana}/, scripts/backup.sh"

# Agent CI/CD (terminal 2 - EN PARALLÈLE)
claude-agent start \
  --doc "docs/implementation/15_CICD_DEPLOYMENT.md" \
  --branch "feature/15-cicd" \
  --description "GitHub Actions, Deploy script, Rollback" \
  --deliverables ".github/workflows/{ci,deploy,backup}.yml, scripts/deploy.sh"

# Agent Testing Strategy (terminal 3 - EN PARALLÈLE)
claude-agent start \
  --doc "docs/implementation/16_TESTING_STRATEGY.md" \
  --branch "feature/16-tests" \
  --description "Tests unitaires, integration, E2E (testify, Playwright)" \
  --deliverables "backend/**/*_test.go, frontend/**/*.test.tsx, e2e/"

# Agent Security (terminal 4 - EN PARALLÈLE)
claude-agent start \
  --doc "docs/implementation/17_SECURITY.md" \
  --branch "feature/17-security" \
  --description "OWASP Top 10, Validation, Sanitization, Headers" \
  --deliverables "backend/internal/validator/, security-audit.md"

# Agent Performance (terminal 5 - EN PARALLÈLE)
claude-agent start \
  --doc "docs/implementation/18_PERFORMANCE.md" \
  --branch "feature/18-performance" \
  --description "Caching Redis, DB optimization, Benchmarks" \
  --deliverables "backend/benchmarks/, scripts/load-test.sh"

# Agent API Reference (terminal 6 - EN PARALLÈLE)
claude-agent start \
  --doc "docs/implementation/19_API_REFERENCE.md" \
  --branch "feature/19-api-ref" \
  --description "OpenAPI spec, Swagger UI" \
  --deliverables "docs/api/openapi.yaml, backend swagger annotations"
```

**Livrables attendus :**

**Infrastructure Production :**
- `docker/nginx/nginx.conf`
- `monitoring/prometheus/prometheus.yml`
- `monitoring/grafana/dashboards/maicivy.json`
- `scripts/backup.sh`

**CI/CD :**
- `.github/workflows/ci.yml`
- `.github/workflows/deploy.yml`
- `scripts/deploy.sh`

**Testing :**
- Tests backend (80%+ coverage)
- Tests frontend (70%+ coverage)
- Tests E2E Playwright

**Security :**
- OWASP Top 10 checklist complète
- Validation/Sanitization
- Security headers

**Performance :**
- Benchmarks (P95 < 100ms)
- Redis caching strategies
- DB indexes optimisés

**API Reference :**
- `docs/api/openapi.yaml`
- Swagger UI `/api/docs`

**Validation :**
```bash
# Infrastructure
docker-compose -f docker-compose.prod.yml up -d
curl https://maicivy.com/health  # SSL + Nginx
curl https://maicivy.com/metrics  # Prometheus
open https://maicivy.com/grafana  # Dashboard public

# CI/CD
git push origin main
# GitHub Actions runs tests → build → deploy
# Health check passes

# Testing
cd backend && go test -cover ./...
# Coverage: 82.5% of statements

cd frontend && npm test -- --coverage
# Coverage: 73.2%

# Security
gosec ./backend/...
# No issues found

npm audit
# 0 vulnerabilities

# Performance
wrk -t12 -c400 -d30s http://localhost:8080/api/cv
# P95 latency: 87ms

# API Reference
open http://localhost:8080/api/docs
# Swagger UI avec tous endpoints
```

---

### Checklist Sprint 6

**Infrastructure Production :**
- [ ] Nginx reverse proxy configuré
- [ ] SSL Let's Encrypt auto-renewal
- [ ] Prometheus scraping fonctionne
- [ ] Grafana dashboards provisionnés
- [ ] Dashboard public accessible
- [ ] Backups automatiques quotidiens
- [ ] Restore backups testé

**CI/CD :**
- [ ] GitHub Actions CI runs tests
- [ ] GitHub Actions Deploy sur push main
- [ ] Docker images build et push
- [ ] Deploy script avec rollback
- [ ] Health checks post-deploy
- [ ] Notifications (succès/échec)

**Testing :**
- [ ] Tests unitaires backend (80%+)
- [ ] Tests integration backend
- [ ] Tests unitaires frontend (70%+)
- [ ] Tests E2E Playwright (scénarios critiques)
- [ ] Performance tests (k6)
- [ ] CI runs tous les tests

**Security :**
- [ ] OWASP Top 10 checklist ✓
- [ ] Input validation (Zod, validator)
- [ ] Sanitization (XSS, SQL injection)
- [ ] Security headers (CSP, HSTS, etc.)
- [ ] Secrets management (.env, vault)
- [ ] Dependency scanning (gosec, npm audit)

**Performance :**
- [ ] Redis caching strategies (CVs, lettres)
- [ ] DB indexes optimisés
- [ ] EXPLAIN ANALYZE sur requêtes lentes
- [ ] Next.js Image optimization
- [ ] Benchmarks P95 < 100ms
- [ ] Grafana dashboard performance

**API Reference :**
- [ ] OpenAPI spec complet
- [ ] Swagger UI accessible
- [ ] Tous endpoints documentés
- [ ] Exemples curl
- [ ] Codes d'erreur

- [ ] Branche `main` mergée avec feature/14, 15, 16, 17, 18, 19

---

## 📅 Timeline Visuelle Complète

```
┌────────────────────────────────────────────────────────────────────────────┐
│  GANTT - DÉVELOPPEMENT MAICIVY (38-44 jours)                              │
└────────────────────────────────────────────────────────────────────────────┘

Week 1  ████ Sprint 1 (Phase 1 - Part 1)
Week 2  ████ Sprint 1 (Phase 1 - Part 2)
        ║
Week 3  ███ Sprint 2 (Phase 2)
        ║
Week 4  █████ Sprint 3 (Phase 3 - Part 1)
Week 5  ███ Sprint 3 (Phase 3 - Part 2)
        ║
Week 6  ███ Sprint 4 (Phase 4)
        ║
Week 7  ████ Sprint 5 (Phase 5)
        ║
Week 8  ███ Sprint 6 (Phase 6)
        ║
        └─► PRODUCTION READY 🚀

Légende:
█ = Jour de développement
```

---

## 🎯 Résumé Exécutif

### Timeline Optimale

| Sprint | Phase | Durée | Agents Max | Gain Parallèle |
|--------|-------|-------|------------|----------------|
| Sprint 1 | Phase 1 | 8-12j | 2 | 20% |
| Sprint 2 | Phase 2 | 5j | 2 | 50% |
| Sprint 3 | Phase 3 | 10j | 2 | 35% |
| Sprint 4 | Phase 4 | 5j | 2 | 50% |
| Sprint 5 | Phase 5 | 5-7j | 4 | 65% |
| Sprint 6 | Phase 6 | 5j | 6 | 70% |
| **TOTAL** | **Toutes** | **38-44j** | **6** | **45%** |

### Métriques de Performance

**Sans parallélisation :**
- Durée : 68-88 jours
- 1 agent à la fois
- 19 documents séquentiels

**Avec parallélisation :**
- Durée : 38-44 jours (-45%)
- Jusqu'à 6 agents simultanés
- 4 phases parallélisables (67%)

---

## ⚠️ Points d'Attention Critiques

### 1. Git Strategy

**Obligatoire :**
- 1 agent = 1 branche feature/XX-nom
- Merge séquentiel après validation
- CI/CD runs tests avant merge

### 2. Coordination Backend ↔ Frontend

**Avant chaque parallélisation :**
- Définir contrat API (OpenAPI spec)
- Frontend peut utiliser mocks au début
- Sync régulier (stand-up quotidien)

### 3. Gestion des Conflits

**Si conflit détecté :**
1. Arrêter les agents en conflit
2. Merger manuellement
3. Relancer agent avec code merged

### 4. Validation Continue

**Après chaque vague :**
- Tests automatisés passent
- Build réussit
- Health checks OK
- Documentation à jour

---

## 🚀 Commencer Maintenant

### Prérequis

1. **Environnement de développement :**
   ```bash
   # Go 1.21+
   go version

   # Node 18+
   node --version

   # Docker
   docker --version
   ```

2. **Outils :**
   ```bash
   # Install migrate (migrations SQL)
   go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest

   # Install air (hot reload Go)
   go install github.com/cosmtrek/air@latest
   ```

3. **Secrets :**
   ```bash
   # Créer .env
   cp .env.example .env

   # Remplir les API keys
   # ANTHROPIC_API_KEY=sk-...
   # OPENAI_API_KEY=sk-...
   ```

### Lancer Sprint 1

```bash
# Clone du repo
git clone <repo-url>
cd maicivy

# Créer branche develop
git checkout -b develop

# Lancer Vague 1 (Infrastructure)
# Voir section "Sprint 1 - Vague 1"
```

---

## 📊 Métriques de Suivi

### Tableau de Bord Sprint

| Métrique | Sprint 1 | Sprint 2 | Sprint 3 | Sprint 4 | Sprint 5 | Sprint 6 |
|----------|----------|----------|----------|----------|----------|----------|
| Agents lancés | 4 | 2 | 3 | 2 | 4 | 6 |
| Fichiers créés | ~50 | ~20 | ~25 | ~15 | ~30 | ~40 |
| Tests coverage | 75% | 80% | 78% | 82% | 76% | 85% |
| LoC ajoutées | ~5000 | ~3000 | ~6000 | ~3500 | ~4000 | ~3000 |
| Bugs détectés | - | - | - | - | - | - |
| Status | ⏳ | ⏳ | ⏳ | ⏳ | ⏳ | ⏳ |

### KPIs Globaux

- **Vélocité moyenne** : X agents/jour
- **Taux de réussite CI** : Y%
- **Coverage global** : Z%
- **Temps moyen merge** : W heures

---

## 📝 Changelog du Séquençage

| Version | Date | Changements |
|---------|------|-------------|
| 1.0 | 2025-12-08 | Création initiale du plan de séquençage |

---

**Prochaine étape :** Lancer Sprint 1 - Vague 1 (Infrastructure)

**Fichier :** `docs/DEVELOPMENT_SEQUENCING.md`
