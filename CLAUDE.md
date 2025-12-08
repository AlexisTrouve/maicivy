# Guide de Documentation - maicivy

**Pour Claude, pour les développeurs, pour tout le monde.**

Ce fichier est votre **boussole** pour naviguer dans la documentation du projet **maicivy**.

---

## 🎯 Vous Êtes...

Choisissez votre profil pour savoir quoi lire :

### 👤 Nouveau sur le Projet
**Objectif :** Comprendre ce qu'on construit et pourquoi

**Parcours recommandé :**
1. 📖 [docs/PROJECT_SPEC.md](docs/PROJECT_SPEC.md) - **COMMENCER ICI**
   - Vision du projet (CV interactif + IA)
   - Stack technique (Go + Next.js + Claude)
   - Fonctionnalités détaillées
   - Roadmap 6 phases
   - **Temps de lecture : 15-20 min**

2. 📋 [docs/IMPLEMENTATION_PLAN.md](docs/IMPLEMENTATION_PLAN.md)
   - Plan global de développement
   - 19 documents d'implémentation
   - Dépendances entre modules
   - Timeline (10-12 semaines)
   - **Temps de lecture : 10 min**

3. 📊 [docs/IMPLEMENTATION_INDEX.md](docs/IMPLEMENTATION_INDEX.md)
   - Index complet des 19 docs d'implémentation
   - Recherche par fonctionnalité
   - Statistiques (priorités, complexités)
   - **Temps de lecture : 5 min**

**Après :** Vous savez ce qu'on construit. Passez à votre rôle spécifique ci-dessous.

---

### 💻 Développeur Backend (Go)
**Objectif :** Implémenter les APIs et services backend

**Documents prioritaires :**

**📦 Phase 1 - Foundation (CRITIQUE)**
1. [docs/implementation/01_SETUP_INFRASTRUCTURE.md](docs/implementation/01_SETUP_INFRASTRUCTURE.md)
   - Docker Compose (PostgreSQL, Redis)
   - Setup environnement dev
   - **À lire avant de coder**

2. [docs/implementation/02_BACKEND_FOUNDATION.md](docs/implementation/02_BACKEND_FOUNDATION.md)
   - Structure projet Go
   - Fiber setup + configuration
   - Connexions DB (GORM, Redis)
   - Logger, error handling
   - **Foundation de tout le backend**

3. [docs/implementation/03_DATABASE_SCHEMA.md](docs/implementation/03_DATABASE_SCHEMA.md)
   - Models GORM (6+ tables)
   - Migrations SQL
   - Relations, indexes
   - **Critique : tous les modules en dépendent**

4. [docs/implementation/04_BACKEND_MIDDLEWARES.md](docs/implementation/04_BACKEND_MIDDLEWARES.md)
   - CORS, Tracking visiteurs
   - Rate limiting (3 visites, 5/jour)
   - Request logging
   - **Sécurité et features métier**

**🎯 Phase 2 - CV API**
5. [docs/implementation/06_BACKEND_CV_API.md](docs/implementation/06_BACKEND_CV_API.md)
   - Algorithme de scoring par thème
   - Endpoints GET /api/cv
   - Export PDF
   - Cache Redis

**🤖 Phase 3 - IA Services**
6. [docs/implementation/08_BACKEND_AI_SERVICES.md](docs/implementation/08_BACKEND_AI_SERVICES.md)
   - Intégration Claude + GPT-4
   - Scraper infos entreprises
   - Génération PDF lettres
   - Prompts engineering
   - **Le plus complexe : 5/5 étoiles**

7. [docs/implementation/09_BACKEND_LETTERS_API.md](docs/implementation/09_BACKEND_LETTERS_API.md)
   - POST /api/letters/generate
   - Queue asynchrone (jobs)
   - Access gate (3 visites)
   - Rate limiting IA (5/jour, 2min)

**📈 Phase 4 - Analytics**
8. [docs/implementation/11_BACKEND_ANALYTICS.md](docs/implementation/11_BACKEND_ANALYTICS.md)
   - Service analytics temps réel
   - WebSocket /ws/analytics
   - Agrégations Redis
   - Métriques Prometheus

