# TODO - Prochaines Actions

**Date:** 2025-12-08
**Après:** Sprint 1 - Phase 1 MVP Foundation COMPLET ✅

---

## Actions Immédiates (À Faire Maintenant)

### 1. Installer les Dépendances Backend

```bash
cd backend
go mod download
go mod tidy
```

**Résultat attendu:**
- Toutes les dépendances Go téléchargées
- go.sum mis à jour
- Aucune erreur

### 2. Installer les Dépendances Frontend

```bash
cd frontend
npm install
```

**Résultat attendu:**
- node_modules/ créé
- package-lock.json généré
- Aucune erreur

---

## Validation (À Faire Après Installation)

### 3. Tester le Build Backend

```bash
cd backend
go build -o bin/maicivy ./cmd
```

**Résultat attendu:**
- Binaire `bin/maicivy` créé
- Aucune erreur de compilation

### 4. Tester le Build Frontend

```bash
cd frontend
npm run build
```

**Résultat attendu:**
- Dossier `.next/` créé avec build production
- Aucune erreur

```bash
npm run type-check
```

**Résultat attendu:**
- Aucune erreur TypeScript

```bash
npm run lint
```

**Résultat attendu:**
- Aucune erreur ESLint

---

## Lancer les Services (Première Fois)

### 5. Démarrer Docker Compose

```bash
# Option 1: Docker Compose classique
docker-compose up -d

# Option 2: Script de développement (recommandé)
./scripts/dev-start.sh
```

**Services démarrés:**
- PostgreSQL (localhost:5432)
- Redis (localhost:6379)
- Backend (localhost:8080)
- Frontend (localhost:3000)

### 6. Vérifier les Services

**Backend Health Check:**
```bash
curl http://localhost:8080/health
```

**Frontend:**
Ouvrir http://localhost:3000 dans un navigateur

**Logs:**
```bash
# Option 1: Docker Compose
docker-compose logs -f

# Option 2: Script
./scripts/dev-logs.sh
```

---

## Debugging (Si Problèmes)

### Backend ne démarre pas

```bash
# Vérifier les logs
docker-compose logs backend

# Vérifier PostgreSQL
docker-compose logs postgres

# Vérifier Redis
docker-compose logs redis
```

**Problèmes courants:**
- PostgreSQL pas prêt → attendre 10-15 secondes
- Port 8080 déjà utilisé → changer dans .env
- Variables d'env manquantes → vérifier .env

### Frontend ne démarre pas

```bash
# Vérifier les logs
docker-compose logs frontend

# Vérifier le build
cd frontend
npm run build
```

**Problèmes courants:**
- node_modules manquant → npm install
- Port 3000 déjà utilisé → changer dans .env
- Backend non accessible → vérifier NEXT_PUBLIC_API_URL

### CORS Errors

Si le frontend ne peut pas contacter le backend:

1. Vérifier que le backend autorise `http://localhost:3000`
2. Vérifier les logs backend pour erreurs CORS
3. Vérifier `NEXT_PUBLIC_API_URL` dans frontend/.env.local

---

## Arrêter les Services

```bash
# Option 1: Docker Compose
docker-compose down

# Option 2: Script
./scripts/dev-stop.sh
```

---

## Nettoyer (Si Besoin de Recommencer)

```bash
# Arrêter et supprimer volumes
./scripts/dev-clean.sh

# Ou manuellement
docker-compose down -v
```

⚠️ **ATTENTION:** Ceci supprime toutes les données PostgreSQL et Redis

---

## Après Validation

Une fois que tout fonctionne:

### Sprint 2 - Database Schema

**Document:** `docs/implementation/03_DATABASE_SCHEMA.md`

**À créer:**
- Models GORM (6+ tables: experiences, skills, projects, etc.)
- Migrations SQL
- Seed data (données de test)
- Relations entre tables
- Indexes pour performance

**Commande:**
```bash
# Demander à Claude d'implémenter 03_DATABASE_SCHEMA.md
```

### Sprint 2 - Backend Middlewares

**Document:** `docs/implementation/04_BACKEND_MIDDLEWARES.md`

**À créer:**
- Middleware tracking visiteurs
- Middleware rate limiting
- Middleware request logging
- CORS avancé

**Commande:**
```bash
# Demander à Claude d'implémenter 04_BACKEND_MIDDLEWARES.md
```

---

## Checklist de Validation

Avant de passer au Sprint 2, valider que:

- [ ] go mod download réussi
- [ ] npm install réussi
- [ ] go build réussi sans erreurs
- [ ] npm build réussi sans erreurs
- [ ] npm run type-check réussi
- [ ] npm run lint réussi
- [ ] docker-compose up démarre les 4 services
- [ ] curl http://localhost:8080/health retourne 200 OK
- [ ] http://localhost:3000 affiche la homepage
- [ ] Dark mode toggle fonctionne
- [ ] Navigation entre pages fonctionne
- [ ] Aucune erreur dans les logs

---

## Aide

### Documentation

- **Infrastructure:** `INFRASTRUCTURE_SETUP_COMPLETE.md`
- **Backend:** `BACKEND_FOUNDATION_COMPLETE.md`
- **Frontend:** `FRONTEND_FOUNDATION_COMPLETE.md`
- **Sprint 1:** `SPRINT1_COMPLETE.md`
- **Docker:** `DOCKER_COMMANDS.md`

### Commandes Utiles

```bash
# Afficher tous les services
docker-compose ps

# Redémarrer un service
docker-compose restart backend

# Voir les logs d'un service
docker-compose logs -f backend

# Exécuter une commande dans un container
docker-compose exec backend sh

# Vérifier l'espace disque
docker system df

# Nettoyer Docker
docker system prune
```

---

**Bonne chance! 🚀**

Si tout fonctionne, le projet est prêt pour le Sprint 2.
