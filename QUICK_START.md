# Quick Start Guide - maicivy

**Sprint 1 COMPLET ✅** - Prêt pour développement

---

## 1. Installation des Dépendances (5 min)

### Backend
```bash
cd backend
go mod download
go mod tidy
```

### Frontend
```bash
cd frontend
npm install
```

---

## 2. Validation (5 min)

### Backend
```bash
cd backend
go build -o bin/maicivy ./cmd
# Doit compiler sans erreur
```

### Frontend
```bash
cd frontend
npm run build
npm run type-check
npm run lint
# Tout doit passer sans erreur
```

---

## 3. Démarrage (2 min)

```bash
# À la racine du projet
./scripts/dev-start.sh
```

**Services démarrés:**
- PostgreSQL: localhost:5432
- Redis: localhost:6379
- Backend: http://localhost:8080
- Frontend: http://localhost:3000

---

## 4. Vérification (1 min)

### Backend
```bash
curl http://localhost:8080/health
# Doit retourner: {"status":"ok","timestamp":"..."}
```

### Frontend
Ouvrir http://localhost:3000 dans un navigateur
- Homepage doit s'afficher
- Dark mode toggle doit fonctionner
- Navigation doit fonctionner

---

## 5. Logs (optionnel)

```bash
./scripts/dev-logs.sh
# Suit les logs de tous les services
```

---

## 6. Arrêt

```bash
./scripts/dev-stop.sh
```

---

## En Cas de Problème

### Backend ne démarre pas
```bash
docker-compose logs backend
```

### Frontend ne démarre pas
```bash
docker-compose logs frontend
```

### Tout recommencer
```bash
./scripts/dev-clean.sh
# ⚠️ Supprime toutes les données
./scripts/dev-start.sh
```

---

## Documentation Complète

- **Sprint 1:** `SPRINT1_COMPLETE.md`
- **Backend:** `BACKEND_FOUNDATION_COMPLETE.md`
- **Frontend:** `FRONTEND_FOUNDATION_COMPLETE.md`
- **Infrastructure:** `INFRASTRUCTURE_SETUP_COMPLETE.md`
- **TODO:** `TODO_NEXT.md`
- **Structure:** `PROJECT_STRUCTURE.md`

---

## Prochaines Étapes

Une fois que tout fonctionne:

1. **Sprint 2 - Database Schema**
   - Voir `docs/implementation/03_DATABASE_SCHEMA.md`
   - Models GORM, migrations, seed data

2. **Sprint 2 - Backend Middlewares**
   - Voir `docs/implementation/04_BACKEND_MIDDLEWARES.md`
   - Tracking, rate limiting, logging

---

**Temps total:** ~15 minutes

**Besoin d'aide?** Lire `TODO_NEXT.md` pour plus de détails
