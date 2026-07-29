# 18. PERFORMANCE

## 📋 Métadonnées

- **Phase:** 6
- **Priorité:** 🟢 MOYENNE
- **Complexité:** ⭐⭐⭐ (3/5)
- **Prérequis:** 14 (Infrastructure Production), tous modules fonctionnels
- **Temps estimé:** 2-3 jours
- **Status:** 🔲 À faire

---

## 🎯 Objectif

Optimiser les performances du système maicivy (backend, frontend, base de données, cache) pour atteindre des métriques cibles de latence, throughput et utilisabilité. Focus sur les optimisations standards : caching Redis, indexes base de données, Next.js Image optimization, lazy loading, et profiling.

---

## 🏗️ Architecture

### Vue d'Ensemble

```
┌─────────────────────────────────────────────────────────────┐
│                    CLIENT BROWSER                           │
│  ┌────────────────────────────────────────────────────────┐ │
│  │  Frontend Performance (Next.js Image, lazy loading)    │ │
│  │  - Code splitting                                      │ │
│  │  - Dynamic imports                                     │ │
│  │  - Prefetching (next/link)                            │ │
│  │  - Image optimization                                 │ │
│  └────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────┘
                            │
                     HTTP/Compression
                            ▼
┌─────────────────────────────────────────────────────────────┐
│                    NGINX (Reverse Proxy)                    │
│  - Gzip/Brotli compression                                │
│  - Static files caching (Cache-Control headers)           │
│  - Keep-Alive connections                                 │
│  - Request/Response header optimization                   │
└─────────────────────────────────────────────────────────────┘
                            │
                   API Requests + Caching
                            ▼
┌─────────────────────────────────────────────────────────────┐
│               BACKEND (Go + Fiber)                          │
│  ┌────────────────────────────────────────────────────────┐ │
│  │ API Optimization                                       │ │
│  │ - Pagination (LIMIT/OFFSET)                           │ │
│  │ - Field selection (sparse fields)                      │ │
│  │ - Connection pooling                                  │ │
│  └────────────────────────────────────────────────────────┘ │
│                            │                                 │
│          ┌─────────────────┼─────────────────┐               │
│          ▼                 ▼                 ▼               │
│  ┌──────────────┐ ┌──────────────┐ ┌──────────────┐         │
│  │   REDIS      │ │  PostgreSQL  │ │  External    │         │
│  │   CACHE      │ │  (Optimized) │ │  APIs        │         │
│  └──────────────┘ └──────────────┘ └──────────────┘         │
└─────────────────────────────────────────────────────────────┘
```

### Design Decisions

1. **Caching Stratégique :**
   - Redis pour données chaudes (CVs, lettres, infos entreprises)
   - HTTP cache headers pour assets statiques
   - CDN optionnel pour distribution géographique

2. **DB Optimization :**
   - Indexes sur colonnes fréquemment queryées (WHERE, JOIN)
   - Connection pooling pour réduire overhead
   - EXPLAIN ANALYZE pour identifier requêtes lentes

3. **Frontend Performance :**
   - Next.js Image pour lazy loading et formats modernes (WebP, AVIF)
   - Code splitting automatique par route
   - Dynamic imports pour components lourds
   - Prefetching intelligent (next/link)

4. **API Optimization :**
   - Pagination par défaut (éviter responses énormes)
   - Field selection (GET /api/experiences?fields=id,title,company)
   - Compression gzip/brotli au niveau Nginx
   - Keep-Alive HTTP/1.1

---

## 📦 Dépendances

### Bibliothèques Go

```bash
# Benchmarking & Profiling
go get github.com/pkg/profile                      # CPU profiling
go get github.com/google/pprof                     # Go profiling tools

# Connection Pooling & Database
go get github.com/jackc/pgx/v5                     # PostgreSQL driver (pgx)
go get github.com/jackc/pgx/v5/pgxpool             # Connection pooling

# Monitoring/Metrics (déjà installé par INFRASTRUCTURE_PRODUCTION)
go get github.com/prometheus/client_golang         # Prometheus metrics

# Compression
# Fiber inclut gzip/brotli en natif
```

### Bibliothèques NPM

```bash
# Image optimization (Next.js inclut next/image natif)
npm install sharp                                  # Image processing backend

# Performance monitoring (Frontend)
npm install web-vitals                             # Core Web Vitals metrics
npm install @sentry/nextjs                         # Error tracking + performance

# Bundle analysis (development)
npm install --save-dev @next/bundle-analyzer       # Analyze bundle size
npm install --save-dev webpack-bundle-analyzer     # Detailed bundle analysis

# Lazy loading
npm install react-lazyload                         # Component lazy loading
# ou native: Intersection Observer (intégré navigateur)
```

### Tools Externes (pour benchmarking)

```bash
# Backend load testing
# wrk : https://github.com/wg/wrk (installer séparément)
# k6 : https://k6.io/

# Frontend profiling
# Chrome DevTools (natif)
# Lighthouse (natif dans Chrome)
```

### Services Externes

- CDN (optionnel): Cloudflare, CloudFront, ou Bunny CDN
- APM (optionnel): Sentry, New Relic (pour monitoring production)

---

## 🔨 Implémentation

### Étape 1 : Caching Strategies - Redis

**Description:** Configurer cache Redis multi-niveaux pour données chaudes.

**Code (backend/internal/services/cache.go):**

