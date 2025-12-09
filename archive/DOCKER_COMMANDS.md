# Docker Commands - Aide-Mémoire

Guide rapide des commandes Docker pour le projet maicivy.

---

## 🚀 Démarrage Initial

```bash
# 1. Copier le fichier d'environnement
cp .env.example .env

# 2. Éditer .env et remplir vos API keys
# CLAUDE_API_KEY=sk-ant-...
# OPENAI_API_KEY=sk-...

# 3. Démarrer tous les services
docker compose up -d
# OU (avec docker-compose v1)
docker-compose up -d

# 4. Vérifier la santé des services
# Linux/Mac/WSL
./scripts/health-check.sh

# Windows PowerShell
.\scripts\health-check.ps1
```

---

## 📊 Monitoring et Logs

```bash
# Voir l'état de tous les services
docker compose ps

# Voir les logs de tous les services (temps réel)
docker compose logs -f

# Voir les logs d'un service spécifique
docker compose logs -f backend
docker compose logs -f frontend
docker compose logs -f postgres
docker compose logs -f redis

# Voir les dernières 100 lignes
docker compose logs --tail=100 backend

# Voir les logs depuis une date
docker compose logs --since 2h backend
```

---

## 🔄 Gestion des Services

### Démarrer/Arrêter

```bash
# Démarrer tous les services
docker compose up -d

# Démarrer un service spécifique
docker compose up -d backend

# Arrêter tous les services
docker compose down

# Arrêter SANS supprimer les volumes
docker compose stop

# Redémarrer tous les services
docker compose restart

# Redémarrer un service spécifique
docker compose restart backend
```

### Rebuild

```bash
# Rebuilder toutes les images (sans cache)
docker compose build --no-cache

# Rebuilder une image spécifique
docker compose build --no-cache backend

# Rebuilder ET redémarrer
docker compose up -d --build
```

---

## 🗃️ Base de Données (PostgreSQL)

```bash
# Accéder à la console PostgreSQL
docker exec -it maicivy-postgres psql -U maicivy -d maicivy_db

# Vérifier la connexion
docker exec maicivy-postgres pg_isready -U maicivy

# Lister les bases de données
docker exec maicivy-postgres psql -U maicivy -c "\l"

# Lister les tables
docker exec maicivy-postgres psql -U maicivy -d maicivy_db -c "\dt"

# Exécuter une requête SQL
docker exec maicivy-postgres psql -U maicivy -d maicivy_db -c "SELECT * FROM visitors LIMIT 10;"

# Dump de la base de données
docker exec maicivy-postgres pg_dump -U maicivy maicivy_db > backup.sql

# Restaurer un dump
docker exec -i maicivy-postgres psql -U maicivy -d maicivy_db < backup.sql
```

### Commandes SQL Utiles (dans psql)

```sql
-- Lister les tables
\dt

-- Décrire une table
\d table_name

-- Lister les index
\di

-- Voir les connexions actives
SELECT * FROM pg_stat_activity;

-- Taille de la base
SELECT pg_size_pretty(pg_database_size('maicivy_db'));

-- Quitter psql
\q
```

---

## 🔴 Cache (Redis)

```bash
# Accéder au Redis CLI
docker exec -it maicivy-redis redis-cli

# Tester la connexion
docker exec maicivy-redis redis-cli ping
# Réponse attendue : PONG

# Voir toutes les clés
docker exec maicivy-redis redis-cli KEYS '*'

# Voir la valeur d'une clé
docker exec maicivy-redis redis-cli GET "key_name"

# Voir les infos serveur
docker exec maicivy-redis redis-cli INFO server

# Voir la mémoire utilisée
docker exec maicivy-redis redis-cli INFO memory

# Vider le cache (⚠️ ATTENTION)
docker exec maicivy-redis redis-cli FLUSHALL
```

### Commandes Redis Utiles (dans redis-cli)

```bash
# Tester la connexion
ping

# Voir toutes les clés
KEYS *

# Voir une valeur
GET key_name

# Supprimer une clé
DEL key_name

# Voir le TTL (temps restant avant expiration)
TTL key_name

# Voir le nombre de clés
DBSIZE

# Infos serveur
INFO

# Quitter redis-cli
quit
```

---

## 🐳 Conteneurs

```bash
# Lister tous les conteneurs
docker ps -a

# Lister les conteneurs maicivy
docker ps -a | grep maicivy

# Inspecter un conteneur
docker inspect maicivy-backend

# Voir les stats (CPU, RAM, etc.)
docker stats

# Accéder à un shell dans un conteneur
docker exec -it maicivy-backend sh
docker exec -it maicivy-frontend sh

# Copier un fichier depuis/vers un conteneur
docker cp maicivy-backend:/app/logs/app.log ./local-logs/
docker cp ./local-file.txt maicivy-backend:/app/
```

---

## 💾 Volumes

