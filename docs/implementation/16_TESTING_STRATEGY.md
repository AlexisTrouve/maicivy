# 16. TESTING_STRATEGY - Stratégie de Tests

## 📋 Métadonnées

- **Phase:** 6 - Production & Qualité
- **Priorité:** 🟡 HAUTE
- **Complexité:** ⭐⭐⭐⭐ (4/5)
- **Prérequis:** Tous modules fonctionnels (docs 01-13)
- **Temps estimé:** Continu (tests écrits au fur et à mesure)
- **Status:** 🔲 À faire

---

## 🎯 Objectif

Définir et implémenter une stratégie de tests complète couvrant l'ensemble du projet **maicivy**, de l'unité au end-to-end, garantissant la qualité, la fiabilité et la maintenabilité du code. Cette stratégie inclut les tests unitaires backend (Go), les tests d'intégration (PostgreSQL, Redis), les tests frontend (React/Next.js), et les tests E2E, avec des objectifs de couverture et une intégration continue.

---

## 🏗️ Architecture de Testing

### Vue d'Ensemble

```
┌─────────────────────────────────────────────────────────┐
│                    PYRAMID DE TESTS                     │
├─────────────────────────────────────────────────────────┤
│                                                         │
│           E2E Tests (Playwright)                        │
│           ▲ 5-10 scénarios critiques                    │
│          ╱ ╲                                            │
│         ╱   ╲                                           │
│        ╱     ╲  Integration Tests                       │
│       ╱       ╲  ▲ Tests API complètes                  │
│      ╱         ╲ ▲ PostgreSQL, Redis                    │
│     ╱           ╲                                       │
│    ╱             ╲  Unit Tests                          │
│   ╱_______________╲  ▲ Services, algorithmes            │
│                      ▲ Components, hooks                │
│                      ▲ Coverage: 80% backend, 70% front │
└─────────────────────────────────────────────────────────┘
```

### Design Decisions

1. **Test-Driven Development (TDD) recommandé** : Écrire les tests au fur et à mesure, pas à la fin
2. **Isolation des tests** : Chaque test doit être indépendant (pas de dépendance entre tests)
3. **Fixtures réutilisables** : Seed data standardisé pour tests cohérents
4. **Mocks vs Real** :
   - Tests unitaires : mocks (testify/gomock)
   - Tests intégration : vraies instances (testcontainers)
5. **CI/CD First** : Tous les tests doivent passer avant merge/deploy

---

## 📦 Dépendances

### Bibliothèques Go

```bash
# Framework de testing standard
go test (built-in)

# Assertions et helpers
go get github.com/stretchr/testify

# Mocking
go get github.com/golang/mock/gomock
go get github.com/golang/mock/mockgen

# Testcontainers (PostgreSQL, Redis)
go get github.com/testcontainers/testcontainers-go
go get github.com/testcontainers/testcontainers-go/modules/postgres
go get github.com/testcontainers/testcontainers-go/modules/redis

# HTTP testing
go get github.com/gofiber/fiber/v2 (built-in test utils)

# Coverage
go get github.com/axw/gocov/gocov
go get github.com/AlekSi/gocov-xml
```

### Bibliothèques NPM

```bash
# Testing framework
npm install --save-dev jest @types/jest

# React Testing Library
npm install --save-dev @testing-library/react
npm install --save-dev @testing-library/jest-dom
npm install --save-dev @testing-library/user-event

# E2E Testing
npm install --save-dev @playwright/test

# Coverage
npm install --save-dev @jest/coverage

# Mocks
npm install --save-dev msw (Mock Service Worker)
```

### Services Externes

- **Codecov** : Rapports de couverture (optionnel, gratuit pour open source)
- **Testcontainers Cloud** : Exécution containers CI (optionnel)

---

## 🔨 Implémentation

### Étape 1: Tests Unitaires Backend (Go)

**Description:** Écrire des tests unitaires pour tous les services, algorithmes et handlers backend avec testify

**Structure:**

```
backend/
├── internal/
│   ├── services/
│   │   ├── cv_scoring.go
│   │   ├── cv_scoring_test.go      # Tests unitaires service CV
│   │   ├── ai.go
│   │   ├── ai_test.go              # Tests avec mocks API IA
│   │   ├── analytics.go
│   │   └── analytics_test.go
│   ├── api/
│   │   ├── cv.go
│   │   ├── cv_test.go              # Tests handlers HTTP
│   │   ├── letters.go
│   │   └── letters_test.go
│   └── middleware/
│       ├── tracking.go
│       └── tracking_test.go
└── pkg/
    └── utils/
        └── utils_test.go
```

**Code exemple - Test unitaire algorithme scoring CV:**

