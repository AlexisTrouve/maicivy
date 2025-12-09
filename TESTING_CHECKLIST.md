# Testing Checklist - maicivy

Guide de test progressif pour valider chaque composant indépendamment.

**Date:** 2025-12-09
**Objectif:** Tester chaque système individuellement avant l'intégration complète

---

## 📋 Vue d'ensemble

```
Level 0: Prérequis système
   ↓
Level 1: Infrastructure (Docker)
   ↓
Level 2: Backend (Go)
   ↓
Level 3: Frontend (Next.js)
   ↓
Level 4: Intégration complète
```

---

## Level 0: Prérequis Système

### Vérifier les outils installés

```bash
# Docker & Docker Compose
docker --version          # Doit être >= 20.x
docker-compose --version  # Doit être >= 2.x

# Go
go version               # Doit être >= 1.21

# Node.js & npm
node --version           # Doit être >= 18.x
npm --version            # Doit être >= 9.x

# Git
git --version            # N'importe quelle version récente
```

**Checklist:**
- [ ] Docker installé et fonctionnel
- [ ] Docker Compose installé
- [ ] Go >= 1.21 installé
- [ ] Node.js >= 18.x installé
- [ ] npm >= 9.x installé
- [ ] Git installé

---

## Level 1: Infrastructure (Docker)

### Test 1.1: PostgreSQL seul

```bash
cd /mnt/c/Users/alexi/Documents/projects/maicivy

# Lancer uniquement PostgreSQL
docker-compose up -d postgres

# Vérifier les logs
docker-compose logs postgres

# Tester la connexion
docker exec -it maicivy-postgres psql -U maicivy_user -d maicivy_db -c "SELECT version();"
```

**Résultat attendu:**
```
PostgreSQL 15.x on x86_64-pc-linux-gnu
```

**Checklist:**
- [ ] Container postgres démarre sans erreur
- [ ] Connexion psql fonctionne
- [ ] Base de données `maicivy_db` créée
- [ ] User `maicivy_user` existe

**Si erreur:** Vérifier les logs avec `docker-compose logs postgres`

---

### Test 1.2: Redis seul

```bash
# Lancer uniquement Redis
docker-compose up -d redis

# Vérifier les logs
docker-compose logs redis

# Tester la connexion
docker exec -it maicivy-redis redis-cli ping
```

**Résultat attendu:**
```
PONG
```

**Checklist:**
- [ ] Container redis démarre sans erreur
- [ ] Commande PING retourne PONG
- [ ] Port 6379 accessible

**Si erreur:** Vérifier les logs avec `docker-compose logs redis`

---

### Test 1.3: Infrastructure complète (sans apps)

```bash
# Arrêter tout
docker-compose down

# Relancer postgres + redis
docker-compose up -d postgres redis

# Attendre 10 secondes
sleep 10

# Vérifier le status
docker-compose ps
```

**Résultat attendu:**
```
NAME                 STATUS
maicivy-postgres     Up (healthy)
maicivy-redis        Up (healthy)
```

**Checklist:**
- [ ] Les 2 containers sont UP
- [ ] Les 2 containers sont HEALTHY (health checks passent)
- [ ] Aucune erreur dans les logs

---

## Level 2: Backend (Go)

### Test 2.1: Compilation du code Go

```bash
cd backend

# Télécharger les dépendances
go mod download

# Vérifier qu'il n'y a pas d'erreurs de compilation
go build -o /tmp/maicivy-backend cmd/main.go
```

**Résultat attendu:**
```
# Aucune erreur, fichier /tmp/maicivy-backend créé
```

**Checklist:**
- [ ] `go mod download` réussit
- [ ] Aucune erreur de compilation
- [ ] Binaire créé dans /tmp/

**Si erreur:**
- Erreur d'import : Vérifier que tous les packages existent
- Erreur de syntaxe : Corriger le code Go
- Dépendance manquante : `go get <package>`

---

### Test 2.2: Variables d'environnement

```bash
cd backend

# Vérifier que .env existe
ls -la .env

# Afficher le contenu (sans les secrets)
cat .env | grep -v "API_KEY\|PASSWORD"
```

