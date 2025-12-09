# Commandes Essentielles - maicivy

Guide rapide des commandes les plus utilisées.

---

## 🚀 Démarrage Initial (Première Fois)

### 1. Installation Backend
```bash
cd backend
go mod download
go mod tidy
go build -o bin/maicivy ./cmd
```

### 2. Installation Frontend
```bash
cd frontend
npm install
npm run build
npm run type-check
npm run lint
```

### 3. Démarrage Docker
```bash
# À la racine du projet
./scripts/dev-start.sh

# OU
docker-compose up -d
```

### 4. Vérification
```bash
# Backend
curl http://localhost:8080/health

# Frontend
open http://localhost:3000
```

---

## 💻 Développement au Quotidien

### Démarrer les Services
```bash
./scripts/dev-start.sh
```

### Voir les Logs
```bash
./scripts/dev-logs.sh

# OU logs d'un service spécifique
docker-compose logs -f backend
docker-compose logs -f frontend
```

### Arrêter les Services
```bash
./scripts/dev-stop.sh

# OU
docker-compose down
```

### Redémarrer un Service
```bash
docker-compose restart backend
docker-compose restart frontend
```

---

## 🔧 Backend (Go)

### Développement
```bash
cd backend

# Lancer le serveur (mode dev)
go run cmd/main.go

# Avec hot reload (nécessite Air)
air
```

### Build
```bash
go build -o bin/maicivy ./cmd
```

### Tests
```bash
go test -v ./...
go test -cover ./...
```

### Dépendances
```bash
go mod download
go mod tidy
go mod vendor  # Optionnel
```

### Linting
```bash
golangci-lint run
gofmt -w .
```

---

## 🎨 Frontend (Next.js)

### Développement
```bash
cd frontend

# Lancer le serveur dev (port 3000)
npm run dev
```

### Build
```bash
npm run build
npm start  # Lancer le build production
```

### Tests
```bash
npm run type-check  # TypeScript
npm run lint        # ESLint
npm run lint:fix    # Corriger auto
npm run format      # Prettier
```

### Dépendances
```bash
npm install
npm install <package>
npm install -D <package>  # Dev dependency
```

---

## 🐳 Docker Compose

### Services
```bash
# Démarrer tous les services
docker-compose up -d

# Démarrer un service spécifique
docker-compose up -d postgres
docker-compose up -d redis
docker-compose up -d backend
docker-compose up -d frontend

# Arrêter tous les services
docker-compose down

# Arrêter et supprimer volumes (⚠️ supprime données)
docker-compose down -v
```

### Logs
```bash
# Tous les services
docker-compose logs -f

# Un service spécifique
docker-compose logs -f backend
docker-compose logs -f postgres
```

### Status
```bash
# Voir tous les containers
docker-compose ps

# Détails d'un service
docker-compose ps backend
```

### Rebuild
```bash
# Rebuild un service
docker-compose build backend
docker-compose build frontend

# Rebuild et redémarrer
docker-compose up -d --build backend
```

### Exécuter Commandes
```bash
# Shell dans un container
docker-compose exec backend sh
docker-compose exec postgres psql -U maicivy maicivy

# Commande unique
docker-compose exec backend go version
```

---

## 🗄️ Base de Données

### PostgreSQL
```bash
# Se connecter à la DB
docker-compose exec postgres psql -U maicivy maicivy

# Exécuter SQL
docker-compose exec postgres psql -U maicivy maicivy -c "SELECT * FROM users;"

# Dump DB
docker-compose exec postgres pg_dump -U maicivy maicivy > backup.sql

# Restore DB
docker-compose exec -T postgres psql -U maicivy maicivy < backup.sql
```

### Redis
```bash
# Redis CLI
docker-compose exec redis redis-cli

# Voir toutes les clés
docker-compose exec redis redis-cli KEYS "*"

# Get une valeur
docker-compose exec redis redis-cli GET visitor:123:count

# Flush all (⚠️ supprime tout)
docker-compose exec redis redis-cli FLUSHALL
```

---

## 🧹 Nettoyage

### Nettoyer Projet
```bash
# Nettoyer Docker (⚠️ supprime volumes)
./scripts/dev-clean.sh

# OU
docker-compose down -v
```

### Nettoyer Backend
```bash
cd backend
rm -rf bin/
go clean
```

### Nettoyer Frontend
```bash
cd frontend
rm -rf .next/
rm -rf node_modules/
npm install
```

### Nettoyer Docker Système
```bash
# Voir l'espace utilisé
docker system df

# Nettoyer images non utilisées
docker system prune

# Nettoyer tout (⚠️ supprime tout Docker)
docker system prune -a --volumes
```

---

## 🔍 Debug

### Vérifier Services
```bash
# Backend health
curl http://localhost:8080/health

# Backend status
curl http://localhost:8080/api/status

# Frontend
curl http://localhost:3000

# PostgreSQL
docker-compose exec postgres pg_isready

# Redis
docker-compose exec redis redis-cli PING
```

### Logs Détaillés
```bash
# Backend avec niveau debug
docker-compose logs --tail=100 backend

# Frontend avec erreurs
docker-compose logs --tail=100 frontend | grep -i error
```

### Ressources
```bash
# Voir ressources containers
docker stats

# Voir processus dans container
docker-compose exec backend ps aux
```

---

## 📊 Monitoring

### Métriques Backend
```bash
# Prometheus metrics
curl http://localhost:8080/metrics

# Health check
curl http://localhost:8080/health
```

### Métriques System
```bash
# Containers stats
docker stats

# Espace disque
df -h

# Mémoire
free -h
```

---

## 🚀 Scripts Personnalisés

```bash
# Démarrer tout
./scripts/dev-start.sh

# Arrêter tout
./scripts/dev-stop.sh

# Logs continus
./scripts/dev-logs.sh

# Nettoyer et recommencer
./scripts/dev-clean.sh
```

---

## 📝 Git

### Workflow
```bash
# Créer branche
git checkout -b feature/nom-feature

# Committer
git add .
git commit -m "feat: description"

# Pusher
git push origin feature/nom-feature

# Merge main
git checkout main
git pull
git merge feature/nom-feature
```

---

## 🔑 Variables d'Environnement

### Backend (.env)
```bash
# Copier l'exemple
cp .env.example .env

# Éditer
nano .env
```

### Frontend (.env.local)
```bash
# Créer
echo "NEXT_PUBLIC_API_URL=http://localhost:8080" > frontend/.env.local
```

---

## ⚡ Raccourcis Utiles

### Tout Démarrer (Fresh Start)
```bash
./scripts/dev-clean.sh && ./scripts/dev-start.sh
```

### Rebuild Complet
```bash
docker-compose down -v
docker-compose build --no-cache
docker-compose up -d
```

### Logs Temps Réel
```bash
# Tous services
./scripts/dev-logs.sh

# Backend uniquement
docker-compose logs -f backend

# Avec timestamps
docker-compose logs -f -t backend
```

### Status Rapide
```bash
docker-compose ps && curl -s http://localhost:8080/health | jq
```

---

**Astuce:** Créez des alias dans votre shell pour les commandes fréquentes :

```bash
# Ajouter à ~/.bashrc ou ~/.zshrc
alias dcu="docker-compose up -d"
alias dcd="docker-compose down"
alias dcl="docker-compose logs -f"
alias dcr="docker-compose restart"
alias dps="docker-compose ps"
```

---

**Dernière mise à jour:** 2025-12-08