```go
// backend/internal/services/cv_scoring_test.go
package services

import (
    "testing"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/suite"
)

// Suite de tests pour le service de scoring CV
type CVScoringTestSuite struct {
    suite.Suite
    service *CVScoringService
}

// Setup avant chaque test
func (suite *CVScoringTestSuite) SetupTest() {
    suite.service = NewCVScoringService()
}

// Test scoring avec tags exacts
func (suite *CVScoringTestSuite) TestScoreExperience_ExactTagMatch() {
    experience := &Experience{
        Title:  "Backend Developer",
        Tags:   []string{"go", "postgresql", "redis"},
        Years:  3,
    }

    theme := &Theme{
        Name:     "backend",
        Keywords: []string{"go", "postgresql"},
        Weight:   1.0,
    }

    score := suite.service.ScoreExperience(experience, theme)

    // Assert score élevé (2 tags sur 3 matchent)
    assert.Greater(suite.T(), score, 0.6, "Score devrait être > 0.6 pour 2 tags matchés")
    assert.LessOrEqual(suite.T(), score, 1.0, "Score max est 1.0")
}

// Test scoring sans match
func (suite *CVScoringTestSuite) TestScoreExperience_NoMatch() {
    experience := &Experience{
        Title: "Designer",
        Tags:  []string{"photoshop", "illustrator"},
        Years: 2,
    }

    theme := &Theme{
        Name:     "backend",
        Keywords: []string{"go", "postgresql"},
        Weight:   1.0,
    }

    score := suite.service.ScoreExperience(experience, theme)

    // Assert score très bas ou 0
    assert.Equal(suite.T(), 0.0, score, "Score devrait être 0 sans match")
}

// Test tri par score décroissant
func (suite *CVScoringTestSuite) TestSortExperiencesByScore() {
    experiences := []*Experience{
        {Title: "Backend Dev", Tags: []string{"go"}, Score: 0.8},
        {Title: "Frontend Dev", Tags: []string{"react"}, Score: 0.3},
        {Title: "Full-Stack Dev", Tags: []string{"go", "react"}, Score: 0.9},
    }

    sorted := suite.service.SortByScore(experiences)

    // Assert ordre décroissant
    assert.Equal(suite.T(), "Full-Stack Dev", sorted[0].Title)
    assert.Equal(suite.T(), "Backend Dev", sorted[1].Title)
    assert.Equal(suite.T(), "Frontend Dev", sorted[2].Title)
}

// Lancer la suite
func TestCVScoringTestSuite(t *testing.T) {
    suite.Run(t, new(CVScoringTestSuite))
}
```

**Code exemple - Test handler avec mock database:**

```go
// backend/internal/api/cv_test.go
package api

import (
    "net/http/httptest"
    "testing"
    "github.com/gofiber/fiber/v2"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/mock"
)

// Mock du repository
type MockCVRepository struct {
    mock.Mock
}

func (m *MockCVRepository) GetExperiences() ([]*Experience, error) {
    args := m.Called()
    return args.Get(0).([]*Experience), args.Error(1)
}

// Test GET /api/cv?theme=backend
func TestGetCVByTheme(t *testing.T) {
    // Setup
    app := fiber.New()
    mockRepo := new(MockCVRepository)
    handler := NewCVHandler(mockRepo)

    app.Get("/api/cv", handler.GetCV)

    // Mock data
    experiences := []*Experience{
        {ID: 1, Title: "Backend Dev", Tags: []string{"go"}},
        {ID: 2, Title: "Frontend Dev", Tags: []string{"react"}},
    }
    mockRepo.On("GetExperiences").Return(experiences, nil)

    // Request
    req := httptest.NewRequest("GET", "/api/cv?theme=backend", nil)
    resp, err := app.Test(req)

    // Assertions
    assert.NoError(t, err)
    assert.Equal(t, 200, resp.StatusCode)
    mockRepo.AssertExpectations(t)
}
```

**Explications:**

- **testify/suite** : Structure tests en suites avec setup/teardown
- **testify/assert** : Assertions lisibles et expressives
- **testify/mock** : Mocking de dépendances (DB, services externes)
- **Fiber test utilities** : Tester handlers HTTP sans serveur réel

---

### Étape 2: Tests IA avec Mocks API

**Description:** Tester le service IA sans appeler les vraies APIs Claude/OpenAI (économie coûts + rapidité)

**Code:**

```go
// backend/internal/services/ai_test.go
package services

import (
    "context"
    "testing"
    "github.com/stretchr/testify/assert"
    "github.com/golang/mock/gomock"
)

// Mock généré par mockgen
// go:generate mockgen -source=ai.go -destination=ai_mock.go -package=services

func TestAIService_GenerateLetter_Success(t *testing.T) {
    ctrl := gomock.NewController(t)
    defer ctrl.Finish()

    // Mock du client Claude
    mockClient := NewMockClaudeClient(ctrl)

    // Expect appel API avec prompt spécifique
    mockClient.EXPECT().
        SendMessage(gomock.Any(), gomock.Any()).
        Return(&AIResponse{
            Content: "Chère entreprise XYZ, je suis motivé car...",
            Tokens:  150,
        }, nil)

    // Test service avec mock
    service := NewAIService(mockClient)
    letter, err := service.GenerateMotivationLetter(context.Background(), "XYZ Corp", "backend")

    assert.NoError(t, err)
    assert.NotEmpty(t, letter.Content)
    assert.Contains(t, letter.Content, "motivé")
}

func TestAIService_Fallback_ClaudeToGPT(t *testing.T) {
    ctrl := gomock.NewController(t)
    defer ctrl.Finish()

    mockClaude := NewMockClaudeClient(ctrl)
    mockGPT := NewMockGPTClient(ctrl)

    // Claude échoue
    mockClaude.EXPECT().
        SendMessage(gomock.Any(), gomock.Any()).
        Return(nil, errors.New("API rate limit"))

    // Fallback sur GPT
    mockGPT.EXPECT().
        SendMessage(gomock.Any(), gomock.Any()).
        Return(&AIResponse{Content: "Lettre générée par GPT"}, nil)

    service := NewAIService(mockClaude, mockGPT)
    letter, err := service.GenerateMotivationLetter(context.Background(), "ABC Inc", "frontend")

    assert.NoError(t, err)
    assert.Contains(t, letter.Content, "GPT")
}
```

**Génération des mocks:**

