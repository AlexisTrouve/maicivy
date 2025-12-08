# 05. FRONTEND FOUNDATION

## 📋 Métadonnées

- **Phase:** 1
- **Priorité:** CRITIQUE
- **Complexité:** ⭐⭐⭐ (3/5)
- **Prérequis:** 01. SETUP_INFRASTRUCTURE.md (peut être parallèle à 02-04)
- **Temps estimé:** 2-3 jours
- **Status:** 🔲 À faire

---

## 🎯 Objectif

Établir la fondation frontend complète du projet avec Next.js 14, TypeScript, et Tailwind CSS. Ce document couvre la configuration initiale, la structure du projet, l'API client, les composants de base, et le système de design.

**Livrables clés:**
- Application Next.js 14 configurée avec App Router
- Tailwind CSS avec thème personnalisé et dark mode
- Structure de dossiers standardisée
- API client avec gestion d'erreurs centralisée
- Layout principal avec navigation
- shadcn/ui configuré avec composants de base

---

## 🏗️ Architecture

### Vue d'Ensemble

```
frontend/
├── app/                          # Next.js 14 App Router
│   ├── layout.tsx               # Layout racine
│   ├── page.tsx                 # Homepage
│   ├── cv/
│   │   └── page.tsx            # CV dynamique (Phase 2)
│   ├── letters/
│   │   └── page.tsx            # Générateur lettres (Phase 3)
│   └── analytics/
│       └── page.tsx            # Dashboard analytics (Phase 4)
│
├── components/
│   ├── layout/                  # Layout components
│   │   ├── Header.tsx
│   │   ├── Footer.tsx
│   │   └── Navigation.tsx
│   ├── ui/                      # shadcn/ui components
│   │   ├── button.tsx
│   │   ├── card.tsx
│   │   ├── dialog.tsx
│   │   └── ...
│   └── shared/                  # Shared components
│       ├── LoadingSpinner.tsx
│       └── ErrorBoundary.tsx
│
├── lib/
│   ├── api.ts                   # API client wrapper
│   ├── utils.ts                 # Utility functions
│   └── types.ts                 # TypeScript types
│
├── public/
│   ├── images/
│   └── fonts/
│
├── styles/
│   └── globals.css              # Global styles
│
├── tailwind.config.ts
├── next.config.js
├── tsconfig.json
└── package.json
```

### Design Decisions

**Pourquoi Next.js 14 App Router:**
- Server Components par défaut (performance)
- Streaming SSR natif
- Route handlers intégrés
- Meilleure organisation du code

**Pourquoi Tailwind CSS:**
- Utility-first (développement rapide)
- Dark mode natif
- Optimisation production (purge CSS)
- Excellent DX avec IntelliSense

**Pourquoi shadcn/ui:**
- Composants accessibles (Radix UI)
- Ownership du code (copy-paste, pas npm)
- Customisation totale
- Pas de bundle bloat

---

## 📦 Dépendances

### Bibliothèques NPM

```bash
# Framework
npm install next@14 react react-dom

# TypeScript
npm install -D typescript @types/react @types/node

# Styling
npm install tailwindcss postcss autoprefixer
npm install tailwindcss-animate class-variance-authority clsx tailwind-merge

# UI Components (shadcn/ui dependencies)
npm install @radix-ui/react-slot @radix-ui/react-dialog @radix-ui/react-dropdown-menu

# Forms & Validation
npm install react-hook-form zod @hookform/resolvers

# Animations
npm install framer-motion

# Icons
npm install lucide-react

# Utils
npm install date-fns
```

### Setup shadcn/ui

```bash
npx shadcn-ui@latest init
```

Configuration:
- Style: Default
- Base color: Slate
- CSS variables: Yes

```bash
# Installer composants de base
npx shadcn-ui@latest add button
npx shadcn-ui@latest add card
npx shadcn-ui@latest add dialog
npx shadcn-ui@latest add dropdown-menu
npx shadcn-ui@latest add input
npx shadcn-ui@latest add label
npx shadcn-ui@latest add toast
npx shadcn-ui@latest add skeleton
```

---

## 🔨 Implémentation

### Étape 1: Initialisation Next.js

**Description:** Créer le projet Next.js avec TypeScript et configuration initiale.

**Code:**

```bash
# Depuis le dossier racine maicivy/
npx create-next-app@14 frontend --typescript --tailwind --app --src-dir=false --import-alias="@/*"
```

**Configuration `next.config.js`:**