**Checklist:**
- [ ] Fichier `.env` existe
- [ ] Toutes les variables requises sont définies :
  - [ ] `DB_HOST`, `DB_PORT`, `DB_USER`, `DB_PASSWORD`, `DB_NAME`
  - [ ] `REDIS_HOST`, `REDIS_PORT`, `REDIS_PASSWORD`
  - [ ] `PORT`, `ENVIRONMENT`
  - [ ] `ANTHROPIC_API_KEY` (si test IA)
  - [ ] `OPENAI_API_KEY` (si test IA)

**Si manquant:** Créer le fichier à partir de `.env.example`

---

### Test 2.3: Tests unitaires Go

```bash
cd backend

# Lancer les tests unitaires (sans DB)
go test ./internal/models -v
go test ./internal/services -v

# Avec coverage
go test ./... -coverprofile=coverage.out
go tool cover -func=coverage.out
```

**Résultat attendu:**
```
PASS
coverage: XX% of statements
```

**Checklist:**
- [ ] Tests des models passent
- [ ] Tests des services passent
- [ ] Coverage > 50% (idéalement > 70%)

**Si erreur:**
- Test fail : Lire le message d'erreur et corriger
- Import manquant : Vérifier les dépendances

---

### Test 2.4: Backend standalone (sans Docker)

**Prérequis:** PostgreSQL et Redis doivent tourner (Test 1.3)

```bash
cd backend

# Source les variables d'environnement
export $(cat .env | xargs)

# Lancer le backend
go run cmd/main.go
```

**Résultat attendu dans les logs:**
```
INF PostgreSQL connected successfully
INF Redis connected successfully
INF Server starting on port 5000
```

**Dans un autre terminal, tester les endpoints:**

```bash
# Health check
curl http://localhost:5000/health

# Résultat attendu:
# {"status":"ok","timestamp":"..."}

# Deep health check
curl http://localhost:5000/health/deep

# Résultat attendu:
# {"status":"ok","database":"ok","redis":"ok","timestamp":"..."}
```

**Checklist:**
- [ ] Backend démarre sans erreur
- [ ] Connexion PostgreSQL OK
- [ ] Connexion Redis OK
- [ ] Endpoint `/health` retourne 200
- [ ] Endpoint `/health/deep` retourne 200 avec DB et Redis OK

**Si erreur:**
- Connexion DB refusée : Vérifier que postgres tourne
- Connexion Redis refusée : Vérifier que redis tourne
- Port 5000 déjà utilisé : Changer le port dans `.env`

**Arrêter le backend:** Ctrl+C

---

### Test 2.5: Migrations de base de données

```bash
cd backend

# Lancer les migrations
go run scripts/migrate.go up

# Vérifier que les tables existent
docker exec -it maicivy-postgres psql -U maicivy_user -d maicivy_db -c "\dt"
```

**Résultat attendu:**
```
              List of relations
 Schema |       Name        | Type  |    Owner
--------+-------------------+-------+-------------
 public | experiences       | table | maicivy_user
 public | skills            | table | maicivy_user
 public | projects          | table | maicivy_user
 public | visitors          | table | maicivy_user
 public | generated_letters | table | maicivy_user
 public | analytics_events  | table | maicivy_user
 public | github_profiles   | table | maicivy_user
```

**Checklist:**
- [ ] Migrations s'exécutent sans erreur
- [ ] Les 7 tables sont créées
- [ ] Les indexes sont créés (vérifier avec `\di`)

**Si erreur:**
- Migration déjà appliquée : Normal si déjà fait
- Erreur SQL : Vérifier le fichier de migration

---

### Test 2.6: Seed data

```bash
cd backend

# Peupler la base avec des données de test
go run scripts/seed.go

# Vérifier les données
docker exec -it maicivy-postgres psql -U maicivy_user -d maicivy_db -c "SELECT COUNT(*) FROM experiences;"
docker exec -it maicivy-postgres psql -U maicivy_user -d maicivy_db -c "SELECT COUNT(*) FROM skills;"
docker exec -it maicivy-postgres psql -U maicivy_user -d maicivy_db -c "SELECT COUNT(*) FROM projects;"
```

**Résultat attendu:**
```
 count
-------
     3  (experiences)
    10  (skills)
     3  (projects)
```

**Checklist:**
- [ ] Seed s'exécute sans erreur
- [ ] Données insérées dans experiences
- [ ] Données insérées dans skills
- [ ] Données insérées dans projects

---

### Test 2.7: Endpoints API Backend

**Prérequis:** Backend doit tourner (Test 2.4) et seed data inséré (Test 2.6)