```bash
# Installer mockgen
go install github.com/golang/mock/mockgen@latest

# Générer mocks depuis interfaces
mockgen -source=backend/internal/services/ai.go \
        -destination=backend/internal/services/ai_mock.go \
        -package=services
```

---

### Étape 3: Tests d'Intégration Backend (PostgreSQL, Redis)

**Description:** Tester les interactions réelles avec bases de données via testcontainers

**Code:**

```go
// backend/internal/database/integration_test.go
//go:build integration
// +build integration

package database

import (
    "context"
    "testing"
    "time"

    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/suite"
    "github.com/testcontainers/testcontainers-go"
    "github.com/testcontainers/testcontainers-go/modules/postgres"
    "github.com/testcontainers/testcontainers-go/modules/redis"
)

type IntegrationTestSuite struct {
    suite.Suite
    pgContainer    *postgres.PostgresContainer
    redisContainer *redis.RedisContainer
    db             *DB
    cache          *Cache
}

// Setup avant toute la suite
func (suite *IntegrationTestSuite) SetupSuite() {
    ctx := context.Background()

    // Démarrer PostgreSQL container
    pgContainer, err := postgres.RunContainer(ctx,
        testcontainers.WithImage("postgres:15-alpine"),
        postgres.WithDatabase("testdb"),
        postgres.WithUsername("test"),
        postgres.WithPassword("test"),
    )
    suite.NoError(err)
    suite.pgContainer = pgContainer

    // Démarrer Redis container
    redisContainer, err := redis.RunContainer(ctx,
        testcontainers.WithImage("redis:7-alpine"),
    )
    suite.NoError(err)
    suite.redisContainer = redisContainer

    // Connect
    connStr, _ := pgContainer.ConnectionString(ctx)
    suite.db, err = NewDB(connStr)
    suite.NoError(err)

    redisAddr, _ := redisContainer.Endpoint(ctx, "")
    suite.cache, err = NewCache(redisAddr)
    suite.NoError(err)

    // Run migrations
    suite.db.AutoMigrate()
}

// Teardown après toute la suite
func (suite *IntegrationTestSuite) TearDownSuite() {
    ctx := context.Background()
    suite.pgContainer.Terminate(ctx)
    suite.redisContainer.Terminate(ctx)
}

// Reset DB avant chaque test
func (suite *IntegrationTestSuite) SetupTest() {
    suite.db.TruncateAll() // Helper pour vider tables
}

// Test création et récupération expérience
func (suite *IntegrationTestSuite) TestCreateAndGetExperience() {
    ctx := context.Background()

    // Create
    exp := &Experience{
        Title:       "Software Engineer",
        Company:     "Tech Corp",
        StartDate:   time.Now().AddDate(-2, 0, 0),
        Tags:        []string{"go", "postgresql"},
        Description: "Backend development",
    }

    err := suite.db.CreateExperience(ctx, exp)
    suite.NoError(err)
    suite.NotZero(exp.ID)

    // Get
    retrieved, err := suite.db.GetExperienceByID(ctx, exp.ID)
    suite.NoError(err)
    suite.Equal(exp.Title, retrieved.Title)
    suite.Equal(exp.Company, retrieved.Company)
    suite.ElementsMatch(exp.Tags, retrieved.Tags)
}

// Test cache Redis
func (suite *IntegrationTestSuite) TestRedisCache() {
    ctx := context.Background()

    key := "test:key"
    value := "test value"

    // Set
    err := suite.cache.Set(ctx, key, value, 5*time.Second)
    suite.NoError(err)

    // Get
    result, err := suite.cache.Get(ctx, key)
    suite.NoError(err)
    suite.Equal(value, result)

    // Expire
    time.Sleep(6 * time.Second)
    _, err = suite.cache.Get(ctx, key)
    suite.Error(err) // Key devrait avoir expiré
}

func TestIntegrationTestSuite(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping integration tests in short mode")
    }
    suite.Run(t, new(IntegrationTestSuite))
}
```

**Commandes:**

```bash
# Tests unitaires seulement (rapides)
go test -v -short ./...

# Tests d'intégration (avec containers)
go test -v -tags=integration ./...

# Tous les tests avec coverage
go test -v -tags=integration -coverprofile=coverage.out ./...
go tool cover -html=coverage.out -o coverage.html
```

---

### Étape 4: Tests Unitaires Frontend (Jest + React Testing Library)

**Description:** Tester components React, hooks et utilities en isolation

**Configuration:**

```javascript
// frontend/jest.config.js
const nextJest = require('next/jest')

const createJestConfig = nextJest({
  dir: './',
})

const customJestConfig = {
  setupFilesAfterEnv: ['<rootDir>/jest.setup.js'],
  moduleDirectories: ['node_modules', '<rootDir>/'],
  testEnvironment: 'jest-environment-jsdom',
  collectCoverageFrom: [
    'app/**/*.{ts,tsx}',
    'components/**/*.{ts,tsx}',
    'lib/**/*.{ts,tsx}',
    '!**/*.d.ts',
    '!**/node_modules/**',
  ],
  coverageThreshold: {
    global: {
      branches: 70,
      functions: 70,
      lines: 70,
      statements: 70,
    },
  },
}

module.exports = createJestConfig(customJestConfig)
```

```javascript
// frontend/jest.setup.js
import '@testing-library/jest-dom'
```

**Code exemple - Test component:**

