# 17. SECURITY

## 📋 Métadonnées

- **Phase:** 6
- **Priorité:** 🔴 CRITIQUE
- **Complexité:** ⭐⭐⭐⭐ (4/5)
- **Prérequis:** Tous modules fonctionnels
- **Temps estimé:** 2-3 jours (audit + fixes)
- **Status:** 🔲 À faire

---

## 🎯 Objectif

Implémenter une stratégie de sécurité complète basée sur les standards OWASP Top 10 pour protéger l'application contre les vulnérabilités les plus critiques. Ce document couvre la validation des entrées, la sanitization, la gestion des secrets, le rate limiting, les headers de sécurité et la mise en place d'un processus d'audit continu.

**Principes clés:**
- Defense in depth (sécurité en couches)
- Fail securely (échec sécurisé par défaut)
- Least privilege (moindre privilège)
- Security by design (sécurité dès la conception)

---

## 🏗️ Architecture

### Vue d'Ensemble de la Sécurité

```
┌─────────────────────────────────────────────────────────┐
│                    HTTPS/TLS Layer                       │
│  (Let's Encrypt + Nginx) - Chiffrement transport        │
└───────────────────────┬─────────────────────────────────┘
                        │
┌───────────────────────▼─────────────────────────────────┐
│                  Security Headers                        │
│  CSP, HSTS, X-Frame-Options, X-Content-Type-Options     │
└───────────────────────┬─────────────────────────────────┘
                        │
┌───────────────────────▼─────────────────────────────────┐
│                 Rate Limiting (Nginx + Redis)            │
│  Middleware: Global + per-endpoint limits               │
└───────────────────────┬─────────────────────────────────┘
                        │
┌───────────────────────▼─────────────────────────────────┐
│                  CORS Middleware                         │
│  Whitelist origins, credentials handling                │
└───────────────────────┬─────────────────────────────────┘
                        │
┌───────────────────────▼─────────────────────────────────┐
│            Input Validation (Backend)                    │
│  Fiber validator + custom validation functions          │
└───────────────────────┬─────────────────────────────────┘
                        │
┌───────────────────────▼─────────────────────────────────┐
│         Input Validation (Frontend)                      │
│  Zod schemas - client-side validation                   │
└───────────────────────┬─────────────────────────────────┘
                        │
┌───────────────────────▼─────────────────────────────────┐
│              Sanitization Layer                          │
│  bluemonday (HTML), SQL prepared statements             │
└───────────────────────┬─────────────────────────────────┘
                        │
┌───────────────────────▼─────────────────────────────────┐
│           Database Access (GORM)                         │
│  Parameterized queries, least privilege DB user         │
└───────────────────────┬─────────────────────────────────┘
                        │
┌───────────────────────▼─────────────────────────────────┐
│              Secrets Management                          │
│  .env (development), environment variables (prod)       │
└─────────────────────────────────────────────────────────┘
```

### OWASP Top 10 2021 Mapping

| # | Vulnérabilité | Mitigation dans maicivy |
|---|---------------|-------------------------|
| A01 | Broken Access Control | Middleware tracking, session validation, rate limiting |
| A02 | Cryptographic Failures | HTTPS enforcement, secure cookies (HttpOnly, Secure) |
| A03 | Injection | Prepared statements (GORM), input validation, sanitization |
| A04 | Insecure Design | Architecture review, threat modeling |
| A05 | Security Misconfiguration | Hardening (headers, CORS), dependency scanning |
| A06 | Vulnerable Components | Automated scanning (gosec, npm audit, trivy) |
| A07 | Authentication Failures | Rate limiting, secure session management |
| A08 | Software & Data Integrity | Checksums, signed builds, HTTPS |
| A09 | Logging Failures | Structured logs, monitoring, no sensitive data |
| A10 | SSRF | URL validation, whitelist domains (scraper) |

---

## 📦 Dépendances

### Bibliothèques Go

```bash
# Validation
go get github.com/go-playground/validator/v10

# HTML Sanitization
go get github.com/microcosm-cc/bluemonday

# Security scanning (dev)
go install github.com/securego/gosec/v2/cmd/gosec@latest

# Rate limiting (déjà dans middlewares)
# Redis (go-redis) déjà présent
```

### Bibliothèques NPM

```bash
# Validation frontend
npm install zod

# Security auditing
npm audit

# Content Security Policy
npm install next-secure-headers
```

### Outils Externes

- **gosec** : Security scanner Go
- **npm audit / Snyk** : Dependency vulnerability scanner
- **trivy** : Container security scanner
- **OWASP ZAP** : Automated security testing (optionnel)

---

## 🔨 Implémentation

### Étape 1: Input Validation Backend (Fiber)

**Description:** Valider TOUTES les entrées utilisateur côté backend avec whitelist approach.

**Fichier:** `backend/internal/middleware/validation.go`

```go
package middleware

import (
	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
)

var validate = validator.New()

// ValidateRequest valide une struct avec go-playground/validator
func ValidateRequest(c *fiber.Ctx, obj interface{}) error {
	if err := c.BodyParser(obj); err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	if err := validate.Struct(obj); err != nil {
		validationErrors := err.(validator.ValidationErrors)
		errors := make(map[string]string)

		for _, e := range validationErrors {
			field := e.Field()
			tag := e.Tag()
			errors[field] = "Field validation for '" + field + "' failed on the '" + tag + "' tag"
		}

		return c.Status(422).JSON(fiber.Map{
			"error":  "Validation failed",
			"fields": errors,
		})
	}

	return nil
}
```

