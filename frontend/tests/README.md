# Frontend Tests

Ce dossier contient tous les tests pour le frontend de maicivy.

## Structure

```
tests/
├── e2e/                    # Tests end-to-end avec Playwright
│   └── example.spec.ts
├── README.md               # Ce fichier
```

Les tests unitaires sont colocalisés avec les composants dans des dossiers `__tests__/`.

## Tests Unitaires (Jest + React Testing Library)

### Commandes

```bash
# Lancer tous les tests
npm run test

# Mode watch (relance automatiquement)
npm run test:watch

# Avec coverage
npm run test:coverage
```

### Écrire un test unitaire

Créez un fichier `__tests__/ComponentName.test.tsx` à côté de votre composant:

```tsx
import { render, screen } from '@testing-library/react'
import { MyComponent } from '../MyComponent'

describe('MyComponent', () => {
  it('should render correctly', () => {
    render(<MyComponent />)
    expect(screen.getByText('Hello')).toBeInTheDocument()
  })
})
```

### Utiliser les fixtures

Importez les données de test depuis `lib/testutil/fixtures.ts`:

```tsx
import { mockCVData, mockLetterRequest } from '@/lib/testutil/fixtures'

describe('CVComponent', () => {
  it('should render CV data', () => {
    render(<CVComponent data={mockCVData} />)
    expect(screen.getByText(mockCVData.name)).toBeInTheDocument()
  })
})
```

### Mock Service Worker (MSW)

Les mocks API sont configurés automatiquement via `__mocks__/handlers.ts` et `__mocks__/server.ts`.

Pour ajouter un nouveau mock:

```tsx
// Dans __mocks__/handlers.ts
http.get('/api/new-endpoint', () => {
  return HttpResponse.json({ data: 'test' })
})
```

## Tests E2E (Playwright)

### Commandes

```bash
# Installer les browsers (à faire une fois)
npx playwright install

# Lancer les tests E2E
npm run test:e2e

# Mode UI interactif
npm run test:e2e:ui

# Mode headed (voir le browser)
npm run test:e2e:headed

# Mode debug
npm run test:e2e:debug
```

### Écrire un test E2E

Créez un fichier `tests/e2e/feature.spec.ts`:

```tsx
import { test, expect } from '@playwright/test'

test.describe('Feature Name', () => {
  test('should do something', async ({ page }) => {
    await page.goto('/feature')
    await expect(page.locator('h1')).toHaveText('Feature Title')
  })
})
```

## Configuration

### Jest

Configuration dans `jest.config.js`:
- Utilise `jest-environment-jsdom` pour simuler le DOM
- Supporte les modules CSS et images
- Coverage threshold: 70%
- Setup dans `jest.setup.js` (MSW, mocks globaux)

### Playwright

Configuration dans `playwright.config.ts`:
- Tests sur Chrome, Firefox, Safari
- Tests mobile (iPhone, Pixel)
- Capture screenshots sur échec
- Server de dev lancé automatiquement

## Bonnes Pratiques

1. **Tests unitaires**
   - Un fichier de test par composant
   - Tester le comportement, pas l'implémentation
   - Utiliser `screen.getByRole` plutôt que `getByTestId`
   - Mocker les appels API avec MSW

2. **Tests E2E**
   - Tester les parcours utilisateurs complets
   - Éviter les sélecteurs CSS fragiles
   - Utiliser `data-testid` si nécessaire
   - Vérifier les états de chargement et d'erreur

3. **Coverage**
   - Viser 70%+ de couverture
   - Prioriser les flux critiques
   - Ne pas oublier les cas d'erreur

## Debugging

### Jest

```bash
# Lancer un seul fichier de test
npm run test -- path/to/test.tsx

# Lancer en mode debug
node --inspect-brk node_modules/.bin/jest --runInBand
```

### Playwright

```bash
# Mode debug avec UI
npm run test:e2e:debug

# Voir le rapport HTML
npx playwright show-report
```

## CI/CD

Les tests sont lancés automatiquement dans la pipeline CI/CD:
- Tests unitaires sur chaque push
- Tests E2E sur les pull requests
- Coverage report publié sur chaque merge

## Ressources

- [Jest Documentation](https://jestjs.io/)
- [React Testing Library](https://testing-library.com/react)
- [Playwright Documentation](https://playwright.dev/)
- [MSW Documentation](https://mswjs.io/)