```typescript
// frontend/components/cv/CVThemeSelector.test.tsx
import { render, screen, fireEvent, waitFor } from '@testing-library/react'
import { CVThemeSelector } from './CVThemeSelector'
import { rest } from 'msw'
import { setupServer } from 'msw/node'

// Mock API server
const server = setupServer(
  rest.get('/api/cv/themes', (req, res, ctx) => {
    return res(ctx.json([
      { id: 1, name: 'backend', keywords: ['go', 'postgresql'] },
      { id: 2, name: 'frontend', keywords: ['react', 'typescript'] },
    ]))
  })
)

beforeAll(() => server.listen())
afterEach(() => server.resetHandlers())
afterAll(() => server.close())

describe('CVThemeSelector', () => {
  it('should render theme selector with fetched themes', async () => {
    render(<CVThemeSelector onThemeChange={jest.fn()} />)

    // Vérifie loading state
    expect(screen.getByText(/loading/i)).toBeInTheDocument()

    // Attends chargement thèmes
    await waitFor(() => {
      expect(screen.getByText('backend')).toBeInTheDocument()
      expect(screen.getByText('frontend')).toBeInTheDocument()
    })
  })

  it('should call onThemeChange when theme is selected', async () => {
    const mockOnChange = jest.fn()
    render(<CVThemeSelector onThemeChange={mockOnChange} />)

    await waitFor(() => screen.getByText('backend'))

    // Click sur thème
    fireEvent.click(screen.getByText('backend'))

    expect(mockOnChange).toHaveBeenCalledWith('backend')
  })

  it('should display error message on API failure', async () => {
    // Override API avec erreur
    server.use(
      rest.get('/api/cv/themes', (req, res, ctx) => {
        return res(ctx.status(500))
      })
    )

    render(<CVThemeSelector onThemeChange={jest.fn()} />)

    await waitFor(() => {
      expect(screen.getByText(/error loading themes/i)).toBeInTheDocument()
    })
  })
})
```

**Code exemple - Test hook:**

```typescript
// frontend/lib/hooks/useVisitorTracking.test.ts
import { renderHook, waitFor } from '@testing-library/react'
import { useVisitorTracking } from './useVisitorTracking'
import { rest } from 'msw'
import { setupServer } from 'msw/node'

const server = setupServer(
  rest.get('/api/visitor/status', (req, res, ctx) => {
    return res(ctx.json({ visitCount: 3, profileDetected: false }))
  })
)

beforeAll(() => server.listen())
afterEach(() => server.resetHandlers())
afterAll(() => server.close())

describe('useVisitorTracking', () => {
  it('should fetch visitor status and return visit count', async () => {
    const { result } = renderHook(() => useVisitorTracking())

    // Initial state
    expect(result.current.loading).toBe(true)

    // Après fetch
    await waitFor(() => {
      expect(result.current.loading).toBe(false)
      expect(result.current.visitCount).toBe(3)
      expect(result.current.hasAIAccess).toBe(true) // >= 3 visites
    })
  })

  it('should indicate no AI access for < 3 visits', async () => {
    server.use(
      rest.get('/api/visitor/status', (req, res, ctx) => {
        return res(ctx.json({ visitCount: 2, profileDetected: false }))
      })
    )

    const { result } = renderHook(() => useVisitorTracking())

    await waitFor(() => {
      expect(result.current.visitCount).toBe(2)
      expect(result.current.hasAIAccess).toBe(false)
    })
  })
})
```

**Commandes:**

```bash
# Tests unitaires
npm test

# Tests avec coverage
npm test -- --coverage

# Watch mode (développement)
npm test -- --watch
```

---

### Étape 5: Tests End-to-End (Playwright)

**Description:** Tester scénarios complets utilisateur sur l'application déployée

**Configuration:**

```typescript
// frontend/playwright.config.ts
import { defineConfig, devices } from '@playwright/test'

export default defineConfig({
  testDir: './e2e',
  fullyParallel: true,
  forbidOnly: !!process.env.CI,
  retries: process.env.CI ? 2 : 0,
  workers: process.env.CI ? 1 : undefined,
  reporter: 'html',
  use: {
    baseURL: process.env.BASE_URL || 'http://localhost:3000',
    trace: 'on-first-retry',
    screenshot: 'only-on-failure',
  },
  projects: [
    {
      name: 'chromium',
      use: { ...devices['Desktop Chrome'] },
    },
    {
      name: 'firefox',
      use: { ...devices['Desktop Firefox'] },
    },
    {
      name: 'webkit',
      use: { ...devices['Desktop Safari'] },
    },
    {
      name: 'Mobile Chrome',
      use: { ...devices['Pixel 5'] },
    },
  ],
  webServer: {
    command: 'npm run dev',
    url: 'http://localhost:3000',
    reuseExistingServer: !process.env.CI,
  },
})
```

**Code exemple - E2E scénario génération lettre:**