**Exemple d'utilisation dans un handler:**

`backend/internal/api/letters.go`

```go
package api

import (
	"github.com/gofiber/fiber/v2"
	"maicivy/internal/middleware"
)

type GenerateLetterRequest struct {
	CompanyName string `json:"company_name" validate:"required,min=2,max=100,alpha_space"`
}

func GenerateLetter(c *fiber.Ctx) error {
	req := new(GenerateLetterRequest)

	if err := middleware.ValidateRequest(c, req); err != nil {
		return err
	}

	// Suite du traitement...
	return c.JSON(fiber.Map{"message": "OK"})
}
```

**Validators customs:**

`backend/internal/utils/validators.go`

```go
package utils

import (
	"regexp"
	"github.com/go-playground/validator/v10"
)

// RegisterCustomValidators enregistre les validators custom
func RegisterCustomValidators(v *validator.Validate) {
	v.RegisterValidation("alpha_space", validateAlphaSpace)
	v.RegisterValidation("safe_url", validateSafeURL)
}

// validateAlphaSpace permet uniquement lettres et espaces
func validateAlphaSpace(fl validator.FieldLevel) bool {
	matched, _ := regexp.MatchString(`^[a-zA-ZÀ-ÿ\s]+$`, fl.Field().String())
	return matched
}

// validateSafeURL valide que l'URL est sûre (whitelist schemes)
func validateSafeURL(fl validator.FieldLevel) bool {
	matched, _ := regexp.MatchString(`^https?://`, fl.Field().String())
	return matched
}
```

**Explications:**
- **Whitelist approach** : on définit ce qui est autorisé, pas ce qui est interdit
- **Validation à plusieurs niveaux** : frontend (UX) + backend (sécurité)
- **Messages d'erreur clairs** sans révéler d'informations sensibles

---

### Étape 2: Input Validation Frontend (Zod)

**Description:** Valider les inputs côté client pour UX et première ligne de défense.

**Fichier:** `frontend/lib/validations.ts`

```typescript
import { z } from 'zod';

// Schema pour génération de lettre
export const letterGenerationSchema = z.object({
  companyName: z
    .string()
    .min(2, 'Le nom doit contenir au moins 2 caractères')
    .max(100, 'Le nom ne peut pas dépasser 100 caractères')
    .regex(/^[a-zA-ZÀ-ÿ\s]+$/, 'Seules les lettres et espaces sont autorisés'),
});

// Schema pour événements analytics
export const analyticsEventSchema = z.object({
  eventType: z.enum(['page_view', 'click', 'cv_theme_change', 'letter_generated']),
  eventData: z.record(z.unknown()).optional(),
});

// Type inference
export type LetterGenerationInput = z.infer<typeof letterGenerationSchema>;
export type AnalyticsEventInput = z.infer<typeof analyticsEventSchema>;
```

**Utilisation dans un composant:**

`frontend/components/letters/LetterGenerator.tsx`

```typescript
'use client';

import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { letterGenerationSchema, LetterGenerationInput } from '@/lib/validations';

export function LetterGenerator() {
  const {
    register,
    handleSubmit,
    formState: { errors, isSubmitting },
  } = useForm<LetterGenerationInput>({
    resolver: zodResolver(letterGenerationSchema),
  });

  const onSubmit = async (data: LetterGenerationInput) => {
    try {
      const response = await fetch('/api/letters/generate', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(data),
      });

      if (!response.ok) {
        throw new Error('Generation failed');
      }

      // Handle success...
    } catch (error) {
      // Handle error...
    }
  };

  return (
    <form onSubmit={handleSubmit(onSubmit)}>
      <input
        {...register('companyName')}
        type="text"
        placeholder="Nom de l'entreprise"
      />
      {errors.companyName && (
        <p className="text-red-500">{errors.companyName.message}</p>
      )}
      <button type="submit" disabled={isSubmitting}>
        Générer
      </button>
    </form>
  );
}
```

**Explications:**
- **Validation côté client** : améliore UX, empêche requêtes invalides inutiles
- **Type safety** : TypeScript + Zod = types automatiques
- **Réutilisabilité** : schemas centralisés dans `lib/validations.ts`

---

### Étape 3: Sanitization (XSS Prevention)

**Description:** Nettoyer les inputs HTML pour prévenir XSS attacks.

**Backend:** `backend/internal/utils/sanitize.go`

```go
package utils

import (
	"github.com/microcosm-cc/bluemonday"
)

var (
	// Policy stricte: retire tout HTML
	strictPolicy = bluemonday.StrictPolicy()

	// Policy permissive pour markdown rendu (si besoin)
	ugcPolicy = bluemonday.UGCPolicy()
)

// SanitizeString retire tout HTML d'une string
func SanitizeString(input string) string {
	return strictPolicy.Sanitize(input)
}

// SanitizeHTML permet HTML safe (pour markdown rendu)
func SanitizeHTML(input string) string {
	return ugcPolicy.Sanitize(input)
}
```

**Usage dans les handlers:**

```go
func GenerateLetter(c *fiber.Ctx) error {
	req := new(GenerateLetterRequest)

	if err := middleware.ValidateRequest(c, req); err != nil {
		return err
	}

	// Sanitize input (defense in depth)
	req.CompanyName = utils.SanitizeString(req.CompanyName)

	// Continue...
}
```

**Frontend:** Next.js échappe automatiquement les variables dans JSX, mais attention aux `dangerouslySetInnerHTML`:

```typescript
// ❌ DANGEREUX
<div dangerouslySetInnerHTML={{ __html: userInput }} />