**Qualité & Sécurité**
9. [docs/implementation/17_SECURITY.md](docs/implementation/17_SECURITY.md)
   - OWASP Top 10
   - Validation, sanitization
   - **À lire AVANT de coder**

10. [docs/implementation/18_PERFORMANCE.md](docs/implementation/18_PERFORMANCE.md)
    - Caching strategies Redis
    - DB optimization (indexes)
    - Benchmarks (wrk, k6)

11. [docs/implementation/19_API_REFERENCE.md](docs/implementation/19_API_REFERENCE.md)
    - OpenAPI/Swagger setup
    - Annotations swaggo
    - Documentation auto-générée

**Ordre recommandé :** 01 → 02 → 03 → 04 → 06 → 08 → 09 → 11 → 17 → 18 → 19

---

### 🎨 Développeur Frontend (Next.js + TypeScript)
**Objectif :** Créer les interfaces utilisateur et intégrations API

**Documents prioritaires :**

**📦 Phase 1 - Foundation (CRITIQUE)**
1. [docs/implementation/01_SETUP_INFRASTRUCTURE.md](docs/implementation/01_SETUP_INFRASTRUCTURE.md)
   - Docker Compose
   - Variables d'environnement
   - **Setup dev local**

2. [docs/implementation/05_FRONTEND_FOUNDATION.md](docs/implementation/05_FRONTEND_FOUNDATION.md)
   - Next.js 14 (App Router)
   - Tailwind CSS + dark mode
   - API client wrapper
   - Layout principal
   - shadcn/ui setup
   - **Foundation de tout le frontend**

**🎯 Phase 2 - CV Dynamique**
3. [docs/implementation/07_FRONTEND_CV_DYNAMIC.md](docs/implementation/07_FRONTEND_CV_DYNAMIC.md)
   - Page /cv avec query params
   - Components : CVThemeSelector, ExperienceTimeline, SkillsCloud, ProjectsGrid
   - Animations Framer Motion
   - Export PDF button
   - **Feature principale du CV**

**✉️ Phase 3 - Générateur Lettres**
4. [docs/implementation/10_FRONTEND_LETTERS.md](docs/implementation/10_FRONTEND_LETTERS.md)
   - Page /letters
   - LetterGenerator (form + validation Zod)
   - LetterPreview **DUAL** (2 lettres côte à côte)
   - AccessGate (teaser 3 visites)
   - Loading states, error handling
   - **Feature signature du projet**

**📊 Phase 4 - Dashboard Analytics**
5. [docs/implementation/12_FRONTEND_ANALYTICS_DASHBOARD.md](docs/implementation/12_FRONTEND_ANALYTICS_DASHBOARD.md)
   - Page /analytics
   - RealtimeVisitors (WebSocket)
   - ThemeStats (Chart.js)
   - Heatmap interactions
   - Filters date
   - **Dashboard public temps réel**

**Qualité**
6. [docs/implementation/17_SECURITY.md](docs/implementation/17_SECURITY.md)
   - Validation Zod frontend
   - XSS prevention
   - **À lire AVANT de coder**

7. [docs/implementation/18_PERFORMANCE.md](docs/implementation/18_PERFORMANCE.md)
   - Next.js Image optimization
   - Lazy loading, code splitting
   - Core Web Vitals

**Ordre recommandé :** 01 → 05 → 07 → 10 → 12 → 17 → 18

---

### 🚀 DevOps / Infrastructure
**Objectif :** Déployer et monitorer l'application

**Documents prioritaires :**

1. [docs/implementation/01_SETUP_INFRASTRUCTURE.md](docs/implementation/01_SETUP_INFRASTRUCTURE.md)
   - Docker Compose architecture
   - Services (postgres, redis, backend, frontend)
   - **Base de l'infra**

2. [docs/implementation/14_INFRASTRUCTURE_PRODUCTION.md](docs/implementation/14_INFRASTRUCTURE_PRODUCTION.md)
   - Nginx reverse proxy + SSL
   - Prometheus + Grafana
   - Health checks
   - Backups PostgreSQL/Redis
   - Logging
   - **Configuration production complète**