```typescript
// frontend/e2e/letter-generation.spec.ts
import { test, expect } from '@playwright/test'

test.describe('Letter Generation Flow', () => {
  test('should generate letters after 3 visits', async ({ page, context }) => {
    // Visite 1
    await page.goto('/')
    await expect(page.locator('h1')).toContainText('maicivy')

    // Aller sur page lettres
    await page.goto('/letters')

    // Devrait afficher teaser (< 3 visites)
    await expect(page.locator('text=Encore 2 visites')).toBeVisible()

    // Visite 2 (reload page)
    await page.reload()
    await expect(page.locator('text=Encore 1 visite')).toBeVisible()

    // Visite 3
    await page.reload()

    // Form devrait être accessible
    const companyInput = page.locator('input[name="company"]')
    await expect(companyInput).toBeVisible()
    await expect(companyInput).toBeEnabled()

    // Remplir form
    await companyInput.fill('Google')
    await page.click('button[type="submit"]')

    // Attendre génération (loading state)
    await expect(page.locator('text=Génération en cours')).toBeVisible()

    // Résultats affichés
    await expect(page.locator('text=Lettre de Motivation')).toBeVisible({ timeout: 30000 })
    await expect(page.locator('text=Lettre d\'Anti-Motivation')).toBeVisible()

    // Vérifier contenu contient "Google"
    const motivationLetter = page.locator('[data-testid="motivation-letter"]')
    await expect(motivationLetter).toContainText('Google')

    // Test export PDF
    const downloadPromise = page.waitForEvent('download')
    await page.click('button:has-text("Télécharger PDF")')
    const download = await downloadPromise
    expect(download.suggestedFilename()).toMatch(/lettre.*\.pdf/i)
  })

  test('should handle rate limiting', async ({ page }) => {
    // Simuler 5 générations successives
    for (let i = 0; i < 5; i++) {
      await page.goto('/letters')
      await page.fill('input[name="company"]', `Company${i}`)
      await page.click('button[type="submit"]')
      await page.waitForSelector('text=Lettre de Motivation', { timeout: 30000 })
    }

    // 6ème tentative devrait être rate-limitée
    await page.goto('/letters')
    await page.fill('input[name="company"]', 'TooMany Corp')
    await page.click('button[type="submit"]')

    await expect(page.locator('text=limite quotidienne atteinte')).toBeVisible()
  })
})

test.describe('CV Dynamic Themes', () => {
  test('should display different CV content for different themes', async ({ page }) => {
    // CV Backend
    await page.goto('/cv?theme=backend')
    await expect(page.locator('text=Go')).toBeVisible()
    await expect(page.locator('text=PostgreSQL')).toBeVisible()

    // CV Frontend
    await page.goto('/cv?theme=frontend')
    await expect(page.locator('text=React')).toBeVisible()
    await expect(page.locator('text=TypeScript')).toBeVisible()

    // Vérifier que contenu change réellement
    const backendPage = await page.goto('/cv?theme=backend')
    const backendContent = await page.textContent('body')

    const frontendPage = await page.goto('/cv?theme=frontend')
    const frontendContent = await page.textContent('body')

    expect(backendContent).not.toEqual(frontendContent)
  })
})

test.describe('Analytics Dashboard', () => {
  test('should display realtime analytics', async ({ page }) => {
    await page.goto('/analytics')

    // Vérifier éléments dashboard
    await expect(page.locator('text=Visiteurs actuels')).toBeVisible()
    await expect(page.locator('[data-testid="realtime-visitors"]')).toBeVisible()

    // Vérifier graphiques
    await expect(page.locator('[data-testid="theme-stats-chart"]')).toBeVisible()
    await expect(page.locator('[data-testid="letters-chart"]')).toBeVisible()

    // Attendre mise à jour temps réel (WebSocket)
    await page.waitForTimeout(2000)
    const initialCount = await page.locator('[data-testid="realtime-visitors"]').textContent()

    // Ouvrir nouvelle page (nouveau visiteur)
    const newPage = await page.context().newPage()
    await newPage.goto('/')

    // Vérifier compteur a augmenté
    await page.waitForTimeout(1000)
    const newCount = await page.locator('[data-testid="realtime-visitors"]').textContent()
    expect(parseInt(newCount || '0')).toBeGreaterThan(parseInt(initialCount || '0'))
  })
})
```

**Commandes:**

```bash
# Installer Playwright
npx playwright install

# Run E2E tests
npx playwright test

# Run avec UI mode (debug)
npx playwright test --ui

# Run spécifique browser
npx playwright test --project=chromium

# Générer rapport HTML
npx playwright show-report
```

---

### Étape 6: Fixtures et Seed Data

**Description:** Créer des fixtures réutilisables pour tests cohérents

**Code:**

```go
// backend/internal/testutil/fixtures.go
package testutil

import (
    "time"
    "github.com/maicivy/internal/models"
)

// Fixtures pré-définies pour tests
var (
    FixtureExperienceBackend = &models.Experience{
        Title:       "Backend Developer",
        Company:     "Tech Corp",
        Description: "Developed APIs in Go",
        StartDate:   time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
        EndDate:     nil, // Current
        Tags:        []string{"go", "postgresql", "redis", "fiber"},
        Category:    "backend",
    }

    FixtureExperienceFrontend = &models.Experience{
        Title:       "Frontend Developer",
        Company:     "Design Studio",
        Description: "Built React applications",
        StartDate:   time.Date(2019, 6, 1, 0, 0, 0, 0, time.UTC),
        EndDate:     &time.Time{}, // Ended
        Tags:        []string{"react", "typescript", "nextjs"},
        Category:    "frontend",
    }

    FixtureSkillGo = &models.Skill{
        Name:            "Go",
        Level:           "Advanced",
        Category:        "backend",
        YearsExperience: 4,
        Tags:            []string{"backend", "api", "microservices"},
    }

    FixtureThemeBackend = &models.Theme{
        Name:     "backend",
        Keywords: []string{"go", "api", "database", "microservices"},
        Weight:   1.0,
    }
)

// Helper pour créer dataset complet pour tests
func CreateFullCVDataset(db *DB) error {
    experiences := []*models.Experience{
        FixtureExperienceBackend,
        FixtureExperienceFrontend,
    }

    skills := []*models.Skill{
        FixtureSkillGo,
        {Name: "React", Level: "Intermediate", Category: "frontend"},
        {Name: "PostgreSQL", Level: "Advanced", Category: "database"},
    }

    for _, exp := range experiences {
        if err := db.Create(exp).Error; err != nil {
            return err
        }
    }

    for _, skill := range skills {
        if err := db.Create(skill).Error; err != nil {
            return err
        }
    }

    return nil
}
```

