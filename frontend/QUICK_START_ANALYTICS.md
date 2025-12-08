# Quick Start - Analytics Dashboard

Guide rapide pour tester le dashboard analytics.

---

## Lancement Rapide

```bash
cd frontend

# 1. Installer les dépendances (si pas déjà fait)
npm install

# 2. Vérifier la configuration
cat .env.local
# NEXT_PUBLIC_API_URL=http://localhost:8080

# 3. Lancer le serveur dev
npm run dev

# 4. Ouvrir le dashboard
# http://localhost:3000/analytics
```

---

## Ce Que Vous Devriez Voir

### Sans Backend (Mode Mock)

Le dashboard fonctionne avec des données de démonstration:

```
┌─────────────────────────────────────────────┐
│ Analytics Dashboard                         │
│                                             │
│ [Filters: Aujourd'hui | 7j | 30j | Tout]   │
│                                             │
│ ┌──────┐ ┌──────┐ ┌──────┐ ┌──────┐        │
│ │ 1543 │ │ 8234 │ │ 456  │ │29.6% │        │
│ │Visit.│ │Pages │ │Lettr.│ │Conv. │        │
│ └──────┘ └──────┘ └──────┘ └──────┘        │
│                                             │
│ ┌─────────────────────────────────┐        │
│ │ 🔴 Déconnecté                   │        │
│ │ 0 personnes en ce moment        │        │
│ └─────────────────────────────────┘        │
│                                             │
│ ┌────────────────┐ ┌─────────────┐        │
│ │ Top Thèmes CV  │ │ Lettres IA  │        │
│ │ ████████ 523   │ │   456       │        │
│ │ █████ 312      │ │   📈        │        │
│ └────────────────┘ └─────────────┘        │
│                                             │
│ ┌─────────────────────────────────┐        │
│ │ Heatmap des Interactions        │        │
│ │ 🟦 🟩 🟨 🔴                      │        │
│ └─────────────────────────────────┘        │
└─────────────────────────────────────────────┘
```

**Note:** L'indicateur WebSocket sera rouge (déconnecté) car le backend n'est pas lancé.

### Avec Backend (Mode Réel)

Si le backend analytics (Doc 11) est lancé:

1. **WebSocket:** 🟢 En ligne (vert pulsé)
2. **Visiteurs:** Nombre réel affiché
3. **Stats:** Données réelles du backend
4. **Auto-refresh:** Toutes les 30-60s

---

## Console Browser

Ouvrez la console (F12) pour voir:

```
[WS] Connected to analytics               # Si backend lancé
[useAnalyticsWebSocket] Connected         # WebSocket OK
Error fetching theme stats: ...           # Si backend pas lancé (normal)
```

**Mode Mock Actif = Données de démonstration affichées**

---

## Tests Manuels

### 1. Responsive

```bash
# Tester différentes tailles d'écran:

# Mobile (375px)
# - Components empilés verticalement
# - Scroll vertical uniquement

# Tablet (768px)
# - Grid 2 colonnes

# Desktop (1920px)
# - Grid 3 colonnes optimale
```

### 2. Interactions

- [ ] Cliquer sur les boutons période (Jour/Semaine/Mois)
- [ ] Hover sur points heatmap → Tooltip
- [ ] Changer les filtres de date
- [ ] Observer l'indicateur WebSocket

### 3. Dark Mode

```bash
# Dans votre navigateur:
# Settings > Appearance > Dark

# Le dashboard s'adapte automatiquement
```

---

## Vérification Build

```bash
# Type check
npm run type-check
# ✅ Expected: 0 errors

# Lint
npm run lint
# ✅ Expected: 0 errors

# Build production
npm run build
# ✅ Expected: Build successful

# Tester build
npm run start
```

---

## Intégration Backend

### Lancer Backend (Doc 11)

```bash
cd ../backend

# Start backend with analytics
go run cmd/main.go

# Backend lance sur http://localhost:8080
# WebSocket disponible ws://localhost:8080/ws/analytics
```