3. [docs/implementation/15_CICD_DEPLOYMENT.md](docs/implementation/15_CICD_DEPLOYMENT.md)
   - GitHub Actions workflows
   - Tests automatisés
   - Build & push Docker images
   - Deploy script SSH vers VPS
   - Rollback strategy
   - **Pipeline CI/CD complet**

4. [docs/implementation/17_SECURITY.md](docs/implementation/17_SECURITY.md)
   - Security headers (Nginx)
   - Secrets management
   - HTTPS enforcement
   - Dependency scanning

5. [docs/implementation/18_PERFORMANCE.md](docs/implementation/18_PERFORMANCE.md)
   - Nginx compression
   - Redis caching
   - Monitoring (Grafana dashboards)

**Ordre recommandé :** 01 → 14 → 15 → 17 → 18

---

### 🧪 QA / Testeur
**Objectif :** Stratégie de tests et validation qualité

**Documents prioritaires :**

1. [docs/PROJECT_SPEC.md](docs/PROJECT_SPEC.md)
   - Spécifications fonctionnelles
   - **Base pour écrire les tests**

2. [docs/implementation/16_TESTING_STRATEGY.md](docs/implementation/16_TESTING_STRATEGY.md)
   - Tests unitaires (testify Go, Jest React)
   - Tests integration (PostgreSQL, Redis)
   - Tests E2E (Playwright)
   - Fixtures & mocks
   - Coverage targets (80% backend, 70% frontend)
   - CI integration
   - **Guide complet de testing**

3. [docs/implementation/17_SECURITY.md](docs/implementation/17_SECURITY.md)
   - OWASP Top 10 checklist
   - Security tests
   - Penetration testing

4. [docs/implementation/18_PERFORMANCE.md](docs/implementation/18_PERFORMANCE.md)
   - Load testing (k6, wrk)
   - Benchmarks
   - Profiling

**Ordre recommandé :** PROJECT_SPEC → 16 → 17 → 18

---

### 📋 Chef de Projet / Product Owner
**Objectif :** Planifier, séquencer et tracker le développement

**Documents essentiels :**

1. [docs/PROJECT_SPEC.md](docs/PROJECT_SPEC.md)
   - Vision, roadmap 6 phases
   - Fonctionnalités détaillées
   - **Document de référence produit**

2. [docs/IMPLEMENTATION_PLAN.md](docs/IMPLEMENTATION_PLAN.md)
   - Plan global 19 documents
   - Dépendances techniques
   - Timeline 10-12 semaines
   - **Plan macro**

3. [docs/DEVELOPMENT_SEQUENCING.md](docs/DEVELOPMENT_SEQUENCING.md)
   - Séquençage optimal (agents parallèles)
   - 6 sprints détaillés
   - Gain de temps : -45% (38-44 jours)
   - Commandes par vague
   - Checklists par sprint
   - **Plan d'exécution opérationnel**

4. [docs/IMPLEMENTATION_INDEX.md](docs/IMPLEMENTATION_INDEX.md)
   - Index complet des 19 docs
   - Statistiques (priorités, complexités)
   - Recherche par fonctionnalité

**Ordre recommandé :** PROJECT_SPEC → IMPLEMENTATION_PLAN → DEVELOPMENT_SEQUENCING → IMPLEMENTATION_INDEX

---

### 🤖 Agent IA (Claude en mode dev)
**Objectif :** Implémenter un module spécifique

**Instructions :**

1. **Lire TOUJOURS en premier :**
   - [docs/PROJECT_SPEC.md](docs/PROJECT_SPEC.md) - Spécifications du projet
   - [docs/IMPLEMENTATION_PLAN.md](docs/IMPLEMENTATION_PLAN.md) - Structure globale

2. **Lire le document d'implémentation assigné :**
   - Voir [docs/implementation/](docs/implementation/) - 19 documents numérotés
   - **Suivre EXACTEMENT** la structure et le code fourni

3. **Vérifier les prérequis :**
   - Voir section "Prérequis" du document
   - S'assurer que les modules dépendants sont terminés

