# 🪟 Installation Windows Native (Sans Docker)

Guide pour installer et tester maicivy sur Windows sans Docker.

---

## Étape 1: Installer PostgreSQL + Redis

**Ouvrir PowerShell EN ADMINISTRATEUR** (clic droit > "Exécuter en tant qu'administrateur")

```powershell
cd C:\Users\alexi\Documents\projects\maicivy
.\install-windows-native.ps1
```

**OU** manuellement :

```powershell
choco install postgresql14 -y --params "/Password:postgres"
choco install redis-64 -y
```

**Temps estimé:** 5-10 minutes

---

## Étape 2: Configurer la Base de Données

**Ouvrir Git Bash** (terminal normal, pas admin)

```bash
cd /c/Users/alexi/Documents/projects/maicivy
chmod +x setup-database-native.sh
./setup-database-native.sh
```

Cela va :
- ✅ Créer la base `maicivy_db`
- ✅ Créer l'utilisateur `maicivy`
- ✅ Charger le schéma SQL
- ✅ Charger les données de test (7 expériences, 20 skills, 8 projets)

---

## Étape 3: Démarrer Redis

**Dans un terminal Git Bash séparé :**

```bash
"/c/Program Files/Redis/redis-server.exe"
```

**Laisser tourner** (ne pas fermer ce terminal)

---

## Étape 4: Configurer le Backend

Créer `backend/.env` (copie de `.env.example`) :

```bash
cd backend
cp .env.example .env
```

**Éditer `backend/.env`** et vérifier ces lignes :

```env
# PostgreSQL natif Windows (pas Docker)
DB_HOST=localhost
DB_PORT=5432
DB_USER=maicivy
DB_PASSWORD=maicivy_password
DB_NAME=maicivy_db

# Redis natif Windows (pas Docker)
REDIS_URL=localhost:6379
REDIS_PASSWORD=

# API Keys (optionnel pour tests de base)
ANTHROPIC_API_KEY=
OPENAI_API_KEY=
```

---

## Étape 5: Démarrer le Backend

**Dans Git Bash :**

```bash
cd backend
go run cmd/main.go
```

Vous devriez voir :

```
2025-12-15T... INF Server starting on :8080
2025-12-15T... INF Database connected
2025-12-15T... INF Redis connected
2025-12-15T... INF All services initialized
```

---

## Étape 6: Tester l'API

**Dans un nouveau terminal :**

```bash
# Health check
curl http://localhost:8080/health

# CV adaptatif (thème backend)
curl "http://localhost:8080/api/v1/cv?theme=backend"

# Liste des skills
curl http://localhost:8080/api/v1/skills

# Liste des projets
curl http://localhost:8080/api/v1/projects

# Swagger UI
# Ouvrir dans navigateur: http://localhost:8080/api/docs
```

---

## 🛠️ Commandes Utiles

### PostgreSQL

```bash
# Se connecter à la base
"/c/Program Files/PostgreSQL/14/bin/psql.exe" -U maicivy -d maicivy_db

# Vérifier les tables
\dt

# Compter les expériences
SELECT COUNT(*) FROM experiences;

# Quitter
\q
```

### Redis

```bash
# Se connecter à Redis
"/c/Program Files/Redis/redis-cli.exe"

# Tester
PING
# Réponse: PONG

# Lister les clés
KEYS *

# Quitter
EXIT
```

### Arrêter les Services

```bash
# Arrêter le backend: Ctrl+C dans le terminal du backend
# Arrêter Redis: Ctrl+C dans le terminal Redis
# PostgreSQL tourne en service Windows (s'arrête avec l'ordinateur)
```

---

## ❌ Dépannage

### PostgreSQL ne démarre pas

```powershell
# PowerShell ADMIN
Get-Service postgresql*
Start-Service postgresql-x64-14
```

### Redis ne démarre pas

```bash
# Vérifier si le port 6379 est déjà utilisé
netstat -ano | findstr :6379

# Tuer le processus si besoin
taskkill /PID <PID> /F
```

### Backend erreur "database connection failed"

1. Vérifier que PostgreSQL tourne : `Get-Service postgresql-x64-14`
2. Vérifier les credentials dans `backend/.env`
3. Tester la connexion manuellement : `psql -U maicivy -d maicivy_db`

### Backend erreur "redis connection failed"

1. Vérifier que Redis tourne : `redis-cli PING`
2. Vérifier `REDIS_URL` dans `backend/.env`

---

## 📊 Architecture (Sans Docker)

```
Windows Services:
├── PostgreSQL 14 (port 5432)
└── Redis (port 6379, manuel)

Backend Go:
└── Fiber API (port 8080)
    ├── Connexion PostgreSQL (GORM)
    ├── Connexion Redis
    └── 37+ endpoints REST
```

---

## ✅ Avantages / Inconvénients

**✅ Avantages :**
- Pas besoin de Docker Desktop
- Installation rapide (< 15 min)
- PostgreSQL et Redis directement dans Windows

**❌ Inconvénients :**
- Besoin de gérer 2 terminaux (Redis + Backend)
- Services Windows à démarrer manuellement
- Moins "portable" que Docker

---

**Prochaine étape :** Frontend Next.js (Phase 2)
