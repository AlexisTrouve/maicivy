# Feature GitHub Import - Exemples d'Utilisation

**Guide pratique avec exemples de code concrets**

---

## 🚀 Quick Start

### 1. Configuration GitHub OAuth App

1. Aller sur https://github.com/settings/developers
2. Cliquer **OAuth Apps** → **New OAuth App**
3. Remplir le formulaire :
   ```
   Application name: maicivy
   Homepage URL: http://localhost:3000
   Authorization callback URL: http://localhost:8080/api/v1/github/callback
   ```
4. Copier **Client ID** et **Client Secret**

### 2. Configuration Backend

```bash
# backend/.env
# ⚠️ EXAMPLE VALUES - Replace with your actual GitHub OAuth credentials
GITHUB_CLIENT_ID=Iv1.YOUR_GITHUB_CLIENT_ID
GITHUB_CLIENT_SECRET=your_github_client_secret_here
GITHUB_REDIRECT_URI=http://localhost:8080/api/v1/github/callback
```

### 3. Démarrer les Services

```bash
# Terminal 1 - Services
docker-compose up -d postgres redis

# Terminal 2 - Backend
cd backend
go run cmd/main.go

# Terminal 3 - Frontend
cd frontend
npm run dev
```

---

## 📱 Exemples Frontend

### Exemple 1: Page Profil avec Import GitHub

```tsx
// app/profile/page.tsx
'use client';

import { GitHubConnect, GitHubStatus, RepoList } from '@/components/github';
import { useGitHubSync } from '@/hooks/useGitHubSync';
import { useState } from 'react';

export default function ProfilePage() {
  const [username, setUsername] = useState('alexiventura'); // Depuis auth session
  const { state, status, repos, error, connect, sync, disconnect } = useGitHubSync(username);

  return (
    <div className="container mx-auto px-4 py-8">
      <h1 className="text-3xl font-bold mb-8">Mon Profil</h1>

      {/* Section GitHub */}
      <section className="mb-12">
        <h2 className="text-2xl font-semibold mb-4">Connexion GitHub</h2>

        {!status?.connected ? (
          // Non connecté: afficher bouton connexion
          <div className="bg-gray-50 rounded-lg p-8 text-center">
            <p className="text-gray-600 mb-4">
              Connectez votre compte GitHub pour importer automatiquement vos projets.
            </p>
            <GitHubConnect
              onConnectSuccess={(username) => {
                console.log('GitHub connecté:', username);
                setUsername(username);
              }}
              onConnectError={(error) => {
                console.error('Erreur connexion:', error);
              }}
            />
          </div>
        ) : (
          // Connecté: afficher status + repos
          <>
            <GitHubStatus
              username={username}
              onSync={() => console.log('Sync déclenchée')}
              onDisconnect={() => {
                console.log('GitHub déconnecté');
                setUsername('');
              }}
            />

            <div className="mt-8">
              <h3 className="text-xl font-semibold mb-4">
                Mes Projets GitHub ({repos.length})
              </h3>
              <RepoList username={username} showPrivate={false} />
            </div>
          </>
        )}

        {/* Afficher erreurs */}
        {error && (
          <div className="mt-4 bg-red-50 border border-red-200 rounded-lg p-4">
            <p className="text-red-600">Erreur: {error}</p>
          </div>
        )}

        {/* Afficher état de sync */}
        {state === 'syncing' && (
          <div className="mt-4 bg-blue-50 border border-blue-200 rounded-lg p-4">
            <p className="text-blue-600">Synchronisation en cours...</p>
          </div>
        )}
      </section>
    </div>
  );
}
```

---

### Exemple 2: Section CV avec Projets GitHub