```go
package services

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// CacheService encapsule la logique de caching Redis
type CacheService struct {
	client *redis.Client
}

func NewCacheService(redisClient *redis.Client) *CacheService {
	return &CacheService{
		client: redisClient,
	}
}

// TTL constants
const (
	// Données quasi-statiques
	TTL_SKILLS    = 24 * time.Hour
	TTL_PROJECTS  = 24 * time.Hour
	TTL_CV        = 1 * time.Hour        // CV adapté par thème change rarement

	// Données temps réel
	TTL_LETTERS   = 24 * time.Hour        // Lettres générées cachées 24h
	TTL_COMPANY   = 7 * 24 * time.Hour    // Infos entreprises cachées 7 jours
	TTL_ANALYTICS = 5 * time.Minute       // Stats temps réel, courte TTL
)

// GetCV récupère CV du cache ou retourne nil si absent
func (cs *CacheService) GetCV(ctx context.Context, theme string) (string, error) {
	key := fmt.Sprintf("cv:%s", theme)
	val, err := cs.client.Get(ctx, key).Result()
	if err == redis.Nil {
		return "", nil // Cache miss
	}
	return val, err
}

// SetCV stocke CV en cache avec TTL
func (cs *CacheService) SetCV(ctx context.Context, theme string, cvData interface{}) error {
	key := fmt.Sprintf("cv:%s", theme)
	cvJSON, _ := json.Marshal(cvData)
	return cs.client.Set(ctx, key, string(cvJSON), TTL_CV).Err()
}

// GetLetter récupère lettre générée du cache
func (cs *CacheService) GetLetter(ctx context.Context, companyName, letterType string) (string, error) {
	// Hash du nom entreprise pour éviter clés trop longues
	hash := fmt.Sprintf("%x", companyName) // Simplified, use SHA256 in prod
	key := fmt.Sprintf("letter:%s:%s", hash, letterType) // "motivation" ou "anti-motivation"
	val, err := cs.client.Get(ctx, key).Result()
	if err == redis.Nil {
		return "", nil
	}
	return val, err
}

// SetLetter stocke lettre générée
func (cs *CacheService) SetLetter(ctx context.Context, companyName, letterType, content string) error {
	hash := fmt.Sprintf("%x", companyName)
	key := fmt.Sprintf("letter:%s:%s", hash, letterType)
	return cs.client.Set(ctx, key, content, TTL_LETTERS).Err()
}

// GetCompanyInfo récupère infos entreprise scrapées
func (cs *CacheService) GetCompanyInfo(ctx context.Context, companyName string) (string, error) {
	hash := fmt.Sprintf("%x", companyName)
	key := fmt.Sprintf("company_info:%s", hash)
	val, err := cs.client.Get(ctx, key).Result()
	if err == redis.Nil {
		return "", nil
	}
	return val, err
}

// SetCompanyInfo stocke infos entreprise
func (cs *CacheService) SetCompanyInfo(ctx context.Context, companyName, info string) error {
	hash := fmt.Sprintf("%x", companyName)
	key := fmt.Sprintf("company_info:%s", hash)
	return cs.client.Set(ctx, key, info, TTL_COMPANY).Err()
}

// InvalidateCV invalide cache CV (après modification)
func (cs *CacheService) InvalidateCV(ctx context.Context, theme string) error {
	key := fmt.Sprintf("cv:%s", theme)
	return cs.client.Del(ctx, key).Err()
}

// InvalidateAllCVs invalide tous les CVs en cache
func (cs *CacheService) InvalidateAllCVs(ctx context.Context) error {
	keys, err := cs.client.Keys(ctx, "cv:*").Result()
	if err != nil {
		return err
	}
	if len(keys) > 0 {
		return cs.client.Del(ctx, keys...).Err()
	}
	return nil
}
```

**Configuration Redis dans docker-compose.yml:**

```yaml
services:
  redis:
    image: redis:7-alpine
    ports:
      - "6379:6379"
    volumes:
      - redis_data:/data
    command: >
      redis-server
      --maxmemory 256mb
      --maxmemory-policy allkeys-lru
      --appendonly yes
    healthcheck:
      test: ["CMD", "redis-cli", "ping"]
      interval: 10s
      timeout: 5s
      retries: 5

volumes:
  redis_data:
```

**Explications:**
- Cache hit rate = coûts API réduits (Claude/OpenAI)
- TTL stratégique : données statiques longues, temps réel courtes
- LRU policy pour éviter mémoire infinies
- Persistence (AOF) pour survie redémarrages

---

### Étape 2 : HTTP Caching Headers

**Description:** Configurer headers Cache-Control pour assets statiques et API responses.

**Code (backend/internal/middleware/cache.go):**

