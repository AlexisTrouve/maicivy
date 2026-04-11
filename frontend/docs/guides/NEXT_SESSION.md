# Prompt Successeur - Session Frontend Tests

## 📋 Contexte du Projet

**Projet :** maicivy - CV interactif avec IA
**Stack :** Next.js 14 (App Router), TypeScript, Tailwind CSS
**Backend :** Go + Fiber
**État :** Phase de correction des tests frontend

---

## ✅ Ce qui a été accompli (Session précédente)

### Tests corrigés : 231/238 tests passants (97.1%)

#### Fichiers complètement fonctionnels (14/15) :
1. ✅ `hooks/__tests__/useProfileDetection.test.ts` - 11/11 tests
2. ✅ `hooks/__tests__/useAnalyticsWebSocket.test.ts` - 8/8 tests
3. ✅ `components/__tests__/TimelineView.test.tsx` - 19/19 tests
4. ✅ `hooks/__tests__/useTimelineData.test.ts` - 8/8 tests
5. ✅ `components/analytics/__tests__/RealtimeVisitors.test.tsx` - 12/12 tests
6. ✅ `components/__tests__/TimelineModal.test.tsx` - 30/30 tests
7. ✅ `hooks/__tests__/use3DSupport.test.ts` - 11/11 tests
8. ✅ `lib/__tests__/lazy-load.test.ts` - 47/47 tests
9. ✅ `hooks/__tests__/useTimelineScroll.test.ts` - 13/13 tests
10. ✅ `components/analytics/__tests__/LettersGenerated.test.tsx` - 26/26 tests
11. ✅ `components/letters/__tests__/LetterGenerator.test.tsx` - 13/13 tests
12. ✅ `hooks/__tests__/useGitHubSync.test.ts` - 10/10 tests
13. ✅ `components/__tests__/GitHubStatus.test.tsx` - 13/13 tests
14. ✅ `components/__tests__/GitHubConnect.test.tsx` - 10/10 tests

#### Fichier partiellement fonctionnel (1/15) :
- ⚠️ `components/cv/__tests__/ExportPDFButton.test.tsx` - 7/14 tests (50%)

---

## 🔧 Solutions techniques appliquées

### 1. Remplacement de MSW par Jest Mocks
**Problème :** MSW (Mock Service Worker) ne fonctionne pas bien avec `undici/fetch` utilisé par Jest en Node.js

**Solution appliquée :**
```typescript
// Avant (MSW)
server.use(
  rest.get('*/api/v1/profile/current', (req, res, ctx) => {
    return res(ctx.json({ data }))
  })
)

// Après (Jest mocks)
jest.mock('@/lib/api', () => ({
  profileApi: {
    getCurrent: jest.fn(),
    detect: jest.fn(),
  }
}))

const mockedProfileApi = profileApi as jest.Mocked<typeof profileApi>
mockedProfileApi.getCurrent.mockResolvedValueOnce({ data })
```

### 2. Mock des icônes lucide-react
**Problème :** Les composants lucide-react causent des erreurs de rendu

**Solution :**
```typescript
jest.mock('lucide-react', () => ({
  Activity: () => <div data-testid="activity-icon">Activity Icon</div>,
  Download: () => <div data-testid="download-icon">Download</div>,
  // ... autres icônes
}))
```

### 3. Assertions avec éléments multiples
**Problème :** `getByText('3')` échoue quand plusieurs éléments contiennent '3'

**Solution :**
```typescript
// Avant
expect(screen.getByText('3')).toBeInTheDocument()

// Après
const threeElements = screen.getAllByText('3')
expect(threeElements.length).toBeGreaterThanOrEqual(1)
```

### 4. Gestion des timers React
**Problème :** Warnings "not wrapped in act()"

**Solution :**
```typescript
afterEach(() => {
  act(() => {
    jest.runOnlyPendingTimers()
  })
  jest.useRealTimers()
})
```

### 5. Installation Next.js SWC
**Problème :** Binary manquant pour Next.js

**Solution :**
```bash
npm install --save-dev @next/swc-linux-x64-gnu --legacy-peer-deps
```

---

## 🚨 Problème restant à résoudre

### ExportPDFButton.test.tsx (7/14 tests échouent)

**Symptômes :**
- Les tests avec interactions utilisateur (clicks, async) fonctionnent ✅
- Les tests de rendu simple échouent ❌
- Erreur : Le composant ne se rend pas, body HTML vide

**Tests échouants :**
1. should render button with correct text
2. should render Download icon when not loading
3. should use fallback filename if Content-Disposition missing
4. should handle API error gracefully
5. should create and trigger download link correctly
6. should render with gradient styling
7. should show Loader2 icon when loading

**Hypothèse :**
Problème de configuration Jest/module mocking qui affecte uniquement les tests synchrones sans interactions.

**Fichiers concernés :**
- `/mnt/c/Users/alexi/Documents/projects/maicivy/frontend/components/cv/__tests__/ExportPDFButton.test.tsx`
- `/mnt/c/Users/alexi/Documents/projects/maicivy/frontend/components/cv/ExportPDFButton.tsx`

