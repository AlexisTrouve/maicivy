# 📋 NEXT STEPS - Compléter le Coverage Frontend

**Objectif :** Passer de **7% à 70%** de coverage frontend
**Temps estimé :** 40-60 heures
**Status actuel :** 48/53 tests passent (90%), mais seulement 7% de coverage

---

## 🚨 Actions Immédiates (Avant de créer de nouveaux tests)

### 1. Fixer les 5 Tests qui Échouent (2-3h)

#### Problème #1: CVThemeSelector échoue malgré Select créé
**Fichier :** `frontend/components/cv/__tests__/CVThemeSelector.test.tsx`

**Action :**
```bash
# Voir l'erreur exacte
npm test -- CVThemeSelector

# Probablement il manque d'autres composants ui/
# Vérifier les imports dans CVThemeSelector.tsx
```

**Possible solution :**
- Créer les composants manquants (button, card, etc.)
- Ou mocker les imports manquants dans le test

---

#### Problème #2: LetterGenerator & LetterPreview échouent (config Jest)
**Fichiers :**
- `frontend/components/letters/__tests__/LetterGenerator.test.tsx`
- `frontend/components/letters/__tests__/LetterPreview.test.tsx`

**Erreur probable :** `SyntaxError: Unexpected token 'export'` dans module `until-async`

**Action :**
```bash
# Vérifier si until-async est dans transformIgnorePatterns
cat frontend/jest.config.js | grep until-async

# Si absent, l'ajouter
```

**Solution :** Le fichier `jest.config.js` ligne 28 contient déjà :
```javascript
transformIgnorePatterns: [
  'node_modules/(?!(msw|@mswjs|@bundled-es-modules|until-async)/)',
  // ...
]
```

Si ça échoue encore :
1. Vérifier la version de MSW installée
2. Peut-être downgrade MSW de v2 à v1 (plus stable avec Jest)

---

#### Problème #3: Tests Flaky (timing issues)
**Fichiers :**
- `RealtimeVisitors.test.tsx` - 1 test WebSocket échoue
- `ExperienceTimeline.test.tsx` - 1 test "Présent" échoue

**Action :**
```typescript
// Dans RealtimeVisitors.test.tsx
// Augmenter le timeout waitFor
await waitFor(() => {
  expect(screen.getByText('7')).toBeInTheDocument()
}, { timeout: 3000 }) // Au lieu de 1000ms par défaut

// Dans ExperienceTimeline.test.tsx
// Vérifier le mock de données pour end_date: null
const mockExperiences = [
  {
    ...mockExperiences[0],
    endDate: null, // Ou end_date selon votre API
  }
]
```

---

## 📝 Tests à Créer (par ordre de priorité)

### Priority 1 - Hooks (10 fichiers, ~15h)

Les hooks sont **critiques** car utilisés partout dans les composants.

#### 1. `hooks/__tests__/useVisitCount.test.ts` (1.5h)
**Coverage actuel :** 0%
**Impact :** HAUTE (utilisé dans AccessGate)

```typescript
import { renderHook, waitFor } from '@testing-library/react'
import { useVisitCount } from '../useVisitCount'
import { server } from '@/__mocks__/server'
import { rest } from 'msw'

describe('useVisitCount', () => {
  beforeAll(() => server.listen())
  afterEach(() => server.resetHandlers())
  afterAll(() => server.close())

  it('should fetch visit count from API', async () => {
    const { result } = renderHook(() => useVisitCount())

    await waitFor(() => {
      expect(result.current.count).toBe(3)
      expect(result.current.hasAccess).toBe(true)
    })
  })

  it('should increment count locally', () => {
    const { result } = renderHook(() => useVisitCount())

    act(() => {
      result.current.increment()
    })

    expect(result.current.count).toBe(4)
  })

  it('should persist count in localStorage', () => {
    const { result } = renderHook(() => useVisitCount())

    expect(localStorage.getItem('visitCount')).toBe('3')
  })

  // 5-7 tests au total
})
```

**Tests à créer :**
- Fetch initial
- Increment local
- Persistence localStorage
- Error handling
- hasAccess logic (< 3 vs >= 3)