4. **Ne PAS inventer :**
   - Utiliser UNIQUEMENT ce qui est dans PROJECT_SPEC.md
   - Suivre la structure standardisée du document
   - Respecter les technologies spécifiées

5. **Livrer :**
   - Créer les fichiers listés dans "Livrables"
   - Écrire les tests (section "Tests")
   - Valider avec les commandes de la section "Validation"
   - Cocher la checklist de complétion

**Documents par phase :**
- Phase 1 : 01, 02, 03, 04, 05
- Phase 2 : 06, 07
- Phase 3 : 08, 09, 10
- Phase 4 : 11, 12
- Phase 5 : 13
- Phase 6 : 14, 15, 16, 17, 18, 19

---

## 📚 Catalogue Complet de la Documentation

### Documents Racine (Specs & Plans)

| Fichier | Description | Quand le lire |
|---------|-------------|---------------|
| [CLAUDE.md](CLAUDE.md) | **Ce fichier** - Guide de navigation | En premier |
| [docs/PROJECT_SPEC.md](docs/PROJECT_SPEC.md) | Spécifications du projet (vision, features, stack) | Avant tout dev |
| [docs/IMPLEMENTATION_PLAN.md](docs/IMPLEMENTATION_PLAN.md) | Plan d'implémentation (19 docs, dépendances, timeline) | Pour comprendre le plan global |
| [docs/DEVELOPMENT_SEQUENCING.md](docs/DEVELOPMENT_SEQUENCING.md) | Séquençage agents parallèles (6 sprints, commandes) | Pour exécuter le dev |
| [docs/IMPLEMENTATION_INDEX.md](docs/IMPLEMENTATION_INDEX.md) | Index des 19 docs d'implémentation | Pour naviguer rapidement |

---

### Documents d'Implémentation (19 docs)

**Voir [docs/implementation/README.md](docs/implementation/README.md) pour le guide du dossier.**

#### Phase 1 - MVP Foundation

| Doc | Fichier | Description | Complexité | Temps |
|-----|---------|-------------|------------|-------|
| 01 | [SETUP_INFRASTRUCTURE.md](docs/implementation/01_SETUP_INFRASTRUCTURE.md) | Docker Compose, PostgreSQL, Redis, .env | ⭐⭐ | 1-2j |
| 02 | [BACKEND_FOUNDATION.md](docs/implementation/02_BACKEND_FOUNDATION.md) | Structure Go, Fiber, GORM, Redis, Logger | ⭐⭐⭐ | 2-3j |
| 03 | [DATABASE_SCHEMA.md](docs/implementation/03_DATABASE_SCHEMA.md) | Models, Migrations, Seed data, Relations | ⭐⭐⭐⭐ | 3-4j |
| 04 | [BACKEND_MIDDLEWARES.md](docs/implementation/04_BACKEND_MIDDLEWARES.md) | CORS, Tracking, Rate limiting, Logger | ⭐⭐⭐ | 2-3j |
| 05 | [FRONTEND_FOUNDATION.md](docs/implementation/05_FRONTEND_FOUNDATION.md) | Next.js 14, Tailwind, API client, Layout | ⭐⭐⭐ | 2-3j |

#### Phase 2 - CV Dynamique

| Doc | Fichier | Description | Complexité | Temps |
|-----|---------|-------------|------------|-------|
| 06 | [BACKEND_CV_API.md](docs/implementation/06_BACKEND_CV_API.md) | Algorithme scoring, Endpoints CV, PDF export | ⭐⭐⭐⭐ | 3-5j |
| 07 | [FRONTEND_CV_DYNAMIC.md](docs/implementation/07_FRONTEND_CV_DYNAMIC.md) | Page /cv, Components, Animations Framer Motion | ⭐⭐⭐⭐ | 4-5j |

#### Phase 3 - IA Lettres