```go
package middleware

import (
	"github.com/gofiber/fiber/v2"
)

// CacheControlMiddleware ajoute les headers Cache-Control appropriés
func CacheControlMiddleware(c *fiber.Ctx) error {
	path := c.Path()

	// Assets statiques : caching long terme (1 an)
	// Next.js hash les assets, donc safe avec expires longs
	if isStaticAsset(path) {
		c.Set("Cache-Control", "public, max-age=31536000, immutable")
		return c.Next()
	}

	// HTML pages : pas de cache (toujours frais)
	if path == "/" || path == "/cv" || path == "/letters" || path == "/analytics" {
		c.Set("Cache-Control", "no-cache, must-revalidate")
		c.Set("Pragma", "no-cache")
		return c.Next()
	}

	// API responses (dépend du endpoint)
	if isPublicAPI(path) {
		// /api/cv/themes - rarement change
		if path == "/api/cv/themes" {
			c.Set("Cache-Control", "public, max-age=3600") // 1 heure
			return c.Next()
		}

		// /api/analytics/realtime - temps réel
		if isRealtimeAPI(path) {
			c.Set("Cache-Control", "no-cache, must-revalidate")
			return c.Next()
		}

		// Default API : 5 minutes
		c.Set("Cache-Control", "public, max-age=300")
		return c.Next()
	}

	return c.Next()
}

func isStaticAsset(path string) bool {
	staticExts := []string{".js", ".css", ".woff2", ".png", ".jpg", ".webp", ".svg"}
	for _, ext := range staticExts {
		if len(path) > len(ext) && path[len(path)-len(ext):] == ext {
			return true
		}
	}
	return false
}

func isPublicAPI(path string) bool {
	// Vérifier si c'est une route /api/...
	return len(path) > 5 && path[:4] == "/api"
}

func isRealtimeAPI(path string) bool {
	realtimeRoutes := []string{
		"/api/analytics/realtime",
		"/api/letters/history",
		"/ws/",
	}
	for _, route := range realtimeRoutes {
		if path == route || len(path) > len(route) && path[:len(route)] == route {
			return true
		}
	}
	return false
}
```

**Utilisation dans main.go:**

```go
// Dans backend/cmd/main.go
import "yourapp/internal/middleware"

func main() {
	app := fiber.New()

	// Ajouter middleware cache en haut du stack
	app.Use(middleware.CacheControlMiddleware)

	// ... autres middlewares et routes
}
```

**Explications:**
- Assets Next.js (JS/CSS) : immutable, max-age très long
- Pages HTML : no-cache (toujours requête au serveur, mais peut utiliser 304 Not Modified)
- API statiques : cache court
- API temps réel : pas de cache (ou revalidate instantanée)

---

### Étape 3 : Database Optimization

**Description:** Indexes stratégiques, connection pooling, EXPLAIN ANALYZE.

**Code (backend/migrations/add_indexes.sql):**

```sql
-- Indexes pour optimiser requêtes fréquentes

-- Experiences table
CREATE INDEX idx_experiences_category ON experiences(category);
CREATE INDEX idx_experiences_tags ON experiences USING GIN(tags);
CREATE INDEX idx_experiences_dates ON experiences(start_date, end_date);

-- Skills table
CREATE INDEX idx_skills_category ON skills(category);
CREATE INDEX idx_skills_tags ON skills USING GIN(tags);
CREATE INDEX idx_skills_level ON skills(level);

-- Projects table
CREATE INDEX idx_projects_category ON projects(category);
CREATE INDEX idx_projects_tags ON projects USING GIN(tags);
CREATE INDEX idx_projects_featured ON projects(featured);

-- Generated letters table
CREATE INDEX idx_letters_company ON generated_letters(company_name);
CREATE INDEX idx_letters_visitor ON generated_letters(visitor_id);
CREATE INDEX idx_letters_created_at ON generated_letters(created_at);

-- Analytics events table
CREATE INDEX idx_analytics_events_visitor ON analytics_events(visitor_id);
CREATE INDEX idx_analytics_events_type ON analytics_events(event_type);
CREATE INDEX idx_analytics_events_created ON analytics_events(created_at);

-- Visitors table
CREATE INDEX idx_visitors_session ON visitors(session_id);
CREATE INDEX idx_visitors_profile ON visitors(profile_detected);
CREATE INDEX idx_visitors_last_visit ON visitors(last_visit);

-- Composite indexes pour queries complexes
CREATE INDEX idx_letters_lookup ON generated_letters(visitor_id, company_name);
CREATE INDEX idx_analytics_timerange ON analytics_events(event_type, created_at);
```

**Code (backend/internal/database/postgres.go) - Connection Pooling:**

```go
package database

import (
	"context"
	"fmt"
	"log"

	"github.com/jackc/pgx/v5/pgxpool"
)

// InitPostgres initialise le pool de connexions PostgreSQL
func InitPostgres(databaseURL string) (*pgxpool.Pool, error) {
	config, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, fmt.Errorf("unable to parse database URL: %w", err)
	}

	// Pool configuration
	config.MaxConns = 25                 // Max connexions simultanées
	config.MinConns = 5                  // Min connexions gardées ouvertes
	config.MaxConnLifetime = 15 * 60     // Recycle connexions après 15 min
	config.MaxConnIdleTime = 5 * 60      // Fermer connexions inactives après 5 min
	config.HealthCheckInterval = 30 * 1  // Vérifier health chaque 30s
	config.ConnectTimeout = 5 * 1        // Timeout connexion 5s

	pool, err := pgxpool.NewWithConfig(context.Background(), config)
	if err != nil {
		return nil, fmt.Errorf("unable to create connection pool: %w", err)
	}

	// Vérifier connexion
	ctx, cancel := context.WithTimeout(context.Background(), 5*1)
	defer cancel()
	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("unable to ping database: %w", err)
	}

	log.Println("PostgreSQL connection pool initialized successfully")
	return pool, nil
}
```

**Script pour identifier requêtes lentes (backend/scripts/analyze_slow_queries.sql):**