```javascript
/** @type {import('next').NextConfig} */
const nextConfig = {
  reactStrictMode: true,

  // API Backend
  async rewrites() {
    return [
      {
        source: '/api/:path*',
        destination: process.env.NEXT_PUBLIC_API_URL + '/api/:path*',
      },
    ];
  },

  // Images
  images: {
    remotePatterns: [
      {
        protocol: 'https',
        hostname: 'avatars.githubusercontent.com',
      },
    ],
  },

  // Headers de sécurité
  async headers() {
    return [
      {
        source: '/:path*',
        headers: [
          {
            key: 'X-Frame-Options',
            value: 'DENY',
          },
          {
            key: 'X-Content-Type-Options',
            value: 'nosniff',
          },
          {
            key: 'Referrer-Policy',
            value: 'strict-origin-when-cross-origin',
          },
        ],
      },
    ];
  },
};

module.exports = nextConfig;
```

**Variables d'environnement `.env.local`:**

```bash
# API Backend
NEXT_PUBLIC_API_URL=http://localhost:8080

# Analytics (si applicable)
NEXT_PUBLIC_ANALYTICS_ENABLED=true
```

**Explications:**
- `rewrites()` permet de proxy les requêtes API vers le backend Go
- Configuration images pour GitHub avatars (import projets)
- Headers de sécurité de base

---

### Étape 2: Configuration Tailwind CSS

**Description:** Configurer Tailwind avec palette personnalisée, dark mode, et fonts.

**Code `tailwind.config.ts`:**

```typescript
import type { Config } from 'tailwindcss';

const config: Config = {
  darkMode: ['class'],
  content: [
    './pages/**/*.{ts,tsx}',
    './components/**/*.{ts,tsx}',
    './app/**/*.{ts,tsx}',
    './src/**/*.{ts,tsx}',
  ],
  theme: {
    container: {
      center: true,
      padding: '2rem',
      screens: {
        '2xl': '1400px',
      },
    },
    extend: {
      colors: {
        border: 'hsl(var(--border))',
        input: 'hsl(var(--input))',
        ring: 'hsl(var(--ring))',
        background: 'hsl(var(--background))',
        foreground: 'hsl(var(--foreground))',
        primary: {
          DEFAULT: 'hsl(var(--primary))',
          foreground: 'hsl(var(--primary-foreground))',
        },
        secondary: {
          DEFAULT: 'hsl(var(--secondary))',
          foreground: 'hsl(var(--secondary-foreground))',
        },
        destructive: {
          DEFAULT: 'hsl(var(--destructive))',
          foreground: 'hsl(var(--destructive-foreground))',
        },
        muted: {
          DEFAULT: 'hsl(var(--muted))',
          foreground: 'hsl(var(--muted-foreground))',
        },
        accent: {
          DEFAULT: 'hsl(var(--accent))',
          foreground: 'hsl(var(--accent-foreground))',
        },
        popover: {
          DEFAULT: 'hsl(var(--popover))',
          foreground: 'hsl(var(--popover-foreground))',
        },
        card: {
          DEFAULT: 'hsl(var(--card))',
          foreground: 'hsl(var(--card-foreground))',
        },
      },
      borderRadius: {
        lg: 'var(--radius)',
        md: 'calc(var(--radius) - 2px)',
        sm: 'calc(var(--radius) - 4px)',
      },
      fontFamily: {
        sans: ['var(--font-inter)', 'sans-serif'],
        heading: ['var(--font-poppins)', 'sans-serif'],
      },
      keyframes: {
        'accordion-down': {
          from: { height: '0' },
          to: { height: 'var(--radix-accordion-content-height)' },
        },
        'accordion-up': {
          from: { height: 'var(--radix-accordion-content-height)' },
          to: { height: '0' },
        },
      },
      animation: {
        'accordion-down': 'accordion-down 0.2s ease-out',
        'accordion-up': 'accordion-up 0.2s ease-out',
      },
    },
  },
  plugins: [require('tailwindcss-animate')],
};

export default config;
```

**Code `styles/globals.css`:**

