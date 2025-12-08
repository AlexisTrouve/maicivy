# Structure Complète du Projet maicivy

**Date:** 2025-12-08
**Sprint:** 1 - Phase 1 MVP Foundation COMPLET ✅

---

## Vue d'Ensemble

```
maicivy/
├── backend/                 ✅ API Go + Fiber
├── frontend/                ✅ Next.js 14 App Router
├── docker/                  ✅ Configuration Docker
├── scripts/                 ✅ Scripts de développement
├── docs/                    ✅ Documentation
└── [fichiers racine]        ✅ Configuration projet
```

---

## Structure Détaillée

### Racine du Projet

```
maicivy/
├── .env.example                              ✅ Variables d'environnement exemple
├── .gitignore                                ✅ Git ignore
├── docker-compose.yml                        ✅ Orchestration Docker
├── README.md                                 ✅ Documentation principale
│
├── CLAUDE.md                                 ✅ Guide Claude (navigation docs)
├── BACKEND_FOUNDATION_COMPLETE.md            ✅ Doc backend
├── FRONTEND_FOUNDATION_COMPLETE.md           ✅ Doc frontend
├── INFRASTRUCTURE_SETUP_COMPLETE.md          ✅ Doc infrastructure
├── SPRINT1_COMPLETE.md                       ✅ Récap Sprint 1
├── SPRINT1_VAGUE2_COMPLETE.md                ✅ Récap Vague 2
├── DOCKER_COMMANDS.md                        ✅ Commandes Docker
├── NEXT_STEPS.md                             ✅ Prochaines étapes
├── TODO_NEXT.md                              ✅ Actions immédiates
└── PROJECT_STRUCTURE.md                      ✅ Ce fichier
```

---

### Backend (Go)

```
backend/
├── cmd/
│   └── main.go                               ✅ Entry point application
│
├── internal/
│   ├── api/
│   │   ├── handlers/
│   │   │   ├── health.go                     ✅ Health check handler
│   │   │   └── status.go                     ✅ Status handler
│   │   └── routes/
│   │       └── routes.go                     ✅ Configuration routes
│   │
│   ├── config/
│   │   └── config.go                         ✅ Configuration (Viper)
│   │
│   ├── database/
│   │   ├── postgres.go                       ✅ Connexion PostgreSQL (GORM)
│   │   └── redis.go                          ✅ Connexion Redis
│   │
│   ├── middleware/
│   │   └── cors.go                           ✅ Middleware CORS
│   │
│   └── utils/
│       └── response.go                       ✅ Helpers réponses HTTP
│
├── pkg/
│   └── logger/
│       └── logger.go                         ✅ Logger structuré (Logrus)
│
├── .env.example                              ✅ Variables d'env backend
├── go.mod                                    ✅ Dépendances Go
├── go.sum                                    ✅ Checksums dépendances
├── Dockerfile                                ✅ Dockerfile multistage
└── README.md                                 ✅ Documentation backend
```

**Total Backend:** 14 fichiers Go + configuration

---

### Frontend (Next.js)

