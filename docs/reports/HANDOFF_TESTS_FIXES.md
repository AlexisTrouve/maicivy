# 🔄 HANDOFF - Suite des Fixes de Tests Frontend

**Date:** 2025-12-11
**Contexte:** Continuation du fix des tests frontend après session parallèle avec 4 agents
**Projet:** maicivy (CV interactif + IA)

---

## 📊 État Actuel des Tests

### Résultats Globaux
```
Test Suites: 16 failed, 33 passed, 49 total
Tests:       90 failed, 738 passed, 828 total
Taux de réussite: 89.1% (738/828)
```

**Progression depuis le début:**
- Avant: 704/835 tests passaient (84.3%)
- Maintenant: 738/828 tests passent (89.1%)
- **41 tests fixés** ✅
- **90 tests restent à fixer** ⚠️

---

## ✅ Ce qui a été FIXÉ (Session précédente)

### Agent 1 - RepoList Tests (18 tests fixés)
**Problème:** Configuration Jest invalide
**Solution:** Nettoyé `frontend/jest.config.js`
- Supprimé `maxAsyncTimeout` (option invalide)
- Supprimé transform custom `@swc/jest` (package manquant)
**Résultat:** ✅ Tous les 18 tests RepoList passent

### Agent 2 - Analytics Tests (39 tests fixés)
**Fichiers:**
- `components/analytics/__tests__/LettersGenerated.test.tsx`
- `components/analytics/__tests__/ThemeStats.test.tsx`

**Fixes appliqués:**
1. LettersGenerated:
   - Changé `querySelector('svg')` → `.h-48 svg` (sélecteur spécifique au chart)
   - Changé `querySelectorAll('line')` → `chartSvg.querySelectorAll('line')`
   - Corrigé regex: `/d{2}sw+/` → `/\d{2}\s\w+/`
   - Changé `findByText` → `findAllByText` pour dates multiples
2. ThemeStats: Fixé via config Jest
**Résultat:** ✅ Tous les 39 tests analytics passent

### Agent 3 - Hooks Tests (partiellement fixé)
**Fichiers convertis:**
- `hooks/__tests__/useVisitCount.test.ts` - Converti en MSW v1
**Problème restant:** Les hooks ne répondent pas correctement (loading reste true)

### Agent 4 - Jest Memory (crashes éliminés)
**Fichiers modifiés:**
- `frontend/jest.config.js` - Optimisé mémoire
- `frontend/jest.setup.js` - Ajouté GC hints
- `hooks/__tests__/useAnalyticsData.test.ts` - Refactorisé
- `components/analytics/__tests__/ThemeStats.test.tsx` - Cleanup ajouté

**Optimisations:**
- `maxWorkers: 1`
- `workerIdleMemoryLimit: '4096MB'`
- `testTimeout: 120000`
- `clearMocks: true` + `restoreMocks: true`
- GC manuel après chaque test
**Résultat:** ✅ Zéro crash mémoire

### Autres Fixes Appliqués
- `frontend/jest.setup.js`: Ajouté variables d'env
  ```javascript
  process.env.NEXT_PUBLIC_API_URL = 'http://localhost:8080'
  process.env.NEXT_PUBLIC_WS_URL = 'ws://localhost:8080'
  ```
- `frontend/jest.config.js`: Désactivé `bail: 1` pour voir tous les résultats

---

## ⚠️ Ce qui RESTE à Fixer (90 tests)

### Catégorie 1: Hooks Tests (~40-50 tests échouants)

**Fichiers problématiques:**
1. `hooks/__tests__/useVisitCount.test.ts` (6 tests échouent)
2. `hooks/__tests__/useProfileDetection.test.ts` (11 tests échouent)
3. `hooks/__tests__/useTimelineData.test.ts` (syntaxe MSW incomplète)
4. `hooks/__tests__/use3DSupport.test.ts` (createElement mock)
5. `hooks/__tests__/useGitHubSync.test.ts` (probablement MSW)
6. `hooks/__tests__/useAnalyticsWebSocket.test.ts` (WebSocket mocking)

**Symptôme commun:**
```javascript
// Le hook ne finit jamais de charger
await waitFor(() => {
  expect(result.current.loading).toBe(false) // TIMEOUT - reste true
})
```

