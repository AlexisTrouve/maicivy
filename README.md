# maicivy - My CV AI

**CV interactif intelligent avec génération de lettres de motivation/anti-motivation par IA**

---

## 🎯 Qu'est-ce que maicivy ?

Un CV en ligne qui sert de **démo technique complète**, démontrant des compétences en :
- ✅ Backend (Go + Fiber)
- ✅ Frontend (Next.js 14 + TypeScript)
- ✅ Intelligence Artificielle (Claude + GPT-4)
- ✅ DevOps (Docker, CI/CD, Monitoring)
- ✅ Architecture système (PostgreSQL, Redis, APIs)

### Fonctionnalités Principales

1. **CV Dynamique Adaptatif** - Se personnalise selon le thème (Backend, C++, Artistique, etc.)
2. **Générateur de Lettres IA** ⭐ - Génère 2 lettres : Motivation + **Anti-Motivation** (humoristique)
3. **Analytics Publiques Temps Réel** - Dashboard visible par tous
4. **Import GitHub** - Synchronisation automatique de vos projets
5. **Timeline Interactive** - Visualisation chronologique avec animations

---

## 📚 Documentation

**👉 DÉMARRAGE RAPIDE : [QUICK_START.md](QUICK_START.md)** - Lancez le projet en 15 minutes

**📋 NAVIGATION : [INDEX.md](INDEX.md)** - Index complet de toute la documentation

**🧭 GUIDE : [CLAUDE.md](CLAUDE.md)** - Guide de navigation détaillé

### Documents Clés

| Fichier | Description | Pour Qui |
|---------|-------------|----------|
| **[CLAUDE.md](CLAUDE.md)** | 🧭 **Guide de navigation** | Tout le monde |
| [docs/PROJECT_SPEC.md](docs/PROJECT_SPEC.md) | 📖 Spécifications complètes | Tous (à lire en premier) |
| [docs/IMPLEMENTATION_PLAN.md](docs/IMPLEMENTATION_PLAN.md) | 📋 Plan global (19 docs) | Chef de projet, Devs |
| [docs/DEVELOPMENT_SEQUENCING.md](docs/DEVELOPMENT_SEQUENCING.md) | 🚀 Séquençage optimal (6 sprints) | Chef de projet, Devs |
| [docs/IMPLEMENTATION_INDEX.md](docs/IMPLEMENTATION_INDEX.md) | 📊 Index des 19 docs | Navigation rapide |
| [docs/implementation/](docs/implementation/) | 🔨 19 guides d'implémentation | Développeurs |

---

## 🏗️ Stack Technique

### Backend
- **Langage:** Go 1.21+
- **Framework:** Fiber
- **Base de données:** PostgreSQL (GORM)
- **Cache:** Redis (go-redis)
- **Logger:** zerolog

### Frontend
- **Framework:** Next.js 14 (App Router)
- **Langage:** TypeScript
- **Styling:** Tailwind CSS
- **Animations:** Framer Motion
- **UI:** shadcn/ui

### IA
- **APIs:** Claude (Anthropic) + GPT-4 (OpenAI)
- **Usage:** Génération lettres, traduction

### Infrastructure
- **Conteneurisation:** Docker + Docker Compose
- **Reverse Proxy:** Nginx + Let's Encrypt SSL
- **Monitoring:** Prometheus + Grafana (dashboard public)
- **CI/CD:** GitHub Actions
- **Hébergement:** VPS OVH

---

## 🚀 Démarrage Rapide

### Prérequis

```bash
# Go 1.21+
go version

# Node 18+
node --version

# Docker
docker --version
```

### Installation

```bash
# Clone du repo
git clone <repo-url>
cd maicivy

# Setup environnement
cp .env.example .env
# Éditer .env avec vos API keys

# Lancer l'infrastructure
docker-compose up -d

# Backend
cd backend
go mod download
go run cmd/main.go

# Frontend (nouveau terminal)
cd frontend
npm install
npm run dev
```

### Accès

- **Frontend:** http://localhost:3000
- **Backend API:** http://localhost:8080
- **Swagger UI:** http://localhost:8080/api/docs
- **Grafana:** http://localhost:3001

---

## 📖 Guide de Développement

### Pour les Nouveaux

1. **Lire [CLAUDE.md](CLAUDE.md)** - Guide de navigation
2. **Lire [docs/PROJECT_SPEC.md](docs/PROJECT_SPEC.md)** - Comprendre le projet
3. **Choisir votre rôle** dans CLAUDE.md (Backend, Frontend, DevOps, etc.)
4. **Suivre le parcours** recommandé pour votre rôle

### Pour les Développeurs

**Ordre d'implémentation :**