```bash
# Test GET /api/v1/cv
curl http://localhost:5000/api/v1/cv?theme=backend | jq

# Test GET /api/v1/experiences
curl http://localhost:5000/api/v1/experiences | jq

# Test GET /api/v1/skills
curl http://localhost:5000/api/v1/skills | jq

# Test GET /api/v1/projects
curl http://localhost:5000/api/v1/projects | jq
```

**Résultat attendu:**
- Status 200
- JSON valide avec données

**Checklist:**
- [ ] Endpoint `/api/v1/cv` retourne des données
- [ ] Endpoint `/api/v1/experiences` retourne des expériences
- [ ] Endpoint `/api/v1/skills` retourne des compétences
- [ ] Endpoint `/api/v1/projects` retourne des projets
- [ ] Tous les JSON sont valides

---

## Level 3: Frontend (Next.js)

### Test 3.1: Installation des dépendances

```bash
cd frontend

# Installer les dépendances npm
npm install
```

**Résultat attendu:**
```
added XXX packages
```

**Checklist:**
- [ ] `npm install` réussit sans erreur
- [ ] `node_modules/` créé
- [ ] `package-lock.json` généré

**Si erreur:**
- Erreur de version Node : Mettre à jour Node.js >= 18
- Erreur de dépendance : `npm install --legacy-peer-deps`

---

### Test 3.2: Variables d'environnement Frontend

```bash
cd frontend

# Vérifier que .env.local existe
ls -la .env.local

# Afficher le contenu
cat .env.local
```