---

#### 2. `hooks/__tests__/useAnalyticsWebSocket.test.ts` (2h)
**Coverage actuel :** 0%
**Impact :** HAUTE (utilisé dans RealtimeVisitors)

```typescript
import { renderHook, waitFor } from '@testing-library/react'
import { useAnalyticsWebSocket } from '../useAnalyticsWebSocket'

describe('useAnalyticsWebSocket', () => {
  it('should connect to WebSocket', () => {
    const { result } = renderHook(() => useAnalyticsWebSocket())

    expect(result.current.connected).toBe(true)
  })

  it('should receive messages', async () => {
    const { result } = renderHook(() => useAnalyticsWebSocket())

    // Simuler message WebSocket
    const mockEvent = new MessageEvent('message', {
      data: JSON.stringify({ visitors: 7 })
    })

    global.WebSocket.onmessage(mockEvent)

    await waitFor(() => {
      expect(result.current.data.visitors).toBe(7)
    })
  })

  it('should reconnect on disconnect', async () => {
    const { result } = renderHook(() => useAnalyticsWebSocket())

    // Simuler déconnexion
    global.WebSocket.close()

    await waitFor(() => {
      expect(result.current.reconnecting).toBe(true)
    })
  })

  it('should cleanup on unmount', () => {
    const { unmount } = renderHook(() => useAnalyticsWebSocket())

    unmount()

    expect(global.WebSocket.close).toHaveBeenCalled()
  })

  // 8-10 tests au total
})
```

**Tests à créer :**
- Connection
- Receiving messages
- Reconnection logic
- Error handling
- Cleanup

---

#### 3-10. Autres Hooks (12h restants)

**Fichiers à créer :**
- `useProfileDetection.test.ts` (1.5h)
- `useAnalyticsData.test.ts` (1.5h)
- `useTheme.test.ts` (1h)
- `useTimelineData.test.ts` (1.5h)
- `useTimelineScroll.test.ts` (2h)
- `useGitHubSync.test.ts` (2h)
- `use3DSupport.test.ts` (1.5h)
- `use3DControls.test.ts` (1.5h)

**Structure commune :**
```typescript
import { renderHook, waitFor, act } from '@testing-library/react'
import { useXXX } from '../useXXX'

describe('useXXX', () => {
  beforeAll(() => server.listen())
  afterEach(() => server.resetHandlers())
  afterAll(() => server.close())

  it('should [feature]', async () => {
    const { result } = renderHook(() => useXXX())

    // Assertions
  })

  // 5-8 tests par hook
})
```

---

### Priority 2 - Lib Utilities (7 fichiers, ~8h)

#### 1. `lib/__tests__/api.test.ts` (2h)
**Coverage actuel :** 0%
**Impact :** CRITIQUE (utilisé partout)

```typescript
import { apiClient } from '../api'
import { server } from '@/__mocks__/server'
import { rest } from 'msw'

describe('API Client', () => {
  beforeAll(() => server.listen())
  afterEach(() => server.resetHandlers())
  afterAll(() => server.close())

  describe('GET requests', () => {
    it('should fetch data successfully', async () => {
      const data = await apiClient.get('/api/cv')

      expect(data.experiences).toBeDefined()
      expect(data.skills).toBeDefined()
    })

    it('should handle 404 errors', async () => {
      server.use(
        rest.get('/api/cv', (req, res, ctx) => {
          return res(ctx.status(404))
        })
      )

      await expect(apiClient.get('/api/cv')).rejects.toThrow('Not Found')
    })

    it('should retry on network error', async () => {
      let attempts = 0

      server.use(
        rest.get('/api/cv', (req, res, ctx) => {
          attempts++
          if (attempts < 3) {
            return res.networkError('Network error')
          }
          return res(ctx.json({ data: 'success' }))
        })
      )

      const data = await apiClient.get('/api/cv')
      expect(attempts).toBe(3)
      expect(data.data).toBe('success')
    })
  })

  describe('POST requests', () => {
    it('should send data successfully', async () => {
      const response = await apiClient.post('/api/letters/generate', {
        company: 'Google'
      })

      expect(response.jobId).toBeDefined()
    })

    it('should include auth headers', async () => {
      // Test headers
    })
  })

  // 12-15 tests au total
})
```