| Doc | Fichier | Description | Complexité | Temps |
|-----|---------|-------------|------------|-------|
| 08 | [BACKEND_AI_SERVICES.md](docs/implementation/08_BACKEND_AI_SERVICES.md) | Claude/GPT, Scraper, PDF lettres, Prompts | ⭐⭐⭐⭐⭐ | 5-7j |
| 09 | [BACKEND_LETTERS_API.md](docs/implementation/09_BACKEND_LETTERS_API.md) | POST /generate, Queue jobs, Rate limiting IA | ⭐⭐⭐⭐ | 3-4j |
| 10 | [FRONTEND_LETTERS.md](docs/implementation/10_FRONTEND_LETTERS.md) | Page /letters, Generator, Preview dual, AccessGate | ⭐⭐⭐⭐ | 4-5j |

#### Phase 4 - Analytics

| Doc | Fichier | Description | Complexité | Temps |
|-----|---------|-------------|------------|-------|
| 11 | [BACKEND_ANALYTICS.md](docs/implementation/11_BACKEND_ANALYTICS.md) | Service analytics, WebSocket, Prometheus | ⭐⭐⭐⭐ | 4-5j |
| 12 | [FRONTEND_ANALYTICS_DASHBOARD.md](docs/implementation/12_FRONTEND_ANALYTICS_DASHBOARD.md) | Page /analytics, Charts, Heatmap, WebSocket | ⭐⭐⭐⭐ | 4-5j |

#### Phase 5 - Features Avancées

| Doc | Fichier | Description | Complexité | Temps |
|-----|---------|-------------|------------|-------|
| 13 | [FEATURES_ADVANCED.md](docs/implementation/13_FEATURES_ADVANCED.md) | GitHub import, Timeline, Profiling, 3D | ⭐⭐⭐ | Variable |

#### Phase 6 - Production & Qualité

| Doc | Fichier | Description | Complexité | Temps |
|-----|---------|-------------|------------|-------|
| 14 | [INFRASTRUCTURE_PRODUCTION.md](docs/implementation/14_INFRASTRUCTURE_PRODUCTION.md) | Nginx, Prometheus, Grafana, SSL, Backups | ⭐⭐⭐⭐ | 3-5j |
| 15 | [CICD_DEPLOYMENT.md](docs/implementation/15_CICD_DEPLOYMENT.md) | GitHub Actions, Deploy script, Rollback | ⭐⭐⭐ | 2-3j |
| 16 | [TESTING_STRATEGY.md](docs/implementation/16_TESTING_STRATEGY.md) | Tests unitaires, integration, E2E, Coverage | ⭐⭐⭐⭐ | Continu |
| 17 | [SECURITY.md](docs/implementation/17_SECURITY.md) | OWASP Top 10, Validation, Sanitization | ⭐⭐⭐⭐ | 2-3j |
| 18 | [PERFORMANCE.md](docs/implementation/18_PERFORMANCE.md) | Caching, DB optimization, Benchmarks | ⭐⭐⭐ | 2-3j |

#### Annexes

| Doc | Fichier | Description | Complexité | Temps |
|-----|---------|-------------|------------|-------|
| 19 | [API_REFERENCE.md](docs/implementation/19_API_REFERENCE.md) | OpenAPI spec, Swagger UI, swaggo | ⭐⭐ | 1-2j |

---

## 🗺️ Parcours de Lecture par Objectif

### Objectif : Comprendre le Projet (30 min)
1. PROJECT_SPEC.md (15 min)
2. IMPLEMENTATION_PLAN.md (10 min)
3. IMPLEMENTATION_INDEX.md (5 min)

### Objectif : Planifier le Développement (45 min)
1. PROJECT_SPEC.md (15 min)
2. IMPLEMENTATION_PLAN.md (10 min)
3. DEVELOPMENT_SEQUENCING.md (15 min)
4. IMPLEMENTATION_INDEX.md (5 min)

### Objectif : Développer Backend (3-4h lecture)
1. PROJECT_SPEC.md (15 min)
2. 01_SETUP_INFRASTRUCTURE.md (20 min)
3. 02_BACKEND_FOUNDATION.md (30 min)
4. 03_DATABASE_SCHEMA.md (45 min)
5. 04_BACKEND_MIDDLEWARES.md (30 min)
6. 17_SECURITY.md (30 min)
7. 18_PERFORMANCE.md (20 min)