```typescript
// frontend/lib/testutil/fixtures.ts
export const mockThemes = [
  { id: 1, name: 'backend', keywords: ['go', 'postgresql'] },
  { id: 2, name: 'frontend', keywords: ['react', 'typescript'] },
  { id: 3, name: 'fullstack', keywords: ['go', 'react'] },
]

export const mockExperiences = [
  {
    id: 1,
    title: 'Backend Developer',
    company: 'Tech Corp',
    startDate: '2020-01-01',
    tags: ['go', 'postgresql'],
    description: 'Backend development',
  },
  {
    id: 2,
    title: 'Frontend Developer',
    company: 'Design Studio',
    startDate: '2019-06-01',
    endDate: '2020-01-01',
    tags: ['react', 'typescript'],
    description: 'Frontend development',
  },
]

export const mockVisitorStatus = {
  visitCount: 3,
  profileDetected: false,
  hasAIAccess: true,
}

export const mockGeneratedLetter = {
  id: '123',
  companyName: 'Google',
  motivationLetter: 'Je suis très motivé par...',
  antiMotivationLetter: 'Je ne suis pas du tout intéressé par...',
  createdAt: new Date().toISOString(),
}
```

---

### Étape 7: Coverage Targets et CI Integration

**Description:** Définir objectifs de couverture et intégrer tests au CI

**Configuration CI (GitHub Actions):**

```yaml
# .github/workflows/ci.yml
name: CI - Tests & Coverage

on:
  push:
    branches: [main, develop]
  pull_request:
    branches: [main]

jobs:
  backend-tests:
    runs-on: ubuntu-latest

    services:
      postgres:
        image: postgres:15-alpine
        env:
          POSTGRES_PASSWORD: test
          POSTGRES_DB: testdb
        options: >-
          --health-cmd pg_isready
          --health-interval 10s
          --health-timeout 5s
          --health-retries 5
        ports:
          - 5432:5432

      redis:
        image: redis:7-alpine
        options: >-
          --health-cmd "redis-cli ping"
          --health-interval 10s
          --health-timeout 5s
          --health-retries 5
        ports:
          - 6379:6379

    steps:
      - uses: actions/checkout@v4

      - name: Setup Go
        uses: actions/setup-go@v5
        with:
          go-version: '1.21'

      - name: Install dependencies
        working-directory: backend
        run: go mod download

      - name: Run unit tests
        working-directory: backend
        run: go test -v -short -coverprofile=coverage.txt -covermode=atomic ./...

      - name: Run integration tests
        working-directory: backend
        run: go test -v -tags=integration -coverprofile=coverage-integration.txt ./...
        env:
          DATABASE_URL: postgres://postgres:test@localhost:5432/testdb?sslmode=disable
          REDIS_URL: redis://localhost:6379

      - name: Check coverage threshold
        working-directory: backend
        run: |
          go tool cover -func=coverage.txt | grep total | awk '{print $3}' | sed 's/%//' | \
          awk '{if ($1 < 80) {print "Coverage " $1 "% is below 80% threshold"; exit 1}}'

      - name: Upload coverage to Codecov
        uses: codecov/codecov-action@v3
        with:
          files: ./backend/coverage.txt,./backend/coverage-integration.txt
          flags: backend
          fail_ci_if_error: true

  frontend-tests:
    runs-on: ubuntu-latest

    steps:
      - uses: actions/checkout@v4

      - name: Setup Node.js
        uses: actions/setup-node@v4
        with:
          node-version: '20'
          cache: 'npm'
          cache-dependency-path: frontend/package-lock.json

      - name: Install dependencies
        working-directory: frontend
        run: npm ci

      - name: Run unit tests
        working-directory: frontend
        run: npm test -- --coverage --watchAll=false

      - name: Check coverage threshold
        working-directory: frontend
        run: |
          COVERAGE=$(cat coverage/coverage-summary.json | jq '.total.lines.pct')
          if (( $(echo "$COVERAGE < 70" | bc -l) )); then
            echo "Coverage $COVERAGE% is below 70% threshold"
            exit 1
          fi

      - name: Upload coverage to Codecov
        uses: codecov/codecov-action@v3
        with:
          files: ./frontend/coverage/coverage-final.json
          flags: frontend
          fail_ci_if_error: true

  e2e-tests:
    runs-on: ubuntu-latest
    timeout-minutes: 30

    steps:
      - uses: actions/checkout@v4

      - name: Setup Node.js
        uses: actions/setup-node@v4
        with:
          node-version: '20'

      - name: Install dependencies
        working-directory: frontend
        run: npm ci

      - name: Install Playwright browsers
        working-directory: frontend
        run: npx playwright install --with-deps

      - name: Start application (Docker Compose)
        run: docker-compose up -d

      - name: Wait for app to be ready
        run: |
          timeout 60 bash -c 'until curl -f http://localhost:3000/health; do sleep 2; done'

      - name: Run Playwright tests
        working-directory: frontend
        run: npx playwright test
        env:
          BASE_URL: http://localhost:3000

      - name: Upload Playwright report
        uses: actions/upload-artifact@v3
        if: always()
        with:
          name: playwright-report
          path: frontend/playwright-report/
          retention-days: 30

      - name: Stop application
        if: always()
        run: docker-compose down
```

**Objectifs de couverture:**

