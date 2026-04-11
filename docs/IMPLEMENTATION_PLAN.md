# Plan d'Implémentation - maicivy

**Version:** 1.0
**Date:** 2025-12-08
**Auteur:** Alexi

---

## 📋 Vue d'Ensemble

Ce document définit le plan complet d'implémentation du projet **maicivy** à travers **19 documents techniques** organisés en **6 phases** correspondant à la roadmap du projet.

### Objectifs du Plan

1. **Structuration claire** : Chaque composant du système a sa documentation dédiée
2. **Progression logique** : Respect des dépendances techniques entre modules
3. **Parallélisation** : Identification des tâches pouvant être menées simultanément
4. **Qualité** : Intégration de la sécurité, tests et performance dès la conception

---

## 🗺️ Vue d'Ensemble des 19 Documents

```
PHASE 1 - MVP FOUNDATION (5 docs)
├─ 01. SETUP_INFRASTRUCTURE.md
├─ 02. BACKEND_FOUNDATION.md
├─ 03. DATABASE_SCHEMA.md
├─ 04. BACKEND_MIDDLEWARES.md
└─ 05. FRONTEND_FOUNDATION.md

PHASE 2 - CV DYNAMIQUE (2 docs)
├─ 06. BACKEND_CV_API.md
└─ 07. FRONTEND_CV_DYNAMIC.md

PHASE 3 - IA LETTRES (3 docs)
├─ 08. BACKEND_AI_SERVICES.md
├─ 09. BACKEND_LETTERS_API.md
└─ 10. FRONTEND_LETTERS.md

PHASE 4 - ANALYTICS (2 docs)
├─ 11. BACKEND_ANALYTICS.md
└─ 12. FRONTEND_ANALYTICS_DASHBOARD.md

PHASE 5 - FEATURES AVANCÉES (1 doc)
└─ 13. FEATURES_ADVANCED.md

PHASE 6 - PRODUCTION & QUALITÉ (5 docs)
├─ 14. INFRASTRUCTURE_PRODUCTION.md
├─ 15. CICD_DEPLOYMENT.md
├─ 16. TESTING_STRATEGY.md
├─ 17. SECURITY.md
└─ 18. PERFORMANCE.md

ANNEXES (1 doc)
└─ 19. API_REFERENCE.md
```

---

## 📊 Graphe de Dépendances

```
Légende:
──► Dépendance bloquante (séquentiel)
═══ Peut être parallélisé
[P] Peut démarrer en parallèle dès le début

                    01. SETUP_INFRASTRUCTURE
                              │
                    ┌─────────┴─────────┐
                    ▼                   ▼
         02. BACKEND_FOUNDATION    05. FRONTEND_FOUNDATION [P]
                    │                   │
                    ▼                   │
         03. DATABASE_SCHEMA            │
                    │                   │
                    ▼                   │
         04. BACKEND_MIDDLEWARES        │
                    │                   │
         ┌──────────┼──────────┐        │
         ▼          ▼          ▼        │
    06. CV_API  08. AI_SRV  11. ANALYTICS_API
         │          │          │        │
         ▼          ▼          ▼        │
    07. CV_FE   09. LETTERS_API  12. ANALYTICS_FE
                    │
                    ▼
               10. LETTERS_FE

    13. FEATURES_ADVANCED (après Phase 2)

    14. INFRASTRUCTURE_PRODUCTION [P] (peut setup dès Phase 1)
    15. CICD_DEPLOYMENT [P] (peut setup dès Phase 1)
    16. TESTING_STRATEGY (après chaque module)
    17. SECURITY (après chaque module)
    18. PERFORMANCE (Phase 6)
    19. API_REFERENCE (continu)
```

### Opportunités de Parallélisation

✅ **Parallèle dès le début:**
- `02. BACKEND_FOUNDATION` ⟷ `05. FRONTEND_FOUNDATION`
- `14. INFRASTRUCTURE_PRODUCTION` (setup initial)
- `15. CICD_DEPLOYMENT` (workflows de base)

✅ **Parallèle après Phase 1:**
- `06. CV_API` ⟷ `08. AI_SERVICES` ⟷ `11. ANALYTICS_API`

✅ **Parallèle continu:**
- `16. TESTING_STRATEGY` (tests écrits au fur et à mesure)
- `17. SECURITY` (revues de sécurité régulières)
- `19. API_REFERENCE` (documentation API progressive)

---

## 📚 Catalogue Détaillé des Documents

### PHASE 1 - MVP FOUNDATION

#### 📦 01. SETUP_INFRASTRUCTURE.md

**Métadonnées:**
- Phase: 1
- Priorité: 🔴 CRITIQUE
- Complexité: ⭐⭐ (2/5)
- Prérequis: Aucun
- Temps estimé: 1-2 jours

