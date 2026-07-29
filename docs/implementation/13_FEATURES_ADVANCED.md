# 13. FEATURES_ADVANCED.md

## 📋 Métadonnées

- **Phase:** 5
- **Priorité:** 🔵 BASSE
- **Complexité:** ⭐⭐⭐ (3/5)
- **Prérequis:** 06, 07 (CV Backend & Frontend)
- **Temps estimé:** Variable selon features (1-3 semaines)
- **Status:** 🔲 À faire

---

## 🎯 Objectif

Implémenter un ensemble de fonctionnalités avancées optionnelles qui enrichissent l'expérience utilisateur et démontrent des compétences techniques supplémentaires :

1. **Import Automatique GitHub** - Synchronisation API GitHub
2. **Timeline Interactive** - Visualisation chronologique animée
3. **Détection Profils Avancée** - Reconnaissance de profils cibles
4. **Effets 3D (optionnel)** - Visualisation 3D interactive
5. **Multi-langue (optionnel)** - Support FR/EN avec traduction IA

---

## 🏗️ Architecture

### Vue d'Ensemble

```
┌─────────────────────────────────────────────────────────┐
│           FEATURES AVANCÉES - Phase 5                   │
└─────────────────────────────────────────────────────────┘
                          │
        ┌─────────────────┼─────────────────┐
        │                 │                 │
        ▼                 ▼                 ▼
   1. GITHUB         2. TIMELINE        3. PROFILING
   IMPORT            INTERACTIVE        DETECTION
        │                 │                 │
        └─────────────────┼─────────────────┘
                          │
        ┌─────────────────┼─────────────────┐
        │                 │                 │
        ▼                 ▼                 ▼
   4. 3D EFFECTS   5. MULTI-LANGUE      INTÉGRATION
   (Optionnel)     (Optionnel)          GLOBALE
```

### Design Decisions

**1. Features Modulaires**
- Chaque feature est indépendante
- Peuvent être activées/désactivées via feature flags
- Dégradation gracieuse si API externe échoue

**2. Dépendances Externes**
- GitHub API (lecture seule, authentification optionnelle)
- Clearbit API (détection entreprises)
- Claude API (traductions multi-langue)

**3. Caching Stratégique**
- GitHub repos: TTL 24h (peu de changements)
- Clearbit: TTL 7j (données profil stables)
- Traductions: TTL infini (contenu statique)

---

## 📦 Dépendances

### Bibliothèques Go

```bash
# Import GitHub
go get github.com/google/go-github/v60

# HTTP Client optimisé
go get github.com/go-resty/resty/v2

# Détection profil (IP lookup)
go get github.com/oschwald/geoip2-golang

# User-Agent parsing
go get github.com/ua-parser/uaparser-go/ua_parser

# Structures de données
go get github.com/iancoleman/strcase
```

### Bibliothèques NPM

```bash
# Timeline & animations
npm install framer-motion react-intersection-observer

# 3D (si utilisé)
npm install three @react-three/fiber @react-three/drei

# Multi-langue
npm install next-intl

# Charts et visualisations
npm install recharts

# Internationalization
npm install i18next react-i18next
```

### Services Externes

- **GitHub API** (gratuit avec rate limit 60 req/h non authentifié, 5000 req/h authentifié)
- **Clearbit API** (API free tier limité, nécessite clé API)
- **Claude API** (pour traductions dynamiques, nécessite clé API)
- **MaxMind GeoIP2** (pour IP lookup, données libres ou entreprise)

---

## 🔨 Implémentation

### Feature 1: Import Automatique GitHub

#### Étape 1.1: Configuration OAuth GitHub

**Description:** Permettre à l'utilisateur de connecter son compte GitHub et d'autoriser l'accès à ses repos.

**Code Backend (Go):**