```tsx
// app/cv/page.tsx
import { RepoList } from '@/components/github';
import { githubApi } from '@/lib/api';

export default async function CVPage() {
  // Fetch repos côté serveur (SSR)
  const username = 'alexiventura'; // Depuis session
  const { repositories } = await githubApi.getRepos(username);

  // Filtrer repos vedettes (stars > 10)
  const featuredRepos = repositories.filter(repo => repo.stars > 10);

  return (
    <div className="container mx-auto px-4 py-8">
      <h1 className="text-4xl font-bold mb-8">Mon CV</h1>

      {/* Autres sections du CV */}
      <section className="mb-12">
        <h2 className="text-2xl font-semibold mb-4">Expériences</h2>
        {/* ... */}
      </section>

      {/* Projets GitHub vedettes */}
      <section className="mb-12">
        <h2 className="text-2xl font-semibold mb-4">
          Projets Open Source ({featuredRepos.length})
        </h2>
        <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
          {featuredRepos.map(repo => (
            <div key={repo.id} className="bg-white border rounded-lg p-6">
              <h3 className="text-lg font-semibold mb-2">
                <a href={repo.url} target="_blank" className="text-blue-600 hover:underline">
                  {repo.repo_name}
                </a>
              </h3>
              <p className="text-sm text-gray-600 mb-3">{repo.description}</p>
              <div className="flex items-center gap-3 text-sm">
                <span className="flex items-center gap-1">
                  ⭐ {repo.stars}
                </span>
                {repo.language && (
                  <span className="px-2 py-1 bg-blue-100 text-blue-800 rounded">
                    {repo.language}
                  </span>
                )}
              </div>
            </div>
          ))}
        </div>
      </section>
    </div>
  );
}
```

---

### Exemple 3: Hook useGitHubSync avec React Query (Optionnel)

```tsx
// hooks/useGitHubSyncQuery.ts (version optimisée avec React Query)
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { githubApi } from '@/lib/api';

export function useGitHubSyncQuery(username?: string) {
  const queryClient = useQueryClient();

  // Query status
  const statusQuery = useQuery({
    queryKey: ['github', 'status', username],
    queryFn: () => username ? githubApi.getStatus(username) : null,
    enabled: !!username,
    staleTime: 60000, // 1 minute
  });

  // Query repos
  const reposQuery = useQuery({
    queryKey: ['github', 'repos', username],
    queryFn: () => username ? githubApi.getRepos(username) : null,
    enabled: !!username && statusQuery.data?.connected,
    staleTime: 3600000, // 1 heure
  });

  // Mutation sync
  const syncMutation = useMutation({
    mutationFn: (username: string) => githubApi.sync(username),
    onSuccess: () => {
      // Invalider queries pour re-fetch
      queryClient.invalidateQueries({ queryKey: ['github', 'status'] });
      queryClient.invalidateQueries({ queryKey: ['github', 'repos'] });
    },
  });

  // Mutation disconnect
  const disconnectMutation = useMutation({
    mutationFn: (username: string) => githubApi.disconnect(username),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['github'] });
    },
  });

  return {
    status: statusQuery.data,
    repos: reposQuery.data?.repositories || [],
    isLoading: statusQuery.isLoading || reposQuery.isLoading,
    error: statusQuery.error || reposQuery.error,
    sync: (username: string) => syncMutation.mutate(username),
    disconnect: (username: string) => disconnectMutation.mutate(username),
    isSyncing: syncMutation.isPending,
  };
}
```

---

## 🔧 Exemples Backend

### Exemple 1: Enregistrer les Routes dans main.go