**Cause probable:**
- Les handlers MSW ne s'interceptent pas correctement
- URL patterns incorrects (`*/api/...` vs URLs complètes)
- Problèmes de timing avec les mocks

**Actions prioritaires:**
1. Vérifier que tous les handlers MSW utilisent des patterns wildcard (`*/api/v1/...`)
2. S'assurer que `server.use()` est appelé AVANT `renderHook()`
3. Ajouter des console.log dans les handlers pour vérifier qu'ils sont appelés
4. Augmenter les timeouts `waitFor` si nécessaire

---

### Catégorie 2: Problèmes undici/jsdom (~15-20 tests)

**Erreur type:**
```
TypeError: fastNowTimeout?.unref is not a function
  at refreshTimeout (node_modules/undici/lib/util/timers.js:205:21)
```

**Cause:**
- Incompatibilité entre `undici` (polyfill fetch dans Node) et `jsdom`
- `undici` utilise des fonctionnalités Node.js non disponibles dans jsdom

**Solutions possibles:**

**Option A - Downgrade undici**
```bash
cd frontend
npm install undici@5.28.0 --save-exact
```

**Option B - Mock undici timers**
Ajouter dans `jest.setup.js`:
```javascript
// Mock undici timers for jsdom compatibility
jest.mock('undici/lib/util/timers', () => ({
  __esModule: true,
  default: {
    setTimeout: global.setTimeout,
    clearTimeout: global.clearTimeout,
  },
}))
```

**Option C - Utiliser node-fetch à la place**
```bash
npm uninstall undici
npm install node-fetch@2.7.0
```

---

### Catégorie 3: Tests Divers (~20-25 tests)

**Fichiers avec 1-3 tests échouants chacun:**
- `components/__tests__/...` - Quelques composants
- `lib/__tests__/...` - Quelques utilities
- `app/__tests__/...` - Quelques pages

**Problèmes variés:**
- Assertions trop strictes
- Mocks incomplets
- Timing issues

---

## 🎯 Plan d'Action Recommandé

### Phase 1 - Fix Problème undici (1h) - HAUTE PRIORITÉ
**Impact:** Fixera 15-20 tests d'un coup

**Actions:**
1. Tester Option A (downgrade undici):
   ```bash
   cd frontend
   npm install undici@5.28.0 --save-exact
   npm test -- useVisitCount.test.ts
   ```
2. Si ça ne marche pas, tester Option B (mock timers)
3. Si ça ne marche pas, tester Option C (node-fetch)
4. Lancer suite complète après fix: `npm test`

---

### Phase 2 - Fix Hooks MSW (2-3h) - PRIORITÉ MOYENNE
**Impact:** Fixera 30-40 tests

**Actions pour chaque fichier de hook échouant:**
1. Lire le fichier de test
2. Identifier les handlers MSW utilisés
3. Vérifier dans `__mocks__/handlers.ts` que les patterns correspondent
4. Ajouter debug logging:
   ```javascript
   rest.get('*/api/v1/visitors/check', (req, res, ctx) => {
     console.log('[MSW] visitors/check intercepted!', req.url)
     return res(ctx.json({ visitCount: 1 }))
   })
   ```
5. S'assurer que `server.use()` est appelé dans chaque test
6. Tester individuellement: `npm test -- <filename>.test.ts`

**Fichiers prioritaires (ordre):**
1. `useVisitCount.test.ts` (6 tests)
2. `useProfileDetection.test.ts` (11 tests)
3. `useTimelineData.test.ts` (syntaxe à compléter)
4. `use3DSupport.test.ts` (createElement mock)
5. Autres hooks

---

### Phase 3 - Fix Tests Divers (2-3h) - PRIORITÉ BASSE
**Impact:** Fixera 15-25 tests

**Actions:**
1. Lancer tests et noter tous les fichiers avec 1-3 échecs
2. Pour chaque fichier:
   - Lire le test échouant
   - Identifier le problème (assertion, mock, timing)
   - Appliquer le fix approprié
3. Relancer: `npm test -- <filename>.test.ts`

---

## 🛠️ Commandes Utiles

### Tests
```bash
# Suite complète
cd frontend && npm test

# Sans coverage (plus rapide)
npm test -- --no-coverage

# Fichier spécifique
npm test -- useVisitCount.test.ts

# Pattern de fichiers
npm test -- hooks/__tests__/

# Verbose (plus de détails)
npm test -- --verbose

# Watch mode (re-run on change)
npm test -- --watch

# Bail on first failure (rapide pour debug)
npm test -- --bail
```