**Contenu:**
- Architecture Docker Compose (4 services: backend, frontend, postgres, redis)
- PostgreSQL: configuration, volumes, stratégie backup
- Redis: configuration, persistence RDB/AOF
- Network Docker & communication inter-services
- Variables d'environnement (.env.example)
- Scripts de vérification santé (health checks)

**Livrables:**
- `docker-compose.yml`
- `.env.example`
- `scripts/health-check.sh`

---

#### 🔧 02. BACKEND_FOUNDATION.md

**Métadonnées:**
- Phase: 1
- Priorité: 🔴 CRITIQUE
- Complexité: ⭐⭐⭐ (3/5)
- Prérequis: 01
- Temps estimé: 2-3 jours

**Contenu:**
- Structure projet Go (`internal/`, `cmd/`, `pkg/`)
- Fiber setup & configuration (port, timeouts, limits)
- Connexion PostgreSQL avec GORM
- Connexion Redis avec go-redis
- Logger (zerolog ou zap)
- Error handling global (custom error types)
- Configuration management (viper ou env)
- Health check endpoint `GET /health`

**Livrables:**
- `backend/cmd/main.go`
- `backend/internal/config/`
- `backend/internal/database/`
- `backend/pkg/logger/`
- `go.mod` avec dépendances

---

#### 📊 03. DATABASE_SCHEMA.md

**Métadonnées:**
- Phase: 1
- Priorité: 🔴 CRITIQUE
- Complexité: ⭐⭐⭐⭐ (4/5)
- Prérequis: 02
- Temps estimé: 3-4 jours

**Contenu:**
- Models GORM pour 8 tables:
  - `experiences` (parcours professionnel)
  - `skills` (compétences)
  - `projects` (projets)
  - `generated_letters` (historique lettres IA)
  - `visitors` (tracking)
  - `analytics_events` (événements)
  - `cv_themes` (thèmes CV)
  - `github_repos` (import GitHub)
- Relations & associations (Has Many, Belongs To, Many2Many)
- Migrations SQL (golang-migrate)
- Indexes (performance) & constraints (intégrité)
- Seed data (fixtures pour dev/test)
- Schema versioning strategy

**Livrables:**
- `backend/internal/models/`
- `backend/migrations/`
- `backend/scripts/seed.go`
- Diagramme ERD (Entity-Relationship Diagram)

---

#### 🛡️ 04. BACKEND_MIDDLEWARES.md

**Métadonnées:**
- Phase: 1
- Priorité: 🟡 HAUTE
- Complexité: ⭐⭐⭐ (3/5)
- Prérequis: 02, 03
- Temps estimé: 2-3 jours

**Contenu:**
- CORS configuration fine (origins autorisées)
- Tracking visiteurs:
  - Génération cookie session
  - Incrémentation compteur Redis
  - Détection profil (User-Agent, IP lookup)
- Rate limiting:
  - Global (100 req/min par IP)
  - Par endpoint (AI: 5/jour)
  - Implémentation Redis (token bucket)
- Request ID (tracing)
- Logging HTTP (request/response)
- Recovery (panic handling)

**Livrables:**
- `backend/internal/middleware/cors.go`
- `backend/internal/middleware/tracking.go`
- `backend/internal/middleware/ratelimit.go`
- `backend/internal/middleware/logger.go`
- `backend/internal/middleware/recovery.go`

---

#### 🎨 05. FRONTEND_FOUNDATION.md

**Métadonnées:**
- Phase: 1
- Priorité: 🔴 CRITIQUE
- Complexité: ⭐⭐⭐ (3/5)
- Prérequis: 01 (peut être parallèle à 02-04)
- Temps estimé: 2-3 jours

**Contenu:**
- Next.js 14 setup (App Router, TypeScript)
- Tailwind CSS configuration:
  - Palette de couleurs custom
  - Dark mode (class strategy)
  - Fonts (Inter, Poppins)
- Structure projet:
  - `app/` (pages)
  - `components/` (UI components)
  - `lib/` (utilities, API client)
- API client wrapper:
  - Fetch wrapper avec retry
  - Error handling centralisé
  - Types TypeScript
- Loading & error states (Suspense, Error Boundaries)
- Layout principal (header, footer, navigation)
- shadcn/ui setup (boutons, cards, dialogs, etc.)

**Livrables:**
- `frontend/app/layout.tsx`
- `frontend/lib/api.ts`
- `frontend/components/ui/`
- `tailwind.config.ts`
- `package.json` avec dépendances

---

### PHASE 2 - CV DYNAMIQUE

#### 🎯 06. BACKEND_CV_API.md

**Métadonnées:**
- Phase: 2
- Priorité: 🟡 HAUTE
- Complexité: ⭐⭐⭐⭐ (4/5)
- Prérequis: 04
- Temps estimé: 3-5 jours

