# Instructions de Test - Backend Foundation

Ce fichier contient les commandes exactes pour tester le backend foundation.

## ⚡ Test Rapide (2 minutes)

### 1. Installer les dépendances

```bash
cd backend
go mod download
```

**Résultat attendu :** Téléchargement de ~20 packages sans erreur.

### 2. Vérifier la compilation

```bash
go build ./cmd/main.go
```

**Résultat attendu :** Création du fichier `main` (ou `main.exe` sur Windows) sans erreur.

### 3. Lancer les tests unitaires

```bash
go test -v -short ./...
```

**Résultat attendu :**
```
=== RUN   TestLoad
--- PASS: TestLoad
=== RUN   TestGetEnv
--- PASS: TestGetEnv
=== RUN   TestAppError
--- PASS: TestAppError
=== RUN   TestErrorConstructors
--- PASS: TestErrorConstructors
PASS
```

**Si ces 3 étapes passent, le code est correct !** ✅

---

## 🚀 Test Complet (5 minutes)

### 1. Démarrer les services Docker

Depuis la racine du projet:

```bash
docker-compose up -d postgres redis
```

Vérifier:
```bash
docker-compose ps
```

Vous devez voir `maicivy-postgres` et `maicivy-redis` en status `Up`.

### 2. Configurer l'environnement

```bash
cd backend
cp .env.example .env
```

Éditez `.env` et vérifiez ces lignes (pour dev local):
```env
DB_HOST=localhost
DB_PASSWORD=maicivy_secure_password_2024
REDIS_HOST=localhost
```

### 3. Lancer le backend

```bash
go run cmd/main.go
```

**Logs attendus:**
```
11:30AM INF Logger initialized environment=development
11:30AM INF PostgreSQL connected successfully database=maicivy host=localhost
11:30AM INF Redis connected successfully addr=localhost:6379 db=0
11:30AM INF Starting server addr=0.0.0.0:8080 environment=development
```

### 4. Tester les endpoints

Dans un autre terminal:

```bash
# Test 1: Health check simple
curl http://localhost:8080/health

# Résultat attendu:
# {"status":"ok","services":{"api":"up"}}

# Test 2: Health check complet
curl http://localhost:8080/health/deep

# Résultat attendu:
# {"status":"ok","services":{"api":"up","postgres":"up","redis":"up"}}

# Test 3: API info
curl http://localhost:8080/api/v1/

# Résultat attendu:
# {"message":"maicivy API v1","version":"1.0.0"}
```

### 5. Vérifier les logs

Retournez au terminal du backend. Vous devez voir:
```
11:31AM INF HTTP request duration_ms=0.5 method=GET path=/health request_id=... status=200
11:31AM INF HTTP request duration_ms=2.3 method=GET path=/health/deep request_id=... status=200
11:31AM INF HTTP request duration_ms=0.2 method=GET path=/api/v1/ request_id=... status=200
```

### 6. Arrêter proprement

Dans le terminal du backend, appuyez sur `Ctrl+C`.

**Log attendu:**
```
11:32AM INF Shutting down server...
11:32AM INF Server stopped
```

**Si tous les tests passent, le backend foundation est 100% opérationnel !** 🎉

---

## 🐛 En Cas d'Erreur

### Erreur de compilation

```
# Vérifier la version de Go
go version

# Doit être >= 1.22

# Nettoyer et réinstaller
go clean -modcache
go mod download
```

### Erreur "cannot find package"

```
# Vérifier que go.mod existe
cat go.mod

# Réinstaller les dépendances
go mod tidy
go mod download
```

### Erreur "Failed to connect to PostgreSQL"

```
# Vérifier que le container tourne
docker-compose ps postgres

# Vérifier les logs
docker-compose logs postgres

# Redémarrer si nécessaire
docker-compose restart postgres

# Attendre 5 secondes puis réessayer
```

### Erreur "address already in use"

```
# Un autre processus utilise le port 8080
# Option 1: Arrêter l'autre processus
# Option 2: Changer le port
echo "SERVER_PORT=8081" >> .env
```

---

## ✅ Checklist Finale

- [ ] `go mod download` réussit
- [ ] `go build ./cmd/main.go` compile sans erreur
- [ ] `go test -v -short ./...` passe tous les tests
- [ ] Docker PostgreSQL et Redis démarrés
- [ ] Fichier `.env` créé
- [ ] `go run cmd/main.go` démarre le serveur
- [ ] Log "PostgreSQL connected successfully"
- [ ] Log "Redis connected successfully"
- [ ] `curl /health` retourne `{"status":"ok"}`
- [ ] `curl /health/deep` retourne tous les services "up"
- [ ] Logs HTTP visibles dans le terminal
- [ ] `Ctrl+C` arrête proprement le serveur

**Si toutes les cases sont cochées : Backend Foundation validé ! ✅**

---

## 📊 Statistiques Attendues

**Compilation:**
- Temps : ~5-10 secondes (première fois)
- Taille binaire : ~15-20 MB

**Tests unitaires:**
- Durée : <1 seconde
- Coverage : ~60-70% (avec les fichiers de test actuels)

**Démarrage:**
- Temps : ~1-2 secondes (après connexions DB)
- Mémoire : ~20-30 MB (sans charge)

**Performance:**
- `/health` : <5ms
- `/health/deep` : <20ms (avec DB pings)

---

## 🎯 Objectif Accompli

Une fois ces tests validés, vous avez:

- ✅ Backend Go structuré et organisé
- ✅ Framework Fiber configuré
- ✅ Connexions DB (PostgreSQL + Redis) opérationnelles
- ✅ Logging structuré actif
- ✅ Error handling robuste
- ✅ Tests unitaires passants
- ✅ Health checks fonctionnels
- ✅ Graceful shutdown

**Prêt pour la Phase suivante: Database Schema (03)** 🚀