```sql
-- Identifier requêtes lentes (execution time > 100ms)
SELECT
    mean_exec_time,
    calls,
    query
FROM pg_stat_statements
WHERE mean_exec_time > 100  -- ms
ORDER BY mean_exec_time DESC
LIMIT 20;

-- Utiliser EXPLAIN ANALYZE pour une query spécifique
EXPLAIN ANALYZE
SELECT e.*, s.* FROM experiences e
JOIN skills s ON e.id = s.experience_id
WHERE e.category = 'backend'
ORDER BY e.start_date DESC;

-- Voir indexes inutilisés
SELECT schemaname, tablename, indexname, idx_scan
FROM pg_stat_user_indexes
WHERE idx_scan = 0
ORDER BY pg_relation_size(indexrelid) DESC;
```

**Explications:**
- GIN indexes pour array/JSON fields (tags)
- Composite indexes pour requêtes multi-colonnes
- Connection pooling réduit overhead création connexions (coûteux)
- EXPLAIN ANALYZE = outil indispensable pour debug

---

### Étape 4 : Frontend Performance - Next.js Image Optimization

**Description:** Utiliser next/image pour lazy loading et formats modernes.

**Code (frontend/components/cv/ProjectsGrid.tsx):**

```tsx
import Image from 'next/image'
import { FC } from 'react'

interface Project {
  id: string
  title: string
  description: string
  image?: string
  github_url: string
  stars: number
  languages: string[]
}

interface ProjectsGridProps {
  projects: Project[]
}

export const ProjectsGrid: FC<ProjectsGridProps> = ({ projects }) => {
  return (
    <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
      {projects.map(project => (
        <div
          key={project.id}
          className="bg-white dark:bg-gray-800 rounded-lg shadow hover:shadow-lg transition-shadow"
        >
          {/* Next.js Image : lazy loading + WebP/AVIF automatic */}
          {project.image && (
            <div className="relative w-full h-48">
              <Image
                src={project.image}
                alt={project.title}
                fill
                className="object-cover rounded-t-lg"
                // loading="lazy" est défaut dans Next.js 12+
                // Sizes pour responsive images
                sizes="(max-width: 768px) 100vw, (max-width: 1200px) 50vw, 33vw"
                // Priority si au-dessus du fold
                priority={projects.indexOf(project) < 3}
              />
            </div>
          )}

          <div className="p-4">
            <h3 className="font-bold text-lg mb-2">{project.title}</h3>
            <p className="text-sm text-gray-600 dark:text-gray-400 mb-3">
              {project.description}
            </p>

            <div className="flex justify-between items-center">
              <div className="flex gap-1 flex-wrap">
                {project.languages.slice(0, 3).map(lang => (
                  <span
                    key={lang}
                    className="px-2 py-1 bg-blue-100 dark:bg-blue-900 text-xs rounded"
                  >
                    {lang}
                  </span>
                ))}
              </div>
              <a
                href={project.github_url}
                target="_blank"
                rel="noopener noreferrer"
                className="text-sm font-medium text-blue-600 hover:underline"
              >
                ⭐ {project.stars}
              </a>
            </div>
          </div>
        </div>
      ))}
    </div>
  )
}
```

**Code (frontend/components/analytics/RealtimeVisitors.tsx) - avec dynamic import:**

```tsx
'use client'

import dynamic from 'next/dynamic'
import { Suspense } from 'react'

// Dynamic import pour component lourd (WebSocket + animations)
const RealtimeVisualization = dynamic(
  () => import('./RealtimeVisualization'),
  {
    loading: () => (
      <div className="bg-gray-200 dark:bg-gray-700 h-32 rounded animate-pulse" />
    ),
    ssr: false, // Éviter SSR pour WebSocket
  }
)

export function RealtimeVisitors() {
  return (
    <Suspense fallback={<div>Loading...</div>}>
      <RealtimeVisualization />
    </Suspense>
  )
}
```

**Next.js Configuration (frontend/next.config.js):**

```js
/** @type {import('next').NextConfig} */
const nextConfig = {
  // Image optimization
  images: {
    remotePatterns: [
      {
        protocol: 'https',
        hostname: 'avatars.githubusercontent.com',
        port: '',
        pathname: '/u/**',
      },
    ],
    // Formats modernes automatiques
    formats: ['image/avif', 'image/webp'],
    // Devicess breakpoints pour responsive images
    deviceSizes: [640, 750, 828, 1080, 1200, 1920, 2048, 3840],
    imageSizes: [16, 32, 48, 64, 96, 128, 256, 384],
  },

  // Code splitting
  experimental: {
    // Optimized package imports (if using large libraries)
    optimizePackageImports: [
      '@mantine/core',
      '@mantine/hooks',
    ],
  },

  // Compression
  compress: true,

  // SWR configuration for better caching
  swcMinify: true,
}

module.exports = nextConfig
```

**Explications:**
- next/image gère lazy loading automatiquement
- Formats modernes (WebP, AVIF) au client (fallback JPEG)
- Priority flag pour images au-dessus du fold
- Sizes attribute pour responsive images
- Dynamic imports réduisent bundle taille initiale

---

### Étape 5 : API Optimization - Pagination & Field Selection

**Description:** Implémenter pagination et field selection pour réduire payload.

**Code (backend/internal/api/cv.go):**