**Contenu:**
- Algorithme de filtrage/scoring:
  - Tags matching
  - Pondération par thème
  - Tri par pertinence
- Endpoints:
  - `GET /api/cv?theme=backend` (CV adapté)
  - `GET /api/cv/themes` (liste thèmes disponibles)
  - `GET /api/experiences` (toutes expériences)
  - `GET /api/skills` (toutes compétences)
  - `GET /api/projects` (tous projets)
  - `GET /api/cv/export?theme=backend&format=pdf` (export PDF)
- Export PDF basique (gofpdf ou chromedp)
- Caching Redis (CV par thème, TTL 1h)
- Tests unitaires (algorithme scoring)

**Livrables:**
- `backend/internal/api/cv.go`
- `backend/internal/services/cv_scoring.go`
- `backend/internal/services/pdf.go`
- Tests

---

#### 💼 07. FRONTEND_CV_DYNAMIC.md

**Métadonnées:**
- Phase: 2
- Priorité: 🟡 HAUTE
- Complexité: ⭐⭐⭐⭐ (4/5)
- Prérequis: 05, 06
- Temps estimé: 4-5 jours

**Contenu:**
- Page `/cv` avec query params (`?theme=backend`)
- Components:
  - `CVThemeSelector` (dropdown thèmes + preview)
  - `ExperienceTimeline` (timeline verticale avec Framer Motion)
  - `SkillsCloud` (tag cloud interactif avec niveaux)
  - `ProjectsGrid` (grid de cards GitHub avec stars/languages)
- Export PDF button (téléchargement via API)
- Animations Framer Motion:
  - Transitions entre thèmes
  - Scroll animations
  - Hover effects
- Responsive design (mobile, tablet, desktop)
- SEO optimization (metadata dynamique)

**Livrables:**
- `frontend/app/cv/page.tsx`
- `frontend/components/cv/`
- Tests e2e (Playwright)

---

### PHASE 3 - IA LETTRES

#### 🤖 08. BACKEND_AI_SERVICES.md

**Métadonnées:**
- Phase: 3
- Priorité: 🟡 HAUTE
- Complexité: ⭐⭐⭐⭐⭐ (5/5)
- Prérequis: 04
- Temps estimé: 5-7 jours

**Contenu:**
- **Service IA:**
  - Client Claude (Anthropic API)
  - Client GPT-4 (OpenAI API)
  - Fallback strategy (Claude → GPT-4 si erreur)
  - Streaming responses (Server-Sent Events)
  - Error handling & retry logic (exponential backoff)
- **Prompts Engineering:**
  - Prompt lettre motivation (professionnel, structuré)
  - Prompt lettre anti-motivation (humoristique, créatif)
  - Variables dynamiques (nom entreprise, infos, profil)
  - Few-shot examples
- **Service Scraper:**
  - Scraping infos entreprises (site web, LinkedIn)
  - API alternatives (Clearbit, Hunter.io)
  - Parsing & extraction données clés
  - Caching résultats (Redis, TTL 7j)
- **Service PDF Lettres:**
  - chromedp (rendu HTML→PDF)
  - Templates HTML lettres (design soigné)
  - Dual PDF (motivation + anti-motivation)
- **Cost Tracking:**
  - Logs tokens utilisés
  - Métriques Prometheus (coûts estimés)

**Livrables:**
- `backend/internal/services/ai.go`
- `backend/internal/services/scraper.go`
- `backend/internal/services/pdf_letters.go`
- `backend/templates/letter_*.html`
- Tests (mocks API)

---

#### ✉️ 09. BACKEND_LETTERS_API.md

**Métadonnées:**
- Phase: 3
- Priorité: 🟡 HAUTE
- Complexité: ⭐⭐⭐⭐ (4/5)
- Prérequis: 08
- Temps estimé: 3-4 jours

**Contenu:**
- Endpoints:
  - `POST /api/letters/generate` (génération lettres)
  - `GET /api/letters/:id` (récupération lettre)
  - `GET /api/letters/:id/pdf` (téléchargement PDF)
  - `GET /api/letters/history` (historique utilisateur)
- **Access Control:**
  - Middleware vérification (3 visites OU profil détecté)
  - Réponse 403 avec teaser si accès refusé
- **Rate Limiting:**
  - Max 5 générations/jour par session
  - Cooldown 2 minutes entre générations
  - Messages d'erreur clairs (retry-after header)
- **Queue système:**
  - Jobs asynchrones (génération longue)
  - Status polling endpoint
  - WebSocket pour notifications temps réel
- **Caching:**
  - Lettres en cache Redis (par entreprise + hash profil)
  - TTL 24h
