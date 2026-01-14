# maicivy - My CV AI

**Tagline:** CV interactif intelligent avec génération de lettres de motivation/anti-motivation par IA

---

## 🎯 Vision du Projet

Un CV en ligne qui sert de démo technique complète, démontrant des compétences en :
- Backend (Go)
- Frontend (Next.js + TypeScript)
- Intelligence Artificielle (génération de contenu)
- DevOps (Docker, CI/CD, monitoring)
- Architecture système (PostgreSQL, Redis, APIs)

---

## 🏗️ Architecture Technique

### Stack

**Backend:**
- Language: Go
- Framework: Fiber
- Base de données: PostgreSQL (données principales)
- Cache: Redis (sessions, rate-limiting, compteur visites)

**Frontend:**
- Framework: Next.js 14 (App Router)
- Language: TypeScript
- Styling: Tailwind CSS
- Animations: Framer Motion
- Effets 3D (optionnel): Three.js / React Three Fiber

**IA:**
- API: Claude (Anthropic) et/ou GPT-4 (OpenAI)
- Use cases: Génération lettres motivation/anti-motivation

**Infrastructure:**
- Conteneurisation: Docker + Docker Compose
- Reverse Proxy: Nginx
- SSL: Let's Encrypt (HTTPS)
- Hébergement: VPS OVH
- CI/CD: GitHub Actions + Gitea
- Monitoring: Prometheus + Grafana (dashboard public)

---

## 🚀 Fonctionnalités Principales

### 1. CV Dynamique Adaptatif

Le CV se personnalise automatiquement selon le contexte demandé :

- **CV Backend** : met en avant expériences/compétences backend (Go, Node.js, APIs, databases)
- **CV C++** : focus sur expériences C++, systèmes bas niveau
- **CV Artistique** : met en avant créativité, design, projets visuels
- **CV Full-Stack**, **CV DevOps**, etc.

**Mécanisme:**
- Base de données contient TOUTES les expériences/compétences avec tags
- Algorithme de filtrage/scoring adapte le contenu selon le thème
- Interface permet de sélectionner le thème ou URL paramétrable (`/cv?theme=backend`)

**Export PDF:**
- Génération PDF du CV personnalisé selon thème choisi
- Design professionnel adapté au print

### 2. Générateur de Lettres par IA

**Fonctionnalité signature du projet:**

**Input:**
- Nom de l'entreprise (champ texte)

**Process:**
1. Recherche automatique d'informations sur l'entreprise (API ou scraping)
2. Analyse IA pour identifier:
   - Ce qui vous rendrait bon pour cette entreprise
   - Ce qui vous plairait dans ce poste/entreprise
   - Match entre votre profil et leurs besoins
3. Génération de **deux lettres** :
   - ✅ **Lettre de Motivation** classique et professionnelle
   - ❌ **Lettre d'Anti-Motivation** humoristique et créative

**Export:**
- PDF de chaque lettre avec design soigné

**Restrictions d'accès:**
- Accessible seulement à partir de la **3ème visite** (tracking via cookies + Redis)
- **Exception** : détection automatique de profils cibles (recruteurs, tech leads, dirigeants) → accès immédiat dès la 1ère visite
- Rate limiting pour contrôler coûts API IA

### 3. Analytics Publiques en Temps Réel

Dashboard de monitoring visible par tous les visiteurs :

**Métriques affichées:**
- Nombre de visiteurs actuels (temps réel)
- Total de visites (jour/semaine/mois)
- Top thèmes CV les plus consultés
- Nombre de lettres générées (anonymisé)
- Heatmap des clics/interactions
- Graphiques de visualisation de données

**Technologies:**
- Prometheus (collecte métriques)
- Grafana (visualisation) ou dashboard custom (Chart.js, D3.js)
- WebSocket pour mise à jour temps réel

### 4. Import Automatique Projets GitHub

- Connexion API GitHub
- Import automatique de vos projets publics
- Affichage avec stars, languages, description
- Mise à jour automatique

### 5. Timeline Interactive

- Visualisation chronologique des expériences professionnelles
- Animations au scroll
- Filtrage par catégorie/technologie

---

## 🗄️ Modèle de Données

### PostgreSQL Tables

```sql
-- Expériences professionnelles
experiences (
  id, title, company, description, start_date, end_date,
  technologies[], tags[], category
)

-- Compétences
skills (
  id, name, level, category, tags[], years_experience
)

-- Projets
projects (
  id, title, description, github_url, demo_url,
  technologies[], category, featured
)

-- Lettres générées (historique)
generated_letters (
  id, company_name, letter_type, content,
  visitor_id, created_at
)

-- Analytics visiteurs
visitors (
  id, session_id, ip_hash, user_agent,
  visit_count, first_visit, last_visit,
  profile_detected
)

-- Analytics events
analytics_events (
  id, visitor_id, event_type, event_data,
  created_at
)
```