```css
@tailwind base;
@tailwind components;
@tailwind utilities;

@layer base {
  :root {
    --background: 0 0% 100%;
    --foreground: 222.2 84% 4.9%;
    --card: 0 0% 100%;
    --card-foreground: 222.2 84% 4.9%;
    --popover: 0 0% 100%;
    --popover-foreground: 222.2 84% 4.9%;
    --primary: 221.2 83.2% 53.3%;
    --primary-foreground: 210 40% 98%;
    --secondary: 210 40% 96.1%;
    --secondary-foreground: 222.2 47.4% 11.2%;
    --muted: 210 40% 96.1%;
    --muted-foreground: 215.4 16.3% 46.9%;
    --accent: 210 40% 96.1%;
    --accent-foreground: 222.2 47.4% 11.2%;
    --destructive: 0 84.2% 60.2%;
    --destructive-foreground: 210 40% 98%;
    --border: 214.3 31.8% 91.4%;
    --input: 214.3 31.8% 91.4%;
    --ring: 221.2 83.2% 53.3%;
    --radius: 0.5rem;
  }

  .dark {
    --background: 222.2 84% 4.9%;
    --foreground: 210 40% 98%;
    --card: 222.2 84% 4.9%;
    --card-foreground: 210 40% 98%;
    --popover: 222.2 84% 4.9%;
    --popover-foreground: 210 40% 98%;
    --primary: 217.2 91.2% 59.8%;
    --primary-foreground: 222.2 47.4% 11.2%;
    --secondary: 217.2 32.6% 17.5%;
    --secondary-foreground: 210 40% 98%;
    --muted: 217.2 32.6% 17.5%;
    --muted-foreground: 215 20.2% 65.1%;
    --accent: 217.2 32.6% 17.5%;
    --accent-foreground: 210 40% 98%;
    --destructive: 0 62.8% 30.6%;
    --destructive-foreground: 210 40% 98%;
    --border: 217.2 32.6% 17.5%;
    --input: 217.2 32.6% 17.5%;
    --ring: 224.3 76.3% 48%;
  }
}

@layer base {
  * {
    @apply border-border;
  }
  body {
    @apply bg-background text-foreground;
    font-feature-settings: 'rlig' 1, 'calt' 1;
  }
}
```

**Explications:**
- Système de design basé sur CSS variables pour faciliter le theming
- Dark mode via classe `dark` sur l'élément HTML
- Couleurs HSL pour meilleure manipulation
- Fonts Inter (texte) et Poppins (headings)

---

### Étape 3: Configuration Fonts

**Description:** Configurer les fonts Google Fonts optimisées avec Next.js.

**Code `app/layout.tsx`:**

```typescript
import type { Metadata } from 'next';
import { Inter, Poppins } from 'next/font/google';
import './globals.css';

const inter = Inter({
  subsets: ['latin'],
  variable: '--font-inter',
  display: 'swap',
});

const poppins = Poppins({
  subsets: ['latin'],
  weight: ['400', '500', '600', '700'],
  variable: '--font-poppins',
  display: 'swap',
});

export const metadata: Metadata = {
  title: {
    default: 'maicivy - CV Interactif Intelligent',
    template: '%s | maicivy',
  },
  description: 'CV interactif avec génération de lettres de motivation par IA',
  keywords: ['CV', 'portfolio', 'IA', 'développeur', 'full-stack'],
  authors: [{ name: 'Alexi' }],
  openGraph: {
    type: 'website',
    locale: 'fr_FR',
    url: 'https://maicivy.com',
    title: 'maicivy - CV Interactif Intelligent',
    description: 'CV interactif avec génération de lettres de motivation par IA',
  },
};

export default function RootLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <html lang="fr" suppressHydrationWarning>
      <body className={`${inter.variable} ${poppins.variable} font-sans antialiased`}>
        {children}
      </body>
    </html>
  );
}
```

**Explications:**
- `next/font/google` optimise le chargement des fonts (self-hosting)
- `display: 'swap'` améliore les performances (FOIT → FOUT)
- CSS variables pour utilisation dans Tailwind
- Metadata SEO de base

---

### Étape 4: API Client Wrapper

**Description:** Créer un client API centralisé avec gestion d'erreurs et retry logic.

**Code `lib/types.ts`:**

```typescript
// Types communs API
export interface ApiResponse<T> {
  data: T;
  success: boolean;
  message?: string;
}

export interface ApiError {
  success: false;
  message: string;
  code?: string;
  statusCode: number;
}

export interface PaginatedResponse<T> {
  data: T[];
  meta: {
    page: number;
    limit: number;
    total: number;
    totalPages: number;
  };
}
```

**Code `lib/api.ts`:**