- Historique PostgreSQL (tracking génération)

**Livrables:**
- `backend/internal/api/letters.go`
- `backend/internal/middleware/access_gate.go`
- Tests integration

---

#### 📝 10. FRONTEND_LETTERS.md

**Métadonnées:**
- Phase: 3
- Priorité: 🟡 HAUTE
- Complexité: ⭐⭐⭐⭐ (4/5)
- Prérequis: 05, 09
- Temps estimé: 4-5 jours

**Contenu:**
- Page `/letters`
- **LetterGenerator Component:**
  - Form (nom entreprise)
  - Validation Zod
  - Submit avec loading state
  - Error handling (403, 429, etc.)
- **LetterPreview Component:**
  - Affichage dual (2 colonnes)
  - Lettre Motivation (gauche)
  - Lettre Anti-Motivation (droite)
  - Markdown rendering
  - Export PDF buttons (individuels + dual)
- **Access Gate:**
  - Teaser si < 3 visites
  - Compteur visites affiché
  - Message "Encore X visites avant déblocage"
- **Loading States:**
  - Skeleton pendant génération
  - Progress bar
  - Animation "IA en train de travailler"
- **Error Handling:**
  - Messages d'erreur contextuels
  - Retry button
  - Rate limit countdown

**Livrables:**
- `frontend/app/letters/page.tsx`
- `frontend/components/letters/LetterGenerator.tsx`
- `frontend/components/letters/LetterPreview.tsx`
- `frontend/components/letters/AccessGate.tsx`
- Tests e2e

---

### PHASE 4 - ANALYTICS

#### 📈 11. BACKEND_ANALYTICS.md

**Métadonnées:**
- Phase: 4
- Priorité: 🟢 MOYENNE
- Complexité: ⭐⭐⭐⭐ (4/5)
- Prérequis: 04
- Temps estimé: 4-5 jours

**Contenu:**
- **Service Analytics:**
  - Collecte événements (page views, clicks, etc.)
  - Enregistrement PostgreSQL (analytics_events)
  - Agrégations Redis (temps réel)
- **Endpoints:**
  - `GET /api/analytics/realtime` (visiteurs actuels, 5s refresh)
  - `GET /api/analytics/stats?period=day|week|month` (agrégations)
  - `GET /api/analytics/themes` (top thèmes CV consultés)
  - `GET /api/analytics/letters` (nb lettres générées)
  - `POST /api/analytics/event` (enregistrement événement custom)
- **WebSocket:**
  - `/ws/analytics` (broadcast temps réel)
  - Pub/Sub Redis (communication multi-instances)
  - Heartbeat mechanism
- **Agrégations Redis:**
  - HyperLogLog (comptage unique visiteurs)
  - Sorted Sets (top thèmes)
  - Hashes (stats jour/semaine/mois)
  - TTL adapté (cleanup auto)
- **Métriques Prometheus:**
  - Custom metrics (visitors_total, letters_generated_total, etc.)
  - Histogram (response times)
  - Gauge (visiteurs actuels)
- **Data Retention:**
  - Événements bruts: 90 jours
  - Agrégations: 1 an
  - Archivage optionnel (S3)

**Livrables:**
- `backend/internal/services/analytics.go`
- `backend/internal/api/analytics.go`
- `backend/internal/websocket/analytics.go`
- Tests

---

#### 📊 12. FRONTEND_ANALYTICS_DASHBOARD.md

**Métadonnées:**
- Phase: 4
- Priorité: 🟢 MOYENNE
- Complexité: ⭐⭐⭐⭐ (4/5)
- Prérequis: 05, 11
- Temps estimé: 4-5 jours

**Contenu:**
- Page `/analytics` (dashboard public)
- **Components:**
  - `RealtimeVisitors`:
    - Nombre visiteurs actuels (gros chiffre animé)
    - WebSocket connection
    - Auto-reconnect
  - `ThemeStats`:
    - Bar chart (Chart.js ou Recharts)
    - Top 5 thèmes CV consultés
    - Pourcentages
  - `LettersGenerated`:
    - Line chart (évolution dans le temps)
    - Filters (jour/semaine/mois)
  - `Heatmap`:
    - Carte cliquable des interactions
    - Gradient de chaleur
  - `VisitorFlow`:
    - Sankey diagram (parcours utilisateurs)
- **Filters:**
  - Date range picker
  - Groupement (heure/jour/semaine)
- **Auto-refresh:**
  - Polling 30s (stats agrégées)
  - WebSocket (temps réel)
- **Responsive:**
  - Grid adaptatif
  - Mobile-friendly charts
- **Animations:**
  - CountUp (chiffres)
  - Smooth transitions (Framer Motion)

**Livrables:**
- `frontend/app/analytics/page.tsx`
- `frontend/components/analytics/`
- Tests e2e