```go
// cmd/main.go
package main

import (
    "log"
    "os"

    "github.com/gofiber/fiber/v2"
    "github.com/gofiber/fiber/v2/middleware/cors"
    "github.com/redis/go-redis/v9"
    "gorm.io/driver/postgres"
    "gorm.io/gorm"

    "maicivy/backend/internal/api"
    "maicivy/backend/internal/jobs"
    "maicivy/backend/internal/models"
    "maicivy/backend/internal/services"
)

func main() {
    // Init DB
    db, err := gorm.Open(postgres.Open(os.Getenv("DATABASE_URL")), &gorm.Config{})
    if err != nil {
        log.Fatal("Failed to connect to database:", err)
    }

    // Migrer tables GitHub
    db.AutoMigrate(&models.GitHubProfile{}, &models.GitHubRepository{})

    // Init Redis
    redisClient := redis.NewClient(&redis.Options{
        Addr: os.Getenv("REDIS_ADDR"),
    })

    // Init services GitHub
    oauthService := services.NewGitHubOAuthService(db, redisClient)
    syncService := services.NewGitHubSyncService(db, redisClient)

    // Init handler GitHub
    githubHandler := api.NewGitHubHandler(oauthService, syncService)

    // Init Fiber
    app := fiber.New(fiber.Config{
        AppName: "maicivy API",
    })

    // Middlewares
    app.Use(cors.New(cors.Config{
        AllowOrigins: "http://localhost:3000",
        AllowCredentials: true,
    }))

    // Routes
    v1 := app.Group("/api/v1")
    githubHandler.RegisterRoutes(v1)

    // Start cron job GitHub auto-sync
    cronJob := jobs.NewGitHubAutoSyncJob(db, syncService)
    if err := cronJob.Start(); err != nil {
        log.Fatal("Failed to start cron job:", err)
    }
    defer cronJob.Stop()

    log.Println("Server starting on :8080")
    log.Fatal(app.Listen(":8080"))
}
```

---

### Exemple 2: Ajouter Endpoint Custom

```go
// internal/api/github.go (ajouter méthode)

// GetTopRepositories retourne les repos avec le plus de stars
// GET /api/v1/github/repos/top?username=xxx&limit=10
func (h *GitHubHandler) GetTopRepositories(c *fiber.Ctx) error {
    username := c.Query("username")
    limit := c.QueryInt("limit", 10)

    if username == "" {
        return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
            "error": "username_required",
        })
    }

    // Custom query DB
    var repos []models.GitHubRepository
    err := h.syncService.db.
        Where("username = ? AND is_private = false", username).
        Order("stars DESC").
        Limit(limit).
        Find(&repos).Error

    if err != nil {
        return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
            "error": "failed_to_fetch_top_repos",
        })
    }

    return c.JSON(fiber.Map{
        "repositories": repos,
        "count": len(repos),
    })
}

// Enregistrer la nouvelle route
func (h *GitHubHandler) RegisterRoutes(router fiber.Router) {
    github := router.Group("/github")

    // Routes existantes...
    github.Get("/auth-url", h.GetAuthURL)
    github.Get("/callback", h.HandleCallback)
    // ...

    // Nouvelle route
    github.Get("/repos/top", h.GetTopRepositories)
}
```

---

### Exemple 3: Webhooks GitHub (Optionnel)

```go
// internal/api/github_webhooks.go
package api

import (
    "crypto/hmac"
    "crypto/sha256"
    "encoding/hex"
    "encoding/json"
    "os"

    "github.com/gofiber/fiber/v2"
)

// GitHubWebhookHandler gère les webhooks GitHub
type GitHubWebhookHandler struct {
    syncService *services.GitHubSyncService
}

// WebhookPayload structure générique
type WebhookPayload struct {
    Action     string `json:"action"`
    Repository struct {
        FullName string `json:"full_name"`
        Owner    struct {
            Login string `json:"login"`
        } `json:"owner"`
    } `json:"repository"`
}

// HandleWebhook traite les webhooks GitHub (push, star, etc.)
// POST /api/v1/github/webhook
func (h *GitHubWebhookHandler) HandleWebhook(c *fiber.Ctx) error {
    // Vérifier signature GitHub
    signature := c.Get("X-Hub-Signature-256")
    if !h.verifySignature(c.Body(), signature) {
        return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
            "error": "invalid_signature",
        })
    }

    // Parser payload
    var payload WebhookPayload
    if err := json.Unmarshal(c.Body(), &payload); err != nil {
        return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
            "error": "invalid_payload",
        })
    }

    username := payload.Repository.Owner.Login

    // Déclencher sync en background
    go h.syncService.SyncRepositories(c.Context(), username, "")

    return c.JSON(fiber.Map{
        "success": true,
        "message": "webhook_received",
    })
}

// verifySignature vérifie la signature HMAC du webhook
func (h *GitHubWebhookHandler) verifySignature(body []byte, signature string) bool {
    if signature == "" {
        return false
    }

    // Extraire signature (format: sha256=xxx)
    if len(signature) < 7 || signature[:7] != "sha256=" {
        return false
    }
    expectedSig := signature[7:]

    // Calculer HMAC
    secret := os.Getenv("GITHUB_WEBHOOK_SECRET")
    mac := hmac.New(sha256.New, []byte(secret))
    mac.Write(body)
    calculatedSig := hex.EncodeToString(mac.Sum(nil))

    return hmac.Equal([]byte(expectedSig), []byte(calculatedSig))
}
```

