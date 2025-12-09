# 🚀 Guide Rapide - Lancer les Tests Backend

**Temps estimé:** 5-10 minutes (incluant installation Go)

---

## Étape 1: Installer Go (si pas déjà fait)

### ✅ Vérifier si Go est déjà installé

```bash
go version
```

**Si vous voyez:** `go version go1.22.x` → ✅ Passer à l'Étape 2

**Si erreur:** `command not found` → ⬇️ Continuer ci-dessous

### 📦 Installation Go

#### Windows
1. Télécharger: https://golang.org/dl/go1.22.9.windows-amd64.msi
2. Double-cliquer sur le .msi et suivre l'installeur
3. Redémarrer votre terminal
4. Vérifier: `go version`

#### macOS
```bash
brew install go
# OU
# Télécharger: https://golang.org/dl/go1.22.9.darwin-amd64.pkg
```

#### Linux
```bash
wget https://golang.org/dl/go1.22.9.linux-amd64.tar.gz
sudo tar -C /usr/local -xzf go1.22.9.linux-amd64.tar.gz
echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc
source ~/.bashrc
go version
```

---

## Étape 2: Lancer les Tests

### Option A: Script Automatique (Recommandé)

```bash
cd backend
chmod +x run_tests.sh
./run_tests.sh
```

Le script va:
- ✅ Vérifier Go installé
- ✅ Télécharger les dépendances
- ✅ Lancer tous les tests
- ✅ Générer le coverage report
- ✅ Créer coverage.html

### Option B: Commandes Manuelles

```bash
cd backend

# 1. Installer dépendances
go mod download
go mod tidy

# 2. Lancer tests
go test ./...

# 3. Avec verbose + coverage
go test -v -cover ./...

# 4. Générer coverage HTML
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out -o coverage.html
```

---

## Étape 3: Interpréter les Résultats

### ✅ Succès

```
=== RUN   TestGetCV_DefaultTheme
--- PASS: TestGetCV_DefaultTheme (0.01s)
=== RUN   TestGetThemes
--- PASS: TestGetThemes (0.00s)
...
PASS
coverage: 75.2% of statements
ok      maicivy/internal/api    1.234s
```

**Résultat:** Tous les tests passent! 🎉

### ❌ Échec

```
--- FAIL: TestGetCV_DefaultTheme (0.01s)
    cv_test.go:45: Expected 200, got 500
FAIL
FAIL    maicivy/internal/api    0.234s
```

**Action:**
1. Noter le nom du test qui échoue
2. Consulter `BACKEND_TESTS_ANALYSIS_AND_FIXES.md`
3. Chercher le fichier concerné dans le doc

---

## 🎯 Objectif Coverage

**Objectif:** 80% coverage

**Vérifier coverage actuel:**
```bash
go test -cover ./... | grep coverage
```

**Voir détails par fichier:**
```bash
go test -coverprofile=coverage.out ./...
go tool cover -func=coverage.out
```

**Fichiers < 80%:**
```bash
go tool cover -func=coverage.out | awk '{if ($3+0 < 80) print $1, $3}'
```

---

## 🐛 Problèmes Courants

### Problème: `go: command not found`

**Solution:** Installer Go (voir Étape 1)

### Problème: `cannot find module`

**Solution:**
```bash
cd backend
go mod download
go mod tidy
```

### Problème: Tests échouent avec `models.CVTheme not found`

**Solution:** ✅ Déjà fixé dans `testing_helpers.go`

Si toujours présent:
```bash
# Vérifier que testing_helpers.go a été corrigé
grep "models.Experience" backend/internal/middleware/testing_helpers.go
```

Devrait retourner une ligne (si fixé).

### Problème: `connection refused` ou Redis errors

**Solution:** ✅ Déjà fixé - utilise miniredis (mock en mémoire)

Si toujours présent:
```bash
# Vérifier que miniredis est utilisé
grep "miniredis" backend/internal/middleware/testing_helpers.go
```

### Problème: Coverage < 80%

**Solution:** Identifier fichiers avec faible coverage

```bash
# Coverage par package
go test -cover ./...

# Fichiers spécifiques < 80%
go test -coverprofile=coverage.out ./...
go tool cover -func=coverage.out | grep -E "\.go" | awk '{if ($3+0 < 80) print}'
```

Ajouter tests pour ces fichiers.

---

## 📊 Tests par Module

### Lancer tests d'un module spécifique

```bash
# Tests API uniquement
go test -v ./internal/api

# Tests Services uniquement
go test -v ./internal/services

# Tests Middleware uniquement
go test -v ./internal/middleware

# Tests Models uniquement
go test -v ./internal/models
```

### Lancer un test spécifique

```bash
# Format: go test -run NomDuTest ./chemin
go test -v -run TestGetCV_DefaultTheme ./internal/api
go test -v -run TestHealth ./internal/api
```

---

## 🏎️ Benchmarks

```bash
# Benchmarks tous modules
go test -bench=. ./...

# Benchmark spécifique
go test -bench=BenchmarkGetCV ./internal/api

# Avec memory stats
go test -bench=. -benchmem ./internal/api
```

---

## 🔍 Debug Tests

### Verbose mode

```bash
go test -v ./internal/api
```

### Avec race detector

```bash
go test -race ./...
```

### Avec logs

```bash
go test -v ./... 2>&1 | tee test.log
```

---

## ✅ Checklist Rapide

- [ ] Go installé (`go version` fonctionne)
- [ ] Dépendances installées (`go mod download`)
- [ ] Tests lancés (`go test ./...`)
- [ ] Tous les tests passent (PASS)
- [ ] Coverage >= 80% (`go test -cover ./...`)
- [ ] Coverage HTML généré (`coverage.html`)

---

## 📚 Documentation Complète

Pour plus de détails, voir:

| Document | Description |
|----------|-------------|
| `TESTS_STATUS.md` | Status court (5 pages) |
| `BACKEND_TESTS_ANALYSIS_AND_FIXES.md` | Analyse complète (40 pages) |
| `FIXES_APPLIED.md` | Résumé corrections (15 pages) |
| `testutil/README.md` | Doc package testutil (10 pages) |

---

## 🆘 Aide

**Si bloqué:**
1. Lire l'erreur complète
2. Chercher dans `BACKEND_TESTS_ANALYSIS_AND_FIXES.md`
3. Vérifier section "Problèmes Courants" ci-dessus

**Temps estimé total:** 5-10 min (avec Go déjà installé)

---

**Prêt? C'est parti! 🚀**

```bash
cd backend
./run_tests.sh
```