### Vérifier Connexion

1. Refresh page `/analytics`
2. WebSocket devrait passer à 🟢 En ligne
3. Données réelles affichées
4. Auto-refresh fonctionne

---

## Troubleshooting

### WebSocket Reste Rouge

**Symptôme:** Indicateur "Déconnecté" même avec backend lancé

**Causes possibles:**
1. Backend pas lancé
2. CORS pas configuré
3. WebSocket endpoint incorrect

**Solution:**
```bash
# Vérifier backend
curl http://localhost:8080/health

# Vérifier logs console browser (F12)
# Chercher: [WS] Error: ...
```

### Données Pas Réelles

**Symptôme:** Toujours les mêmes chiffres (1543, 8234, etc.)

**Cause:** Mode mock actif (backend pas accessible)

**Solution:**
```bash
# Vérifier console browser
# Si erreur fetch → backend pas lancé ou endpoints manquants
```

### Erreur de Build

**Symptôme:** `npm run build` échoue

**Solution:**
```bash
# Clean install
rm -rf node_modules .next
npm install
npm run build
```

---

## Features Testées

### ✅ Fonctionnalités Principales

- [✅] Page `/analytics` accessible
- [✅] Stats cards affichées (4 cards)
- [✅] WebSocket connection tentée
- [✅] Charts rendus (themes, letters)
- [✅] Heatmap affichée
- [✅] Filters fonctionnels
- [✅] Responsive design
- [✅] Dark mode compatible
- [✅] Loading skeletons
- [✅] Mock data fallback

### ⏳ À Tester avec Backend

- [ ] WebSocket données réelles
- [ ] Auto-refresh stats
- [ ] Persistence données
- [ ] Performance load
- [ ] Multi-users temps réel

---

## Performance

### Metrics Attendus

**Lighthouse Score (Target):**
- Performance: > 90
- Accessibility: > 95
- Best Practices: > 90
- SEO: > 85

**Bundle Size:**
- Page analytics: ~50 KB (gzipped)
- First Load JS: ~100 KB
- No Chart.js: Bundle optimisé

**Runtime:**
- Time to Interactive: < 2s
- First Contentful Paint: < 1s
- WebSocket latency: < 100ms

---

## Développement

### Ajouter un Nouveau Composant Analytics

1. Créer `/components/analytics/MonComposant.tsx`
2. Implémenter avec même pattern:
   - useState pour data
   - useEffect pour fetch/polling
   - Loading skeleton
   - Error handling (mock data)
3. Ajouter à `/app/analytics/page.tsx`
4. Tester responsive

### Modifier API Endpoints

1. Modifier `/lib/analytics-api.ts`
2. Mettre à jour types dans `/lib/types.ts`
3. Adapter composants si format change

---

## Documentation Complète

- **Résumé:** `ANALYTICS_DASHBOARD_IMPLEMENTATION_SUMMARY.md`
- **Validation:** `ANALYTICS_VALIDATION.md`
- **Quick Start:** Ce fichier
- **Doc officielle:** `/docs/implementation/12_FRONTEND_ANALYTICS_DASHBOARD.md`

---

## Support

### Questions Backend

Voir Document 11 (Backend Analytics) pour:
- Endpoints à implémenter
- Format des réponses
- WebSocket protocol
- Configuration CORS

### Questions Frontend

Issues courantes:
- TypeScript errors → `npm run type-check`
- Style issues → Vérifier Tailwind classes
- WebSocket issues → Console logs
- Performance → Chrome DevTools Performance tab

---

## Next Steps

Après validation frontend analytics:

1. ✅ Frontend complet (Ce doc)
2. ⏳ Backend analytics (Doc 11)
3. ⏳ Tests integration
4. ⏳ Tests E2E (Playwright)
5. ⏳ Performance optimization
6. ⏳ Production deployment

---

**Status:** ✅ Dashboard Frontend Prêt
**Date:** 2025-12-08

Enjoy your Analytics Dashboard! 📊