### Objectif : Développer Frontend (2-3h lecture)
1. PROJECT_SPEC.md (15 min)
2. 01_SETUP_INFRASTRUCTURE.md (20 min)
3. 05_FRONTEND_FOUNDATION.md (30 min)
4. 07_FRONTEND_CV_DYNAMIC.md (30 min)
5. 10_FRONTEND_LETTERS.md (30 min)
6. 12_FRONTEND_ANALYTICS_DASHBOARD.md (30 min)

### Objectif : Setup DevOps (2h lecture)
1. 01_SETUP_INFRASTRUCTURE.md (20 min)
2. 14_INFRASTRUCTURE_PRODUCTION.md (40 min)
3. 15_CICD_DEPLOYMENT.md (30 min)
4. 17_SECURITY.md (20 min)
5. 18_PERFORMANCE.md (10 min)

### Objectif : Écrire les Tests (2h lecture)
1. PROJECT_SPEC.md (15 min)
2. 16_TESTING_STRATEGY.md (60 min)
3. 17_SECURITY.md (30 min)
4. 18_PERFORMANCE.md (15 min)

---

## 🔍 Recherche par Technologie

### Go / Backend
- 02_BACKEND_FOUNDATION.md - Fiber, GORM, Redis
- 03_DATABASE_SCHEMA.md - Models GORM
- 06_BACKEND_CV_API.md - Endpoints, caching
- 08_BACKEND_AI_SERVICES.md - Intégration Claude/GPT
- 09_BACKEND_LETTERS_API.md - Queue, rate limiting
- 11_BACKEND_ANALYTICS.md - WebSocket, Prometheus

### Next.js / Frontend
- 05_FRONTEND_FOUNDATION.md - App Router, Tailwind
- 07_FRONTEND_CV_DYNAMIC.md - Components, Framer Motion
- 10_FRONTEND_LETTERS.md - Forms, validation Zod
- 12_FRONTEND_ANALYTICS_DASHBOARD.md - Charts, WebSocket

### Docker / Infrastructure
- 01_SETUP_INFRASTRUCTURE.md - Docker Compose
- 14_INFRASTRUCTURE_PRODUCTION.md - Nginx, monitoring

### IA / Machine Learning
- 08_BACKEND_AI_SERVICES.md - Claude (Anthropic), GPT-4
- 09_BACKEND_LETTERS_API.md - Génération lettres

### Base de Données
- 01_SETUP_INFRASTRUCTURE.md - PostgreSQL, Redis setup
- 03_DATABASE_SCHEMA.md - Schema, migrations, indexes
- 18_PERFORMANCE.md - Optimizations DB

### Sécurité
- 04_BACKEND_MIDDLEWARES.md - Rate limiting, tracking
- 17_SECURITY.md - OWASP Top 10, validation

### Tests
- 16_TESTING_STRATEGY.md - testify, Playwright, k6

### CI/CD
- 15_CICD_DEPLOYMENT.md - GitHub Actions, deploy

---

## 🚦 Indicateurs de Priorité

### 🔴 CRITIQUE - À lire absolument avant de coder
- PROJECT_SPEC.md
- 01_SETUP_INFRASTRUCTURE.md
- 02_BACKEND_FOUNDATION.md
- 03_DATABASE_SCHEMA.md
- 05_FRONTEND_FOUNDATION.md
- 14_INFRASTRUCTURE_PRODUCTION.md
- 17_SECURITY.md

### 🟡 HAUTE - Fonctionnalités principales
- 04_BACKEND_MIDDLEWARES.md
- 06_BACKEND_CV_API.md
- 07_FRONTEND_CV_DYNAMIC.md
- 08_BACKEND_AI_SERVICES.md
- 09_BACKEND_LETTERS_API.md
- 10_FRONTEND_LETTERS.md
- 15_CICD_DEPLOYMENT.md
- 16_TESTING_STRATEGY.md

### 🟢 MOYENNE - Fonctionnalités secondaires
- 11_BACKEND_ANALYTICS.md
- 12_FRONTEND_ANALYTICS_DASHBOARD.md
- 18_PERFORMANCE.md
- 19_API_REFERENCE.md