```go
// backend/internal/models/github_oauth.go
package models

import (
    "database/sql/driver"
    "encoding/json"
)

// GitHubToken structure OAuth token
type GitHubToken struct {
    AccessToken string `json:"access_token"`
    TokenType   string `json:"token_type"`
    ExpiresAt   int64  `json:"expires_at"`
}

// Scan pour GORM
func (gt *GitHubToken) Scan(value interface{}) error {
    bytes, _ := value.([]byte)
    return json.Unmarshal(bytes, &gt)
}

// Value pour GORM
func (gt GitHubToken) Value() (driver.Value, error) {
    return json.Marshal(gt)
}

// GitHubProfile utilisateur GitHub
type GitHubProfile struct {
    ID        uint          `gorm:"primaryKey"`
    Username  string        `gorm:"uniqueIndex"`
    Token     GitHubToken   `gorm:"type:jsonb"`
    ConnectedAt int64
    SyncedAt  int64
}

// backend/internal/api/github_oauth.go
package api

import (
    "crypto/rand"
    "encoding/base64"
    "github.com/gofiber/fiber/v2"
    "os"
)

// GitHub OAuth endpoints
type GitHubOAuthHandler struct {
    // ... dependencies
}

// GetAuthURL génère l'URL d'authentification GitHub
func (h *GitHubOAuthHandler) GetAuthURL(c *fiber.Ctx) error {
    state := generateRandomState()

    // Stocker state en Redis (TTL 10 min)
    h.redis.Set(c.Context(), "github:oauth:state:"+state, "true", 10*time.Minute)

    url := fmt.Sprintf(
        "https://github.com/login/oauth/authorize?client_id=%s&redirect_uri=%s&scope=%s&state=%s",
        os.Getenv("GITHUB_CLIENT_ID"),
        os.Getenv("GITHUB_REDIRECT_URI"),
        "user:email,public_repo",
        state,
    )

    return c.JSON(fiber.Map{"auth_url": url})
}

// HandleCallback traite la réponse OAuth
func (h *GitHubOAuthHandler) HandleCallback(c *fiber.Ctx) error {
    code := c.Query("code")
    state := c.Query("state")

    // Vérifier state
    exists, _ := h.redis.Exists(c.Context(), "github:oauth:state:"+state).Result()
    if exists == 0 {
        return c.Status(400).JSON(fiber.Map{"error": "invalid_state"})
    }

    // Échanger code contre token
    token, err := h.exchangeCodeForToken(code)
    if err != nil {
        return c.Status(400).JSON(fiber.Map{"error": "token_exchange_failed"})
    }

    // Récupérer infos utilisateur GitHub
    githubUser, err := h.getGitHubUser(token.AccessToken)
    if err != nil {
        return c.Status(400).JSON(fiber.Map{"error": "failed_to_fetch_user"})
    }

    // Sauvegarder/mettre à jour profil
    profile := &GitHubProfile{
        Username: githubUser.Login,
        Token:    token,
        ConnectedAt: time.Now().Unix(),
    }

    h.db.Save(profile)

    // Déclencher sync initial
    go h.syncRepositories(githubUser.Login, token.AccessToken)

    return c.JSON(fiber.Map{
        "success": true,
        "username": githubUser.Login,
    })
}

func (h *GitHubOAuthHandler) exchangeCodeForToken(code string) (*GitHubToken, error) {
    client := resty.New()

    resp, err := client.R().
        SetHeader("Accept", "application/json").
        SetFormData(map[string]string{
            "client_id": os.Getenv("GITHUB_CLIENT_ID"),
            "client_secret": os.Getenv("GITHUB_CLIENT_SECRET"),
            "code": code,
        }).
        Post("https://github.com/login/oauth/access_token")

    if err != nil {
        return nil, err
    }

    var token GitHubToken
    if err := json.Unmarshal(resp.Body(), &token); err != nil {
        return nil, err
    }

    return &token, nil
}

func generateRandomState() string {
    b := make([]byte, 32)
    rand.Read(b)
    return base64.StdEncoding.EncodeToString(b)
}
```

#### Étape 1.2: Synchronisation API GitHub

**Description:** Récupérer les repos GitHub et les stocker en base de données.

**Code Backend (Go):**

```go
// backend/internal/services/github_sync.go
package services

import (
    "context"
    "github.com/google/go-github/v60/github"
)

type GitHubService struct {
    db    *gorm.DB
    redis *redis.Client
}

// SyncRepositories récupère tous les repos de l'utilisateur
func (s *GitHubService) SyncRepositories(ctx context.Context, username, token string) error {
    client := github.NewClient(nil).WithAuthToken(token)

    // Lister tous les repos (publics et privés accessibles)
    opts := &github.ListOptions{PerPage: 100}
    var allRepos []*github.Repository

    for {
        repos, resp, err := client.Repositories.List(ctx, "", opts)
        if err != nil {
            return fmt.Errorf("failed to list repos: %w", err)
        }

        allRepos = append(allRepos, repos...)

        if resp.NextPage == 0 {
            break
        }
        opts.Page = resp.NextPage
    }

    // Sauvegarder dans PostgreSQL
    for _, ghRepo := range allRepos {
        repo := &models.GitHubRepository{
            Username:    username,
            RepoName:    ghRepo.GetName(),
            FullName:    ghRepo.GetFullName(),
            Description: ghRepo.GetDescription(),
            URL:         ghRepo.GetHTMLURL(),
            Stars:       int32(ghRepo.GetStargazersCount()),
            Language:    ghRepo.GetLanguage(),
            Topics:      pq.Array(ghRepo.Topics),
            IsPrivate:   ghRepo.GetPrivate(),
            UpdatedAt:   ghRepo.GetUpdatedAt().Time,
        }

        // Upsert (créer ou mettre à jour)
        s.db.Save(repo)
    }

    // Mettre en cache le timestamp du sync
    s.redis.Set(ctx, "github:sync:"+username, time.Now().Unix(), 24*time.Hour)

    return nil
}

// GetPublicRepositories retourne les repos publics à afficher
func (s *GitHubService) GetPublicRepositories(ctx context.Context, username string) ([]models.GitHubRepository, error) {
    // Essayer le cache d'abord
    cached, err := s.redis.Get(ctx, "github:repos:"+username).Result()
    if err == nil {
        var repos []models.GitHubRepository
        json.Unmarshal([]byte(cached), &repos)
        return repos, nil
    }

    // Requête DB
    var repos []models.GitHubRepository
    s.db.Where("username = ? AND is_private = false", username).
        Order("stars DESC").
        Find(&repos)

    // Mettre en cache
    data, _ := json.Marshal(repos)
    s.redis.Set(ctx, "github:repos:"+username, string(data), 24*time.Hour)

    return repos, nil
}

// ScheduleSyncJob crée un cron job pour sync automatique
func (s *GitHubService) ScheduleSyncJob(username, token string) {
    scheduler := cron.New()

    // Sync quotidien à 2h du matin
    scheduler.AddFunc("0 2 * * *", func() {
        s.SyncRepositories(context.Background(), username, token)
    })

    scheduler.Start()
}
```