### Redis Keys

```
visitor:{session_id}:count        # Compteur de visites
visitor:{session_id}:profile      # Profil détecté (recruteur, etc.)
ratelimit:ai:{session_id}         # Rate limiting génération IA
analytics:realtime:visitors       # Set des visiteurs actuels
analytics:stats:cv_themes         # Hash des thèmes consultés
```

---

## 🔐 Système de Tracking et Accès IA

### Tracking Visiteurs

1. **Cookie de session** : identifiant unique visiteur
2. **Compteur Redis** : nombre de visites par session
3. **Détection de profil** :
   - Analyse User-Agent
   - Lookup IP → entreprise (via API type Clearbit)
   - Détection LinkedIn referrer
   - Patterns de navigation

### Règles d'Accès IA

```
SI visite_count >= 3 OU profile_detected IN ['recruiter', 'tech_lead', 'cto', 'ceo']
  ALORS activer_fonctionnalités_IA()
SINON
  afficher_teaser("Fonctionnalités IA disponibles à partir de la 3ème visite")
```

### Rate Limiting

- Max générations IA par session : 5/jour
- Cooldown entre générations : 2 minutes
- Protection contre abus (coûts API)

---

## 📦 Structure du Projet

```
maicivy/
├── backend/                    # API Go
│   ├── cmd/
│   │   └── main.go            # Entry point
│   ├── internal/
│   │   ├── api/               # HTTP handlers
│   │   │   ├── cv.go          # CV endpoints
│   │   │   ├── letters.go     # Génération lettres IA
│   │   │   ├── analytics.go   # Analytics endpoints
│   │   │   └── pdf.go         # Export PDF
│   │   ├── services/
│   │   │   ├── ai.go          # Service IA (Claude/GPT)
│   │   │   ├── scraper.go     # Scraping infos entreprises
│   │   │   ├── pdf.go         # Génération PDF
│   │   │   └── analytics.go   # Collecte analytics
│   │   ├── models/            # DB models (GORM)
│   │   ├── middleware/
│   │   │   ├── tracking.go    # Tracking visiteurs
│   │   │   ├── ratelimit.go   # Rate limiting
│   │   │   └── cors.go
│   │   ├── database/
│   │   │   ├── postgres.go
│   │   │   └── redis.go
│   │   └── utils/
│   ├── migrations/            # SQL migrations
│   ├── pkg/                   # Libs réutilisables
│   ├── go.mod
│   ├── go.sum
│   └── Dockerfile
│
├── frontend/                  # Next.js App
│   ├── app/
│   │   ├── page.tsx          # Homepage
│   │   ├── cv/
│   │   │   └── page.tsx      # CV dynamique
│   │   ├── letters/
│   │   │   └── page.tsx      # Générateur lettres
│   │   ├── analytics/
│   │   │   └── page.tsx      # Dashboard analytics
│   │   └── layout.tsx
│   ├── components/
│   │   ├── cv/
│   │   │   ├── CVThemeSelector.tsx
│   │   │   ├── ExperienceTimeline.tsx
│   │   │   ├── SkillsCloud.tsx
│   │   │   └── ProjectsGrid.tsx
│   │   ├── letters/
│   │   │   ├── LetterGenerator.tsx
│   │   │   └── LetterPreview.tsx
│   │   ├── analytics/
│   │   │   ├── RealtimeVisitors.tsx
│   │   │   ├── ThemeStats.tsx
│   │   │   └── Heatmap.tsx
│   │   └── ui/               # shadcn/ui components
│   ├── lib/
│   │   ├── api.ts            # API client
│   │   └── utils.ts
│   ├── public/
│   ├── package.json
│   └── Dockerfile
│
├── docker/
│   ├── nginx/
│   │   ├── nginx.conf
│   │   └── Dockerfile
│   └── docker-compose.yml
│
├── monitoring/
│   ├── prometheus/
│   │   └── prometheus.yml
│   └── grafana/
│       └── dashboards/
│
├── .github/
│   └── workflows/
│       ├── ci.yml
│       └── deploy.yml
│
├── docs/
│   ├── PROJECT_SPEC.md       # Ce document
│   ├── ARCHITECTURE.md       # Architecture détaillée
│   ├── API.md                # Documentation API
│   └── DEPLOYMENT.md         # Guide déploiement
│
├── .gitignore
└── README.md
```

---

## 🚢 Déploiement

### Environnement

**VPS OVH:**
- Docker + Docker Compose
- Nginx reverse proxy
- Let's Encrypt SSL (certbot)
- Domaine: À définir

### CI/CD Pipeline