**Mocks actuels :**
```typescript
jest.mock('lucide-react', () => ({
  Download: () => <div data-testid="download-icon">Download</div>,
  Loader2: () => <div data-testid="loader-icon">Loading</div>,
}))

jest.mock('@/components/ui/button', () => ({
  Button: ({ children, ...props }: any) => (
    <button {...props}>{children}</button>
  ),
}))
```

---

## 🎯 Prochaines étapes recommandées

### Priorité 1 : Résoudre ExportPDFButton (Temps estimé : 2-3h)

**Approches à tester :**

1. **Vérifier les imports du composant**
   ```bash
   # Lire le composant ExportPDFButton
   cat components/cv/ExportPDFButton.tsx
   ```

2. **Tester avec différentes stratégies de mocking**
   - Essayer `jest.mock` au niveau du fichier vs `beforeEach`
   - Vérifier si d'autres dépendances manquent (framer-motion, etc.)
   - Tester avec `jest.resetModules()` avant chaque test

3. **Comparer avec les tests fonctionnels**
   - Analyser la différence entre tests qui passent et qui échouent
   - Les tests fonctionnels utilisent `fireEvent` et `waitFor` - peut-être nécessaire même pour les tests simples ?

4. **Vérifier jest.config.js**
   ```typescript
   // Peut-être besoin d'ajuster :
   moduleNameMapper: {
     '^@/components/(.*)$': '<rootDir>/components/$1',
   }
   ```

### Priorité 2 : Optimisation des tests (Optionnel)

1. **Réduire les warnings React act()**
   - Certains tests ont encore des warnings même s'ils passent
   - Particulièrement dans `useTimelineData.test.ts`

2. **Améliorer la couverture de code**
   ```bash
   npm test -- --coverage
   ```

3. **Paralléliser l'exécution des tests**
   ```bash
   # Tester avec plus de workers
   npm test -- --maxWorkers=4
   ```

### Priorité 3 : Documentation

1. **Créer un guide de test**
   - Documenter les patterns de mocking utilisés
   - Exemples pour les nouveaux tests

2. **CI/CD Integration**
   - Ajouter les tests dans GitHub Actions
   - Configurer le seuil de couverture minimum

---

## 📝 Commandes utiles

```bash
# Lancer tous les tests
npm test

# Tests spécifiques
npm test -- components/cv/__tests__/ExportPDFButton.test.tsx --no-coverage

# Tests avec coverage
npm test -- --coverage

# Tests en mode watch
npm test -- --watch

# Lancer un seul test
npm test -- --testNamePattern="should render button with correct text"

# Debug mode
node --inspect-brk node_modules/.bin/jest --runInBand
```

---

## 🔍 Fichiers clés à connaître

### Configuration
- `jest.config.js` - Configuration Jest
- `jest.setup.js` - Setup global des tests (polyfills, mocks globaux)
- `__mocks__/` - Mocks globaux (server MSW, webAPIs)

### Utilitaires de test
- `lib/testutil/fixtures.tsx` - Données de test mockées
- `__mocks__/handlers.ts` - Handlers MSW (si réactivés)

### Tests problématiques
- `components/cv/__tests__/ExportPDFButton.test.tsx` - 50% échec

---

## 💡 Conseils pour la session suivante

1. **Commencer par lire ce fichier** pour comprendre le contexte
2. **Ne pas réintroduire MSW** - Les Jest mocks fonctionnent mieux
3. **Toujours mocker lucide-react** dans les nouveaux tests
4. **Utiliser `getAllByText()` pour les éléments en double**
5. **Wrapper les timers dans `act()`** si warnings React
6. **Consulter les tests fonctionnels** comme référence pour les patterns

---

## 📊 Métriques du projet

- **Tests totaux :** 238
- **Tests passants :** 231 (97.1%)
- **Tests échouants :** 7 (2.9%)
- **Fichiers de tests :** 49 (incluant node_modules)
- **Fichiers du projet testés :** 15
- **Couverture estimée :** ~70-80% (à vérifier)

---

## 🤝 Collaboration

**Dernier commit :**
- Hash: `48f2fcc`
- Message: "fix: resolve 231 out of 238 frontend tests (97.1% success rate)"
- Branche: `main`
- Remote: `https://git.etheryale.com/StillHammer/maicivy.git`

**État Git :**
- ✅ Tous les changements committés
- ✅ Poussés vers origin/main
- ✅ Working tree propre

---

## ❓ Questions à clarifier

1. **ExportPDFButton :** Est-ce critique de corriger les 7 tests restants ou 50% suffit ?
2. **Couverture de code :** Quel est le seuil minimum acceptable ?
3. **CI/CD :** Les tests doivent-ils bloquer le merge des PR ?
4. **Performance :** Les tests prennent ~20-30s par fichier, est-ce acceptable ?

---

**Créé le :** 2025-12-11
**Auteur :** Claude (Session de correction des tests)
**Prochaine action recommandée :** Corriger les 7 tests restants d'ExportPDFButton

---

Bonne chance pour la suite ! 🚀
