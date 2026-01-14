# Analytics Components

Dashboard analytics temps réel pour **maicivy** - Phase 4

---

## Composants

### RealtimeVisitors.tsx

**WebSocket temps réel des visiteurs actuels**

```tsx
import RealtimeVisitors from '@/components/analytics/RealtimeVisitors';

<RealtimeVisitors />
```

**Features:**
- WebSocket natif (`ws://backend/ws/analytics`)
- Auto-reconnect avec exponential backoff
- Count-up animation custom
- Indicateur connexion (vert pulsé / rouge)
- Cleanup automatique

**Props:** Aucune

---

### StatsOverview.tsx

**4 cards métriques principales**

```tsx
import StatsOverview from '@/components/analytics/StatsOverview';

<StatsOverview />
```

**Features:**
- Visiteurs, Pages vues, Lettres, Conversion
- Icônes Lucide (Users, Eye, FileText, TrendingUp)
- Auto-refresh 30s
- Grid responsive (1/2/4 cols)

**Props:** Aucune

---

### ThemeStats.tsx

**Bar chart horizontal des thèmes CV populaires**

```tsx
import ThemeStats from '@/components/analytics/ThemeStats';

<ThemeStats />
```

**Features:**
- Bar chart CSS pur (pas de Chart.js)
- Top thèmes avec count + pourcentage
- Colors gradient par thème
- Auto-refresh 30s

**Props:** Aucune

---

### LettersGenerated.tsx

**Line chart évolution lettres générées**

```tsx
import LettersGenerated from '@/components/analytics/LettersGenerated';

<LettersGenerated />
```

**Features:**
- Line chart SVG custom
- Period selector (Jour/Semaine/Mois)
- Total counter en haut
- Area fill + line + points
- Auto-refresh 30s

**Props:** Aucune

---

### Heatmap.tsx

**Carte de chaleur des interactions utilisateurs**

```tsx
import Heatmap from '@/components/analytics/Heatmap';

<Heatmap />
```

**Features:**
- Points blurred avec gradient couleur
- Tooltip au hover (élément + intensity)
- Grille 10x10 référence
- Auto-refresh 60s

**Props:** Aucune

---

### DateFilter.tsx

**Sélecteur de période pour filtrer les données**

```tsx
import DateFilter from '@/components/analytics/DateFilter';

<DateFilter />
```