**Repositories:**
- **Gitea** (principal) : repo privé de développement
- **GitHub** (vitrine) : mirror public avec README attractif

**GitHub Actions Workflow:**

```yaml
1. Tests
   - Backend: go test ./...
   - Frontend: npm run test

2. Build
   - Docker build backend
   - Docker build frontend

3. Deploy (sur push main)
   - SSH vers VPS
   - Pull images
   - Docker Compose up -d
   - Health check
```

### Monitoring

**Prometheus + Grafana:**
- Dashboard public accessible à tous
- Métriques applicatives + système
- Alerting (optionnel)

---

## 🎨 Design et UX

### Principes

- **Moderne et professionnel** : design épuré, typographie soignée
- **Interactif** : animations fluides, micro-interactions
- **Performant** : optimisation images, lazy loading, SSR
- **Responsive** : mobile-first design
- **Accessible** : WCAG 2.1 AA compliance

### Thème Visuel

- Palette de couleurs : À définir (dark mode + light mode)
- Typographie : Inter, Poppins ou SF Pro
- Animations : Framer Motion (transitions page, hover effects)
- Effets spéciaux : Possibilité de Three.js pour avatar 3D ou visualisation compétences

---

## 🔮 Roadmap

### Phase 1 - MVP (Minimal Viable Product)
- ✅ Structure projet
- 🔲 Setup Docker Compose (PostgreSQL, Redis, backend, frontend)
- 🔲 Backend Go : API basique + DB models
- 🔲 Frontend Next.js : pages principales + design
- 🔲 CV statique avec données en dur
- 🔲 Déploiement VPS basique

### Phase 2 - CV Dynamique
- 🔲 Système de thèmes/filtrage
- 🔲 Interface de sélection thème
- 🔲 Algorithme de scoring/adaptation contenu
- 🔲 Export PDF CV

### Phase 3 - IA Lettres
- 🔲 Intégration API Claude/GPT
- 🔲 Service de recherche infos entreprises
- 🔲 Génération lettres motivation + anti-motivation
- 🔲 Export PDF lettres
- 🔲 Système de tracking visites (Redis)
- 🔲 Rate limiting IA

### Phase 4 - Analytics
- 🔲 Collecte événements (PostgreSQL + Redis)
- 🔲 Dashboard temps réel
- 🔲 Visualisations graphiques
- 🔲 Heatmap interactions

### Phase 5 - Features Avancées
- 🔲 Import automatique GitHub
- 🔲 Timeline interactive
- 🔲 Détection profil visiteurs (recruteurs)
- 🔲 Exploration cookies avancée
- 🔲 Effets 3D (Three.js)

### Phase 6 - Production
- 🔲 CI/CD complet
- 🔲 Monitoring Prometheus + Grafana
- 🔲 Tests automatisés (unitaires, e2e)
- 🔲 Optimisations performances
- 🔲 SEO optimization
- 🔲 Documentation complète

---

## 💡 Features Futures (Post-Launch)

- **Blog technique** : articles avec génération IA assistée
- **Recommandation de projets** : IA suggère projets pertinents selon profil visiteur
- **Chatbot conversationnel** : discussion sur le parcours professionnel
- **A/B Testing** : tester différentes versions CV
- **Multi-langue** : FR/EN avec traduction IA
- **API publique** : exposer certaines données via API REST
- **Webhooks** : notifications sur événements (nouvelle visite recruteur, etc.)

---

## 📝 Notes Techniques

### Bibliothèques Go à Considérer

**Framework web:**
- Fiber (express-like, très rapide)

**Base de données:**
- GORM (ORM)
- pgx (driver PostgreSQL performant)

**Redis:**
- go-redis

**PDF:**
- gofpdf (génération pure Go)
- chromedp (rendu HTML→PDF via Chrome headless)

**IA:**
- Clients HTTP custom pour APIs Claude/OpenAI

**Tests:**
- testify (assertions)
- gomock (mocking)

### Frontend Libraries

- **UI:** shadcn/ui, Radix UI
- **Forms:** React Hook Form + Zod
- **Charts:** Chart.js, Recharts, ou D3.js
- **3D:** React Three Fiber (si effets 3D)
- **PDF:** react-pdf ou jsPDF

---

## 🎯 Objectifs du Projet

1. **Vitrine technique** : démontrer compétences full-stack + DevOps + IA
2. **Originalité** : lettres d'anti-motivation = différenciation créative
3. **Fonctionnel** : vraiment utilisable pour candidatures
4. **Open Source** : GitHub public avec documentation exemplaire
5. **Performance** : monitoring public = transparence et démo compétences
6. **Évolutif** : architecture permettant ajouts features facilement

---

**Auteur:** Alexi
**Date création:** 2025-12-06
**Version:** 1.0