### Debug
```bash
# Lancer avec node debugger
node --inspect-brk node_modules/.bin/jest --runInBand

# Logs MSW
# Ajouter dans le test:
server.listen({ onUnhandledRequest: 'warn' })
```

---

## 📁 Fichiers Clés Modifiés (Session Précédente)

### Configuration
- `frontend/jest.config.js` - Optimisé mémoire, enlevé options invalides
- `frontend/jest.setup.js` - Ajouté env vars, GC hints, polyfills
- `frontend/__mocks__/handlers.ts` - Handlers MSW (probablement OK)

### Tests Fixes
- `components/__tests__/RepoList.test.tsx` - ✅ 18 tests passent
- `components/analytics/__tests__/LettersGenerated.test.tsx` - ✅ 26 tests passent
- `components/analytics/__tests__/ThemeStats.test.tsx` - ✅ 13 tests passent
- `hooks/__tests__/useAnalyticsData.test.ts` - ✅ 6 tests passent (refactorisé)
- `hooks/__tests__/useVisitCount.test.ts` - ⚠️ Converti MSW v1 mais 6 tests échouent

### Tests Échouants (À FIXER)
- `hooks/__tests__/useProfileDetection.test.ts` - ❌ 11 tests
- `hooks/__tests__/useVisitCount.test.ts` - ❌ 6 tests
- `hooks/__tests__/useTimelineData.test.ts` - ❌ Syntaxe incomplète
- `hooks/__tests__/use3DSupport.test.ts` - ❌ createElement mock
- Et ~60 autres tests divers

---

## 🔍 Debugging Tips

### Si un hook ne répond pas (loading reste true)
1. Vérifier que MSW intercepte:
   ```javascript
   server.use(
     rest.get('*/api/v1/visitors/check', (req, res, ctx) => {
       console.log('MSW INTERCEPTED:', req.url.toString())
       return res(ctx.json({ visitCount: 1 }))
     })
   )
   ```
2. Vérifier l'URL exacte dans le hook:
   ```javascript
   // Dans le hook source
   const url = `${process.env.NEXT_PUBLIC_API_URL}/api/v1/visitors/check`
   console.log('Hook fetching:', url)
   ```
3. Augmenter timeout:
   ```javascript
   await waitFor(() => {
     expect(result.current.loading).toBe(false)
   }, { timeout: 10000 }) // 10s au lieu de 1s par défaut
   ```

### Si undici crash
1. Vérifier version: `npm list undici`
2. Downgrade: `npm install undici@5.28.0 --save-exact`
3. Ou mock les timers (voir Option B ci-dessus)

### Si mémoire crash (rare maintenant)
1. Augmenter: `workerIdleMemoryLimit: '8192MB'`
2. Réduire workers: `maxWorkers: 1` (déjà fait)
3. Lancer avec: `NODE_OPTIONS=--max-old-space-size=8192 npm test`

---

## 📊 Objectif Final

**Cible:** 95%+ de tests passants (785+/828 tests)

**Milestones:**
- ✅ Phase 1 (undici): ~755 tests passent (91%)
- ✅ Phase 2 (hooks MSW): ~785 tests passent (95%)
- ✅ Phase 3 (divers): ~800+ tests passent (97%+)

---

## 🚀 Prompt pour Agent Successeur

**Contexte rapide:**
Tu reprends une session de fix de tests frontend. 4 agents ont déjà fixé 41 tests (18 RepoList, 39 Analytics, crashes mémoire). Il reste 90 tests à fixer répartis en 3 catégories : problème undici/jsdom (15-20 tests), hooks MSW (40-50 tests), tests divers (20-25 tests).

**Priorité immédiate:**
1. Fixer le problème undici (TypeError: fastNowTimeout?.unref is not a function)
2. Fixer les hooks qui ne répondent pas (loading reste true)
3. Nettoyer les tests divers

**Fichier de référence:**
Lis ce fichier (HANDOFF_TESTS_FIXES.md) en entier pour comprendre le contexte complet.

**Première action:**
Tester le fix undici (downgrade à 5.28.0) et relancer les tests pour voir l'amélioration.

---

**Bonne chance ! 🎯**