```typescript
const API_BASE_URL = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080';

export class ApiClient {
  private baseUrl: string;
  private defaultHeaders: HeadersInit;

  constructor() {
    this.baseUrl = API_BASE_URL;
    this.defaultHeaders = {
      'Content-Type': 'application/json',
    };
  }

  /**
   * Fetch wrapper avec retry logic
   */
  private async fetchWithRetry<T>(
    url: string,
    options: RequestInit = {},
    retries = 3
  ): Promise<T> {
    const controller = new AbortController();
    const timeout = setTimeout(() => controller.abort(), 30000); // 30s timeout

    try {
      const response = await fetch(url, {
        ...options,
        headers: {
          ...this.defaultHeaders,
          ...options.headers,
        },
        credentials: 'include', // Pour les cookies de session
        signal: controller.signal,
      });

      clearTimeout(timeout);

      // Gestion des erreurs HTTP
      if (!response.ok) {
        const errorData = await response.json().catch(() => ({
          message: 'Une erreur est survenue',
        }));

        throw {
          success: false,
          message: errorData.message || `Erreur ${response.status}`,
          code: errorData.code,
          statusCode: response.status,
        } as ApiError;
      }

      return await response.json();
    } catch (error) {
      clearTimeout(timeout);

      // Retry logic pour erreurs réseau
      if (retries > 0 && this.isRetryable(error)) {
        await this.delay(1000 * (4 - retries)); // Exponential backoff
        return this.fetchWithRetry<T>(url, options, retries - 1);
      }

      throw error;
    }
  }

  /**
   * Détermine si l'erreur est retryable
   */
  private isRetryable(error: any): boolean {
    // Ne pas retry les erreurs client (4xx)
    if (error.statusCode && error.statusCode >= 400 && error.statusCode < 500) {
      return false;
    }

    // Retry erreurs réseau et serveur (5xx)
    return true;
  }

  /**
   * Delay helper pour retry
   */
  private delay(ms: number): Promise<void> {
    return new Promise((resolve) => setTimeout(resolve, ms));
  }

  /**
   * GET request
   */
  async get<T>(endpoint: string, params?: Record<string, string>): Promise<T> {
    const url = new URL(`${this.baseUrl}${endpoint}`);

    if (params) {
      Object.entries(params).forEach(([key, value]) => {
        url.searchParams.append(key, value);
      });
    }

    return this.fetchWithRetry<T>(url.toString(), {
      method: 'GET',
    });
  }

  /**
   * POST request
   */
  async post<T>(endpoint: string, data?: any): Promise<T> {
    return this.fetchWithRetry<T>(`${this.baseUrl}${endpoint}`, {
      method: 'POST',
      body: data ? JSON.stringify(data) : undefined,
    });
  }

  /**
   * PUT request
   */
  async put<T>(endpoint: string, data?: any): Promise<T> {
    return this.fetchWithRetry<T>(`${this.baseUrl}${endpoint}`, {
      method: 'PUT',
      body: data ? JSON.stringify(data) : undefined,
    });
  }

  /**
   * DELETE request
   */
  async delete<T>(endpoint: string): Promise<T> {
    return this.fetchWithRetry<T>(`${this.baseUrl}${endpoint}`, {
      method: 'DELETE',
    });
  }
}

// Export singleton
export const api = new ApiClient();

// Export helpers typés (seront enrichis dans les phases suivantes)
export const cvApi = {
  getCV: (theme?: string) =>
    api.get<any>('/api/cv', theme ? { theme } : undefined),
  getThemes: () =>
    api.get<any>('/api/cv/themes'),
};

export const healthApi = {
  check: () => api.get<{ status: string; timestamp: string }>('/health'),
};
```

**Code `lib/utils.ts`:**

```typescript
import { type ClassValue, clsx } from 'clsx';
import { twMerge } from 'tailwind-merge';

/**
 * Merge Tailwind classes (shadcn/ui utility)
 */
export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs));
}

/**
 * Format date helper
 */
export function formatDate(date: Date | string): string {
  return new Date(date).toLocaleDateString('fr-FR', {
    year: 'numeric',
    month: 'long',
    day: 'numeric',
  });
}

/**
 * Delay utility
 */
export function sleep(ms: number): Promise<void> {
  return new Promise((resolve) => setTimeout(resolve, ms));
}
```

**Explications:**
- Wrapper fetch avec timeout (30s)
- Retry automatique avec exponential backoff (3 tentatives)
- Gestion centralisée des erreurs
- Support cookies (credentials: 'include') pour tracking sessions
- Types TypeScript pour toutes les réponses

---

### Étape 5: Composants Loading & Error States

**Description:** Créer les composants pour gérer les états de chargement et erreurs.

**Code `components/shared/LoadingSpinner.tsx`:**