```
Backend (Go):
- Unitaire: 80%+
- Integration: 70%+
- Total: 75%+

Frontend (TypeScript):
- Unitaire: 70%+
- E2E: Scénarios critiques couverts (5-10 tests)

Composants critiques:
- Services IA: 90%+
- Algorithme scoring CV: 95%+
- Middlewares sécurité: 85%+
```

---

### Étape 8: Performance Tests (Load Testing)

**Description:** Tester performance et scalabilité avec k6

**Code:**

```javascript
// backend/tests/load/cv_api_load.js
import http from 'k6/http'
import { check, sleep } from 'k6'
import { Rate } from 'k6/metrics'

const errorRate = new Rate('errors')

export const options = {
  stages: [
    { duration: '30s', target: 50 },   // Ramp up à 50 users
    { duration: '1m', target: 100 },   // Ramp up à 100 users
    { duration: '2m', target: 100 },   // Maintenir 100 users
    { duration: '30s', target: 0 },    // Ramp down
  ],
  thresholds: {
    http_req_duration: ['p(95)<200'],  // 95% requests < 200ms
    http_req_failed: ['rate<0.01'],    // < 1% errors
    errors: ['rate<0.05'],             // < 5% errors
  },
}

const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080'

export default function () {
  // Test GET /api/cv?theme=backend
  const themes = ['backend', 'frontend', 'fullstack', 'devops']
  const theme = themes[Math.floor(Math.random() * themes.length)]

  const res = http.get(`${BASE_URL}/api/cv?theme=${theme}`)

  const success = check(res, {
    'status is 200': (r) => r.status === 200,
    'response time < 200ms': (r) => r.timings.duration < 200,
    'has experiences': (r) => {
      const body = JSON.parse(r.body)
      return body.experiences && body.experiences.length > 0
    },
  })

  errorRate.add(!success)

  sleep(1)
}

export function handleSummary(data) {
  return {
    'summary.html': htmlReport(data),
    stdout: textSummary(data, { indent: ' ', enableColors: true }),
  }
}
```

```javascript
// backend/tests/load/ai_load.js
import http from 'k6/http'
import { check, sleep } from 'k6'

export const options = {
  stages: [
    { duration: '1m', target: 10 },    // Ramp up (IA coûteux)
    { duration: '2m', target: 10 },    // Maintenir
    { duration: '30s', target: 0 },    // Ramp down
  ],
  thresholds: {
    http_req_duration: ['p(95)<30000'], // 95% < 30s (génération IA lente)
    http_req_failed: ['rate<0.05'],     // < 5% errors
  },
}

const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080'

export default function () {
  const payload = JSON.stringify({
    company_name: `Company-${__VU}-${__ITER}`,
    theme: 'backend',
  })

  const params = {
    headers: {
      'Content-Type': 'application/json',
      'Cookie': `session_id=load-test-${__VU}`,
    },
  }

  const res = http.post(`${BASE_URL}/api/letters/generate`, payload, params)

  check(res, {
    'status is 200 or 429': (r) => r.status === 200 || r.status === 429,
    'has letter content': (r) => {
      if (r.status === 200) {
        const body = JSON.parse(r.body)
        return body.motivation_letter && body.anti_motivation_letter
      }
      return true
    },
  })

  sleep(5) // Respecter rate limiting
}
```

**Commandes:**

```bash
# Installer k6
# macOS: brew install k6
# Linux: sudo apt install k6

# Run load test
k6 run backend/tests/load/cv_api_load.js

# Run avec config custom
k6 run --vus 50 --duration 3m backend/tests/load/cv_api_load.js

# Export résultats JSON
k6 run --out json=results.json backend/tests/load/cv_api_load.js
```

---

## 🧪 Tests

### Structure Complète des Tests

```
maicivy/
├── backend/
│   ├── internal/
│   │   ├── api/
│   │   │   ├── cv_test.go
│   │   │   ├── letters_test.go
│   │   │   └── analytics_test.go
│   │   ├── services/
│   │   │   ├── cv_scoring_test.go
│   │   │   ├── ai_test.go
│   │   │   ├── ai_mock.go (generated)
│   │   │   └── analytics_test.go
│   │   ├── middleware/
│   │   │   ├── tracking_test.go
│   │   │   ├── ratelimit_test.go
│   │   │   └── cors_test.go
│   │   └── database/
│   │       └── integration_test.go
│   ├── testutil/
│   │   ├── fixtures.go
│   │   └── helpers.go
│   └── tests/
│       └── load/
│           ├── cv_api_load.js
│           └── ai_load.js
│
└── frontend/
    ├── components/
    │   ├── cv/
    │   │   ├── CVThemeSelector.test.tsx
    │   │   ├── ExperienceTimeline.test.tsx
    │   │   └── SkillsCloud.test.tsx
    │   ├── letters/
    │   │   ├── LetterGenerator.test.tsx
    │   │   └── LetterPreview.test.tsx
    │   └── analytics/
    │       └── RealtimeVisitors.test.tsx
    ├── lib/
    │   ├── hooks/
    │   │   └── useVisitorTracking.test.ts
    │   ├── testutil/
    │   │   └── fixtures.ts
    │   └── api.test.ts
    └── e2e/
        ├── letter-generation.spec.ts
        ├── cv-themes.spec.ts
        └── analytics.spec.ts
```

### Commandes Consolidées

```bash
# Backend
cd backend

# Tests unitaires rapides
go test -v -short ./...

# Tests integration
go test -v -tags=integration ./...

# Coverage complet
go test -v -tags=integration -coverprofile=coverage.out ./...
go tool cover -html=coverage.out

# Frontend
cd frontend

# Tests unitaires
npm test

# Coverage
npm test -- --coverage

# E2E
npx playwright test

# E2E avec UI (debug)
npx playwright test --ui

# Load testing
cd backend/tests/load
k6 run cv_api_load.js
```