**Tests à créer :**
- GET/POST/PUT/DELETE requests
- Error handling (400, 401, 403, 404, 500)
- Retry logic
- Request interceptors
- Response transformations
- Headers (auth, content-type)

---

#### 2-7. Autres Utilities (6h restants)

**Fichiers à créer :**
- `analytics-api.test.ts` (1.5h) - Endpoints analytics
- `validations.test.ts` (1.5h) - Schemas Zod
- `utils.test.ts` (1h) - Helpers (cn, formatDate, etc.)
- `3d-utils.test.ts` (1h) - Utilities 3D
- `lazy-load.test.ts` (1h) - Lazy loading

**Structure :**
```typescript
import { functionToTest } from '../file'

describe('functionToTest', () => {
  it('should [expected behavior]', () => {
    const result = functionToTest(input)
    expect(result).toBe(expected)
  })

  // 3-5 tests par fonction
  // Focus sur edge cases
})
```

---

### Priority 3 - Composants Manquants (15 fichiers, ~20h)

#### CV Components (5 fichiers, 6h)
- `ProjectsGrid.test.tsx` (1.5h) - Grid layout, filtrage
- `ExportPDFButton.test.tsx` (1h) - Click, download, loading
- `CVSkeleton.test.tsx` (30min) - Loading skeleton

#### Analytics Components (4 fichiers, 5h)
- `ThemeStats.test.tsx` (1.5h) - Charts (déjà créé mais échoue)
- `DateFilter.test.tsx` (1h) - Date picker, validation
- `Heatmap.test.tsx` (1.5h) - Heatmap interactions
- `LettersGenerated.test.tsx` (1h) - Stats lettres
- `StatsOverview.test.tsx` (1h) - Cards metrics

#### GitHub Components (3 fichiers, 4h)
- `GitHubConnect.test.tsx` (1.5h) - OAuth flow
- `GitHubStatus.test.tsx` (1h) - Sync status
- `RepoList.test.tsx` (1.5h) - Liste repos, filtrage

#### Timeline Components (6 fichiers, 8h)
- `TimelineView.test.tsx` (2h)
- `TimelineItem.test.tsx` (1.5h)
- `TimelineFilters.test.tsx` (1h)
- `TimelineMilestones.test.tsx` (1h)
- `TimelineModal.test.tsx` (1.5h)
- `TimelineNavigation.test.tsx` (1h)

#### Layout & Shared (3 fichiers, 2h)
- `Header.test.tsx` (1h)
- `Footer.test.tsx` (30min)
- `LoadingSpinner.test.tsx` (30min)

---

### Priority 4 - Pages (7 fichiers, ~5h)

**Fichiers à créer :**
- `app/__tests__/page.test.tsx` (1h) - Homepage
- `app/cv/__tests__/page.test.tsx` (1h) - Page CV
- `app/letters/__tests__/page.test.tsx` (1h) - Page Lettres
- `app/analytics/__tests__/page.test.tsx` (1h) - Page Analytics
- `app/__tests__/error.test.tsx` (30min) - Error page
- `app/__tests__/loading.test.tsx` (30min) - Loading page
- `app/__tests__/not-found.test.tsx` (30min) - 404 page

**Structure page :**
```typescript
import { render, screen } from '@testing-library/react'
import Page from '../page'

describe('Page', () => {
  it('should render page content', () => {
    render(<Page />)

    expect(screen.getByRole('heading', { name: /titre/i })).toBeInTheDocument()
  })

  it('should fetch data on mount', async () => {
    render(<Page />)

    await waitFor(() => {
      expect(screen.getByText('Data loaded')).toBeInTheDocument()
    })
  })
})
```

---

## 🎯 Plan d'Exécution Recommandé