```typescript
import { cn } from '@/lib/utils';

interface LoadingSpinnerProps {
  size?: 'sm' | 'md' | 'lg';
  className?: string;
}

export function LoadingSpinner({ size = 'md', className }: LoadingSpinnerProps) {
  const sizeClasses = {
    sm: 'h-4 w-4 border-2',
    md: 'h-8 w-8 border-3',
    lg: 'h-12 w-12 border-4',
  };

  return (
    <div
      className={cn(
        'animate-spin rounded-full border-primary border-t-transparent',
        sizeClasses[size],
        className
      )}
      role="status"
      aria-label="Chargement"
    >
      <span className="sr-only">Chargement...</span>
    </div>
  );
}
```

**Code `app/loading.tsx`:**

```typescript
import { LoadingSpinner } from '@/components/shared/LoadingSpinner';

export default function Loading() {
  return (
    <div className="flex min-h-screen items-center justify-center">
      <div className="text-center">
        <LoadingSpinner size="lg" />
        <p className="mt-4 text-muted-foreground">Chargement...</p>
      </div>
    </div>
  );
}
```

**Code `app/error.tsx`:**

```typescript
'use client';

import { useEffect } from 'react';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';

export default function Error({
  error,
  reset,
}: {
  error: Error & { digest?: string };
  reset: () => void;
}) {
  useEffect(() => {
    // Log l'erreur (pourrait envoyer à un service de monitoring)
    console.error('Error boundary:', error);
  }, [error]);

  return (
    <div className="container flex min-h-screen items-center justify-center py-10">
      <Card className="max-w-md">
        <CardHeader>
          <CardTitle>Une erreur est survenue</CardTitle>
          <CardDescription>
            Désolé, quelque chose s'est mal passé. Veuillez réessayer.
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          {process.env.NODE_ENV === 'development' && (
            <div className="rounded-md bg-muted p-4">
              <p className="text-sm font-mono text-muted-foreground">
                {error.message}
              </p>
            </div>
          )}
          <Button onClick={reset} className="w-full">
            Réessayer
          </Button>
        </CardContent>
      </Card>
    </div>
  );
}
```

**Code `app/not-found.tsx`:**

```typescript
import Link from 'next/link';
import { Button } from '@/components/ui/button';

export default function NotFound() {
  return (
    <div className="container flex min-h-screen flex-col items-center justify-center">
      <h1 className="text-6xl font-bold">404</h1>
      <p className="mt-4 text-xl text-muted-foreground">Page non trouvée</p>
      <Button asChild className="mt-8">
        <Link href="/">Retour à l'accueil</Link>
      </Button>
    </div>
  );
}
```

**Explications:**
- `loading.tsx` : affichage automatique pendant chargement Suspense
- `error.tsx` : Error Boundary pour capturer les erreurs React
- `not-found.tsx` : page 404 personnalisée
- Accessible (sr-only, role="status")

---

### Étape 6: Layout Principal

**Description:** Créer le layout principal avec Header, Footer, et Navigation.

**Code `components/layout/Header.tsx`:**

```typescript
'use client';

import Link from 'next/link';
import { usePathname } from 'next/navigation';
import { Button } from '@/components/ui/button';
import { cn } from '@/lib/utils';
import { Moon, Sun } from 'lucide-react';
import { useTheme } from '@/hooks/useTheme';

const navigation = [
  { name: 'Accueil', href: '/' },
  { name: 'CV', href: '/cv' },
  { name: 'Lettres', href: '/letters' },
  { name: 'Analytics', href: '/analytics' },
];

export function Header() {
  const pathname = usePathname();
  const { theme, toggleTheme } = useTheme();

  return (
    <header className="sticky top-0 z-50 w-full border-b bg-background/95 backdrop-blur supports-[backdrop-filter]:bg-background/60">
      <div className="container flex h-16 items-center justify-between">
        <div className="flex items-center gap-8">
          <Link href="/" className="font-heading text-xl font-bold">
            maicivy
          </Link>

          <nav className="hidden md:flex items-center gap-6">
            {navigation.map((item) => (
              <Link
                key={item.href}
                href={item.href}
                className={cn(
                  'text-sm font-medium transition-colors hover:text-primary',
                  pathname === item.href
                    ? 'text-foreground'
                    : 'text-muted-foreground'
                )}
              >
                {item.name}
              </Link>
            ))}
          </nav>
        </div>

        <div className="flex items-center gap-4">
          <Button
            variant="ghost"
            size="icon"
            onClick={toggleTheme}
            aria-label="Changer de thème"
          >
            {theme === 'dark' ? (
              <Sun className="h-5 w-5" />
            ) : (
              <Moon className="h-5 w-5" />
            )}
          </Button>
        </div>
      </div>
    </header>
  );
}
```