// ✅ BON - utiliser une lib comme DOMPurify si besoin de rendu HTML
import DOMPurify from 'isomorphic-dompurify';

<div dangerouslySetInnerHTML={{ __html: DOMPurify.sanitize(userInput) }} />
```

**Explications:**
- **Backend sanitization** : toujours nettoyer avant stockage DB
- **Frontend escaping** : React/Next.js échappe par défaut
- **Double protection** : validation + sanitization

---

### Étape 4: SQL Injection Prevention (GORM)

**Description:** GORM utilise des prepared statements par défaut. Éviter raw queries.

**✅ BON - Utiliser GORM Query Builder:**

```go
// Safe: GORM utilise des prepared statements
func GetExperiencesByTheme(theme string) ([]models.Experience, error) {
	var experiences []models.Experience

	err := db.Where("tags @> ?", pq.Array([]string{theme})).
		Order("start_date DESC").
		Find(&experiences).Error

	return experiences, err
}
```

**❌ DANGEREUX - Raw SQL non préparé:**

```go
// VULNÉRABLE À SQL INJECTION - NE PAS FAIRE
func GetExperiencesByThemeBad(theme string) ([]models.Experience, error) {
	var experiences []models.Experience

	// ❌ String concatenation = SQL injection
	query := "SELECT * FROM experiences WHERE theme = '" + theme + "'"
	err := db.Raw(query).Scan(&experiences).Error

	return experiences, err
}
```

**✅ Si raw SQL nécessaire, utiliser des placeholders:**

```go
func CustomQuery(param string) error {
	// ✅ Utiliser des placeholders ? pour paramètres
	err := db.Raw("SELECT * FROM custom_table WHERE field = ?", param).Scan(&result).Error
	return err
}
```

**Principes:**
- **Toujours utiliser GORM Query Builder** quand possible
- **Si raw SQL:** utiliser placeholders (`?`) jamais de concaténation
- **Least privilege:** utilisateur DB avec permissions minimales

---

### Étape 5: Secrets Management

**Description:** Ne JAMAIS commit de secrets. Utiliser variables d'environnement.

**Fichier:** `backend/internal/config/config.go`

```go
package config

import (
	"os"
	"log"
)

type Config struct {
	// Database
	DatabaseURL string

	// Redis
	RedisURL string

	// AI APIs
	ClaudeAPIKey  string
	OpenAIAPIKey  string

	// Security
	JWTSecret     string // Si on ajoute JWT plus tard

	// External APIs
	ClearbitAPIKey string

	// Server
	Port          string
	Environment   string // "development" | "production"
}

func Load() *Config {
	return &Config{
		DatabaseURL:    getEnv("DATABASE_URL", ""),
		RedisURL:       getEnv("REDIS_URL", "redis://localhost:6379"),
		ClaudeAPIKey:   getEnv("CLAUDE_API_KEY", ""),
		OpenAIAPIKey:   getEnv("OPENAI_API_KEY", ""),
		JWTSecret:      getEnv("JWT_SECRET", ""),
		ClearbitAPIKey: getEnv("CLEARBIT_API_KEY", ""),
		Port:           getEnv("PORT", "8080"),
		Environment:    getEnv("ENVIRONMENT", "development"),
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}

	if defaultValue == "" && isCriticalKey(key) {
		log.Fatalf("CRITICAL: Environment variable %s is required but not set", key)
	}

	return defaultValue
}

func isCriticalKey(key string) bool {
	critical := []string{"DATABASE_URL", "CLAUDE_API_KEY", "OPENAI_API_KEY"}
	for _, k := range critical {
		if k == key {
			return true
		}
	}
	return false
}
```

**Fichier:** `.env.example` (à commit)

```bash
# Database
DATABASE_URL=postgresql://user:password@localhost:5432/maicivy?sslmode=disable

# Redis
REDIS_URL=redis://localhost:6379

# AI APIs
CLAUDE_API_KEY=your_claude_api_key_here
OPENAI_API_KEY=your_openai_api_key_here

# External APIs
CLEARBIT_API_KEY=your_clearbit_key_here

# Server
PORT=8080
ENVIRONMENT=development

# Security (generate with: openssl rand -base64 32)
JWT_SECRET=your_random_secret_here
```

**Fichier:** `.env` (DANS .gitignore - NE PAS COMMIT)

```bash
# Real secrets go here
CLAUDE_API_KEY=sk-ant-real-key-here
OPENAI_API_KEY=sk-real-key-here
# ...
```

**`.gitignore` doit contenir:**

```
.env
.env.local
.env.production
*.key
*.pem
secrets/
```

**Production:** Utiliser secrets manager (ex: GitHub Secrets, HashiCorp Vault, AWS Secrets Manager) ou variables d'environnement injectées par Docker/CI/CD.

---

### Étape 6: HTTPS Enforcement

**Description:** Forcer HTTPS pour tout le trafic.

**Nginx Configuration:** `docker/nginx/nginx.conf`

```nginx
# HTTP → HTTPS redirect
server {
    listen 80;
    listen [::]:80;
    server_name maicivy.example.com;

    # Redirect all HTTP to HTTPS
    return 301 https://$server_name$request_uri;
}