### Semaine 1 (20h)
- ✅ Fixer les 5 tests qui échouent (3h)
- ✅ Hooks Priority 1: useVisitCount, useAnalyticsWebSocket, useProfileDetection (5h)
- ✅ Lib utilities: api.test.ts, validations.test.ts, analytics-api.test.ts (5h)
- ✅ CV Components: ProjectsGrid, ExportPDFButton (3h)
- ✅ Analytics: DateFilter, LettersGenerated, StatsOverview (3h)
- ✅ Pages: Homepage, CV page (2h)

**Coverage estimé après Semaine 1 :** ~35-40%

### Semaine 2 (20h)
- ✅ Hooks restants (7h)
- ✅ GitHub Components (4h)
- ✅ Timeline Components (8h)
- ✅ Layout & Shared (2h)

**Coverage estimé après Semaine 2 :** ~60-65%

### Semaine 3 (10-15h)
- ✅ Compléter les tests manquants
- ✅ Fixer les tests flaky
- ✅ Atteindre 70% coverage
- ✅ Documentation

**Coverage final :** 70%+ ✅

---

## 🛠️ Outils et Commandes Utiles

### Lancer les tests
```bash
cd frontend

# Tous les tests
npm test

# Tests spécifiques
npm test -- useVisitCount
npm test -- components/cv
npm test -- hooks

# Avec coverage
npm run test:coverage

# Mode watch (développement)
npm run test:watch

# Voir les tests disponibles
npm test -- --listTests
```

### Coverage par catégorie
```bash
# Voir coverage d'un dossier spécifique
npm test -- --coverage hooks/
npm test -- --coverage components/cv/
npm test -- --coverage lib/

# Coverage détaillé
npm run test:coverage
# Ouvrir coverage/lcov-report/index.html dans le browser
```

### Debug un test qui échoue
```bash
# Verbose output
npm test -- --verbose ComponentName

# Voir les assertions qui échouent
npm test -- --no-coverage ComponentName

# Detect open handles (pour tests qui ne terminent pas)
npm test -- --detectOpenHandles
```

---

## 📚 Ressources et Exemples

### Tests Déjà Créés (Utilisez-les comme référence !)

**Excellents exemples :**
- `components/letters/__tests__/AccessGate.test.tsx` (100% coverage)
- `components/cv/__tests__/SkillsCloud.test.tsx` (100% coverage)
- `components/analytics/__tests__/RealtimeVisitors.test.tsx` (96% coverage)

**Structure à réutiliser :**
```typescript
import { render, screen, fireEvent, waitFor } from '@testing-library/react'
import { server } from '@/__mocks__/server'
import { rest } from 'msw'
import { mockData } from '@/lib/testutil/fixtures'

describe('ComponentName', () => {
  beforeAll(() => server.listen())
  afterEach(() => server.resetHandlers())
  afterAll(() => server.close())

  it('should render correctly', () => {
    render(<ComponentName />)
    expect(screen.getByText('Expected text')).toBeInTheDocument()
  })

  it('should handle user interaction', async () => {
    render(<ComponentName />)

    const button = screen.getByRole('button', { name: /click/i })
    fireEvent.click(button)

    await waitFor(() => {
      expect(screen.getByText('Result')).toBeInTheDocument()
    })
  })

  it('should handle API error', async () => {
    server.use(
      rest.get('/api/endpoint', (req, res, ctx) => {
        return res(ctx.status(500))
      })
    )

    render(<ComponentName />)

    await waitFor(() => {
      expect(screen.getByText(/error/i)).toBeInTheDocument()
    })
  })
})
```

### Fixtures Disponibles

Utilisez les fixtures dans `lib/testutil/fixtures.ts` :
```typescript
import {
  mockThemes,
  mockExperiences,
  mockSkills,
  mockProjects,
  mockVisitorStatus,
  mockGeneratedLetter,
  mockAnalyticsData
} from '@/lib/testutil/fixtures'
```

### MSW Handlers Disponibles

Les handlers API sont déjà configurés dans `__mocks__/handlers.ts` :
- `/api/cv` - CV data
- `/api/cv/themes` - Themes
- `/api/letters/generate` - Generate letters
- `/api/visitor/status` - Visitor status
- `/api/analytics/stats` - Analytics

Vous pouvez les override dans vos tests :
```typescript
server.use(
  rest.get('/api/cv', (req, res, ctx) => {
    return res(ctx.json(customData))
  })
)
```