```bash
# Lister tous les volumes
docker volume ls

# Lister les volumes maicivy
docker volume ls | grep maicivy

# Inspecter un volume
docker volume inspect maicivy_postgres-data

# Voir l'emplacement des données
docker volume inspect maicivy_postgres-data -f '{{ .Mountpoint }}'

# Supprimer un volume (⚠️ PERTE DE DONNÉES)
docker volume rm maicivy_postgres-data

# Supprimer tous les volumes non utilisés (⚠️ DANGER)
docker volume prune
```

---

## 🌐 Networks

```bash
# Lister les networks
docker network ls

# Inspecter le network maicivy
docker network inspect maicivy

# Voir les conteneurs connectés au network
docker network inspect maicivy -f '{{range .Containers}}{{.Name}} {{.IPv4Address}}{{"\n"}}{{end}}'
```

---

## 🧹 Nettoyage

```bash
# Arrêter et supprimer les conteneurs
docker compose down

# Arrêter et supprimer conteneurs + volumes (⚠️ PERTE DE DONNÉES)
docker compose down -v

# Arrêter et supprimer conteneurs + volumes + images
docker compose down -v --rmi all

# Nettoyer les images non utilisées
docker image prune -a

# Nettoyer tout (conteneurs, images, volumes, networks)
docker system prune -a --volumes
```

### Fresh Start Complet

```bash
# ⚠️ ATTENTION : Supprime TOUT et redémarre
docker compose down -v --rmi all
docker compose up -d --build
```

---

## 🔍 Debugging

### Service ne démarre pas

```bash
# Voir les logs détaillés
docker compose logs backend

# Voir l'état du healthcheck
docker inspect maicivy-backend --format='{{.State.Health.Status}}'

# Redémarrer le service
docker compose restart backend

# Rebuilder l'image
docker compose build --no-cache backend
docker compose up -d backend
```

### Problème de connexion entre services

```bash
# Vérifier que les services sont sur le même network
docker network inspect maicivy

# Tester la connexion depuis un conteneur
docker exec -it maicivy-backend ping postgres
docker exec -it maicivy-backend curl http://redis:6379

# Vérifier les variables d'environnement
docker exec maicivy-backend env | grep DB_
```

### Ports déjà utilisés

```bash
# Trouver quel processus utilise un port (Windows)
netstat -ano | findstr :8080
# Puis tuer le processus : taskkill /PID <PID> /F

# Trouver quel processus utilise un port (Linux/Mac)
lsof -i :8080
# Puis tuer le processus : kill -9 <PID>

# Ou modifier le port dans .env
# Exemple : BACKEND_PORT=8081
```

---

## 📈 Performance

```bash
# Voir l'utilisation des ressources
docker stats

# Voir l'utilisation d'un conteneur spécifique
docker stats maicivy-backend

# Limiter la mémoire d'un service (dans docker-compose.yml)
# deploy:
#   resources:
#     limits:
#       memory: 512M
```

---

## 🔐 Sécurité

```bash
# Vérifier les images pour vulnérabilités (si Docker Scout activé)
docker scout cves maicivy-backend

# Voir les logs de sécurité
docker compose logs | grep -i "error\|warning\|security"

# Vérifier que .env n'est pas commité
git status
# .env ne doit PAS apparaître

# Vérifier les permissions des fichiers secrets
ls -la .env
# Devrait être -rw------- (600)
```

---

## ⚡ Astuces

### Alias Utiles

Ajoutez ces alias à votre `.bashrc` ou `.zshrc` (Linux/Mac) :

```bash
alias dc='docker compose'
alias dcup='docker compose up -d'
alias dcdown='docker compose down'
alias dclogs='docker compose logs -f'
alias dcps='docker compose ps'
alias dcrestart='docker compose restart'
```

Ou PowerShell (`$PROFILE`) (Windows) :

```powershell
function dc { docker compose $args }
function dcup { docker compose up -d }
function dcdown { docker compose down }
function dclogs { docker compose logs -f $args }
```

### Watch Mode

```bash
# Suivre les logs avec couleurs
docker compose logs -f --tail=50 | grep --color -E 'ERROR|WARN|INFO|$'

# Surveiller l'état des conteneurs (Linux/Mac)
watch -n 2 'docker compose ps'
```

---

## 📝 Fichiers Importants

- `docker-compose.yml` - Configuration des services
- `.env` - Variables d'environnement (NE PAS COMMITER)
- `.env.example` - Template des variables
- `docker/redis/redis.conf` - Configuration Redis
- `backend/Dockerfile` - Build backend
- `frontend/Dockerfile` - Build frontend

---

## 🆘 Aide

```bash
# Aide Docker Compose
docker compose --help

# Aide pour une commande spécifique
docker compose up --help
docker compose logs --help

# Version Docker
docker --version
docker compose version

# Informations système Docker
docker info
```

---

## 🔗 Liens Utiles

- [Docker Compose Reference](https://docs.docker.com/compose/compose-file/)
- [Docker CLI Reference](https://docs.docker.com/engine/reference/commandline/cli/)
- [PostgreSQL Docker Docs](https://hub.docker.com/_/postgres)
- [Redis Docker Docs](https://hub.docker.com/_/redis)

---

**Dernière mise à jour:** 2025-12-08