```
frontend/
├── app/
│   ├── layout.tsx                            ✅ Layout racine (fonts, Header, Footer)
│   ├── page.tsx                              ✅ Homepage
│   ├── globals.css                           ✅ Styles globaux + dark mode
│   ├── loading.tsx                           ✅ Loading state global
│   ├── error.tsx                             ✅ Error boundary global
│   ├── not-found.tsx                         ✅ Page 404
│   │
│   ├── cv/
│   │   └── page.tsx                          ✅ Page CV (placeholder Phase 2)
│   ├── letters/
│   │   └── page.tsx                          ✅ Page Lettres (placeholder Phase 3)
│   ├── analytics/
│   │   └── page.tsx                          ✅ Page Analytics (placeholder Phase 4)
│   └── api-test/
│       └── page.tsx                          ✅ Page test API
│
├── components/
│   ├── layout/
│   │   ├── Header.tsx                        ✅ Header sticky (nav + dark mode)
│   │   └── Footer.tsx                        ✅ Footer (liens sociaux)
│   │
│   ├── ui/                                   ✅ shadcn/ui components
│   │   ├── button.tsx                        ✅ Button (6 variants)
│   │   └── card.tsx                          ✅ Card + sous-composants
│   │
│   └── shared/
│       └── LoadingSpinner.tsx                ✅ Spinner accessible
│
├── lib/
│   ├── api.ts                                ✅ Client API (retry, timeout)
│   ├── types.ts                              ✅ Types TypeScript
│   └── utils.ts                              ✅ Utilities (cn, formatDate)
│
├── hooks/
│   └── useTheme.ts                           ✅ Hook dark mode
│
├── package.json                              ✅ Dépendances NPM
├── next.config.js                            ✅ Config Next.js
├── tsconfig.json                             ✅ Config TypeScript
├── tailwind.config.ts                        ✅ Config Tailwind
├── postcss.config.js                         ✅ Config PostCSS
├── .env.local                                ✅ Variables d'env frontend
├── .eslintrc.json                            ✅ Config ESLint
├── .prettierrc                               ✅ Config Prettier
├── .gitignore                                ✅ Git ignore
├── next-env.d.ts                             ✅ Types Next.js
├── Dockerfile                                ✅ Dockerfile multistage
├── README.md                                 ✅ Documentation frontend
└── IMPLEMENTATION_CHECKLIST.md               ✅ Checklist implémentation
```

**Total Frontend:** 28 fichiers TypeScript/React + configuration

---

### Docker

```
docker/
├── nginx/
│   └── nginx.conf                            ✅ Reverse proxy (préconfiguré)
└── redis/
    └── redis.conf                            ✅ Configuration Redis (optionnel)
```

**Fichier racine:**
```
docker-compose.yml                            ✅ Orchestration 4 services
```

**Services:**
1. **postgres** - PostgreSQL 16 (port 5432)
2. **redis** - Redis 7 (port 6379)
3. **backend** - API Go (port 8080)
4. **frontend** - Next.js (port 3000)

---

### Scripts de Développement

```
scripts/
├── dev-start.sh                              ✅ Démarre services dev
├── dev-stop.sh                               ✅ Arrête services dev
├── dev-logs.sh                               ✅ Affiche logs
└── dev-clean.sh                              ✅ Nettoie volumes
```

**Usage:**
```bash
./scripts/dev-start.sh    # Démarre Docker Compose
./scripts/dev-logs.sh     # Suit les logs
./scripts/dev-stop.sh     # Arrête les services
./scripts/dev-clean.sh    # Nettoie tout (⚠️ supprime données)
```

---

### Documentation

```
docs/
├── PROJECT_SPEC.md                           ✅ Spécifications complètes
├── IMPLEMENTATION_PLAN.md                    ✅ Plan d'implémentation
├── DEVELOPMENT_SEQUENCING.md                 ✅ Séquençage agents
├── IMPLEMENTATION_INDEX.md                   ✅ Index des 19 docs
│
└── implementation/                           ✅ 19 documents d'implémentation
    ├── 01_SETUP_INFRASTRUCTURE.md            ✅ Infrastructure (FAIT)
    ├── 02_BACKEND_FOUNDATION.md              ✅ Backend foundation (FAIT)
    ├── 03_DATABASE_SCHEMA.md                 ⏳ Database schema (TODO)
    ├── 04_BACKEND_MIDDLEWARES.md             ⏳ Middlewares (TODO)
    ├── 05_FRONTEND_FOUNDATION.md             ✅ Frontend foundation (FAIT)
    ├── 06_BACKEND_CV_API.md                  ⏳ CV API (Phase 2)
    ├── 07_FRONTEND_CV_DYNAMIC.md             ⏳ CV dynamique (Phase 2)
    ├── 08_BACKEND_AI_SERVICES.md             ⏳ Services IA (Phase 3)
    ├── 09_BACKEND_LETTERS_API.md             ⏳ Lettres API (Phase 3)
    ├── 10_FRONTEND_LETTERS.md                ⏳ Lettres frontend (Phase 3)
    ├── 11_BACKEND_ANALYTICS.md               ⏳ Analytics backend (Phase 4)
    ├── 12_FRONTEND_ANALYTICS_DASHBOARD.md    ⏳ Analytics dashboard (Phase 4)
    ├── 13_FEATURES_ADVANCED.md               ⏳ Features avancées (Phase 5)
    ├── 14_INFRASTRUCTURE_PRODUCTION.md       ⏳ Infra production (Phase 6)
    ├── 15_CICD_DEPLOYMENT.md                 ⏳ CI/CD (Phase 6)
    ├── 16_TESTING_STRATEGY.md                ⏳ Tests (Phase 6)
    ├── 17_SECURITY.md                        ⏳ Sécurité (Phase 6)
    ├── 18_PERFORMANCE.md                     ⏳ Performance (Phase 6)
    └── 19_API_REFERENCE.md                   ⏳ API référence (Phase 6)
```