---

## 📊 Exemples d'Utilisation API

### cURL - Flow Complet

```bash
# 1. Obtenir l'URL OAuth
curl http://localhost:8080/api/v1/github/auth-url

# Response:
# {
#   "auth_url": "https://github.com/login/oauth/authorize?client_id=xxx&..."
# }

# 2. User ouvre l'URL dans un navigateur et autorise
# GitHub redirige vers: /api/v1/github/callback?code=xxx&state=yyy

# 3. Vérifier status connexion
curl "http://localhost:8080/api/v1/github/status?username=alexiventura"

# Response:
# {
#   "connected": true,
#   "username": "alexiventura",
#   "last_sync": 1733664000,
#   "repo_count": 42
# }

# 4. Récupérer repos
curl "http://localhost:8080/api/v1/github/repos?username=alexiventura"

# Response:
# {
#   "repositories": [...]
# }

# 5. Déclencher sync manuel
curl -X POST http://localhost:8080/api/v1/github/sync \
  -H "Content-Type: application/json" \
  -d '{"username":"alexiventura"}'

# Response:
# {
#   "status": "sync_started",
#   "username": "alexiventura"
# }

# 6. Déconnecter
curl -X DELETE "http://localhost:8080/api/v1/github/disconnect?username=alexiventura"

# Response:
# {
#   "success": true,
#   "message": "GitHub account alexiventura disconnected"
# }
```

---

### JavaScript - Fetch API

```javascript
// 1. Connexion GitHub
async function connectGitHub() {
  const response = await fetch('http://localhost:8080/api/v1/github/auth-url');
  const { auth_url } = await response.json();

  // Ouvrir popup
  const popup = window.open(auth_url, 'GitHub OAuth', 'width=600,height=700');

  // Attendre fermeture popup
  const interval = setInterval(() => {
    if (popup.closed) {
      clearInterval(interval);
      checkGitHubStatus();
    }
  }, 500);
}

// 2. Vérifier status
async function checkGitHubStatus() {
  const username = 'alexiventura';
  const response = await fetch(
    `http://localhost:8080/api/v1/github/status?username=${username}`
  );
  const status = await response.json();

  console.log('GitHub connecté:', status.connected);
  console.log('Repos importés:', status.repo_count);
}

// 3. Récupérer repos
async function fetchRepos() {
  const username = 'alexiventura';
  const response = await fetch(
    `http://localhost:8080/api/v1/github/repos?username=${username}`
  );
  const { repositories } = await response.json();

  console.log('Repos:', repositories);
}

// 4. Sync manuel
async function syncRepos() {
  const username = 'alexiventura';
  const response = await fetch('http://localhost:8080/api/v1/github/sync', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ username }),
  });

  const result = await response.json();
  console.log('Sync démarrée:', result.status);
}
```

---

## 🗄️ Exemples SQL

### Requêtes Utiles

```sql
-- 1. Lister tous les profils GitHub connectés
SELECT id, username, connected_at, synced_at
FROM github_profiles
ORDER BY connected_at DESC;

-- 2. Compter repos par utilisateur
SELECT username, COUNT(*) as repo_count, SUM(stars) as total_stars
FROM github_repositories
GROUP BY username
ORDER BY total_stars DESC;

-- 3. Top repos par stars
SELECT repo_name, full_name, stars, language
FROM github_repositories
WHERE is_private = false
ORDER BY stars DESC
LIMIT 10;

