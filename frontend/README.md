# maicivy Frontend

Frontend Next.js 14 pour le projet maicivy.

## Technologies

- **Framework:** Next.js 14 (App Router)
- **Language:** TypeScript
- **Styling:** Tailwind CSS
- **UI Components:** shadcn/ui (Radix UI)
- **Animations:** Framer Motion
- **Icons:** Lucide React
- **Forms:** React Hook Form + Zod

## Installation

```bash
npm install
```

## Développement

```bash
npm run dev
```

Ouvre [http://localhost:3000](http://localhost:3000) dans ton navigateur.

## Build Production

```bash
npm run build
npm start
```

## Scripts Disponibles

- `npm run dev` - Lance le serveur de développement
- `npm run build` - Build pour la production
- `npm start` - Lance le serveur de production
- `npm run lint` - Vérifie le code avec ESLint
- `npm run lint:fix` - Corrige automatiquement les erreurs ESLint
- `npm run type-check` - Vérifie les types TypeScript
- `npm run format` - Formate le code avec Prettier
- `npm run format:check` - Vérifie le formatage du code

## Structure

```
frontend/
├── app/                    # Next.js App Router
│   ├── layout.tsx         # Layout racine
│   ├── page.tsx           # Homepage
│   ├── loading.tsx        # Loading state global
│   ├── error.tsx          # Error boundary global
│   ├── not-found.tsx      # Page 404
│   ├── cv/                # Pages CV (Phase 2)
│   ├── letters/           # Pages Lettres (Phase 3)
│   └── analytics/         # Pages Analytics (Phase 4)
├── components/
│   ├── layout/            # Header, Footer
│   ├── ui/                # shadcn/ui components
│   └── shared/            # Composants partagés
├── lib/
│   ├── api.ts             # API client
│   ├── types.ts           # Types TypeScript
│   └── utils.ts           # Fonctions utilitaires
└── hooks/                 # Custom React hooks
```

## Variables d'Environnement

Copie `.env.local` et configure :

```bash
NEXT_PUBLIC_API_URL=http://localhost:8080
NEXT_PUBLIC_ANALYTICS_ENABLED=true
```

## Dark Mode

Le dark mode est activé via un toggle dans le header. Le thème est persisté dans localStorage.

## API Client

Le client API (`lib/api.ts`) inclut :
- Retry logic automatique (3 tentatives)
- Timeout de 30 secondes
- Gestion centralisée des erreurs
- Support des cookies de session

## Prochaines Étapes

- Phase 2: Implémentation du CV Dynamique
- Phase 3: Générateur de Lettres IA
- Phase 4: Dashboard Analytics

## Ressources

- [Next.js Documentation](https://nextjs.org/docs)
- [Tailwind CSS](https://tailwindcss.com/docs)
- [shadcn/ui](https://ui.shadcn.com)