**Modèle de Données:**

```go
// backend/internal/models/github.go
package models

import (
    "github.com/lib/pq"
    "gorm.io/datatypes"
)

type GitHubRepository struct {
    ID        uint      `gorm:"primaryKey"`
    Username  string    `gorm:"index"`
    RepoName  string
    FullName  string    `gorm:"uniqueIndex:,composite:username"`
    Description string
    URL       string
    Stars     int32
    Language  string
    Topics    pq.StringArray `gorm:"type:text[]"`
    IsPrivate bool
    UpdatedAt time.Time
    CreatedAt time.Time
}
```

#### Étape 1.3: Endpoint API

**Code Backend:**

```go
// backend/internal/api/github.go
package api

type GitHubAPIHandler struct {
    service *services.GitHubService
}

// GET /api/github/repos - Lister les repos publics
func (h *GitHubAPIHandler) GetRepositories(c *fiber.Ctx) error {
    username := c.Query("username", "") // Optionnel, défaut = profil connecté

    repos, err := h.service.GetPublicRepositories(c.Context(), username)
    if err != nil {
        return c.Status(500).JSON(fiber.Map{"error": "failed_to_fetch_repos"})
    }

    return c.JSON(fiber.Map{
        "repositories": repos,
        "count": len(repos),
    })
}

// POST /api/github/sync - Déclencher sync manuel
func (h *GitHubAPIHandler) SyncRepositories(c *fiber.Ctx) error {
    var req struct {
        Token string `json:"token"`
    }

    if err := c.BodyParser(&req); err != nil {
        return c.Status(400).JSON(fiber.Map{"error": "invalid_request"})
    }

    // Récupérer username depuis session/JWT
    username := c.Locals("github_username").(string)

    // Déclencher sync en arrière-plan
    go h.service.SyncRepositories(c.Context(), username, req.Token)

    return c.JSON(fiber.Map{"status": "sync_started"})
}
```

---

### Feature 2: Timeline Interactive

#### Étape 2.1: Endpoint Backend pour Données Chronologiques

**Description:** Exposer les expériences et projets avec données chronologiques complètes.

**Code Backend (Go):**

```go
// backend/internal/api/timeline.go
package api

type TimelineEvent struct {
    ID        string    `json:"id"`
    Type      string    `json:"type"` // "experience" ou "project"
    Title     string    `json:"title"`
    Subtitle  string    `json:"subtitle"`
    Content   string    `json:"content"`
    StartDate time.Time `json:"start_date"`
    EndDate   *time.Time `json:"end_date,omitempty"`
    Tags      []string  `json:"tags"`
    Category  string    `json:"category"` // "backend", "frontend", "devops", etc.
    Image     string    `json:"image,omitempty"`
}

// GET /api/timeline - Retourner tous les événements chronologiques
func (h *TimelineHandler) GetTimeline(c *fiber.Ctx) error {
    category := c.Query("category", "") // Optionnel: filtrer par catégorie

    // Récupérer expériences
    var experiences []models.Experience
    query := h.db
    if category != "" {
        query = query.Where("category = ?", category)
    }
    query.Order("start_date DESC").Find(&experiences)

    // Récupérer projets
    var projects []models.Project
    query = h.db
    if category != "" {
        query = query.Where("category = ?", category)
    }
    query.Order("created_at DESC").Find(&projects)

    // Combiner et trier par date
    var events []TimelineEvent

    for _, exp := range experiences {
        events = append(events, TimelineEvent{
            ID:       fmt.Sprintf("exp_%d", exp.ID),
            Type:     "experience",
            Title:    exp.Title,
            Subtitle: exp.Company,
            Content:  exp.Description,
            StartDate: exp.StartDate,
            EndDate:  exp.EndDate,
            Tags:     exp.Technologies,
            Category: exp.Category,
        })
    }

    for _, proj := range projects {
        endDate := proj.CompletedAt
        events = append(events, TimelineEvent{
            ID:       fmt.Sprintf("proj_%d", proj.ID),
            Type:     "project",
            Title:    proj.Title,
            Subtitle: proj.Description,
            StartDate: proj.StartedAt,
            EndDate:  &endDate,
            Tags:     proj.Technologies,
            Category: proj.Category,
        })
    }

    // Trier par date décroissante
    sort.Slice(events, func(i, j int) bool {
        return events[i].StartDate.After(events[j].StartDate)
    })

    return c.JSON(fiber.Map{
        "events": events,
        "total": len(events),
    })
}

// GET /api/timeline/categories - Lister catégories disponibles
func (h *TimelineHandler) GetCategories(c *fiber.Ctx) error {
    var categories []string

    h.db.Model(&models.Experience{}).
        Distinct("category").
        Pluck("category", &categories)

    return c.JSON(fiber.Map{"categories": categories})
}
```

#### Étape 2.2: Composant Frontend Timeline Interactive

**Description:** Afficher timeline chronologique avec animations.

**Code Frontend (React/Next.js):**