---

## ⚠️ Points d'Attention

- **Isolation tests:** Chaque test DOIT être indépendant. Utiliser `SetupTest()` pour reset state
- **Mocks API IA:** TOUJOURS mocker Claude/GPT dans tests unitaires (coûts + déterminisme)
- **Testcontainers CI:** Vérifier Docker disponible dans CI (GitHub Actions: ok)
- **Fixtures réutilisables:** Ne pas dupliquer données test, centraliser dans `testutil/`
- **Coverage seuil CI:** Fail build si coverage < threshold (80% backend, 70% frontend)
- **E2E lents:** Limiter nombre tests E2E (5-10 scénarios critiques). Privilégier tests unitaires/integration
- **Rate limiting tests:** Mocker Redis pour tests rate limiting (pas dépendre timing réel)
- **Timezone tests:** Utiliser `time.UTC` dans tests pour éviter problèmes timezone
- **Parallel tests:** Go tests sont parallel par défaut. Désactiver si conflit ressources: `t.Parallel()` ou `-p 1`
- **Cleanup E2E:** TOUJOURS cleanup état après tests E2E (cookies, localStorage, DB)

**Pièges courants:**

- **Oublier build tags:** Tests integration sans `-tags=integration` ne s'exécutent pas
- **Mocks non mis à jour:** Si interface change, regénérer mocks (`go generate`)
- **Tests E2E flaky:** Ajouter `waitFor` avec timeouts généreux pour éléments async
- **Coverage faux positif:** Coverage code != coverage scénarios. Reviewer tests qualitativement

**Optimisations:**

- **Cache Go modules CI:** `actions/setup-go` avec `cache: true`
- **Parallel CI jobs:** Backend, frontend, E2E en parallèle
- **Skip tests courts:** `go test -short` skip tests integration (dev rapide)
- **Playwright sharding:** `--shard=1/4` pour distribuer E2E sur multiple runners

---

## 📚 Ressources

**Go Testing:**

- [testify documentation](https://github.com/stretchr/testify)
- [gomock guide](https://github.com/golang/mock)
- [testcontainers-go](https://golang.testcontainers.org/)
- [Table-driven tests pattern](https://dave.cheney.net/2019/05/07/prefer-table-driven-tests)

**Frontend Testing:**

- [React Testing Library](https://testing-library.com/react)
- [Jest documentation](https://jestjs.io/)
- [MSW (Mock Service Worker)](https://mswjs.io/)
- [Testing Hooks](https://react-hooks-testing-library.com/)

**E2E Testing:**

- [Playwright documentation](https://playwright.dev/)
- [Best practices E2E](https://playwright.dev/docs/best-practices)

**Load Testing:**

- [k6 documentation](https://k6.io/docs/)
- [k6 cloud](https://k6.io/cloud/)

**Coverage:**

- [Codecov](https://codecov.io/)
- [Go coverage](https://go.dev/blog/cover)

---

## ✅ Checklist de Complétion

### Tests Backend

- [ ] Tests unitaires tous services (cv_scoring, ai, analytics, pdf, scraper)
- [ ] Tests handlers HTTP (cv, letters, analytics)
- [ ] Tests middlewares (tracking, rate limiting, CORS)
- [ ] Mocks générés (mockgen) pour interfaces externes (Claude, GPT, scraper)
- [ ] Tests integration PostgreSQL (CRUD, migrations)
- [ ] Tests integration Redis (cache, rate limiting, sessions)
- [ ] Fixtures réutilisables (testutil/fixtures.go)
- [ ] Coverage >= 80%

### Tests Frontend

- [ ] Tests unitaires composants CV (CVThemeSelector, ExperienceTimeline, SkillsCloud)
- [ ] Tests unitaires composants Letters (LetterGenerator, LetterPreview, AccessGate)
- [ ] Tests unitaires composants Analytics (RealtimeVisitors, ThemeStats, Heatmap)
- [ ] Tests hooks custom (useVisitorTracking, etc.)
- [ ] Tests utilities et helpers (lib/)
- [ ] MSW configuré pour mock API
- [ ] Coverage >= 70%

### Tests E2E

- [ ] Scénario: Génération lettres (3 visites + rate limiting)
- [ ] Scénario: CV dynamique (switch thèmes)
- [ ] Scénario: Dashboard analytics (temps réel)
- [ ] Scénario: Export PDF (CV + lettres)
- [ ] Scénario: Access gate (< 3 visites)
- [ ] Tests multi-navigateurs (Chromium, Firefox, WebKit)
- [ ] Tests responsive (desktop + mobile)

### Performance

- [ ] Load test API CV (k6)
- [ ] Load test génération lettres (k6)
- [ ] Benchmarks Go (testing.B) pour algorithmes critiques
- [ ] Profiling (pprof) si goulets détectés
- [ ] Objectifs: P95 < 200ms (API), < 30s (IA)

### CI/CD

- [ ] Workflow GitHub Actions tests backend
- [ ] Workflow GitHub Actions tests frontend
- [ ] Workflow GitHub Actions E2E
- [ ] Codecov intégré (badges README)
- [ ] Fail build si coverage < threshold
- [ ] Notifications (Discord/Slack) si tests fail

### Documentation

- [ ] README tests (comment run)
- [ ] Documentation fixtures (testutil/)
- [ ] Guide writing tests (conventions)
- [ ] Troubleshooting tests CI

---

**Dernière mise à jour:** 2025-12-08
**Auteur:** Alexi
