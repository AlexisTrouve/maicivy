# Letters Components

**Phase 3 - IA Lettres**

Composants pour la génération de lettres de motivation et anti-motivation par IA.

---

## 📁 Structure

```
/components/letters/
├── AccessGate.tsx         - Vérifie accès (3 visites requises)
├── LetterGenerator.tsx    - Formulaire + génération
├── LetterPreview.tsx      - Affichage dual + export PDF
├── index.ts               - Barrel exports
└── README.md              - Cette documentation
```

---

## 🎯 Usage

### AccessGate

Wrapper pour vérifier l'accès aux fonctionnalités IA.

```tsx
import { AccessGate } from '@/components/letters';

<AccessGate>
  <ProtectedContent />
</AccessGate>
```

**Comportement:**
- Affiche teaser si `visitCount < 3`
- Affiche children si `hasAccess = true`
- Appelle API `/api/v1/visitors/check` au montage

---

### LetterGenerator

Formulaire de génération de lettres.

```tsx
import { LetterGenerator } from '@/components/letters';

<LetterGenerator />
```

**Features:**
- Input validation (Zod)
- Progress bar animée
- Error handling (403, 429, 500)
- LocalStorage history
- Affichage conditionnel Form ↔ Preview

---

### LetterPreview

Affichage dual des lettres + actions.

```tsx
import { LetterPreview } from '@/components/letters';

<LetterPreview
  letters={generatedLetters}
  onReset={() => setLetters(null)}
/>
```

**Features:**
- Dual display (2 colonnes responsive)
- Copy to clipboard
- Download PDF (individuel + dual)
- Scroll indépendant
- Warning footer

---

## 🎨 Design Tokens

### Colors

```tsx
// Gradients
primary: 'from-blue-600 to-purple-600'
progress: 'from-blue-500 to-purple-500'
motivation: 'from-green-500 to-emerald-500'
anti: 'from-orange-500 to-red-500'
warning: 'amber-50' / 'amber-800'
```

### Breakpoints

```tsx
mobile: < 640px   → Stack vertical
tablet: 640-1024  → Stack vertical
desktop: > 1024px → Dual display (2 cols)
```

---

## 🔧 Dependencies

```json
{
  "react-hook-form": "^7.49.2",
  "zod": "^3.22.4",
  "@hookform/resolvers": "^3.3.3",
  "framer-motion": "^10.16.16",
  "lucide-react": "^0.303.0"
}
```

---

## 📊 API Endpoints

### Required Backend Endpoints

1. **POST /api/v1/letters/generate**
   ```json
   Request: { "companyName": "Google" }
   Response: { id, motivationLetter, antiMotivationLetter, ... }
   ```

2. **GET /api/v1/visitors/check**
   ```json
   Response: { visitCount, hasAccess, remainingVisits, ... }
   ```

3. **GET /api/v1/letters/:id/pdf?type=...**
   ```
   Response: application/pdf (blob)
   ```

Voir **API_REQUIREMENTS_LETTERS.md** pour détails complets.

---

## 🧪 Testing

### Unit Tests

```bash
npm run test -- letters
```

Tests recommandés:
- AccessGate: loading, teaser, access granted
- LetterGenerator: validation, API calls, errors
- LetterPreview: dual display, copy, download

### E2E Tests

```bash
npm run test:e2e
```

Flows:
- Visite 1-3 → Access gate → Generator
- Generate → Preview dual → Download PDF

---

## 🚀 Development

### Run Dev Server

```bash
npm run dev
```

Navigate to: http://localhost:3000/letters

### Type Check

```bash
npm run type-check
```

### Lint

```bash
npm run lint
```

---

## 📝 Notes

### LocalStorage

Les lettres générées sont sauvegardées dans:
```typescript
Key: 'letters_history'
Format: Array<{ id, companyName, createdAt }>
Max: 10 items (FIFO)
```

### Error Handling

```typescript
403 → "Accès refusé. 3 visites requises."
429 → "Limite atteinte. Réessayez plus tard."
500 → "Erreur serveur. IA en pause café."
```

### Progress Simulation

Actuellement simulée côté client (800ms interval).
Future: WebSocket pour progrès temps réel.

---

## 🔗 Related Docs

- [LETTERS_FRONTEND_IMPLEMENTATION_SUMMARY.md](../../LETTERS_FRONTEND_IMPLEMENTATION_SUMMARY.md)
- [API_REQUIREMENTS_LETTERS.md](../../API_REQUIREMENTS_LETTERS.md)
- [/docs/implementation/10_FRONTEND_LETTERS.md](/docs/implementation/10_FRONTEND_LETTERS.md)

---

**Author:** Claude (Agent IA)
**Date:** 2025-12-08
**Version:** 1.0