```go
package api

import (
	"github.com/gofiber/fiber/v2"
	"yourapp/internal/database"
	"strconv"
)

type ListExperiencesRequest struct {
	Page      int    `query:"page"`
	Limit     int    `query:"limit"`
	Fields    string `query:"fields"` // Sparse fields: "id,title,company"
	Category  string `query:"category"`
	Search    string `query:"search"`
}

type PaginatedResponse struct {
	Data  interface{} `json:"data"`
	Meta  Meta        `json:"meta"`
}

type Meta struct {
	Page      int `json:"page"`
	Limit     int `json:"limit"`
	Total     int `json:"total"`
	TotalPage int `json:"total_page"`
}

// GetExperiences retourne expériences avec pagination et field selection
func (h *Handler) GetExperiences(c *fiber.Ctx) error {
	var req ListExperiencesRequest

	// Parse query params
	req.Page = c.QueryInt("page", 1)
	req.Limit = c.QueryInt("limit", 20)
	req.Fields = c.Query("fields", "")
	req.Category = c.Query("category", "")
	req.Search = c.Query("search", "")

	// Validation
	if req.Page < 1 {
		req.Page = 1
	}
	if req.Limit < 1 || req.Limit > 100 {
		req.Limit = 20 // Max 100 pour éviter DOS
	}

	// Query database avec pagination
	offset := (req.Page - 1) * req.Limit

	query := h.db.WithContext(c.Context())

	// Filters
	if req.Category != "" {
		query = query.Where("category = ?", req.Category)
	}
	if req.Search != "" {
		query = query.Where("title ILIKE ? OR description ILIKE ?",
			"%"+req.Search+"%", "%"+req.Search+"%")
	}

	// Count total
	var total int64
	query.Model(&database.Experience{}).Count(&total)

	// Pagination
	query = query.Offset(offset).Limit(req.Limit)

	// Sparse fields selection
	if req.Fields != "" {
		// Whitelisted fields pour sécurité
		fields := parseFields(req.Fields, []string{
			"id", "title", "company", "description",
			"start_date", "end_date", "category", "technologies",
		})
		query = query.Select(fields)
	}

	// Fetch data
	var experiences []database.Experience
	if err := query.Order("start_date DESC").Find(&experiences).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Failed to fetch experiences"})
	}

	return c.JSON(PaginatedResponse{
		Data: experiences,
		Meta: Meta{
			Page:      req.Page,
			Limit:     req.Limit,
			Total:     int(total),
			TotalPage: int((total + int64(req.Limit) - 1) / int64(req.Limit)),
		},
	})
}

// parseFields valide et retourne liste fields autorisés
func parseFields(fieldsStr string, allowed []string) []string {
	if fieldsStr == "" {
		return []string{"*"}
	}

	allowedMap := make(map[string]bool)
	for _, f := range allowed {
		allowedMap[f] = true
	}

	fields := []string{}
	for _, f := range strings.Split(fieldsStr, ",") {
		f = strings.TrimSpace(f)
		if allowedMap[f] {
			fields = append(fields, f)
		}
	}

	if len(fields) == 0 {
		return []string{"*"}
	}
	return fields
}
```

**Usage Client:**

```bash
# Pagination
curl "http://localhost:3000/api/experiences?page=1&limit=20"

# Field selection
curl "http://localhost:3000/api/experiences?fields=id,title,company"

# Combined
curl "http://localhost:3000/api/experiences?page=1&limit=50&category=backend&fields=id,title"
```

**Explications:**
- Limite par défaut 20, max 100 pour éviter DOS
- Field selection = client ne reçoit que colonnes nécessaires
- Offset/Limit standard pour scalabilité (éviter COUNT(*) complexe)

---

### Étape 6 : Compression & Keep-Alive

**Description:** Configurer compression gzip/brotli et Keep-Alive HTTP.

**Code (backend/cmd/main.go):**

```go
package main

import (
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/compress"
	"github.com/gofiber/fiber/v2/middleware/requestid"
)

func main() {
	app := fiber.New(fiber.Config{
		// Keep-Alive configuration
		IdleTimeout: 15 * time.Second,
		ReadTimeout: 10 * time.Second,
		WriteTimeout: 10 * time.Second,

		// Body size limits
		BodyLimit: 4 * 1024 * 1024, // 4MB max request body
	})

	// Compression middleware (gzip + brotli)
	app.Use(compress.New(compress.Config{
		Level: compress.LevelDefault, // LevelBestSpeed, LevelDefault, LevelBestCompression
		// Compress types
		Next: func(c *fiber.Ctx) bool {
			// Pas compresser si trop petit (overhead)
			return c.Response().Header("Content-Length") < "100"
		},
	}))

	// Request ID pour tracing
	app.Use(requestid.New())

	// Routes
	app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "ok"})
	})

	// ... autres routes

	app.Listen(":3000")
}
```

**Nginx configuration (docker/nginx/nginx.conf):**