---

### PHASE 5 - FEATURES AVANCÉES

#### 🚀 13. FEATURES_ADVANCED.md

**Métadonnées:**
- Phase: 5
- Priorité: 🔵 BASSE
- Complexité: ⭐⭐⭐ (3/5)
- Prérequis: 06, 07
- Temps estimé: Variable selon features

**Contenu:**
- **Import Automatique GitHub:**
  - OAuth GitHub
  - API GitHub (repos publics)
  - Sync automatique (cron job)
  - Affichage stars, languages, description
  - Mise à jour incrémentale
- **Timeline Interactive:**
  - Timeline horizontale/verticale (responsive)
  - Animations scroll (Intersection Observer)
  - Filtrage par catégorie (backend, frontend, etc.)
  - Zoom sur expérience (modal détails)
- **Détection Profils Avancée:**
  - Clearbit API (IP → entreprise)
  - LinkedIn Sales Navigator detection
  - Patterns User-Agent (bots recruteurs)
  - Enrichissement données visiteur
  - Notifications (nouveau profil cible)
- **Effets 3D (optionnel):**
  - Three.js / React Three Fiber
  - Avatar 3D interactif
  - Visualisation 3D compétences (graphe)
  - Particules (background animé)
- **Multi-langue:**
  - i18n setup (next-intl)
  - Traduction IA (Claude pour contenu dynamique)
  - FR/EN switch
  - SEO multi-langue

**Livrables:**
- Selon features choisies
- Documentation dédiée par feature

---

### PHASE 6 - PRODUCTION & QUALITÉ

#### 🚀 14. INFRASTRUCTURE_PRODUCTION.md

**Métadonnées:**
- Phase: 6
- Priorité: 🔴 CRITIQUE
- Complexité: ⭐⭐⭐⭐ (4/5)
- Prérequis: Tous modules fonctionnels
- Temps estimé: 3-5 jours