```tsx
// frontend/components/timeline/InteractiveTimeline.tsx
import React, { useState } from 'react';
import { motion } from 'framer-motion';
import { useInView } from 'react-intersection-observer';

interface TimelineEvent {
  id: string;
  type: 'experience' | 'project';
  title: string;
  subtitle: string;
  content: string;
  startDate: string;
  endDate?: string;
  tags: string[];
  category: string;
  image?: string;
}

interface TimelineProps {
  events: TimelineEvent[];
  categories: string[];
}

export function InteractiveTimeline({ events, categories }: TimelineProps) {
  const [selectedCategory, setSelectedCategory] = useState<string>('');
  const [selectedEvent, setSelectedEvent] = useState<TimelineEvent | null>(null);

  const filteredEvents = selectedCategory
    ? events.filter(e => e.category === selectedCategory)
    : events;

  return (
    <div className="timeline-container">
      {/* Filtres par catégorie */}
      <div className="category-filters flex gap-2 mb-8 overflow-x-auto pb-2">
        <button
          onClick={() => setSelectedCategory('')}
          className={`px-4 py-2 rounded-full whitespace-nowrap ${
            selectedCategory === ''
              ? 'bg-blue-500 text-white'
              : 'bg-gray-200'
          }`}
        >
          Tous
        </button>
        {categories.map(cat => (
          <button
            key={cat}
            onClick={() => setSelectedCategory(cat)}
            className={`px-4 py-2 rounded-full whitespace-nowrap ${
              selectedCategory === cat
                ? 'bg-blue-500 text-white'
                : 'bg-gray-200'
            }`}
          >
            {cat}
          </button>
        ))}
      </div>

      {/* Timeline verticale */}
      <div className="relative">
        {/* Ligne verticale */}
        <div className="absolute left-1/2 transform -translate-x-1/2 w-1 h-full bg-gradient-to-b from-blue-500 to-purple-500" />

        {/* Événements */}
        <div className="space-y-12">
          {filteredEvents.map((event, index) => (
            <TimelineItem
              key={event.id}
              event={event}
              index={index}
              isSelected={selectedEvent?.id === event.id}
              onSelect={() => setSelectedEvent(event)}
            />
          ))}
        </div>
      </div>

      {/* Modal détails */}
      {selectedEvent && (
        <TimelineModal
          event={selectedEvent}
          onClose={() => setSelectedEvent(null)}
        />
      )}
    </div>
  );
}

interface TimelineItemProps {
  event: TimelineEvent;
  index: number;
  isSelected: boolean;
  onSelect: () => void;
}

function TimelineItem({ event, index, isSelected, onSelect }: TimelineItemProps) {
  const { ref, inView } = useInView({
    threshold: 0.5,
    triggerOnce: true,
  });

  const isEven = index % 2 === 0;

  return (
    <motion.div
      ref={ref}
      initial={{ opacity: 0, y: 50 }}
      animate={inView ? { opacity: 1, y: 0 } : { opacity: 0, y: 50 }}
      transition={{ duration: 0.6, delay: 0.1 * index }}
      className={`relative flex ${isEven ? 'flex-row' : 'flex-row-reverse'}`}
    >
      {/* Contenu */}
      <div className={`w-1/2 ${isEven ? 'pr-12' : 'pl-12'}`}>
        <motion.div
          onClick={onSelect}
          whileHover={{ scale: 1.02 }}
          className={`p-6 rounded-lg cursor-pointer transition-all ${
            event.type === 'experience'
              ? 'bg-blue-50 border-l-4 border-blue-500'
              : 'bg-purple-50 border-l-4 border-purple-500'
          } ${isSelected ? 'ring-2 ring-blue-500' : ''}`}
        >
          <div className="flex items-start justify-between mb-2">
            <h3 className="text-lg font-bold text-gray-900">
              {event.title}
            </h3>
            <span className="text-xs font-semibold text-gray-500 uppercase">
              {event.type === 'experience' ? '💼' : '🚀'}
            </span>
          </div>

          <p className="text-sm text-gray-600 mb-2">{event.subtitle}</p>

          <div className="flex items-center gap-2 text-xs text-gray-500 mb-3">
            📅 {new Date(event.startDate).toLocaleDateString('fr-FR')}
            {event.endDate && ` → ${new Date(event.endDate).toLocaleDateString('fr-FR')}`}
          </div>

          <div className="flex flex-wrap gap-1">
            {event.tags.map(tag => (
              <span
                key={tag}
                className="inline-block px-2 py-1 text-xs bg-gray-200 text-gray-700 rounded"
              >
                {tag}
              </span>
            ))}
          </div>
        </motion.div>
      </div>

      {/* Point central */}
      <motion.div
        animate={isSelected ? { scale: 1.2 } : { scale: 1 }}
        className="absolute left-1/2 transform -translate-x-1/2 w-4 h-4 bg-white border-4 border-blue-500 rounded-full"
      />
    </motion.div>
  );
}

interface TimelineModalProps {
  event: TimelineEvent;
  onClose: () => void;
}

function TimelineModal({ event, onClose }: TimelineModalProps) {
  return (
    <motion.div
      initial={{ opacity: 0 }}
      animate={{ opacity: 1 }}
      exit={{ opacity: 0 }}
      onClick={onClose}
      className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50"
    >
      <motion.div
        initial={{ scale: 0.9 }}
        animate={{ scale: 1 }}
        onClick={e => e.stopPropagation()}
        className="bg-white rounded-lg p-8 max-w-2xl w-full mx-4"
      >
        <button
          onClick={onClose}
          className="absolute top-4 right-4 text-gray-500 hover:text-gray-700"
        >
          ✕
        </button>

        <h2 className="text-2xl font-bold mb-2">{event.title}</h2>
        <p className="text-gray-600 mb-4">{event.subtitle}</p>

        <div className="bg-gray-50 p-4 rounded-lg mb-4">
          <p className="text-gray-700 whitespace-pre-wrap">{event.content}</p>
        </div>

        <div className="flex flex-wrap gap-2">
          {event.tags.map(tag => (
            <span
              key={tag}
              className="px-3 py-1 bg-blue-500 text-white text-sm rounded-full"
            >
              {tag}
            </span>
          ))}
        </div>
      </motion.div>
    </motion.div>
  );
}
```