# HTTPS server
server {
    listen 443 ssl http2;
    listen [::]:443 ssl http2;
    server_name maicivy.example.com;

    # SSL certificates (Let's Encrypt)
    ssl_certificate /etc/letsencrypt/live/maicivy.example.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/maicivy.example.com/privkey.pem;

    # SSL configuration (Mozilla Modern)
    ssl_protocols TLSv1.3 TLSv1.2;
    ssl_ciphers 'ECDHE-ECDSA-AES128-GCM-SHA256:ECDHE-RSA-AES128-GCM-SHA256:ECDHE-ECDSA-AES256-GCM-SHA384:ECDHE-RSA-AES256-GCM-SHA384';
    ssl_prefer_server_ciphers off;

    # HSTS (HTTP Strict Transport Security)
    add_header Strict-Transport-Security "max-age=63072000; includeSubDomains; preload" always;

    # OCSP Stapling
    ssl_stapling on;
    ssl_stapling_verify on;
    ssl_trusted_certificate /etc/letsencrypt/live/maicivy.example.com/chain.pem;

    # Backend proxy
    location /api {
        proxy_pass http://backend:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }

    # Frontend
    location / {
        proxy_pass http://frontend:3000;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
```

**Certificat Let's Encrypt (auto-renewal):**

```bash
# Installer certbot
sudo apt-get install certbot python3-certbot-nginx

# Obtenir certificat
sudo certbot --nginx -d maicivy.example.com

# Auto-renewal (cron)
sudo certbot renew --dry-run
```

**Backend - Secure Cookies:**

```go
// Dans middleware tracking
c.Cookie(&fiber.Cookie{
    Name:     "session_id",
    Value:    sessionID,
    MaxAge:   86400 * 30, // 30 jours
    Secure:   true,       // HTTPS uniquement
    HTTPOnly: true,       // Pas accessible via JavaScript
    SameSite: "Lax",      // Protection CSRF
})
```

---

### Étape 7: Rate Limiting (Détail)

**Description:** Rate limiting déjà implémenté dans `04. BACKEND_MIDDLEWARES.md`. Recap sécurité:

**Backend:** `backend/internal/middleware/ratelimit.go`

```go
package middleware

import (
	"context"
	"fmt"
	"time"
	"github.com/gofiber/fiber/v2"
	"github.com/redis/go-redis/v9"
)

type RateLimiter struct {
	redis  *redis.Client
	ctx    context.Context
}

func NewRateLimiter(redisClient *redis.Client) *RateLimiter {
	return &RateLimiter{
		redis: redisClient,
		ctx:   context.Background(),
	}
}

// GlobalRateLimit: 100 requêtes/minute par IP
func (rl *RateLimiter) GlobalRateLimit() fiber.Handler {
	return func(c *fiber.Ctx) error {
		ip := c.IP()
		key := fmt.Sprintf("ratelimit:global:%s", ip)

		count, err := rl.redis.Incr(rl.ctx, key).Result()
		if err != nil {
			// Fail open en cas d'erreur Redis (log mais continue)
			return c.Next()
		}

		if count == 1 {
			rl.redis.Expire(rl.ctx, key, time.Minute)
		}

		if count > 100 {
			return c.Status(429).JSON(fiber.Map{
				"error": "Too many requests. Please try again later.",
			})
		}

		return c.Next()
	}
}

// AIRateLimit: 5 requêtes/jour par session
func (rl *RateLimiter) AIRateLimit() fiber.Handler {
	return func(c *fiber.Ctx) error {
		sessionID := c.Cookies("session_id")
		if sessionID == "" {
			return c.Status(400).JSON(fiber.Map{
				"error": "Session required",
			})
		}

		key := fmt.Sprintf("ratelimit:ai:%s", sessionID)

		count, err := rl.redis.Incr(rl.ctx, key).Result()
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": "Internal error"})
		}

		if count == 1 {
			rl.redis.Expire(rl.ctx, key, 24*time.Hour)
		}

		if count > 5 {
			ttl := rl.redis.TTL(rl.ctx, key).Val()
			return c.Status(429).JSON(fiber.Map{
				"error":      "Daily limit reached",
				"retry_after": int(ttl.Seconds()),
			})
		}

		// Ajouter header avec remaining requests
		c.Set("X-RateLimit-Limit", "5")
		c.Set("X-RateLimit-Remaining", fmt.Sprintf("%d", 5-count))

		return c.Next()
	}
}
```

**Utilisation dans routes:**

```go
// Global rate limit sur toutes les routes
app.Use(rateLimiter.GlobalRateLimit())

// Rate limit spécifique sur endpoint AI
app.Post("/api/letters/generate", rateLimiter.AIRateLimit(), handlers.GenerateLetter)
```

---

### Étape 8: CORS Configuration

**Description:** Whitelist origins autorisées, pas de wildcard `*`.

**Fichier:** `backend/internal/middleware/cors.go`

```go
package middleware

import (
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
)

func CORS(allowedOrigins []string) fiber.Handler {
	return cors.New(cors.Config{
		AllowOrigins:     getAllowedOrigins(allowedOrigins),
		AllowMethods:     "GET,POST,PUT,DELETE,OPTIONS",
		AllowHeaders:     "Origin,Content-Type,Accept,Authorization",
		AllowCredentials: true, // Nécessaire pour cookies
		MaxAge:           3600,
	})
}

func getAllowedOrigins(origins []string) string {
	// En dev: localhost autorisé
	// En prod: uniquement domaine principal
	if len(origins) == 0 {
		return "http://localhost:3000" // Fallback dev
	}

	result := ""
	for i, origin := range origins {
		result += origin
		if i < len(origins)-1 {
			result += ","
		}
	}
	return result
}
```

**Configuration selon environnement:**

```go
// Dans main.go
var allowedOrigins []string

if config.Environment == "production" {
	allowedOrigins = []string{"https://maicivy.example.com"}
} else {
	allowedOrigins = []string{
		"http://localhost:3000",
		"http://127.0.0.1:3000",
	}
}

app.Use(middleware.CORS(allowedOrigins))
```

**Principes:**
- **Whitelist explicite** : jamais `AllowOrigins: "*"` avec `AllowCredentials: true`
- **Environment-aware** : différentes origines dev/prod
- **Credentials** : nécessaire pour cookies de session

---

### Étape 9: Security Headers

**Description:** Ajouter headers HTTP de sécurité (Defense in Depth).

**Nginx Configuration:**

```nginx
# Security Headers
add_header X-Frame-Options "DENY" always;
add_header X-Content-Type-Options "nosniff" always;
add_header X-XSS-Protection "1; mode=block" always;
add_header Referrer-Policy "strict-origin-when-cross-origin" always;

# Content Security Policy
add_header Content-Security-Policy "
    default-src 'self';
    script-src 'self' 'unsafe-inline' 'unsafe-eval' https://cdn.jsdelivr.net;
    style-src 'self' 'unsafe-inline' https://fonts.googleapis.com;
    font-src 'self' https://fonts.gstatic.com;
    img-src 'self' data: https:;
    connect-src 'self' https://api.anthropic.com https://api.openai.com;
    frame-ancestors 'none';
    base-uri 'self';
    form-action 'self';
" always;

# Permissions Policy (anciennement Feature Policy)
add_header Permissions-Policy "
    geolocation=(),
    microphone=(),
    camera=(),
    payment=(),
    usb=()
" always;
```

**Next.js Configuration (alternative):**

`frontend/next.config.js`

```javascript
const securityHeaders = [
  {
    key: 'X-DNS-Prefetch-Control',
    value: 'on'
  },
  {
    key: 'Strict-Transport-Security',
    value: 'max-age=63072000; includeSubDomains; preload'
  },
  {
    key: 'X-Frame-Options',
    value: 'DENY'
  },
  {
    key: 'X-Content-Type-Options',
    value: 'nosniff'
  },
  {
    key: 'X-XSS-Protection',
    value: '1; mode=block'
  },
  {
    key: 'Referrer-Policy',
    value: 'strict-origin-when-cross-origin'
  },
  {
    key: 'Permissions-Policy',
    value: 'geolocation=(), microphone=(), camera=()'
  }
];

module.exports = {
  async headers() {
    return [
      {
        source: '/:path*',
        headers: securityHeaders,
      },
    ];
  },
};
```

**Explications des headers:**
- **X-Frame-Options: DENY** : empêche embedding iframe (clickjacking)
- **X-Content-Type-Options: nosniff** : empêche MIME type sniffing
- **CSP** : contrôle ressources chargées (XSS protection)
- **HSTS** : force HTTPS pour tous les futurs accès
- **Referrer-Policy** : contrôle infos Referer envoyées

---

### Étape 10: Dependency Scanning

**Description:** Scanner automatiquement les dépendances pour vulnérabilités connues.

**Go - gosec:**

```bash
# Installer gosec
go install github.com/securego/gosec/v2/cmd/gosec@latest

# Scanner le code
gosec ./...

# Scanner avec rapport JSON
gosec -fmt=json -out=security-report.json ./...
```

**Go - Dependency check:**

```bash
# Vérifier vulnérabilités dans go.mod
go list -json -m all | docker run --rm -i sonatypecommunity/nancy:latest sleuth

# Ou utiliser govulncheck (officiel Go)
go install golang.org/x/vuln/cmd/govulncheck@latest
govulncheck ./...
```

**NPM - npm audit:**

```bash
# Audit dépendances
npm audit

# Audit avec fix automatique (mineur)
npm audit fix

# Audit avec fix majeur (peut casser)
npm audit fix --force

# Rapport JSON
npm audit --json > npm-audit.json
```

**Docker - trivy:**

```bash
# Scanner image Docker
docker run --rm -v /var/run/docker.sock:/var/run/docker.sock \
  aquasec/trivy image maicivy-backend:latest

# Scanner avec sévérité minimale
trivy image --severity HIGH,CRITICAL maicivy-backend:latest
```

**GitHub Actions CI Workflow:**

`.github/workflows/security.yml`

```yaml
name: Security Scan

on:
  push:
    branches: [main]
  pull_request:
    branches: [main]
  schedule:
    - cron: '0 0 * * 0' # Weekly scan

jobs:
  security:
    runs-on: ubuntu-latest

    steps:
      - uses: actions/checkout@v4

      # Go security
      - name: Run gosec
        uses: securego/gosec@master
        with:
          args: './...'

      # Go vulnerabilities
      - name: Run govulncheck
        run: |
          go install golang.org/x/vuln/cmd/govulncheck@latest
          govulncheck ./...

      # NPM audit
      - name: NPM audit
        working-directory: ./frontend
        run: npm audit --audit-level=high

      # Docker scan (trivy)
      - name: Run Trivy
        uses: aquasecurity/trivy-action@master
        with:
          image-ref: 'maicivy-backend:latest'
          severity: 'HIGH,CRITICAL'
          exit-code: '1'
```

---

### Étape 11: Logging Sécurisé

**Description:** Logger les événements de sécurité SANS exposer données sensibles.

**Backend:** `backend/pkg/logger/logger.go`

```go
package logger

import (
	"os"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func Init(environment string) {
	if environment == "development" {
		log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
	} else {
		// Production: JSON structured logs
		zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	}
}

// LogSecurityEvent log événements de sécurité
func LogSecurityEvent(eventType, message string, metadata map[string]interface{}) {
	event := log.Warn().
		Str("type", "security").
		Str("event", eventType).
		Str("message", message)

	for k, v := range metadata {
		// Ne JAMAIS logger: passwords, API keys, tokens, PII
		if isSensitiveField(k) {
			event.Str(k, "[REDACTED]")
		} else {
			event.Interface(k, v)
		}
	}

	event.Send()
}

func isSensitiveField(field string) bool {
	sensitive := []string{
		"password", "token", "api_key", "secret",
		"credit_card", "ssn", "email", // Adapter selon besoins
	}

	for _, s := range sensitive {
		if field == s {
			return true
		}
	}
	return false
}
```

**Exemples de logs de sécurité:**

```go
// Rate limit exceeded
logger.LogSecurityEvent("rate_limit_exceeded", "AI endpoint rate limit hit", map[string]interface{}{
	"ip":         c.IP(),
	"session_id": sessionID,
	"endpoint":   "/api/letters/generate",
})

// Validation failed
logger.LogSecurityEvent("validation_failed", "Input validation failed", map[string]interface{}{
	"ip":     c.IP(),
	"field":  "company_name",
	"reason": "invalid_characters",
})

// Suspicious activity
logger.LogSecurityEvent("suspicious_activity", "Multiple failed attempts", map[string]interface{}{
	"ip":         c.IP(),
	"attempts":   attemptCount,
	"time_window": "5m",
})
```

**Principes:**
- **Ne JAMAIS logger** : passwords, tokens, API keys, PII (Personal Identifiable Info)
- **Logger** : IP (hashé si RGPD), timestamps, event types, error codes
- **Structured logs** : JSON en prod pour parsing facile (Loki, ELK)
- **Retention** : logs de sécurité conservés plus longtemps (6-12 mois)

---

### Étape 12: SSRF Prevention (Scraper)

**Description:** Le scraper d'infos entreprises peut être vulnérable à SSRF. Whitelister les domaines.

**Fichier:** `backend/internal/services/scraper.go`

```go
package services

import (
	"errors"
	"net/url"
	"strings"
)

var (
	ErrInvalidURL      = errors.New("invalid URL")
	ErrBlockedDomain   = errors.New("domain not allowed")
	ErrPrivateIP       = errors.New("private IP addresses not allowed")
)

// Domaines autorisés pour scraping
var allowedDomains = []string{
	"linkedin.com",
	"crunchbase.com",
	"glassdoor.com",
	// Ajouter domaines légitimes uniquement
}

// ValidateURL valide qu'une URL est sûre pour scraping
func ValidateURL(rawURL string) error {
	// Parse URL
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return ErrInvalidURL
	}

	// Vérifier scheme (HTTPS uniquement)
	if parsedURL.Scheme != "https" {
		return errors.New("only HTTPS URLs allowed")
	}

	// Vérifier domaine dans whitelist
	if !isDomainAllowed(parsedURL.Host) {
		return ErrBlockedDomain
	}

	// Bloquer IPs privées (127.0.0.1, 192.168.x.x, etc.)
	if isPrivateIP(parsedURL.Host) {
		return ErrPrivateIP
	}

	return nil
}

func isDomainAllowed(host string) bool {
	for _, allowed := range allowedDomains {
		if strings.HasSuffix(host, allowed) {
			return true
		}
	}
	return false
}

func isPrivateIP(host string) bool {
	// Bloquer localhost, IPs privées
	privateRanges := []string{
		"localhost",
		"127.0.0.1",
		"0.0.0.0",
		"192.168.",
		"10.",
		"172.16.",
		"::1",
	}

	for _, private := range privateRanges {
		if strings.HasPrefix(host, private) {
			return true
		}
	}
	return false
}

// ScrapeCompanyInfo scrape infos avec validation
func ScrapeCompanyInfo(companyName string) (*CompanyInfo, error) {
	// Construire URL (exemple LinkedIn)
	searchURL := "https://www.linkedin.com/company/" + url.QueryEscape(companyName)

	// Valider URL avant requête
	if err := ValidateURL(searchURL); err != nil {
		return nil, err
	}

	// Faire la requête HTTP (avec timeout)
	// ... code scraping ...

	return &CompanyInfo{}, nil
}
```

**Principes SSRF protection:**
- **Whitelist domains** : uniquement domaines légitimes
- **Block private IPs** : empêcher accès réseau interne
- **HTTPS only** : pas de HTTP
- **Timeout** : limiter durée requêtes (5-10s max)
- **No redirects** : ou limiter redirects (max 3)

---

## 🧪 Tests

### Tests de Sécurité Backend

**Fichier:** `backend/internal/middleware/validation_test.go`

```go
package middleware_test

import (
	"testing"
	"github.com/stretchr/testify/assert"
	"maicivy/internal/middleware"
)

func TestValidateRequest_ValidInput(t *testing.T) {
	// Test avec input valide
	req := GenerateLetterRequest{
		CompanyName: "Google Inc",
	}

	err := validate.Struct(req)
	assert.NoError(t, err)
}

func TestValidateRequest_InvalidInput(t *testing.T) {
	// Test injection SQL
	req := GenerateLetterRequest{
		CompanyName: "'; DROP TABLE users--",
	}

	err := validate.Struct(req)
	assert.Error(t, err) // Doit échouer validation
}

func TestValidateRequest_XSSAttempt(t *testing.T) {
	// Test XSS
	req := GenerateLetterRequest{
		CompanyName: "<script>alert('xss')</script>",
	}

	err := validate.Struct(req)
	assert.Error(t, err)
}
```

### Tests de Rate Limiting

```go
func TestRateLimit_ExceedsLimit(t *testing.T) {
	// Setup Redis test
	rdb := setupTestRedis()
	defer rdb.FlushDB(context.Background())

	rl := NewRateLimiter(rdb)

	// Simuler 101 requêtes
	for i := 0; i < 101; i++ {
		// ... test rate limit ...
	}

	// La 101ème doit être bloquée (429)
	assert.Equal(t, 429, statusCode)
}
```

### Tests SSRF Prevention

```go
func TestValidateURL_PrivateIP(t *testing.T) {
	urls := []string{
		"http://127.0.0.1:8080/admin",
		"http://192.168.1.1/",
		"http://localhost/secrets",
	}

	for _, url := range urls {
		err := ValidateURL(url)
		assert.Error(t, err, "Should block private IP: %s", url)
	}
}

func TestValidateURL_ValidDomain(t *testing.T) {
	err := ValidateURL("https://www.linkedin.com/company/google")
	assert.NoError(t, err)
}
```

### Test E2E - Security Scan (Playwright)

`e2e/security.spec.ts`

```typescript
import { test, expect } from '@playwright/test';

test('Should prevent XSS in letter generation', async ({ page }) => {
  await page.goto('/letters');

  // Tenter injection XSS
  await page.fill('input[name="companyName"]', '<script>alert("xss")</script>');
  await page.click('button[type="submit"]');

  // Vérifier que le script n'est PAS exécuté
  await expect(page.locator('script:has-text("alert")')).toHaveCount(0);

  // Vérifier message d'erreur validation
  await expect(page.locator('text=/validation failed/i')).toBeVisible();
});

test('Should enforce rate limiting', async ({ page }) => {
  await page.goto('/letters');

  // Faire 6 requêtes rapidement (limite: 5/jour)
  for (let i = 0; i < 6; i++) {
    await page.fill('input[name="companyName"]', `Company ${i}`);
    await page.click('button[type="submit"]');
    await page.waitForTimeout(500);
  }

  // La 6ème doit être bloquée
  await expect(page.locator('text=/daily limit reached/i')).toBeVisible();
});
```

---

## ⚠️ Points d'Attention

### Pièges à Éviter

- **❌ Validation côté client uniquement** : toujours valider côté serveur
- **❌ Blacklist approach** : préférer whitelist (définir ce qui est autorisé)
- **❌ Commit secrets** : vérifier `.gitignore` avant chaque commit
- **❌ Ignorer dependency updates** : vulnérabilités découvertes régulièrement
- **❌ Logs verbeux** : ne JAMAIS logger passwords, tokens, PII

### Edge Cases

- **Rate limiting + Redis down** : fail open (allow) ou fail closed (deny)? → fail open avec logs
- **Validation multi-langue** : attention aux caractères spéciaux (accents, chinois, etc.)
- **Large payloads** : limiter taille body (Fiber: `BodyLimit`)
- **Slow POST attacks** : timeout lecture body (Fiber: `ReadTimeout`)

### Best Practices

- **Defense in Depth** : plusieurs couches de sécurité (validation + sanitization + prepared statements)
- **Fail Securely** : en cas d'erreur, dénier l'accès par défaut
- **Least Privilege** : permissions minimales (DB user, filesystem, etc.)
- **Security by Design** : penser sécurité dès la conception, pas après

---

## 📚 Ressources

### OWASP

- [OWASP Top 10 2021](https://owasp.org/www-project-top-ten/)
- [OWASP Cheat Sheet Series](https://cheatsheetseries.owasp.org/)
- [OWASP ZAP (Security Testing)](https://www.zaproxy.org/)

### Documentation Officielles

- [Go Security Best Practices](https://golang.org/doc/security)
- [Next.js Security Headers](https://nextjs.org/docs/advanced-features/security-headers)
- [Mozilla Web Security Guidelines](https://infosec.mozilla.org/guidelines/web_security)

### Outils

- [gosec - Go Security Scanner](https://github.com/securego/gosec)
- [Trivy - Container Security](https://github.com/aquasecurity/trivy)
- [Snyk - Dependency Scanning](https://snyk.io/)
- [Let's Encrypt - Free SSL](https://letsencrypt.org/)

### Standards

- [CWE Top 25 (Common Weakness Enumeration)](https://cwe.mitre.org/top25/)
- [NIST Cybersecurity Framework](https://www.nist.gov/cyberframework)

---

## ✅ Checklist de Complétion

### OWASP Top 10 2021

- [ ] **A01: Broken Access Control**
  - [ ] Middleware tracking visiteurs implémenté
  - [ ] Validation session pour endpoints sensibles
  - [ ] Rate limiting configuré (global + AI)

- [ ] **A02: Cryptographic Failures**
  - [ ] HTTPS enforced (Nginx redirect + HSTS)
  - [ ] Secure cookies (HttpOnly, Secure, SameSite)
  - [ ] Secrets en variables d'environnement (jamais commit)

- [ ] **A03: Injection**
  - [ ] Validation inputs backend (Fiber validator)
  - [ ] Validation inputs frontend (Zod)
  - [ ] Sanitization HTML (bluemonday)
  - [ ] GORM prepared statements (no raw SQL)

- [ ] **A04: Insecure Design**
  - [ ] Architecture review documentée
  - [ ] Threat modeling effectué
  - [ ] Security requirements définis

- [ ] **A05: Security Misconfiguration**
  - [ ] Security headers configurés (CSP, HSTS, etc.)
  - [ ] CORS whitelist (pas de wildcard)
  - [ ] Error messages génériques (pas d'infos sensibles)
  - [ ] Production vs development configs séparées

- [ ] **A06: Vulnerable and Outdated Components**
  - [ ] gosec scan CI/CD
  - [ ] govulncheck scan CI/CD
  - [ ] npm audit CI/CD
  - [ ] trivy Docker scan CI/CD
  - [ ] Dependency updates régulières

- [ ] **A07: Identification and Authentication Failures**
  - [ ] Rate limiting (login si ajouté)
  - [ ] Secure session management
  - [ ] Cookie security (HttpOnly, Secure)

- [ ] **A08: Software and Data Integrity Failures**
  - [ ] HTTPS pour toutes les communications
  - [ ] Checksums pour assets critiques
  - [ ] Signed Docker images (optionnel)

- [ ] **A09: Security Logging and Monitoring Failures**
  - [ ] Logs structurés (zerolog JSON)
  - [ ] Security events loggés (rate limit, validation failures)
  - [ ] Pas de données sensibles dans logs
  - [ ] Monitoring Prometheus + Grafana
  - [ ] Alerting configuré (optionnel)

- [ ] **A10: Server-Side Request Forgery (SSRF)**
  - [ ] URL validation (scraper)
  - [ ] Domain whitelist
  - [ ] Block private IPs
  - [ ] HTTPS only pour external requests

### Implémentation

- [ ] Input validation backend implémentée
- [ ] Input validation frontend implémentée (Zod)
- [ ] Sanitization HTML configurée
- [ ] Secrets management setup (.env, config)
- [ ] HTTPS enforcement (Nginx + Let's Encrypt)
- [ ] Rate limiting testé
- [ ] CORS configuration validée
- [ ] Security headers configurés (Nginx)
- [ ] Dependency scanning CI/CD setup
- [ ] Logging sécurisé implémenté
- [ ] SSRF protection (scraper) implémentée

### Tests

- [ ] Tests validation inputs (XSS, SQL injection attempts)
- [ ] Tests rate limiting
- [ ] Tests SSRF prevention
- [ ] Tests E2E sécurité (Playwright)
- [ ] Security scan automatisé (CI/CD)

### Documentation

- [ ] Security guidelines documentées
- [ ] Incident response plan (optionnel)
- [ ] SECURITY.md dans repo (responsible disclosure)
- [ ] README avec security badges

### Audit & Review

- [ ] Code review sécurité effectué
- [ ] OWASP ZAP scan (optionnel)
- [ ] Penetration testing (optionnel)
- [ ] Security checklist validée

---

## 🔄 Maintenance Continue

### Processus de Sécurité Régulier

1. **Hebdomadaire:**
   - Vérifier alertes dépendances (GitHub Dependabot)
   - Review logs de sécurité (rate limits, validation failures)

2. **Mensuel:**
   - Mettre à jour dépendances (Go modules, NPM packages)
   - Scanner avec gosec + npm audit
   - Review security headers (Mozilla Observatory)

3. **Trimestriel:**
   - Audit sécurité complet
   - Test pénétration (OWASP ZAP)
   - Review et update OWASP Top 10 checklist
   - Rotation API keys (si applicable)

4. **Annuel:**
   - Audit externe (si budget)
   - Review architecture sécurité
   - Update security documentation
   - Formation sécurité équipe

### Responsible Disclosure

**Fichier:** `SECURITY.md` (à la racine du repo)

```markdown
# Security Policy

## Reporting a Vulnerability

If you discover a security vulnerability in maicivy, please report it by emailing **security@maicivy.example.com**.

**Please do NOT open a public GitHub issue.**

### What to Include

- Description of the vulnerability
- Steps to reproduce
- Potential impact
- Suggested fix (if any)

### Response Timeline

- **Acknowledgment:** Within 48 hours
- **Initial Assessment:** Within 1 week
- **Fix & Disclosure:** Coordinated with reporter

## Supported Versions

| Version | Supported          |
| ------- | ------------------ |
| 1.x     | :white_check_mark: |

## Security Best Practices for Contributors

- Never commit secrets (.env files)
- Always validate user inputs
- Use prepared statements (GORM)
- Follow OWASP Top 10 guidelines

Thank you for helping keep maicivy secure!
```

---

**Dernière mise à jour:** 2025-12-08
**Auteur:** Alexi
**Review:** Security Team