**Features:**
- 4 presets (Aujourd'hui, 7j, 30j, Tout)
- Calcul dates automatique
- Display formatté (date-fns)
- UI buttons

**Props:** Aucune

---

## Hooks Utilisés

### useAnalyticsWebSocket

```typescript
import { useAnalyticsWebSocket } from '@/hooks/useAnalyticsWebSocket';

const { data, isConnected, error, reconnect } = useAnalyticsWebSocket();
```

**Features:**
- Connection WebSocket avec auto-reconnect
- Error handling
- Cleanup automatique

### useAnalyticsData

```typescript
import { useAnalyticsData } from '@/hooks/useAnalyticsData';

const { data, isLoading, error, refetch } = useAnalyticsData({
  endpoint: '/api/v1/analytics/stats',
  refreshInterval: 30000,
  enabled: true,
});
```

**Features:**
- Fetch avec polling configurable
- Loading/error states
- Manual refetch

---

## API Client

```typescript
import {
  fetchAnalyticsStats,
  fetchThemeStats,
  fetchLettersStats,
  fetchHeatmapData,
} from '@/lib/analytics-api';

// Fetch stats
const stats = await fetchAnalyticsStats('week');

// Fetch themes
const themes = await fetchThemeStats();

// Fetch letters
const letters = await fetchLettersStats('month');

// Fetch heatmap
const heatmap = await fetchHeatmapData('home');
```

---

## Types

```typescript
import {
  AnalyticsStats,
  ThemeStat,
  LettersStat,
  HeatmapPoint,
  HeatmapData,
  RealtimeData,
} from '@/lib/types';
```

Tous les types sont dans `/lib/types.ts` section "Analytics Types (Phase 4)"

---

## Usage dans Page

```tsx
// /app/analytics/page.tsx
import { Suspense } from 'react';
import RealtimeVisitors from '@/components/analytics/RealtimeVisitors';
import ThemeStats from '@/components/analytics/ThemeStats';
import LettersGenerated from '@/components/analytics/LettersGenerated';
import Heatmap from '@/components/analytics/Heatmap';
import DateFilter from '@/components/analytics/DateFilter';
import StatsOverview from '@/components/analytics/StatsOverview';

export default function AnalyticsPage() {
  return (
    <div className="container mx-auto px-4 py-8">
      <h1 className="text-4xl font-bold mb-8">Analytics Dashboard</h1>

      <DateFilter />

      <Suspense fallback={<Skeleton />}>
        <StatsOverview />
      </Suspense>

      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
        <div className="lg:col-span-3">
          <Suspense fallback={<Skeleton />}>
            <RealtimeVisitors />
          </Suspense>
        </div>

        <div className="lg:col-span-2">
          <Suspense fallback={<Skeleton />}>
            <ThemeStats />
          </Suspense>
        </div>

        <div className="lg:col-span-1">
          <Suspense fallback={<Skeleton />}>
            <LettersGenerated />
          </Suspense>
        </div>

        <div className="lg:col-span-3">
          <Suspense fallback={<Skeleton />}>
            <Heatmap />
          </Suspense>
        </div>
      </div>
    </div>
  );
}
```

---

## Styling

Tous les composants utilisent:
- **Tailwind CSS** pour le styling
- **shadcn/ui** patterns (cards, borders, etc.)
- **Dark mode** compatible
- **Responsive** design (mobile-first)

### Classes Communes

```tsx
// Card wrapper
"rounded-lg border bg-card p-6"

// Skeleton loading
"animate-pulse h-64 bg-muted rounded"

// Grid responsive
"grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4"
```

---

## Performance

### Optimizations

1. **Pas de Chart.js:** Charts custom CSS/SVG (bundle size optimisé)
2. **WebSocket natif:** Pas de socket.io-client
3. **Polling intelligent:** Intervals différents selon criticité
4. **Cleanup:** Tous les intervals/WebSocket nettoyés
5. **Suspense:** Progressive loading
6. **Memoization:** React.memo si besoin

### Polling Intervals

- StatsOverview: 30s
- ThemeStats: 30s
- LettersGenerated: 30s
- Heatmap: 60s (moins critique)
- RealtimeVisitors: WebSocket (temps réel)

---

## Error Handling

Tous les composants ont:
1. **Try/catch** sur fetch
2. **Mock data fallback** si erreur
3. **Console logs** pour debug
4. **Loading states** élégants
5. **Error boundaries** (Suspense)

### Mode Mock

Si backend pas disponible:
- Données de démonstration affichées
- Console: "Error fetching..."
- UI reste fonctionnelle
- Permet dev frontend seul

---

## Backend Integration

### Endpoints Requis (Doc 11)

```
GET  /api/v1/analytics/stats?period={day|week|month}
GET  /api/v1/analytics/themes
GET  /api/v1/analytics/letters?period={day|week|month}
GET  /api/v1/analytics/heatmap?page={page_name}
WS   /ws/analytics
```

### CORS Configuration

Backend doit autoriser:
```go
AllowOrigins: []string{"http://localhost:3000"}
AllowCredentials: true
AllowWebSockets: true
```

---

## Testing

### Tests Unitaires

```bash
# À créer
components/analytics/__tests__/RealtimeVisitors.test.tsx
components/analytics/__tests__/ThemeStats.test.tsx
components/analytics/__tests__/LettersGenerated.test.tsx
components/analytics/__tests__/Heatmap.test.tsx
```

### Tests E2E

```bash
# À créer
e2e/analytics.spec.ts
```

---

## Troubleshooting

### WebSocket Reste Déconnecté

**Causes:**
1. Backend pas lancé
2. CORS pas configuré
3. WebSocket endpoint incorrect

**Solution:**
```bash
# Vérifier backend
curl http://localhost:8080/health

# Vérifier logs console browser (F12)
```

### Données Toujours Mock

**Cause:** Backend pas accessible

**Solution:** Lancer backend (Doc 11)

### Erreur TypeScript

```bash
npm run type-check
```

---

## Documentation Complète

- **Implementation:** `/ANALYTICS_DASHBOARD_IMPLEMENTATION_SUMMARY.md`
- **Validation:** `/ANALYTICS_VALIDATION.md`
- **Quick Start:** `/QUICK_START_ANALYTICS.md`
- **Specs:** `/docs/implementation/12_FRONTEND_ANALYTICS_DASHBOARD.md`

---

## Contribution

Pour ajouter un nouveau composant analytics:

1. Créer `/components/analytics/MonComposant.tsx`
2. Suivre le pattern existant:
   - useState pour data
   - useEffect pour fetch/polling
   - Loading skeleton
   - Error handling (mock data)
3. Ajouter types dans `/lib/types.ts`
4. Ajouter endpoint dans `/lib/analytics-api.ts`
5. Utiliser dans `/app/analytics/page.tsx`
6. Tester responsive + dark mode

---

**Version:** 1.0
**Date:** 2025-12-08
**Phase:** 4 - Analytics