#### Étape 2.3: Page Timeline

**Code Frontend:**

```tsx
// frontend/app/timeline/page.tsx
import { InteractiveTimeline } from '@/components/timeline/InteractiveTimeline';

async function getTimelineData() {
  const res = await fetch(`${process.env.NEXT_PUBLIC_API_URL}/api/timeline`);
  const data = await res.json();
  return data;
}

async function getCategories() {
  const res = await fetch(`${process.env.NEXT_PUBLIC_API_URL}/api/timeline/categories`);
  const data = await res.json();
  return data.categories || [];
}

export const metadata = {
  title: 'Timeline - Mon Parcours Professionnel',
  description: 'Découvrez mon évolution professionnelle au fil du temps',
};

export default async function TimelinePage() {
  const timelineData = await getTimelineData();
  const categories = await getCategories();

  return (
    <main className="container mx-auto px-4 py-12">
      <h1 className="text-4xl font-bold mb-4">Mon Parcours Professionnel</h1>
      <p className="text-gray-600 mb-8">
        Explorez mon évolution professionnelle au fil du temps. Cliquez sur un événement pour plus de détails.
      </p>

      <InteractiveTimeline
        events={timelineData.events}
        categories={categories}
      />
    </main>
  );
}
```

---

### Feature 3: Détection Profils Avancée

#### Étape 3.1: Service de Détection de Profil

**Description:** Identifier automatiquement les profils cibles (recruteurs, tech leads, etc.).

**Code Backend (Go):**