**Contenu:**
- **Nginx:**
  - Configuration reverse proxy
  - SSL/TLS (Let's Encrypt + certbot)
  - HTTPS redirect (HTTP → HTTPS)
  - Compression gzip/brotli
  - Rate limiting (nginx level)
  - Static files caching
  - Security headers (CSP, HSTS, X-Frame-Options)
- **Prometheus:**
  - Configuration scraping (backends, redis, postgres)
  - Service discovery
  - Retention policy (15 jours)
  - Alerting rules (optionnel)
- **Grafana:**
  - Dashboards JSON (import/export)
  - Dashboard public (readonly)
  - Panels:
    - Visiteurs temps réel
    - Request rate (RPS)
    - Response times (P50, P95, P99)
    - Error rates
    - Database connections
    - Redis memory usage
  - Alerting (optionnel: Slack, Discord)
- **Health Checks:**
  - Kubernetes-style probes (liveness, readiness)
  - `/health` endpoint (shallow check)
  - `/health/deep` endpoint (DB connections, Redis)
- **Logging:**
  - Structured logs JSON
  - Stdout/stderr (Docker logs)
  - Loki optionnel (centralisé)
  - Log rotation
- **Backups:**
  - PostgreSQL: pg_dump quotidien (cron)
  - Redis: RDB snapshots
  - Stockage backups (S3 ou local)
  - Restore testing

**Livrables:**
- `docker/nginx/nginx.conf`
- `monitoring/prometheus/prometheus.yml`
- `monitoring/grafana/dashboards/*.json`
- `scripts/backup.sh`
- Documentation restore

---

#### ⚙️ 15. CICD_DEPLOYMENT.md

**Métadonnées:**
- Phase: 6
- Priorité: 🟡 HAUTE
- Complexité: ⭐⭐⭐ (3/5)
- Prérequis: 14
- Temps estimé: 2-3 jours

**Contenu:**
- **GitHub Actions Workflows:**
  - `.github/workflows/ci.yml` (tests)
  - `.github/workflows/deploy.yml` (déploiement)
  - `.github/workflows/backup.yml` (backup hebdo)
- **CI Workflow:**
  - Triggers: push, pull_request
  - Jobs:
    - Backend tests (go test -v ./...)
    - Frontend tests (npm test)
    - Linting (golangci-lint, eslint)
    - Security scan (gosec, npm audit)
    - Build Docker images (cache layers)
- **Deploy Workflow:**
  - Trigger: push sur main
  - Jobs:
    - Build & push images (Docker Hub ou GitHub Registry)
    - SSH vers VPS
    - Pull images
    - Docker Compose down/up (rolling update)
    - Health check (curl /health)
    - Rollback si échec
- **Secrets Management:**
  - GitHub Secrets (API keys, SSH keys)
  - .env généré dynamiquement
- **Status Badges:**
  - CI status
  - Deployment status
  - Coverage badge (codecov)
- **Notifications:**
  - Discord/Slack webhook (succès/échec deploy)

**Livrables:**
- `.github/workflows/*.yml`
- `scripts/deploy.sh`
- Documentation rollback

---

#### 🧪 16. TESTING_STRATEGY.md

**Métadonnées:**
- Phase: 6
- Priorité: 🟡 HAUTE
- Complexité: ⭐⭐⭐⭐ (4/5)
- Prérequis: Tous modules fonctionnels
- Temps estimé: Continu (tests écrits au fur et à mesure)

**Contenu:**
- **Tests Unitaires Backend (Go):**
  - Framework: testify
  - Tests services (mocks API IA, DB)
  - Tests algorithmes (scoring CV)
  - Coverage target: 80%+
- **Tests Integration Backend:**
  - PostgreSQL (testcontainers)
  - Redis (miniredis ou testcontainers)
  - Tests end-to-end API (routes complètes)
- **Tests Frontend:**
  - Unit: Jest + React Testing Library
  - Components isolation
  - Hooks testing
  - Coverage target: 70%+
- **Tests E2E:**
  - Framework: Playwright
  - Scénarios critiques:
    - Parcours complet génération lettre
    - Navigation CV avec thèmes
    - Dashboard analytics
  - Tests multi-navigateurs
- **Fixtures & Mocks:**
  - Seed data test
  - Mock API IA (réponses pré-enregistrées)
  - Mock scraper (données entreprises)
- **CI Integration:**
  - Tests auto sur chaque PR
  - Coverage reports (codecov)
  - Fail si coverage < threshold
- **Performance Tests:**
  - Load testing (k6, wrk)
  - Stress testing
  - Benchmarks Go (testing.B)

**Livrables:**
- Tests dans chaque module
- `backend/internal/*_test.go`
- `frontend/**/*.test.tsx`
- `e2e/` folder (Playwright tests)
- Documentation tests

---

#### 🔒 17. SECURITY.md

**Métadonnées:**
- Phase: 6
- Priorité: 🔴 CRITIQUE
- Complexité: ⭐⭐⭐⭐ (4/5)
- Prérequis: Tous modules fonctionnels
- Temps estimé: 2-3 jours (audit + fixes)

**Contenu:**
- **Input Validation:**
  - Backend: validation Fiber (validator)
  - Frontend: validation Zod (schemas réutilisables)
  - Whitelist (pas blacklist)
  - Sanitization HTML (bluemonday)
- **OWASP Top 10 Checklist:**
  - ✅ A01: Broken Access Control → middleware auth
  - ✅ A02: Cryptographic Failures → HTTPS, secrets vault
  - ✅ A03: Injection → prepared statements, sanitization
  - ✅ A04: Insecure Design → architecture review
  - ✅ A05: Security Misconfiguration → hardening
  - ✅ A06: Vulnerable Components → dependency scanning
  - ✅ A07: Auth Failures → rate limiting, secure cookies
  - ✅ A08: Software & Data Integrity → checksums, signing
  - ✅ A09: Logging Failures → structured logs, monitoring
  - ✅ A10: SSRF → URL validation, whitelist
- **Secrets Management:**
  - .env (NEVER commit)
  - Vault optionnel (HashiCorp Vault)
  - Rotation API keys
- **HTTPS Enforcement:**
  - Redirect HTTP → HTTPS (Nginx)
  - HSTS header (Strict-Transport-Security)
  - Certificate auto-renewal (certbot)
- **Rate Limiting:**
  - Global: 100 req/min
  - AI endpoints: 5 req/jour
  - Login endpoints: 5 req/15min
  - DDoS mitigation (Cloudflare optionnel)
- **CORS:**
  - Whitelist origins (pas wildcard *)
  - Credentials: true (si cookies)
- **Security Headers:**
  - Content-Security-Policy (CSP)
  - X-Frame-Options: DENY
  - X-Content-Type-Options: nosniff
  - Referrer-Policy: strict-origin
- **Dependency Scanning:**
  - Go: gosec, go mod audit
  - NPM: npm audit, Snyk
  - Docker: trivy (scan images)
- **Penetration Testing:**
  - OWASP ZAP (automated scan)
  - Manual testing (top endpoints)

**Livrables:**
- Checklist sécurité complète
- Rapport audit
- Fixes implémentés
- Documentation bonnes pratiques

---

#### ⚡ 18. PERFORMANCE.md

**Métadonnées:**
- Phase: 6
- Priorité: 🟢 MOYENNE
- Complexité: ⭐⭐⭐ (3/5)
- Prérequis: Tous modules fonctionnels
- Temps estimé: 2-3 jours

**Contenu:**
- **Caching Strategies:**
  - Redis:
    - CV par thème (TTL 1h)
    - Lettres générées (TTL 24h)
    - Infos entreprises (TTL 7j)
  - HTTP caching headers (Cache-Control, ETag)
  - CDN optionnel (Cloudflare, CloudFront)
- **Database Optimization:**
  - Indexes sur colonnes fréquentes (WHERE, JOIN)
  - EXPLAIN ANALYZE (requêtes lentes)
  - Connection pooling (pgx pool)
  - Read replicas optionnel (scale lecture)
- **Frontend Performance:**
  - Next.js Image (lazy loading, formats modernes)
  - Code splitting (dynamic imports)
  - Tree shaking (unused code)
  - Lazy loading components (below fold)
  - Prefetching (next/link)
  - Bundle size analysis (webpack-bundle-analyzer)
- **API Optimization:**
  - Pagination (LIMIT/OFFSET)
  - Field selection (GraphQL-style)
  - Compression (gzip/brotli)
  - Keep-Alive connections
- **Benchmarks:**
  - Backend: wrk, k6 (RPS, latency)
  - Target: < 100ms P95 response time
  - Load testing (1000 concurrent users)
- **Profiling:**
  - Go: pprof (CPU, memory, goroutines)
  - Frontend: Chrome DevTools (Lighthouse)
  - Metrics Prometheus (tracking regressions)
- **Optimizations Identified:**
  - N+1 queries → preloading
  - Large JSON responses → pagination
  - Slow PDF generation → queue + async
  - Heavy frontend bundles → code splitting

**Livrables:**
- Rapport benchmarks
- Optimizations appliquées
- Documentation performance best practices
- Monitoring dashboards (Grafana)

---

### ANNEXES

#### 📖 19. API_REFERENCE.md

**Métadonnées:**
- Phase: Continu
- Priorité: 🟢 MOYENNE
- Complexité: ⭐⭐ (2/5)
- Prérequis: Toutes APIs implémentées
- Temps estimé: 1-2 jours (setup auto-génération)

**Contenu:**
- **OpenAPI/Swagger Specification:**
  - Auto-génération (swaggo/swag pour Go)
  - Spec OpenAPI 3.0
- **Documentation par Endpoint:**
  - HTTP method, URL, description
  - Request params/body (schemas)
  - Response schemas (success, errors)
  - Exemples curl
  - Codes erreur (400, 401, 403, 429, 500)
- **Authentication:**
  - Cookies (session tracking)
  - Rate limiting headers
- **Pagination:**
  - Query params (page, limit)
  - Response format (data, meta)
- **Erreurs:**
  - Format standardisé JSON
  - Error codes custom
- **Swagger UI:**
  - Interface interactive (test endpoints)
  - Hosted sur `/api/docs`

**Livrables:**
- `docs/api/openapi.yaml`
- Swagger UI
- Exemples curl (README)

---

## 📐 Structure Standardisée de Chaque Document

Tous les documents d'implémentation suivront cette structure:

```markdown
# [TITRE DU DOCUMENT]

## 📋 Métadonnées

- **Phase:** X
- **Priorité:** CRITIQUE / HAUTE / MOYENNE / BASSE
- **Complexité:** ⭐⭐⭐ (X/5)
- **Prérequis:** Liste des docs à terminer avant
- **Temps estimé:** X jours
- **Status:** 🔲 À faire / 🔄 En cours / ✅ Terminé

---

## 🎯 Objectif

Description claire et concise de ce qui sera implémenté dans ce document.

---

## 🏗️ Architecture

### Vue d'Ensemble

Schéma, diagramme ou description de l'architecture du composant.

### Design Decisions

Choix techniques et justifications.

---

## 📦 Dépendances

### Bibliothèques Go

```bash
go get github.com/example/package
```

### Bibliothèques NPM

```bash
npm install example-package
```

### Services Externes

- API X (clé requise)
- Service Y

---

## 🔨 Implémentation

### Étape 1: [Titre Étape]

**Description:** Quoi faire

**Code:**

```go
// Code example
```

**Explications:** Pourquoi et comment

---

### Étape 2: [Titre Étape]

...

---

## 🧪 Tests

### Tests Unitaires

```go
// Test example
```

### Tests Integration

...

### Commandes

```bash
# Run tests
go test -v ./...
```

---

## ⚠️ Points d'Attention

- ⚠️ **Piège 1:** Description du piège à éviter
- ⚠️ **Edge case:** Cas particulier à gérer
- 💡 **Astuce:** Optimisation ou bonne pratique

---

## 📚 Ressources

- [Documentation officielle](https://example.com)
- [Tutoriel](https://example.com)
- [Article de blog](https://example.com)

---

## ✅ Checklist de Complétion

- [ ] Code implémenté
- [ ] Tests écrits et passants
- [ ] Documentation code (commentaires)
- [ ] Review sécurité
- [ ] Review performance
- [ ] Commit & Push

---

**Dernière mise à jour:** YYYY-MM-DD
**Auteur:** Alexi
```

---

## ⏱️ Timeline Suggérée

### Phase 1 - MVP (2-3 semaines)

**Semaine 1:**
- Jours 1-2: Doc 01 (Infrastructure)
- Jours 3-5: Doc 02 (Backend Foundation)
- Jours 3-5: Doc 05 (Frontend Foundation) ⟷ Parallèle

**Semaine 2:**
- Jours 1-4: Doc 03 (Database Schema)
- Jours 5-7: Doc 04 (Middlewares)

**Semaine 3:**
- Tests integration Phase 1
- Fixes bugs
- Documentation

---

### Phase 2 - CV Dynamique (2 semaines)

**Semaine 1:**
- Jours 1-5: Doc 06 (Backend CV API)

**Semaine 2:**
- Jours 1-5: Doc 07 (Frontend CV)
- Tests e2e

---

### Phase 3 - IA Lettres (3 semaines)

**Semaine 1-2:**
- Jours 1-7: Doc 08 (AI Services) - Complexe
- Jours 1-4: Doc 09 (Letters API) ⟷ Peut démarrer en parallèle après services AI

**Semaine 3:**
- Jours 1-5: Doc 10 (Frontend Letters)
- Tests e2e

---

### Phase 4 - Analytics (2 semaines)

**Semaine 1:**
- Jours 1-5: Doc 11 (Backend Analytics)

**Semaine 2:**
- Jours 1-5: Doc 12 (Frontend Dashboard)
- Tests

---

### Phase 5 - Features Avancées (Variable)

- À planifier selon features choisies
- 1-3 semaines selon scope

---

### Phase 6 - Production (2 semaines)

**Semaine 1:**
- Jours 1-3: Doc 14 (Infrastructure Prod)
- Jours 4-5: Doc 15 (CI/CD)
- Jours 1-5: Doc 16, 17 (Tests, Security) ⟷ Parallèle

**Semaine 2:**
- Jours 1-2: Doc 18 (Performance)
- Jours 3-5: Doc 19 (API Reference)
- Final testing
- Déploiement production

---

## 🎯 Conseils d'Implémentation

### 1. Suivre l'Ordre de Dépendances

Ne pas sauter d'étapes. Respecter les prérequis pour éviter les refactorisations.

### 2. Maximiser la Parallélisation

Identifier tôt ce qui peut être fait en parallèle (backend ⟷ frontend).

### 3. Tests en Continu

Écrire les tests au fur et à mesure, pas à la fin. TDD recommandé.

### 4. Documentation Progressive

Documenter pendant le développement, pas après (sinon oubli).

### 5. Revues de Code

Self-review avant commit. Checklist sécurité + performance.

### 6. Commits Atomiques

Un commit = une feature/fix. Messages clairs (Conventional Commits).

### 7. CI/CD dès le Début

Setup workflows GitHub Actions dès Phase 1 (tests auto).

### 8. Monitoring dès MVP

Prometheus + Grafana peuvent être configurés dès Phase 1 (itératif).

### 9. Sécurité by Design

Ne pas attendre Phase 6 pour penser sécurité. Valider inputs dès le début.

### 10. Itérations Courtes

Déployer souvent (daily/weekly). Feedback rapide.

---

## 📊 Tracking Progress

### Utilisation d'un Kanban

Outil recommandé: GitHub Projects

**Colonnes:**
- 📋 Backlog
- 🔄 In Progress
- 👀 Review
- ✅ Done

**Cards:** Un card par document d'implémentation

---

### Métriques de Suivi

- **Velocity:** Nombre de docs complétés par semaine
- **Coverage:** Pourcentage de tests
- **Bugs:** Nombre de bugs ouverts vs fermés
- **Déploiements:** Fréquence (target: 1x/semaine minimum)

---

## 🔄 Processus d'Itération

Ce plan n'est pas figé. Iterations attendues:

1. **Review après Phase 1:** Ajustements architecture si nécessaire
2. **Review après Phase 3:** Feedback utilisateurs beta (si applicable)
3. **Review avant Phase 6:** Audit sécurité complet
4. **Post-launch:** Roadmap features futures

---

## 📝 Changelog du Plan

| Version | Date       | Changements                          |
|---------|------------|--------------------------------------|
| 1.0     | 2025-12-08 | Création initiale (19 documents)     |

---

**Prochaine Étape:** Commencer la rédaction de `01. SETUP_INFRASTRUCTURE.md`

---

**Questions / Feedback:** [Ajouter lien issue tracker ou contact]