```
Phase 1 (8-12j)  → Infrastructure + Backend + Frontend Foundation
Phase 2 (5j)     → CV Dynamique
Phase 3 (10j)    → IA Lettres (feature principale)
Phase 4 (5j)     → Analytics
Phase 5 (5-7j)   → Features Avancées
Phase 6 (5j)     → Production

Total : 38-44 jours (avec parallélisation)
```

**Voir [docs/DEVELOPMENT_SEQUENCING.md](docs/DEVELOPMENT_SEQUENCING.md)** pour le plan détaillé.

---

## 🎯 Roadmap

### ✅ Phase 0 - Documentation (TERMINÉ)
- [x] Spécifications complètes
- [x] Plan d'implémentation (19 documents)
- [x] Séquençage optimal
- [x] Guide de navigation

### ✅ Phase 1 - MVP Foundation (TERMINÉ - 2025-12-08)
- [x] Setup Docker Compose
- [x] Backend Go + Fiber
- [x] Frontend Next.js 14
- [ ] Database schema + migrations (Sprint 2)
- [ ] Middlewares (tracking, rate limiting) (Sprint 2)

**Détails:** Voir [SPRINT1_COMPLETE.md](SPRINT1_COMPLETE.md)

### 🔲 Phase 2 - CV Dynamique
- [ ] API Backend (algorithme scoring)
- [ ] Frontend (5 thèmes : Backend, C++, Artistique, Full-Stack, DevOps)
- [ ] Export PDF

### 🔲 Phase 3 - IA Lettres
- [ ] Intégration Claude/GPT-4
- [ ] Génération 2 lettres (motivation + anti-motivation)
- [ ] Access gate (3 visites)
- [ ] Rate limiting (5/jour)

### 🔲 Phase 4 - Analytics
- [ ] Dashboard temps réel
- [ ] WebSocket
- [ ] Prometheus + Grafana public

### 🔲 Phase 5 - Features Avancées
- [ ] Import GitHub
- [ ] Timeline interactive
- [ ] Détection profils avancée

### 🔲 Phase 6 - Production
- [ ] Infrastructure Nginx + SSL
- [ ] CI/CD GitHub Actions
- [ ] Tests (80%+ coverage)
- [ ] Sécurité (OWASP Top 10)
- [ ] Performance (benchmarks)

---

## 🧪 Tests

```bash
# Backend
cd backend
go test -v ./...
go test -cover ./...

# Frontend
cd frontend
npm test
npm test -- --coverage

# E2E
npm run test:e2e
```

**Coverage Targets:**
- Backend : 80%+
- Frontend : 70%+

---

## 🔒 Sécurité

- ✅ OWASP Top 10 compliant
- ✅ Input validation (Zod frontend, validator backend)
- ✅ Sanitization (XSS, SQL injection)
- ✅ Rate limiting (global + AI)
- ✅ HTTPS enforcement
- ✅ Security headers (CSP, HSTS, etc.)

Voir [docs/implementation/17_SECURITY.md](docs/implementation/17_SECURITY.md)

---

## 📊 Monitoring

### Métriques Disponibles

- Visiteurs actuels (temps réel)
- Total visites (jour/semaine/mois)
- Top thèmes CV consultés
- Lettres générées
- Response times (P50, P95, P99)
- Error rates
- Database connections
- Redis memory usage

**Dashboard Public:** https://maicivy.com/grafana (après déploiement)

---

## 🤝 Contribution

### Workflow

1. Lire [CLAUDE.md](CLAUDE.md) pour comprendre la structure
2. Choisir un document d'implémentation dans [docs/implementation/](docs/implementation/)
3. Créer une branche `feature/XX-nom`
4. Implémenter selon le document
5. Écrire les tests
6. Soumettre une Pull Request

### Standards

- **Go:** `gofmt`, `golangci-lint`
- **TypeScript:** ESLint, Prettier
- **Commits:** Conventional Commits
- **Tests:** Obligatoires (coverage > seuils)

---

## 📝 Licence

À définir

---

## 👤 Auteur

**Alexi**

---

## 🔗 Liens Utiles

- **Documentation:** [CLAUDE.md](CLAUDE.md)
- **Spécifications:** [docs/PROJECT_SPEC.md](docs/PROJECT_SPEC.md)
- **Plan d'Implémentation:** [docs/IMPLEMENTATION_PLAN.md](docs/IMPLEMENTATION_PLAN.md)
- **Séquençage Dev:** [docs/DEVELOPMENT_SEQUENCING.md](docs/DEVELOPMENT_SEQUENCING.md)

---

**Status:** ✅ Sprint 1 COMPLET - Backend + Frontend Foundations prêts

**Prochaine étape:** Sprint 2 - Database Schema + Middlewares

**Dernière mise à jour:** 2025-12-08