```go
// backend/internal/services/profile_detection.go
package services

import (
    "context"
    "github.com/ua-parser/uaparser-go/ua_parser"
    "geoip2.io"
)

type ProfileDetectionService struct {
    db      *gorm.DB
    redis   *redis.Client
    geoip   *geoip2.Reader
    parser  *ua_parser.Parser
}

type DetectedProfile struct {
    ProfileType   string   `json:"profile_type"` // "recruiter", "tech_lead", "cto", "normal"
    Confidence    float32  `json:"confidence"`   // 0.0 - 1.0
    Company       string   `json:"company,omitempty"`
    Indicators    []string `json:"indicators"`
    DeviceInfo    DeviceInfo `json:"device_info"`
}

type DeviceInfo struct {
    Browser    string `json:"browser"`
    OS         string `json:"os"`
    DeviceType string `json:"device_type"` // "mobile", "tablet", "desktop"
    IsMobile   bool   `json:"is_mobile"`
}

// DetectProfile analyse l'utilisateur et retourne son profil probable
func (s *ProfileDetectionService) DetectProfile(ctx context.Context,
    ip, userAgent, referrer string) (*DetectedProfile, error) {

    profile := &DetectedProfile{
        ProfileType: "normal",
        Confidence:  0.0,
        Indicators:  []string{},
    }

    // 1. Parser User-Agent
    ua := s.parser.Parse(userAgent)
    profile.DeviceInfo = DeviceInfo{
        Browser:    ua.Browser.Family,
        OS:         ua.OS.Family,
        DeviceType: ua.Device.Family,
        IsMobile:   ua.Device.IsSpider || ua.Device.Family == "Spider",
    }

    // 2. Détecter si c'est un bot recruteur
    if isRecruiterBot(userAgent) {
        profile.ProfileType = "recruiter"
        profile.Confidence = 0.95
        profile.Indicators = append(profile.Indicators, "bot_recruiter_user_agent")
        return profile, nil
    }

    // 3. Lookup IP pour company
    if company, err := s.lookupCompanyByIP(ip); err == nil && company != "" {
        profile.Company = company
        profile.Indicators = append(profile.Indicators, "ip_company_lookup")

        // Vérifier si c'est une entreprise tech connue (recrutement)
        if isTechCompany(company) {
            profile.ProfileType = "tech_recruiter"
            profile.Confidence += 0.3
        }
    }

    // 4. Vérifier Clearbit API pour enrichissement
    if clearbitData, err := s.lookupClearbit(ip); err == nil {
        profile.Company = clearbitData.Company
        profile.Indicators = append(profile.Indicators, "clearbit_enrichment")

        // Analyser le rôle probable
        if clearbitData.Role != "" {
            profile.ProfileType = classifyRole(clearbitData.Role)
            profile.Confidence += 0.4
        }
    }

    // 5. Analyser le referrer LinkedIn
    if isLinkedInReferrer(referrer) {
        profile.Indicators = append(profile.Indicators, "linkedin_referrer")
        profile.Confidence += 0.2
    }

    // 6. Vérifier patterns de navigation (si historique disponible)
    // Implémenté dans une phase suivante avec analyse comportementale

    return profile, nil
}

// isRecruiterBot détecte les bots recruteurs connus
func isRecruiterBot(userAgent string) bool {
    recruiterPatterns := []string{
        "LinkedInBot",
        "Recruiter",
        "HubSpot",
        "Workable",
        "Lever",
        "SmashFly",
        "PeopleClick",
        "ApplicantStack",
        "Jobvite",
    }

    for _, pattern := range recruiterPatterns {
        if strings.Contains(userAgent, pattern) {
            return true
        }
    }
    return false
}

// lookupClearbit utilise Clearbit API pour enrichissement
func (s *ProfileDetectionService) lookupClearbit(ip string) (*ClearbitEnrichment, error) {
    // Vérifier cache
    cached, err := s.redis.Get(context.Background(), "clearbit:"+ip).Result()
    if err == nil {
        var data ClearbitEnrichment
        json.Unmarshal([]byte(cached), &data)
        return &data, nil
    }

    // API Clearbit
    client := resty.New()
    resp, err := client.R().
        SetHeader("Authorization", "Bearer "+os.Getenv("CLEARBIT_API_KEY")).
        Get("https://api.clearbit.com/v1/ip/" + ip)

    if err != nil || resp.StatusCode() != 200 {
        return nil, err
    }

    var data ClearbitEnrichment
    if err := json.Unmarshal(resp.Body(), &data); err != nil {
        return nil, err
    }

    // Cacher 7 jours
    s.redis.Set(context.Background(), "clearbit:"+ip, string(resp.Body()), 7*24*time.Hour)

    return &data, nil
}

type ClearbitEnrichment struct {
    IP       string `json:"ip"`
    Company  string `json:"company"`
    Domain   string `json:"domain"`
    Role     string `json:"role"`
    Title    string `json:"title"`
    GeoIP    struct {
        City    string `json:"city"`
        Country string `json:"country"`
    } `json:"geoIP"`
}

// classifyRole retourne le type de profil selon le rôle
func classifyRole(role string) string {
    roles := map[string]string{
        "CTO":        "cto",
        "CEO":        "ceo",
        "Founder":    "founder",
        "Recruiter":  "recruiter",
        "HR":         "hr",
        "Manager":    "tech_lead",
        "Lead":       "tech_lead",
        "Director":   "director",
    }

    for pattern, profileType := range roles {
        if strings.Contains(strings.ToUpper(role), pattern) {
            return profileType
        }
    }
    return "normal"
}

// isTechCompany check si l'entreprise est une société tech
func isTechCompany(company string) bool {
    techCompanies := []string{
        "Google", "Microsoft", "Apple", "Amazon", "Facebook",
        "Netflix", "Spotify", "Uber", "Airbnb", "Slack",
        "GitLab", "GitHub", "Datadog", "MongoDB", "Stripe",
    }

    company = strings.ToLower(company)
    for _, tc := range techCompanies {
        if strings.Contains(company, strings.ToLower(tc)) {
            return true
        }
    }
    return false
}

// isLinkedInReferrer vérifie si le referrer est LinkedIn
func isLinkedInReferrer(referrer string) bool {
    return strings.Contains(strings.ToLower(referrer), "linkedin")
}
```

#### Étape 3.2: Intégration dans le Middleware

**Code Backend:**

```go
// backend/internal/middleware/profile_detection.go
package middleware

import (
    "github.com/gofiber/fiber/v2"
)

// ProfileDetectionMiddleware détecte et enrichit le profil visiteur
func ProfileDetectionMiddleware(detectionService *services.ProfileDetectionService) fiber.Handler {
    return func(c *fiber.Ctx) error {
        // Récupérer infos requête
        ip := c.IP()
        userAgent := c.Get("User-Agent")
        referrer := c.Get("Referer")

        // Détecter profil
        profile, err := detectionService.DetectProfile(c.Context(), ip, userAgent, referrer)
        if err == nil {
            // Stocker dans context
            c.Locals("detected_profile", profile)

            // Enregistrer en base si recruteur
            if profile.ProfileType != "normal" {
                // Async: enregistrer le visiteur cible
                go recordTargetVisitor(detectionService.db, profile, ip)
            }
        }

        return c.Next()
    }
}

func recordTargetVisitor(db *gorm.DB, profile *services.DetectedProfile, ip string) {
    visitor := &models.TargetVisitor{
        IP:            ip,
        ProfileType:   profile.ProfileType,
        Company:       profile.Company,
        Confidence:    profile.Confidence,
        Indicators:    pq.Array(profile.Indicators),
        DetectedAt:    time.Now(),
    }
    db.Create(visitor)
}
```

#### Étape 3.3: Dashboard de Détection

**Code Backend:**

```go
// Endpoint pour voir les profils détectés
// GET /api/analytics/target-visitors (admin only)
func (h *AnalyticsHandler) GetTargetVisitors(c *fiber.Ctx) error {
    var visitors []models.TargetVisitor
    h.db.Order("detected_at DESC").Limit(100).Find(&visitors)

    return c.JSON(fiber.Map{
        "visitors": visitors,
        "total": len(visitors),
    })
}
```

---