**Code `components/layout/Footer.tsx`:**

```typescript
import Link from 'next/link';
import { Github, Linkedin, Mail } from 'lucide-react';

export function Footer() {
  const currentYear = new Date().getFullYear();

  return (
    <footer className="border-t bg-muted/50">
      <div className="container py-8 md:py-12">
        <div className="grid grid-cols-1 gap-8 md:grid-cols-3">
          <div>
            <h3 className="font-heading text-lg font-semibold">maicivy</h3>
            <p className="mt-2 text-sm text-muted-foreground">
              CV interactif intelligent avec génération de lettres par IA
            </p>
          </div>

          <div>
            <h3 className="font-heading text-lg font-semibold">Navigation</h3>
            <ul className="mt-2 space-y-2 text-sm">
              <li>
                <Link href="/cv" className="text-muted-foreground hover:text-foreground">
                  CV Dynamique
                </Link>
              </li>
              <li>
                <Link href="/letters" className="text-muted-foreground hover:text-foreground">
                  Générateur de Lettres
                </Link>
              </li>
              <li>
                <Link href="/analytics" className="text-muted-foreground hover:text-foreground">
                  Analytics
                </Link>
              </li>
            </ul>
          </div>

          <div>
            <h3 className="font-heading text-lg font-semibold">Contact</h3>
            <div className="mt-2 flex gap-4">
              <a
                href="https://github.com/yourusername"
                target="_blank"
                rel="noopener noreferrer"
                className="text-muted-foreground hover:text-foreground"
              >
                <Github className="h-5 w-5" />
                <span className="sr-only">GitHub</span>
              </a>
              <a
                href="https://linkedin.com/in/yourusername"
                target="_blank"
                rel="noopener noreferrer"
                className="text-muted-foreground hover:text-foreground"
              >
                <Linkedin className="h-5 w-5" />
                <span className="sr-only">LinkedIn</span>
              </a>
              <a
                href="mailto:contact@example.com"
                className="text-muted-foreground hover:text-foreground"
              >
                <Mail className="h-5 w-5" />
                <span className="sr-only">Email</span>
              </a>
            </div>
          </div>
        </div>

        <div className="mt-8 border-t pt-8 text-center text-sm text-muted-foreground">
          <p>&copy; {currentYear} maicivy. Tous droits réservés.</p>
        </div>
      </div>
    </footer>
  );
}
```

**Code `hooks/useTheme.ts`:**

```typescript
'use client';

import { useEffect, useState } from 'react';

type Theme = 'light' | 'dark';

export function useTheme() {
  const [theme, setTheme] = useState<Theme>('light');

  useEffect(() => {
    // Lire le thème stocké ou détecter préférence système
    const stored = localStorage.getItem('theme') as Theme | null;
    const prefersDark = window.matchMedia('(prefers-color-scheme: dark)').matches;

    const initialTheme = stored || (prefersDark ? 'dark' : 'light');
    setTheme(initialTheme);
    document.documentElement.classList.toggle('dark', initialTheme === 'dark');
  }, []);

  const toggleTheme = () => {
    const newTheme: Theme = theme === 'light' ? 'dark' : 'light';
    setTheme(newTheme);
    localStorage.setItem('theme', newTheme);
    document.documentElement.classList.toggle('dark', newTheme === 'dark');
  };

  return { theme, toggleTheme };
}
```

**Code `app/layout.tsx` (mise à jour):**

```typescript
import type { Metadata } from 'next';
import { Inter, Poppins } from 'next/font/google';
import { Header } from '@/components/layout/Header';
import { Footer } from '@/components/layout/Footer';
import './globals.css';

const inter = Inter({
  subsets: ['latin'],
  variable: '--font-inter',
  display: 'swap',
});

const poppins = Poppins({
  subsets: ['latin'],
  weight: ['400', '500', '600', '700'],
  variable: '--font-poppins',
  display: 'swap',
});

export const metadata: Metadata = {
  title: {
    default: 'maicivy - CV Interactif Intelligent',
    template: '%s | maicivy',
  },
  description: 'CV interactif avec génération de lettres de motivation par IA',
  keywords: ['CV', 'portfolio', 'IA', 'développeur', 'full-stack'],
  authors: [{ name: 'Alexi' }],
  openGraph: {
    type: 'website',
    locale: 'fr_FR',
    url: 'https://maicivy.com',
    title: 'maicivy - CV Interactif Intelligent',
    description: 'CV interactif avec génération de lettres de motivation par IA',
  },
};

export default function RootLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <html lang="fr" suppressHydrationWarning>
      <body className={`${inter.variable} ${poppins.variable} font-sans antialiased`}>
        <div className="relative flex min-h-screen flex-col">
          <Header />
          <main className="flex-1">{children}</main>
          <Footer />
        </div>
      </body>
    </html>
  );
}
```