**Checklist:**
- [ ] Fichier `.env.local` existe
- [ ] `NEXT_PUBLIC_API_URL` est défini (ex: http://localhost:5000)

**Si manquant:** Créer le fichier :
```bash
echo "NEXT_PUBLIC_API_URL=http://localhost:5000" > .env.local
```

---

### Test 3.3: Compilation TypeScript

```bash
cd frontend

# Vérifier qu'il n'y a pas d'erreurs TypeScript
npx tsc --noEmit
```

**Résultat attendu:**
```
# Aucune erreur
```

**Checklist:**
- [ ] Aucune erreur TypeScript
- [ ] Aucun type manquant

**Si erreur:**
- Erreur de type : Corriger les types dans le code
- Import manquant : Installer le package manquant

---

### Test 3.4: Build Next.js

```bash
cd frontend

# Build de production
npm run build
```

**Résultat attendu:**
```
✓ Compiled successfully
✓ Linting and checking validity of types
✓ Collecting page data
✓ Generating static pages (X/X)
```

**Checklist:**
- [ ] Build réussit sans erreur
- [ ] Aucune erreur de lint
- [ ] Pages générées avec succès
- [ ] Dossier `.next/` créé

**Si erreur:**
- Erreur de build : Lire le message et corriger
- Erreur de lint : `npm run lint` pour voir les détails

---

### Test 3.5: Frontend standalone (dev mode)

**Prérequis:** Backend doit tourner (Test 2.4)

```bash
cd frontend

# Lancer le dev server
npm run dev
```

**Résultat attendu:**
```
  ▲ Next.js 14.x
  - Local:        http://localhost:3000
  - Ready in XXXms
```

**Dans un navigateur, tester:**

1. **Page d'accueil:** http://localhost:3000
   - [ ] Page se charge sans erreur
   - [ ] Layout s'affiche correctement

2. **Page CV:** http://localhost:3000/cv
   - [ ] Page se charge
   - [ ] ThemeSelector s'affiche
   - [ ] Expériences affichées (si backend connecté)

3. **Page Lettres:** http://localhost:3000/letters
   - [ ] Page se charge
   - [ ] Formulaire s'affiche
   - [ ] AccessGate visible (si < 3 visites)

4. **Page Analytics:** http://localhost:3000/analytics
   - [ ] Page se charge
   - [ ] Dashboard s'affiche

**Checklist:**
- [ ] Dev server démarre
- [ ] Aucune erreur dans la console navigateur (F12)
- [ ] Toutes les pages sont accessibles
- [ ] Styles Tailwind appliqués

**Si erreur:**
- Page 404 : Vérifier que les fichiers page.tsx existent
- Erreur API : Vérifier que backend tourne et CORS configuré
- Erreur de style : Vérifier Tailwind config

**Arrêter le dev server:** Ctrl+C

---

## Level 4: Intégration Complète

### Test 4.1: Docker Compose complet

```bash
cd /mnt/c/Users/alexi/Documents/projects/maicivy

# Arrêter tout
docker-compose down

# Rebuild et relancer tout
docker-compose up --build -d

# Attendre 30 secondes (le temps que tout démarre)
sleep 30

# Vérifier le status
docker-compose ps
```

**Résultat attendu:**
```
NAME                 STATUS
maicivy-postgres     Up (healthy)
maicivy-redis        Up (healthy)
maicivy-backend      Up (healthy)
maicivy-frontend     Up
```

**Checklist:**
- [ ] Les 4 containers démarrent
- [ ] Tous les health checks passent
- [ ] Aucune erreur dans `docker-compose logs`

---

### Test 4.2: Tests d'intégration End-to-End

**Ouvrir un navigateur:**

1. **Frontend:** http://localhost:3000
   - [ ] Page d'accueil se charge
   - [ ] Navigation fonctionne

2. **Backend Health:** http://localhost:5000/health
   - [ ] Retourne JSON `{"status":"ok"}`

3. **API via Frontend:** http://localhost:3000/cv?theme=backend
   - [ ] Données du backend s'affichent
   - [ ] Expériences filtrées par thème

4. **Analytics temps réel:** http://localhost:3000/analytics
   - [ ] WebSocket connecté
   - [ ] Stats en temps réel

**Checklist:**
- [ ] Frontend communique avec Backend
- [ ] Backend accède à PostgreSQL
- [ ] Backend accède à Redis
- [ ] WebSocket fonctionne
- [ ] Tracking visiteur fonctionne

---

### Test 4.3: Logs et Debugging

```bash
# Voir tous les logs
docker-compose logs

# Logs backend seulement
docker-compose logs backend

# Logs frontend seulement
docker-compose logs frontend

# Suivre les logs en temps réel
docker-compose logs -f backend
```

**Checklist:**
- [ ] Aucune erreur critique dans les logs
- [ ] Connexions DB/Redis OK
- [ ] Requêtes HTTP loguées

---

## 🎯 Résumé des Tests

### Tests Réussis ✅

Cocher au fur et à mesure :

**Level 0: Prérequis**
- [ ] Docker installé
- [ ] Go installé
- [ ] Node.js installé

**Level 1: Infrastructure**
- [ ] PostgreSQL fonctionne
- [ ] Redis fonctionne
- [ ] Health checks OK

**Level 2: Backend**
- [ ] Code Go compile
- [ ] Tests unitaires passent
- [ ] Backend démarre standalone
- [ ] Migrations OK
- [ ] Seed data OK
- [ ] Endpoints API répondent

**Level 3: Frontend**
- [ ] npm install OK
- [ ] TypeScript compile
- [ ] Build Next.js OK
- [ ] Dev server démarre
- [ ] Pages accessibles

**Level 4: Intégration**
- [ ] Docker Compose complet OK
- [ ] Frontend ↔ Backend communication
- [ ] Backend ↔ PostgreSQL communication
- [ ] Backend ↔ Redis communication
- [ ] WebSocket fonctionne

---

## 🚨 Dépannage Rapide

### Backend ne démarre pas
```bash
# Vérifier les logs
docker-compose logs backend

# Erreurs communes:
# - DB connection failed → Vérifier PostgreSQL
# - Redis connection failed → Vérifier Redis
# - Port already in use → Changer le port dans .env
```

### Frontend ne se connecte pas au Backend
```bash
# Vérifier CORS
# Dans backend/internal/middleware/cors.go
# AllowOrigins doit inclure http://localhost:3000

# Vérifier .env.local
cat frontend/.env.local
# NEXT_PUBLIC_API_URL doit pointer vers http://localhost:5000
```

### Docker Compose erreurs
```bash
# Nettoyer tout
docker-compose down -v  # -v supprime aussi les volumes

# Rebuild complet
docker-compose build --no-cache

# Relancer
docker-compose up -d
```

---

## 📝 Notes

- **Ne pas tester l'IA (lettres) au début** : Nécessite des API keys Anthropic/OpenAI
- **Tester progressivement** : Ne pas sauter de level
- **Documenter les erreurs** : Noter les erreurs rencontrées pour debug
- **Utiliser les logs** : Toujours vérifier les logs en cas d'erreur

---

**Version:** 1.0
**Date:** 2025-12-09
**Auteur:** Alexi