### 🔵 BASSE - Nice to have
- 13_FEATURES_ADVANCED.md

---

## ⏱️ Temps de Lecture Estimés

| Document | Temps Lecture | Temps Implémentation |
|----------|---------------|----------------------|
| PROJECT_SPEC.md | 15-20 min | - |
| IMPLEMENTATION_PLAN.md | 10 min | - |
| DEVELOPMENT_SEQUENCING.md | 15-20 min | - |
| IMPLEMENTATION_INDEX.md | 5 min | - |
| Chaque doc d'implémentation | 20-45 min | 1-7 jours |

**Total lecture docs d'implémentation :** ~8-10 heures
**Total implémentation projet :** 38-44 jours (avec parallélisation)

---

## 💡 Conseils d'Utilisation

### Pour les Humains

1. **Ne lisez PAS tout d'un coup**
   - Commencez par PROJECT_SPEC.md
   - Lisez les docs d'implémentation au fur et à mesure

2. **Utilisez la recherche**
   - Ctrl+F dans les fichiers Markdown
   - Cherchez par technologie, fonctionnalité, phase

3. **Suivez l'ordre des phases**
   - Phase 1 avant Phase 2, etc.
   - Respectez les prérequis

4. **Validez au fur et à mesure**
   - Utilisez les checklists des documents
   - Lancez les commandes de validation

### Pour les Agents IA (Claude)

1. **Lis TOUJOURS PROJECT_SPEC.md en premier**
   - C'est la source de vérité
   - N'invente JAMAIS rien qui n'y est pas

2. **Suis la structure standardisée**
   - Chaque doc d'implémentation a la même structure
   - Respecte-la exactement

3. **Vérifie les prérequis**
   - Section "Prérequis" de chaque document
   - Ne code pas si les dépendances ne sont pas prêtes

4. **Livre exactement ce qui est demandé**
   - Section "Livrables" de chaque document
   - Pas plus, pas moins

5. **Teste ton code**
   - Section "Tests" de chaque document
   - Lance les commandes de validation

---

## 📞 Aide et Support

### Questions Fréquentes

**Q: Par où commencer ?**
A: Lire PROJECT_SPEC.md, puis votre section de profil ci-dessus.

**Q: Je ne comprends pas une techno (ex: GORM, Fiber)**
A: Chaque doc d'implémentation a une section "Ressources" avec liens documentation officielle.

**Q: Dans quel ordre coder ?**
A: Suivre DEVELOPMENT_SEQUENCING.md qui détaille les 6 sprints.

**Q: Comment paralléliser ?**
A: DEVELOPMENT_SEQUENCING.md explique quels agents lancer en parallèle.

**Q: Un document est trop long, je peux le résumer ?**
A: Non. Suivre EXACTEMENT les instructions. Les docs sont longs car ils contiennent tout le code nécessaire.

**Q: Puis-je utiliser une autre techno (ex: Gin au lieu de Fiber) ?**
A: Non. La stack est définie dans PROJECT_SPEC.md et ne doit pas être changée.

---

## 🎯 Checklist Avant de Commencer à Coder

- [ ] J'ai lu PROJECT_SPEC.md en entier
- [ ] J'ai lu IMPLEMENTATION_PLAN.md pour comprendre la structure
- [ ] Je sais quelle phase je dois implémenter
- [ ] J'ai lu le(s) document(s) d'implémentation de ma phase
- [ ] J'ai vérifié les prérequis de mes documents
- [ ] J'ai les outils installés (Go, Node, Docker)
- [ ] J'ai les API keys nécessaires (ANTHROPIC_API_KEY, OPENAI_API_KEY si IA)
- [ ] Je connais mes livrables (section "Livrables" du doc)
- [ ] J'ai compris les tests à écrire (section "Tests" du doc)
- [ ] Je suis prêt à suivre la checklist de complétion du doc

**Si toutes les cases sont cochées : GO ! 🚀**

---

**Version:** 1.0
**Date:** 2025-12-08
**Auteur:** Alexi

**Note:** Ce fichier est vivant. Il sera mis à jour si la structure de documentation change.