-- 4. Repos par language
SELECT language, COUNT(*) as count
FROM github_repositories
WHERE language IS NOT NULL
GROUP BY language
ORDER BY count DESC;

-- 5. Derniers repos importés
SELECT username, repo_name, stars, pushed_at
FROM github_repositories
ORDER BY created_at DESC
LIMIT 20;

-- 6. Stats globales
SELECT
  COUNT(DISTINCT username) as total_users,
  COUNT(*) as total_repos,
  AVG(stars) as avg_stars,
  SUM(CASE WHEN is_private THEN 1 ELSE 0 END) as private_repos
FROM github_repositories;
```

---

## 🔍 Exemples Redis

### Commandes Debug

```bash
# 1. Vérifier state OAuth
redis-cli GET "github:oauth:state:abc123xyz"

# 2. Vérifier cache repos
redis-cli GET "github:repos:alexiventura"

# 3. Invalider cache (force re-sync)
redis-cli DEL "github:repos:alexiventura"

# 4. Lister toutes les clés GitHub
redis-cli KEYS "github:*"

# 5. Vérifier TTL d'une clé
redis-cli TTL "github:repos:alexiventura"

# 6. Monitorer activité Redis en temps réel
redis-cli MONITOR
```

---

## 🧪 Exemples de Tests

### Test Intégration OAuth Flow

```go
// internal/api/github_integration_test.go
func TestGitHubOAuthFlow_Integration(t *testing.T) {
    // Setup
    app := setupTestApp()
    client := &http.Client{
        CheckRedirect: func(req *http.Request, via []*http.Request) error {
            return http.ErrUseLastResponse // Ne pas suivre redirects
        },
    }

    // 1. Récupérer auth URL
    resp, _ := client.Get("http://localhost:8080/api/v1/github/auth-url")
    var authData map[string]string
    json.NewDecoder(resp.Body).Decode(&authData)

    authURL := authData["auth_url"]
    assert.Contains(t, authURL, "github.com/login/oauth/authorize")

    // 2. Simuler callback GitHub (avec mock token)
    mockCode := "mock_code_123"
    mockState := extractStateFromURL(authURL)

    resp, _ = client.Get(fmt.Sprintf(
        "http://localhost:8080/api/v1/github/callback?code=%s&state=%s",
        mockCode, mockState,
    ))

    var callbackData map[string]interface{}
    json.NewDecoder(resp.Body).Decode(&callbackData)

    assert.True(t, callbackData["success"].(bool))
    assert.NotEmpty(t, callbackData["username"])
}
```

---

## 🎯 Scénarios d'Utilisation

### Scénario 1: Premier Utilisateur

```
1. User visite /profile pour la première fois
2. Voit message "Connectez votre GitHub"
3. Clique "Connecter GitHub" → popup OAuth
4. Autorise l'app sur GitHub
5. Popup se ferme → status passe à "Connecté"
6. Sync automatique démarre en background (10s)
7. Liste de repos s'affiche (42 repos trouvés)
8. User peut cliquer sur un repo pour voir détails
```

### Scénario 2: Utilisateur Existant

```
1. User visite /profile
2. Status GitHub: "Connecté - Dernière synchro: il y a 12h"
3. User push un nouveau projet sur GitHub
4. User clique "Synchroniser maintenant"
5. Loading pendant 5 secondes
6. Nouveau projet apparaît dans la liste
```

### Scénario 3: Cron Job Quotidien

```
# Chaque jour à 2h du matin
1. Cron job démarre
2. Récupère 150 profils GitHub connectés
3. Pour chaque profil:
   - Sync repos via API GitHub
   - Update synced_at timestamp
   - Sleep 2 secondes
4. Logs: "150/150 profiles synced - Duration: 5m30s"
```

---

**Fin des exemples**

Pour plus de détails, voir:
- `backend/GITHUB_IMPORT_IMPLEMENTATION_SUMMARY.md` - Documentation complète
- `GITHUB_FEATURE_DELIVERABLES.md` - Liste des fichiers créés

**Questions ? Consultez la section Dépannage de la documentation.**