---

## Fichiers de Configuration

### Backend (.env)

```bash
# PostgreSQL
POSTGRES_HOST=postgres
POSTGRES_PORT=5432
POSTGRES_USER=maicivy
POSTGRES_PASSWORD=password
POSTGRES_DB=maicivy

# Redis
REDIS_HOST=redis
REDIS_PORT=6379

# Server
SERVER_PORT=8080
SERVER_ENV=development

# API Keys (à ajouter plus tard)
# ANTHROPIC_API_KEY=
# OPENAI_API_KEY=
```

### Frontend (.env.local)

```bash
# API Backend
NEXT_PUBLIC_API_URL=http://localhost:8080

# Analytics
NEXT_PUBLIC_ANALYTICS_ENABLED=true
```

---

## Métriques du Projet

### Statistiques

**Fichiers créés:** 55+ fichiers
- Backend: 14 fichiers Go
- Frontend: 28 fichiers TypeScript/React
- Infrastructure: 6 fichiers Docker/Scripts
- Documentation: 7+ fichiers Markdown

**Lignes de code (estimation):**
- Backend: ~1000 lignes Go
- Frontend: ~1500 lignes TypeScript/React
- Config: ~500 lignes (JSON, YAML, etc.)

**Technologies:**
- Backend: Go 1.21, Fiber, GORM, Redis, Logrus, Viper
- Frontend: Next.js 14, React 18, TypeScript 5.3, Tailwind CSS 3.4
- Infrastructure: Docker, PostgreSQL 16, Redis 7, Nginx

---

## Prochaines Étapes

### Sprint 2 - Database & Middlewares

**Documents à implémenter:**
1. `03_DATABASE_SCHEMA.md` - Models GORM, migrations, seed
2. `04_BACKEND_MIDDLEWARES.md` - Tracking, rate limiting, logging

**Fichiers à créer:**
- `backend/internal/models/*.go` (6+ models)
- `backend/migrations/*.sql` (migrations)
- `backend/internal/middleware/*.go` (tracking, rate limiting)

---

## Commandes Utiles

### Démarrage Rapide

```bash
# Installation
cd backend && go mod download
cd ../frontend && npm install

# Démarrage
cd ..
./scripts/dev-start.sh

# Vérification
curl http://localhost:8080/health
open http://localhost:3000
```

### Développement

```bash
# Backend (hot reload avec Air - à installer)
cd backend
air

# Frontend (hot reload intégré)
cd frontend
npm run dev
```

### Build Production

```bash
# Backend
cd backend
go build -o bin/maicivy ./cmd

# Frontend
cd frontend
npm run build
```

---

**Dernière mise à jour:** 2025-12-08

**Status:** Sprint 1 COMPLET ✅

**Prêt pour:** Sprint 2 - Database Schema + Middlewares