---

## ✅ Checklist de Progression

Cochez au fur et à mesure :

### Tests qui Échouent
- [ ] CVThemeSelector fixed
- [ ] LetterGenerator fixed
- [ ] LetterPreview fixed
- [ ] ThemeStats fixed
- [ ] RealtimeVisitors flaky test fixed
- [ ] ExperienceTimeline flaky test fixed

### Hooks (10 fichiers)
- [ ] useVisitCount.test.ts
- [ ] useAnalyticsWebSocket.test.ts
- [ ] useProfileDetection.test.ts
- [ ] useAnalyticsData.test.ts
- [ ] useTheme.test.ts
- [ ] useTimelineData.test.ts
- [ ] useTimelineScroll.test.ts
- [ ] useGitHubSync.test.ts
- [ ] use3DSupport.test.ts
- [ ] use3DControls.test.ts

### Lib Utilities (7 fichiers)
- [ ] api.test.ts
- [ ] analytics-api.test.ts
- [ ] validations.test.ts
- [ ] utils.test.ts
- [ ] 3d-utils.test.ts
- [ ] lazy-load.test.ts

### Composants CV (3 fichiers)
- [ ] ProjectsGrid.test.tsx
- [ ] ExportPDFButton.test.tsx
- [ ] CVSkeleton.test.tsx

### Composants Analytics (5 fichiers)
- [ ] ThemeStats.test.tsx (fix)
- [ ] DateFilter.test.tsx
- [ ] Heatmap.test.tsx
- [ ] LettersGenerated.test.tsx
- [ ] StatsOverview.test.tsx

### Composants GitHub (3 fichiers)
- [ ] GitHubConnect.test.tsx
- [ ] GitHubStatus.test.tsx
- [ ] RepoList.test.tsx

### Composants Timeline (6 fichiers)
- [ ] TimelineView.test.tsx
- [ ] TimelineItem.test.tsx
- [ ] TimelineFilters.test.tsx
- [ ] TimelineMilestones.test.tsx
- [ ] TimelineModal.test.tsx
- [ ] TimelineNavigation.test.tsx

### Layout & Shared (3 fichiers)
- [ ] Header.test.tsx
- [ ] Footer.test.tsx
- [ ] LoadingSpinner.test.tsx

### Pages (7 fichiers)
- [ ] app/page.test.tsx
- [ ] app/cv/page.test.tsx
- [ ] app/letters/page.test.tsx
- [ ] app/analytics/page.test.tsx
- [ ] app/error.test.tsx
- [ ] app/loading.test.tsx
- [ ] app/not-found.test.tsx

### Milestones Coverage
- [ ] 20% coverage
- [ ] 35% coverage
- [ ] 50% coverage
- [ ] 65% coverage
- [ ] **70% coverage** ✅ OBJECTIF

---

## 🚀 Quick Start

Pour commencer **maintenant** :

```bash
# 1. Fixer les tests qui échouent
cd frontend
npm test -- --verbose

# 2. Créer votre premier nouveau test
touch hooks/__tests__/useVisitCount.test.ts

# 3. Copier la structure depuis AccessGate.test.tsx
cp components/letters/__tests__/AccessGate.test.tsx hooks/__tests__/useVisitCount.test.ts

# 4. Adapter le code

# 5. Lancer le test
npm test -- useVisitCount

# 6. Vérifier le coverage
npm run test:coverage
```

---

## 📞 Support

**Documentation :**
- `TESTING_REPORT_FINAL.md` - Rapport complet
- `frontend/tests/README.md` - Guide de testing
- `docs/implementation/16_TESTING_STRATEGY.md` - Stratégie globale

**Problèmes courants :**
- Tests timeout → Augmenter timeout dans waitFor
- Module not found → Vérifier les imports et jest.config.js
- Tests flaky → Ajouter waitFor, vérifier les mocks
- Coverage bas → Ajouter edge cases, error handling

---

**Bonne chance ! 🚀**

**Temps total estimé :** 40-60 heures
**Objectif :** 70% coverage ✅
**Status actuel :** 7% → Gap de 63%