**Explications:**
- Header sticky avec navigation
- Dark mode toggle avec persistance localStorage
- Footer avec liens sociaux
- Layout flex pour footer en bas de page
- Responsive design

---

### Étape 7: Homepage Basique

**Description:** Créer une homepage simple pour tester le setup.

**Code `app/page.tsx`:**

```typescript
import Link from 'next/link';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { FileText, Sparkles, BarChart3 } from 'lucide-react';

export default function HomePage() {
  return (
    <div className="container py-12 md:py-24">
      <div className="mx-auto max-w-4xl text-center">
        <h1 className="font-heading text-4xl font-bold tracking-tight sm:text-5xl md:text-6xl">
          CV Interactif Intelligent
        </h1>
        <p className="mt-6 text-lg text-muted-foreground">
          Découvrez mon parcours professionnel adaptatif et générez des lettres de motivation
          personnalisées grâce à l'intelligence artificielle.
        </p>

        <div className="mt-10 flex flex-wrap justify-center gap-4">
          <Button asChild size="lg">
            <Link href="/cv">Voir mon CV</Link>
          </Button>
          <Button asChild size="lg" variant="outline">
            <Link href="/letters">Générer une lettre</Link>
          </Button>
        </div>
      </div>

      <div className="mx-auto mt-24 grid max-w-5xl gap-8 md:grid-cols-3">
        <Card>
          <CardHeader>
            <FileText className="h-10 w-10 text-primary" />
            <CardTitle>CV Dynamique</CardTitle>
            <CardDescription>
              Un CV qui s'adapte automatiquement selon le contexte : backend, frontend, DevOps...
            </CardDescription>
          </CardHeader>
          <CardContent>
            <Button asChild variant="ghost" className="w-full">
              <Link href="/cv">Explorer</Link>
            </Button>
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <Sparkles className="h-10 w-10 text-primary" />
            <CardTitle>Lettres IA</CardTitle>
            <CardDescription>
              Génération de lettres de motivation et anti-motivation personnalisées par IA.
            </CardDescription>
          </CardHeader>
          <CardContent>
            <Button asChild variant="ghost" className="w-full">
              <Link href="/letters">Essayer</Link>
            </Button>
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <BarChart3 className="h-10 w-10 text-primary" />
            <CardTitle>Analytics Publiques</CardTitle>
            <CardDescription>
              Dashboard temps réel des statistiques de visite et d'utilisation.
            </CardDescription>
          </CardHeader>
          <CardContent>
            <Button asChild variant="ghost" className="w-full">
              <Link href="/analytics">Voir les stats</Link>
            </Button>
          </CardContent>
        </Card>
      </div>
    </div>
  );
}
```

**Explications:**
- Hero section avec CTAs
- Grid de cards présentant les features
- Utilisation des composants shadcn/ui
- Responsive design

---

### Étape 8: Package.json Scripts

**Description:** Configurer les scripts npm utiles.

**Code `package.json` (scripts section):**

```json
{
  "scripts": {
    "dev": "next dev",
    "build": "next build",
    "start": "next start",
    "lint": "next lint",
    "lint:fix": "next lint --fix",
    "type-check": "tsc --noEmit",
    "format": "prettier --write \"**/*.{ts,tsx,md,json}\"",
    "format:check": "prettier --check \"**/*.{ts,tsx,md,json}\""
  }
}
```

**Configuration Prettier `.prettierrc`:**

```json
{
  "semi": true,
  "trailingComma": "es5",
  "singleQuote": true,
  "tabWidth": 2,
  "useTabs": false,
  "printWidth": 100,
  "plugins": ["prettier-plugin-tailwindcss"]
}
```

**Configuration ESLint `.eslintrc.json`:**

```json
{
  "extends": [
    "next/core-web-vitals",
    "plugin:@typescript-eslint/recommended"
  ],
  "rules": {
    "@typescript-eslint/no-unused-vars": "error",
    "@typescript-eslint/no-explicit-any": "warn",
    "react-hooks/rules-of-hooks": "error",
    "react-hooks/exhaustive-deps": "warn"
  }
}
```

---

## 🧪 Tests

### Tests de Vérification

**Test 1: Build Production**