```nginx
http {
    # Compression
    gzip on;
    gzip_types text/plain text/css text/xml text/javascript
               application/x-javascript application/xml+rss
               application/javascript application/json;
    gzip_min_length 1000;
    gzip_level 6;

    # Brotli (si module installé)
    # brotli on;
    # brotli_types text/plain text/css text/xml text/javascript
    #              application/x-javascript application/xml+rss
    #              application/javascript application/json;

    # Keep-Alive
    keepalive_timeout 65;
    keepalive_requests 100;

    # Buffer sizes (éviter "too many redirects" warnings)
    client_body_buffer_size 1M;
    client_header_buffer_size 1k;

    upstream backend {
        server backend:3000;
        keepalive 32;
    }

    server {
        listen 80;
        server_name _;

        # Disable access log pour reduce I/O overhead
        access_log off;

        location / {
            proxy_pass http://backend;
            proxy_http_version 1.1;
            proxy_set_header Connection "";
            proxy_set_header Host $host;
            proxy_set_header X-Real-IP $remote_addr;
            proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        }
    }
}
```

**Explications:**
- Gzip reduce payload ~80% (très efficace pour JSON/text)
- Keep-Alive évite overhead TCP handshake
- Compression level 6 = sweet spot (speed vs ratio)

---

### Étape 7 : Benchmarking avec wrk et k6

**Description:** Load testing et stress testing.

**Script (backend/benchmarks/load_test.sh):**

```bash
#!/bin/bash
# Load testing avec wrk

echo "=== Benchmarking GET /api/cv ===";
wrk -t4 -c100 -d30s http://localhost:3000/api/cv
# -t4 = 4 threads
# -c100 = 100 connections
# -d30s = 30 secondes

echo "\n=== Benchmarking GET /api/experiences ===";
wrk -t4 -c100 -d30s "http://localhost:3000/api/experiences?limit=20"

echo "\n=== Benchmarking avec custom script (POST) ===";
wrk -t4 -c100 -d30s -s benchmark_post.lua http://localhost:3000/api/letters/generate
```

**Script Lua pour POST (backend/benchmarks/benchmark_post.lua):**

```lua
request = function()
  wrk.method = "POST"
  wrk.body = '{"company_name":"Google"}'
  wrk.headers["Content-Type"] = "application/json"
  return wrk.format(nil)
end
```

**K6 script pour tester responsiveness (backend/benchmarks/load_test.js):**

```javascript
import http from 'k6/http'
import { check } from 'k6'
import { Rate, Trend } from 'k6/metrics'

// Custom metrics
const errorRate = new Rate('errors')
const responseTime = new Trend('response_time')

export const options = {
  // Ramp-up: 10 users per second até 100 users over 20s
  stages: [
    { duration: '20s', target: 100 },  // Ramp-up
    { duration: '1m', target: 100 },   // Stay at 100 users for 1 minute
    { duration: '20s', target: 0 },    // Ramp-down to 0
  ],
  thresholds: {
    'http_req_duration': ['p(95)<100', 'p(99)<300'], // 95% < 100ms, 99% < 300ms
    'errors': ['rate<0.1'],                          // Error rate < 10%
  },
}

export default function() {
  const res = http.get('http://localhost:3000/api/experiences?limit=20')

  const success = check(res, {
    'status is 200': (r) => r.status === 200,
    'response time < 200ms': (r) => r.timings.duration < 200,
  })

  errorRate.add(!success)
  responseTime.add(res.timings.duration)
}
```

**Commandes de lancement:**

```bash
# Installation wrk
brew install wrk  # macOS
sudo apt-get install wrk  # Linux

# Installation k6
brew install k6   # macOS
sudo apt-get install k6  # Linux

# Run benchmarks
./backend/benchmarks/load_test.sh

# Run k6 test
k6 run backend/benchmarks/load_test.js
```

**Explications:**
- wrk = simple, HTTP/1.1 load test
- k6 = plus fonctionnalités, scripting complexe
- Targets: P95 < 100ms, P99 < 300ms (standards industry)

---

### Étape 8 : Profiling - Go pprof

**Description:** Identifier bottlenecks CPU, mémoire, goroutines.

**Code (backend/cmd/main.go):**

```go
package main

import (
	"fmt"
	"log"
	"net/http"
	_ "net/http/pprof" // Importer pour enregistrer pprof endpoints
	"runtime"
	"runtime/pprof"
	"os"
	"time"

	"github.com/gofiber/fiber/v2"
)

func main() {
	// Lancer pprof server en arrière-plan sur port 6060
	go func() {
		log.Println("Starting pprof on http://localhost:6060/debug/pprof")
		log.Println(http.ListenAndServe("localhost:6060", nil))
	}()

	app := fiber.New()

	// Health endpoint
	app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"status": "ok",
			"memory": getMemoryStats(),
			"goroutines": runtime.NumGoroutine(),
		})
	})

	// Routes
	app.Get("/api/cv", handleCV)
	app.Get("/api/experiences", handleExperiences)

	app.Listen(":3000")
}

func getMemoryStats() map[string]interface{} {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	return map[string]interface{}{
		"alloc_mb":       m.Alloc / 1024 / 1024,
		"total_alloc_mb": m.TotalAlloc / 1024 / 1024,
		"sys_mb":         m.Sys / 1024 / 1024,
		"gc_runs":        m.NumGC,
	}
}

// Endpoints dummy pour testing
func handleCV(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{"data": "cv"})
}

func handleExperiences(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{"data": "experiences"})
}
```

**Script pour profiling (backend/scripts/profile.sh):**