### Feature 4: Effets 3D (Optionnel)

#### Étape 4.1: Configuration Three.js

**Code Frontend:**

```tsx
// frontend/components/3d/Avatar3D.tsx
import React, { useRef, useEffect } from 'react';
import { Canvas, useFrame } from '@react-three/fiber';
import { OrbitControls, PerspectiveCamera } from '@react-three/drei';
import * as THREE from 'three';

// Composant Avatar 3D simple (cube interactif)
function AvatarMesh() {
  const meshRef = useRef<THREE.Mesh>(null);

  useFrame(() => {
    if (meshRef.current) {
      meshRef.current.rotation.x += 0.005;
      meshRef.current.rotation.y += 0.01;
    }
  });

  return (
    <mesh ref={meshRef}>
      <boxGeometry args={[2, 2, 2]} />
      <meshStandardMaterial
        color="#3b82f6"
        metalness={0.7}
        roughness={0.3}
      />
    </mesh>
  );
}

// Composant Scene
function Avatar3DScene() {
  return (
    <Canvas>
      <PerspectiveCamera makeDefault position={[0, 0, 5]} />
      <OrbitControls />

      <ambientLight intensity={0.5} />
      <pointLight position={[10, 10, 10]} />

      <AvatarMesh />
    </Canvas>
  );
}

interface Avatar3DProps {
  height?: string;
  width?: string;
}

export function Avatar3D({ height = '400px', width = '100%' }: Avatar3DProps) {
  return (
    <div style={{ height, width, position: 'relative' }}>
      <Canvas>
        <PerspectiveCamera makeDefault position={[0, 0, 5]} />
        <OrbitControls />

        <ambientLight intensity={0.5} />
        <pointLight position={[10, 10, 10]} />
        <pointLight position={[-10, -10, 10]} color="purple" intensity={0.5} />

        <AvatarMesh />
      </Canvas>
    </div>
  );
}
```

#### Étape 4.2: Visualisation 3D des Compétences

**Code Frontend:**

```tsx
// frontend/components/3d/SkillsVisualization3D.tsx
import React, { useRef } from 'react';
import { Canvas, useFrame } from '@react-three/fiber';
import * as THREE from 'three';

interface Skill3D {
  name: string;
  level: number; // 0-1
  color: string;
}

function SkillNode({ name, level, color, position }: any) {
  const meshRef = useRef<THREE.Mesh>(null);

  useFrame(() => {
    if (meshRef.current) {
      meshRef.current.rotation.x += 0.001;
      meshRef.current.rotation.y += 0.002;
    }
  });

  return (
    <mesh ref={meshRef} position={position}>
      <sphereGeometry args={[level * 0.5 + 0.3, 32, 32]} />
      <meshStandardMaterial color={color} emissive={color} emissiveIntensity={0.2} />
    </mesh>
  );
}

interface SkillsVisualization3DProps {
  skills: Skill3D[];
}

export function SkillsVisualization3D({ skills }: SkillsVisualization3DProps) {
  // Positionner les skills en 3D space
  const positions = skills.map((_, i) => {
    const theta = (i / skills.length) * Math.PI * 2;
    const phi = Math.acos(2 * (i / skills.length) - 1);
    return [
      3 * Math.sin(phi) * Math.cos(theta),
      3 * Math.sin(phi) * Math.sin(theta),
      3 * Math.cos(phi),
    ];
  });

  return (
    <Canvas camera={{ position: [0, 0, 8] }}>
      <ambientLight intensity={0.6} />
      <pointLight position={[10, 10, 10]} intensity={1} />

      {skills.map((skill, i) => (
        <SkillNode
          key={skill.name}
          name={skill.name}
          level={skill.level}
          color={skill.color}
          position={positions[i]}
        />
      ))}
    </Canvas>
  );
}
```

---

### Feature 5: Multi-langue (Optionnel)

#### Étape 5.1: Configuration i18n

**Code Frontend:**

```tsx
// frontend/lib/i18n.ts
import { getRequestConfig } from 'next-intl/server';

export default getRequestConfig(async ({ locale }) => ({
  messages: (await import(`../messages/${locale}.json`)).default,
}));

// frontend/middleware.ts
import createMiddleware from 'next-intl/middleware';

export default createMiddleware({
  locales: ['en', 'fr'],
  defaultLocale: 'fr',
});

export const config = {
  matcher: ['/((?!api|_next|.*\\..*).*)'],
};
```

#### Étape 5.2: Fichiers de Traduction

```json
// frontend/messages/fr.json
{
  "nav": {
    "home": "Accueil",
    "cv": "CV",
    "letters": "Lettres",
    "timeline": "Timeline",
    "analytics": "Analytics"
  },
  "timeline": {
    "title": "Mon Parcours Professionnel",
    "description": "Explorez mon évolution professionnelle au fil du temps"
  }
}

// frontend/messages/en.json
{
  "nav": {
    "home": "Home",
    "cv": "CV",
    "letters": "Letters",
    "timeline": "Timeline",
    "analytics": "Analytics"
  },
  "timeline": {
    "title": "My Professional Journey",
    "description": "Explore my professional evolution over time"
  }
}
```

#### Étape 5.3: Service de Traduction IA

**Code Backend:**