```bash
cd frontend
npm run build
```

**Résultat attendu:**
- Build réussi sans erreurs
- Bundle optimisé généré dans `.next/`

**Test 2: Développement**

```bash
npm run dev
```

**Résultat attendu:**
- Serveur démarre sur http://localhost:3000
- Homepage s'affiche correctement
- Dark mode fonctionne
- Navigation fonctionne

**Test 3: Type Safety**

```bash
npm run type-check
```

**Résultat attendu:**
- Aucune erreur TypeScript

**Test 4: API Health Check**

Créer un test simple pour vérifier la connexion API:

**Code `app/api-test/page.tsx`:**

```typescript
'use client';

import { useEffect, useState } from 'react';
import { healthApi } from '@/lib/api';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { LoadingSpinner } from '@/components/shared/LoadingSpinner';

export default function ApiTestPage() {
  const [status, setStatus] = useState<'loading' | 'success' | 'error'>('loading');
  const [data, setData] = useState<any>(null);

  useEffect(() => {
    healthApi
      .check()
      .then((res) => {
        setData(res);
        setStatus('success');
      })
      .catch((err) => {
        console.error(err);
        setStatus('error');
      });
  }, []);

  return (
    <div className="container py-12">
      <Card>
        <CardHeader>
          <CardTitle>API Health Check</CardTitle>
        </CardHeader>
        <CardContent>
          {status === 'loading' && <LoadingSpinner />}
          {status === 'success' && (
            <div className="text-green-600">
              ✓ API connectée: {JSON.stringify(data)}
            </div>
          )}
          {status === 'error' && (
            <div className="text-red-600">✗ Erreur de connexion API</div>
          )}
        </CardContent>
      </Card>
    </div>
  );
}
```

**Accéder à:** http://localhost:3000/api-test

---

## ⚠️ Points d'Attention

- ⚠️ **Hydration Errors:** Utiliser `suppressHydrationWarning` sur `<html>` pour éviter erreurs dark mode
- ⚠️ **CORS:** Vérifier que le backend autorise `http://localhost:3000` en dev
- ⚠️ **Cookies:** `credentials: 'include'` nécessaire pour recevoir cookies de session du backend
- ⚠️ **Environment Variables:** Ne jamais committer `.env.local`, toujours utiliser `.env.example`
- 💡 **Performance:** Utiliser `next/image` pour toutes les images (optimisation auto)
- 💡 **SEO:** Générer des `metadata` dynamiques dans chaque page
- 💡 **Accessibility:** Tester avec un screen reader, vérifier contraste couleurs
- ⚠️ **Fonts Flash:** `display: 'swap'` évite FOIT mais peut causer layout shift (acceptable)
- 💡 **Bundle Size:** Utiliser dynamic imports pour composants lourds (Framer Motion, Charts)

---

## 📚 Ressources

- [Next.js 14 Documentation](https://nextjs.org/docs)
- [Tailwind CSS Documentation](https://tailwindcss.com/docs)
- [shadcn/ui Documentation](https://ui.shadcn.com)
- [Radix UI Primitives](https://www.radix-ui.com/primitives)
- [Framer Motion Documentation](https://www.framer.com/motion/)
- [TypeScript Handbook](https://www.typescriptlang.org/docs/handbook/intro.html)
- [React Hook Form + Zod](https://react-hook-form.com/get-started#SchemaValidation)

---

## ✅ Checklist de Complétion

- [ ] Next.js 14 installé et configuré (App Router)
- [ ] Tailwind CSS configuré avec thème custom
- [ ] Dark mode fonctionnel avec persistance
- [ ] Fonts Google (Inter, Poppins) chargées
- [ ] Structure de dossiers créée (app/, components/, lib/)
- [ ] API client wrapper implémenté avec retry logic
- [ ] Types TypeScript définis (ApiResponse, ApiError)
- [ ] Composants loading et error states
- [ ] Layout principal (Header, Footer) implémenté
- [ ] shadcn/ui installé avec composants de base
- [ ] Homepage basique fonctionnelle
- [ ] Scripts npm configurés (dev, build, lint)
- [ ] ESLint et Prettier configurés
- [ ] Variables d'environnement `.env.local` créées
- [ ] Build production réussi sans erreurs
- [ ] Test API health check fonctionnel
- [ ] Documentation code (commentaires TSDoc)
- [ ] Review performance (Lighthouse > 90)
- [ ] Review accessibilité (contraste, sr-only)
- [ ] Commit & Push

---

**Dernière mise à jour:** 2025-12-08
**Auteur:** Alexi