```bash
#!/bin/bash

# CPU profiling (30 secondes)
echo "Starting CPU profile for 30 seconds..."
go tool pprof -http=:8080 http://localhost:6060/debug/pprof/profile?seconds=30

# Memory profiling (heap)
echo "Memory profile..."
go tool pprof -http=:8081 http://localhost:6060/debug/pprof/heap

# Goroutine analysis
echo "Goroutine profile..."
go tool pprof -http=:8082 http://localhost:6060/debug/pprof/goroutine

# Mutation count (GC allocations)
echo "Allocation profile..."
go tool pprof -http=:8083 http://localhost:6060/debug/pprof/allocs
```

**Commands:**

```bash
# Lancer backend
go run backend/cmd/main.go

# Dans autre terminal, profiler
bash backend/scripts/profile.sh

# Ou commandes manuelles
go tool pprof http://localhost:6060/debug/pprof/heap
> top10        # Top 10 memory allocations
> list funcName # Voir code function
> quit

# Obtenir graphique flamegraph
go tool pprof -http=:8080 http://localhost:6060/debug/pprof/profile?seconds=30
```

**Explications:**
- pprof = outil built-in Go pour profiling
- CPU profile = identifier functions lentes
- Memory profile = memory leaks
- Goroutine profile = identifier goroutine leaks

---

### Étape 9 : Frontend Profiling - Chrome Lighthouse

**Description:** Audit performance frontend avec Lighthouse (intégré Chrome).

**Instructions:**

1. Ouvrir DevTools (F12 dans Chrome)
2. Aller à l'onglet "Lighthouse"
3. Cliquer "Analyse page load"
4. Voir scores :
   - Performance (target: > 90)
   - Accessibility (target: > 90)
   - Best Practices (target: > 90)
   - SEO (target: > 90)

**Metrics clés (Core Web Vitals):**

```
LCP (Largest Contentful Paint) : < 2.5s
FID (First Input Delay) : < 100ms
CLS (Cumulative Layout Shift) : < 0.1
```

**Code pour tracker Web Vitals (frontend/lib/vitals.ts):**

```typescript
import { getCLS, getFID, getFCP, getLCP, getTTFB } from 'web-vitals'

export function reportWebVitals() {
  getCLS(console.log)
  getFID(console.log)
  getFCP(console.log)
  getLCP(console.log)
  getTTFB(console.log)
}

export function trackToAnalytics(metric: any) {
  // Envoyer à backend pour analytics
  fetch('/api/analytics/vitals', {
    method: 'POST',
    body: JSON.stringify(metric),
  }).catch(console.error)
}
```

**Utilisation (frontend/app/layout.tsx):**

```tsx
'use client'

import { useEffect } from 'react'
import { getCLS, getFID, getFCP, getLCP, getTTFB } from 'web-vitals'

export default function RootLayout({
  children,
}: {
  children: React.ReactNode
}) {
  useEffect(() => {
    // Report Web Vitals
    getCLS(console.log)
    getFID(console.log)
    getFCP(console.log)
    getLCP(console.log)
    getTTFB(console.log)
  }, [])

  return (
    <html lang="fr">
      <body>{children}</body>
    </html>
  )
}
```

---

### Étape 10 : Monitoring Performance - Prometheus + Grafana

**Description:** Dashboard de monitoring performance (déjà setup en doc 14).

**Code (backend/internal/metrics/metrics.go):**

```go
package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Request metrics
var (
	RequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name: "http_request_duration_seconds",
			Help: "HTTP request latencies in seconds",
			Buckets: []float64{0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1.0, 2.5, 5.0},
		},
		[]string{"method", "endpoint", "status"},
	)

	RequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total HTTP requests",
		},
		[]string{"method", "endpoint", "status"},
	)

	// Database metrics
	DBConnections = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "db_connections_active",
			Help: "Active database connections",
		},
		[]string{"pool"},
	)

	DBQueryDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name: "db_query_duration_seconds",
			Help: "Database query duration",
			Buckets: []float64{0.001, 0.005, 0.01, 0.05, 0.1, 0.5},
		},
		[]string{"query_type"},
	)

	// Cache metrics
	CacheHits = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "cache_hits_total",
			Help: "Total cache hits",
		},
		[]string{"cache_name"},
	)

	CacheMisses = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "cache_misses_total",
			Help: "Total cache misses",
		},
		[]string{"cache_name"},
	)
)
```

**Middleware pour capture metrics (backend/internal/middleware/metrics.go):**

```go
package middleware

import (
	"time"
	"github.com/gofiber/fiber/v2"
	"yourapp/internal/metrics"
)

func MetricsMiddleware(c *fiber.Ctx) error {
	start := time.Now()

	// Exécuter handler
	err := c.Next()

	// Enregistrer metrics
	duration := time.Since(start).Seconds()
	status := c.Response().StatusCode()

	metrics.RequestDuration.WithLabelValues(
		c.Method(),
		c.Path(),
		string(rune(status)),
	).Observe(duration)

	metrics.RequestsTotal.WithLabelValues(
		c.Method(),
		c.Path(),
		string(rune(status)),
	).Inc()

	return err
}
```

**Grafana dashboard panel (monitoring/grafana/dashboards/performance.json):**

```json
{
  "dashboard": {
    "title": "Performance Metrics",
    "panels": [
      {
        "title": "Request Latency (P95)",
        "targets": [
          {
            "expr": "histogram_quantile(0.95, http_request_duration_seconds_bucket)"
          }
        ]
      },
      {
        "title": "Cache Hit Rate",
        "targets": [
          {
            "expr": "rate(cache_hits_total[5m]) / (rate(cache_hits_total[5m]) + rate(cache_misses_total[5m]))"
          }
        ]
      },
      {
        "title": "Database Connections",
        "targets": [
          {
            "expr": "db_connections_active"
          }
        ]
      }
    ]
  }
}
```