```go
// backend/internal/services/translation.go
package services

type TranslationService struct {
    claudeClient *anthropic.Client
}

func (s *TranslationService) TranslateContent(ctx context.Context,
    content, fromLang, toLang string) (string, error) {

    prompt := fmt.Sprintf(`Translate the following %s text to %s.
Keep the same tone and format. Return only the translated text.

Original text:
%s`, fromLang, toLang, content)

    resp, err := s.claudeClient.Messages.New(ctx, &anthropic.MessageNewParams{
        Model: anthropic.ModelClaude3_5Sonnet,
        Messages: []anthropic.MessageParam{
            anthropic.NewUserMessage(prompt),
        },
    })

    if err != nil {
        return "", err
    }

    return resp.Content[0].Text, nil
}
```

---

## 🧪 Tests

### Tests Unitaires

```go
// backend/internal/services/profile_detection_test.go
package services

import (
    "testing"
    "github.com/stretchr/testify/assert"
)

func TestDetectRecruiterBot(t *testing.T) {
    tests := []struct {
        userAgent string
        expected  bool
    }{
        {"LinkedInBot/2.0", true},
        {"Mozilla/5.0 (Windows)", false},
        {"HubSpot/1.0", true},
    }

    for _, test := range tests {
        result := isRecruiterBot(test.userAgent)
        assert.Equal(t, test.expected, result)
    }
}
```

### Tests Integration

```go
// backend/internal/api/github_test.go
func TestGitHubSync(t *testing.T) {
    // Setup
    db := setupTestDB()
    service := NewGitHubService(db, nil)

    // Exécuter sync
    err := service.SyncRepositories(context.Background(), "testuser", mockToken)

    // Assertions
    assert.NoError(t, err)
    var repos []GitHubRepository
    db.Find(&repos)
    assert.Greater(t, len(repos), 0)
}
```

### Commandes de Test

```bash
# Tests unitaires
go test -v ./... -run TestGithub

# Tests avec couverture
go test -v ./... -coverprofile=coverage.out

# Tests Frontend
npm run test:unit

# Tests E2E
npm run test:e2e
```

---

## ⚠️ Points d'Attention

- ⚠️ **API Rate Limiting:** GitHub API a des limites (60/h public, 5000/h authentifié). Implémenter circuit breaker
- ⚠️ **Clearbit Cost:** API Clearbit est payante. Vérifier les coûts avant déploiement
- ⚠️ **3D Performance:** Three.js peut être lourd sur mobile. Utiliser feature flags pour désactiver sur petits écrans
- ⚠️ **Traductions:** IA traductions peuvent être imprécises. Réviser manuellement contenu critique
- 💡 **Cache Agressif:** Utiliser TTL élevées pour données externes (GitHub, Clearbit)
- 💡 **Feature Flags:** Implémenter feature flags pour chaque feature optionnelle (3D, multi-langue)
- 💡 **Fallbacks:** Si une API externe échoue, afficher gracefully et utiliser cache

---

## 📚 Ressources

- [GitHub API Documentation](https://docs.github.com/en/rest)
- [Clearbit API](https://clearbit.com/resources/api)
- [Next-intl Documentation](https://next-intl-docs.vercel.app/)
- [Three.js Documentation](https://threejs.org/docs/)
- [React Three Fiber](https://docs.pmnd.rs/react-three-fiber/getting-started/introduction)
- [Framer Motion Timeline](https://www.framer.com/motion/)
- [User-Agent Parser](https://github.com/ua-parser/uaparser-go)

---

## ✅ Checklist de Complétion

### Feature 1: Import GitHub
- [ ] OAuth GitHub implémenté
- [ ] Synchronisation API repos
- [ ] Endpoint `/api/github/repos`
- [ ] Cron job sync automatique
- [ ] Tests unitaires
- [ ] Tests integration

### Feature 2: Timeline Interactive
- [ ] Endpoint backend `/api/timeline`
- [ ] Composant React Timeline
- [ ] Animations Framer Motion
- [ ] Filtrage par catégorie
- [ ] Modal détails événement
- [ ] Tests E2E

### Feature 3: Détection Profils
- [ ] Service détection profil
- [ ] Intégration Clearbit API
- [ ] Middleware profile detection
- [ ] Dashboard cible visiteurs
- [ ] Tests unitaires
- [ ] Tests security

### Feature 4: Effets 3D
- [ ] Configuration Three.js
- [ ] Avatar 3D component
- [ ] Skills visualization 3D
- [ ] Feature flag 3D
- [ ] Optimisation mobile
- [ ] Tests performance

### Feature 5: Multi-langue
- [ ] Configuration i18n
- [ ] Fichiers traductions (FR/EN)
- [ ] Service traduction IA
- [ ] Routing i18n
- [ ] SEO multi-langue
- [ ] Tests traductions

---

## 📊 Tracking Progress

| Feature | Status | Priorité | Effort | Notes |
|---------|--------|----------|--------|-------|
| GitHub Import | 🔲 | Moyenne | 3j | Dépend API GitHub |
| Timeline | 🔲 | Haute | 2j | Core feature |
| Profiling | 🔲 | Moyenne | 2j | Nécessite clé Clearbit |
| 3D Effects | 🔲 | Basse | 2j | Optionnel |
| Multi-langue | 🔲 | Basse | 1.5j | Optionnel |

---

**Dernière mise à jour:** 2025-12-08
**Auteur:** Alexi