---

## 🧪 Tests

### Tests Unitaires

**Test pagination (backend/internal/api/cv_test.go):**

```go
package api

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetExperiencesPagination(t *testing.T) {
	// Test sans fixtures pour simplifier
	// En production, utiliser testcontainers PostgreSQL

	tests := []struct {
		name           string
		page           int
		limit          int
		expectedStatus int
	}{
		{"Valid pagination", 1, 20, 200},
		{"Page 0 defaults to 1", 0, 20, 200},
		{"Limit > 100 capped to 100", 1, 200, 200},
		{"Negative limit defaults to 20", 1, -5, 200},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Pseudo code (remplacer avec vrai test)
			// req := parseListRequest(tt.page, tt.limit)
			// assert.Equal(t, req.Page, maxInt(tt.page, 1))
			// assert.Equal(t, req.Limit, minMaxInt(tt.limit, 1, 100))
		})
	}
}
```

### Tests Integration

**Test cache Redis (backend/internal/services/cache_test.go):**

```go
package services

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
)

func TestCacheService(t *testing.T) {
	// Utiliser miniredis pour mock Redis sans serveur
	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})
	defer client.Close()

	cs := NewCacheService(client)
	ctx := context.Background()

	t.Run("Set and Get CV", func(t *testing.T) {
		err := cs.SetCV(ctx, "backend", `{"data":"cv_content"}`)
		assert.NoError(t, err)

		val, err := cs.GetCV(ctx, "backend")
		assert.NoError(t, err)
		assert.Equal(t, `{"data":"cv_content"}`, val)
	})

	t.Run("Cache miss returns empty", func(t *testing.T) {
		val, err := cs.GetCV(ctx, "nonexistent")
		assert.NoError(t, err)
		assert.Equal(t, "", val)
	})

	t.Run("TTL expiration", func(t *testing.T) {
		err := cs.SetCV(ctx, "temp", "data")
		assert.NoError(t, err)

		// Simulate expiration
		mr.FastForward(2 * time.Hour)

		val, err := cs.GetCV(ctx, "temp")
		assert.NoError(t, err)
		assert.Equal(t, "", val) // Expired
	})
}
```

### Commandes

```bash
# Run tests
cd backend && go test -v ./...

# Run tests avec coverage
go test -cover ./...

# Benchmark tests (built-in Go)
go test -bench=. -benchmem ./...
```

---

## ⚠️ Points d'Attention

- ⚠️ **N+1 Queries:** Faire un JOIN plutôt que boucle queries. Utiliser GORM Preload/Joins.
- ⚠️ **Large JSON Responses:** Toujours paginer, jamais retourner tableau entier (ex: 10000 records).
- ⚠️ **Cache Invalidation:** "There are only two hard things in Computer Science: cache invalidation and naming things." Planifier stratégie clear TTLs ou events.
- ⚠️ **Memory Leaks Goroutines:** Chaque goroutine lancée doit terminer. Utiliser context.Context avec cancel.
- ⚠️ **Slow PDF Generation:** Générer PDF synchrone = bloquer request. Utiliser job queue (Redis BullMQ ou Asynq).
- 💡 **Bundle Size:** Utiliser webpack-bundle-analyzer pour identifier packages inutiles.
- 💡 **Image Optimization:** Toujours utiliser next/image, pas <img> HTML natif.
- 💡 **Sentry Integration:** Pour error tracking + performance monitoring en production.

---

## 📚 Ressources

- [Go Profiling Best Practices](https://golang.org/doc/diagnostics)
- [Next.js Image Optimization](https://nextjs.org/docs/basic-features/image-optimization)
- [Core Web Vitals Guide](https://web.dev/vitals/)
- [PostgreSQL Query Optimization](https://www.postgresql.org/docs/current/sql-explain.html)
- [Redis Best Practices](https://redis.io/topics/optimization)
- [Prometheus Metrics Best Practices](https://prometheus.io/docs/practices/instrumentation/)
- [wrk Load Testing](https://github.com/wg/wrk)
- [K6 Load Testing](https://k6.io/docs/)

---

## ✅ Checklist de Complétion

- [ ] Redis cache strategies implémentées (CV, lettres, company info)
- [ ] HTTP Cache-Control headers configurés
- [ ] PostgreSQL indexes créés et EXPLAIN ANALYZE validé
- [ ] Connection pooling configuré (pgx)
- [ ] Next.js Image optimization en place
- [ ] Code splitting + dynamic imports implémentés
- [ ] Pagination implémentée sur toutes les listes
- [ ] Compression gzip/brotli activée (Nginx + Fiber)
- [ ] Load tests runés (wrk, k6) et métriques validées
- [ ] Go profiling (pprof) documenté et exécuté
- [ ] Chrome Lighthouse audit complété (score > 90)
- [ ] Prometheus metrics collect bien (dashboard Grafana fonctionnel)
- [ ] Slow queries identifiées et optimisées
- [ ] Web Vitals tracking implémenté
- [ ] Documentation performance best practices rédigée

---

**Dernière mise à jour:** 2025-12-08
**Auteur:** Alexi
**Version:** 1.0
