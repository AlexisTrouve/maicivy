# Plan d'Implémentation i18n (FR/EN)

**Status:** TOUTES LES PHASES COMPLÉTÉES ✅
**Priorité:** TERMINÉ
**Estimation:** 10h (10h complétées)
**Date:** 2025-12-30
**Dernière mise à jour:** 2025-12-30 11:30

**RÉSUMÉ COMPLET:** Voir `I18N_IMPLEMENTATION_SUMMARY.md` à la racine du projet

---

## Objectif

Ajouter le support multilingue (Français/Anglais) au frontend maicivy avec:
- Détection automatique de la langue du navigateur
- Switch de langue manuel
- URLs localisées (`/fr/cv`, `/en/cv`)
- ~250 clés de traduction

---

## Stack Technique

- **Framework:** `next-intl` (recommandé pour Next.js 14 App Router)
- **Langues:** `fr` (défaut), `en`
- **Stockage préférence:** Cookie `NEXT_LOCALE`

---

## Phases d'Implémentation

### PHASE 1: Setup next-intl
**Status:** [✅] COMPLÉTÉE
**Fichiers à créer/modifier:**

#### 1.1 Installer la dépendance
```bash
cd frontend && npm install next-intl
```

#### 1.2 Créer `frontend/i18n/config.ts`
```typescript
export const locales = ['fr', 'en'] as const;
export const defaultLocale = 'fr' as const;
export type Locale = (typeof locales)[number];
```

#### 1.3 Créer `frontend/i18n/request.ts`
```typescript
import { getRequestConfig } from 'next-intl/server';
import { locales, defaultLocale } from './config';

export default getRequestConfig(async ({ locale }) => {
  if (!locales.includes(locale as any)) {
    locale = defaultLocale;
  }

  return {
    messages: (await import(`../messages/${locale}.json`)).default
  };
});
```

#### 1.4 Créer `frontend/middleware.ts`
```typescript
import createMiddleware from 'next-intl/middleware';
import { locales, defaultLocale } from './i18n/config';

export default createMiddleware({
  locales,
  defaultLocale,
  localePrefix: 'as-needed'
});

export const config = {
  matcher: ['/((?!api|_next|_vercel|.*\\..*).*)']
};
```

#### 1.5 Modifier `frontend/next.config.js`
Ajouter:
```javascript
const withNextIntl = require('next-intl/plugin')('./i18n/request.ts');

module.exports = withNextIntl({
  // existing config...
});
```

---

### PHASE 2: Créer les fichiers de traduction
**Status:** [✅] COMPLÉTÉE
**Fichiers à créer:**

#### 2.1 Créer `frontend/messages/fr.json`
```json
{
  "metadata": {
    "title": "maicivy - CV Interactif Intelligent",
    "description": "CV interactif avec génération de lettres de motivation par IA"
  },
  "common": {
    "loading": "Chargement...",
    "error": "Erreur",
    "retry": "Réessayer",
    "close": "Fermer",
    "backToHome": "Retour à l'accueil",
    "download": "Télécharger",
    "copy": "Copier",
    "copied": "Copié !",
    "viewMore": "Voir plus",
    "toggleTheme": "Changer de thème"
  },
  "nav": {
    "home": "Accueil",
    "cv": "CV",
    "letters": "Lettres",
    "analytics": "Analytics",
    "architecture": "Architecture"
  },
  "home": {
    "title": "CV Interactif Intelligent",
    "subtitle": "Découvrez mon parcours professionnel adaptatif et générez des lettres de motivation personnalisées grâce à l'intelligence artificielle.",
    "cta": {
      "viewCV": "Voir mon CV",
      "generateLetter": "Générer une lettre"
    },
    "features": {
      "cv": {
        "title": "CV Dynamique",
        "description": "Un CV qui s'adapte automatiquement selon le contexte : backend, frontend, DevOps...",
        "action": "Explorer"
      },
      "letters": {
        "title": "Lettres IA",
        "description": "Génération de lettres de motivation et anti-motivation personnalisées par IA.",
        "action": "Essayer"
      },
      "analytics": {
        "title": "Analytics Publiques",
        "description": "Dashboard temps réel des statistiques de visite et d'utilisation.",
        "action": "Voir les stats"
      },
      "architecture": {
        "title": "Architecture",
        "description": "Stack technique complète : Go, Next.js, PostgreSQL, Redis, Claude AI...",
        "action": "Découvrir"
      }
    }
  },
  "cv": {
    "title": "CV Dynamique",
    "adaptedTo": "CV adapté au profil :",
    "selectTheme": "Sélectionner un thème",
    "sections": {
      "experiences": "Expériences Professionnelles",
      "skills": "Compétences",
      "projects": "Projets"
    },
    "duration": {
      "present": "Présent",
      "year": "an",
      "years": "ans",
      "month": "mois",
      "months": "mois",
      "unknownDate": "Date inconnue",
      "invalidDate": "Date invalide",
      "unknownDuration": "Durée inconnue"
    },
    "skills": {
      "all": "Toutes",
      "tooltip": "{name} - Niveau {level}/5 - {years} ans d'expérience",
      "legend": "La taille représente le niveau de compétence et la pertinence par rapport au thème sélectionné"
    },
    "projects": {
      "featured": "Projet Vedette",
      "code": "Code",
      "demo": "Demo",
      "relevance": "Pertinence"
    },
    "export": {
      "downloading": "Export en cours...",
      "download": "Télécharger PDF",
      "error": "Échec de l'export PDF"
    },
    "themes": {
      "backend": "Backend",
      "fullstack": "Full-Stack",
      "devops": "DevOps",
      "cpp": "C++",
      "artistic": "Artistique"
    }
  },
  "letters": {
    "title": "Générateur de Lettres par IA",
    "subtitle": "Générez instantanément une lettre de motivation professionnelle et sa version humoristique \"anti-motivation\"",
    "form": {
      "companyName": "Nom de l'entreprise",
      "placeholder": "Ex: Google, Microsoft, Startup Innovante...",
      "submit": "Générer les lettres",
      "generating": "Génération en cours...",
      "info": "L'IA va générer deux lettres : une motivation professionnelle et une anti-motivation humoristique. La génération prend environ 30-60 secondes."
    },
    "progress": {
      "analyzing": "Analyse de l'entreprise...",
      "writingMotivation": "Rédaction de la lettre de motivation...",
      "writingAnti": "Création de l'anti-motivation...",
      "finalizing": "Finalisation..."
    },
    "preview": {
      "title": "Lettres pour {company}",
      "sector": "Secteur:",
      "motivation": "Lettre de Motivation",
      "antiMotivation": "Lettre d'Anti-Motivation",
      "downloadDual": "PDF Dual",
      "newGeneration": "Nouvelle génération",
      "copyText": "Copier le texte",
      "downloadPDF": "Télécharger PDF",
      "warning": "Note: La lettre d'anti-motivation est générée à titre humoristique et créatif. Elle ne doit PAS être envoyée à l'entreprise. Utilisez uniquement la lettre de motivation professionnelle pour vos candidatures réelles.",
      "downloadError": "Erreur lors du téléchargement du PDF"
    },
    "accessGate": {
      "title": "Fonctionnalité Premium",
      "description": "Le générateur de lettres par IA est accessible à partir de la 3ème visite.",
      "progress": "Votre progression",
      "visits": "{count} / 3 visites",
      "remaining": "Encore {count} visite(s) avant déblocage",
      "encourage": "Revenez explorer mon CV pour débloquer cette fonctionnalité !",
      "unlockTitle": "Vous débloquerez :",
      "features": {
        "motivation": "Génération de lettre de motivation personnalisée",
        "anti": "Lettre d'anti-motivation humoristique unique",
        "pdf": "Export PDF professionnel des deux lettres",
        "analysis": "Analyse IA de l'entreprise cible"
      },
      "exploreCV": "Explorer mon CV"
    },
    "errors": {
      "accessDenied": "Accès refusé. Vous devez effectuer 3 visites pour débloquer cette fonctionnalité.",
      "rateLimit": "Limite atteinte. Réessayez dans quelques minutes.",
      "serverError": "Erreur serveur. Nos IA prennent une pause café. Réessayez dans quelques instants.",
      "generic": "Une erreur est survenue lors de la génération."
    }
  },
  "analytics": {
    "title": "Analytics Dashboard",
    "subtitle": "Statistiques publiques en temps réel",
    "description": "Découvrez comment les visiteurs interagissent avec le site",
    "privacy": "Ce dashboard est public et mis à jour en temps réel. Les données sont collectées de manière anonyme dans le respect de la vie privée des visiteurs.",
    "widgets": {
      "visitors": {
        "title": "Visiteurs Actuels",
        "online": "En ligne",
        "offline": "Déconnecté",
        "person": "personne",
        "people": "personnes",
        "rightNow": "en ce moment",
        "realtime": "Mise à jour en temps réel via WebSocket"
      },
      "themes": {
        "title": "Top Thèmes CV",
        "views": "vues",
        "update": "Mise à jour toutes les 30 secondes"
      },
      "letters": {
        "title": "Lettres IA",
        "total": "lettres générées au total",
        "noData": "Aucune donnée disponible",
        "evolution": "Évolution sur {period}"
      },
      "heatmap": {
        "title": "Heatmap des Interactions",
        "low": "Faible",
        "high": "Fort",
        "noData": "Aucune donnée d'interaction disponible",
        "interactions": "{count} interactions",
        "based": "Basé sur {count} points d'interaction"
      },
      "stats": {
        "visitors": "Visiteurs",
        "pageViews": "Pages Vues",
        "letters": "Lettres",
        "conversion": "Conversion"
      }
    },
    "periods": {
      "today": "Aujourd'hui",
      "last7days": "7 derniers jours",
      "last30days": "30 derniers jours",
      "all": "Tout",
      "day": "Jour",
      "week": "Semaine",
      "month": "Mois",
      "select": "Sélectionner une période",
      "label": "Période:"
    }
  },
  "architecture": {
    "title": "Architecture & Stack Technique",
    "subtitle": "Découvrez les technologies et l'architecture derrière maicivy",
    "overview": "Vue d'Ensemble",
    "fullStack": "Stack Technique Complète",
    "scraper": "Company Scraper Multi-Sources",
    "features": "Fonctionnalités Techniques",
    "security": "Sécurité (OWASP Top 10)",
    "codeExample": "Exemple: Scraper Multi-Sources",
    "sourceCode": "Code Source",
    "sourceCodeDesc": "Explorez le code complet sur GitHub - 100+ fichiers Go, 60+ composants React, 882 tests",
    "viewOnGithub": "Voir sur GitHub",
    "backToCV": "Retour au CV",
    "layers": {
      "client": "CLIENT",
      "backend": "API BACKEND",
      "services": "SERVICES",
      "database": "DATABASE",
      "cache": "CACHE",
      "aiProviders": "AI PROVIDERS"
    },
    "stacks": {
      "backend": {
        "title": "Backend (Go)",
        "description": "API haute performance avec Go et Fiber"
      },
      "frontend": {
        "title": "Frontend (Next.js)",
        "description": "Interface moderne avec React et TypeScript"
      },
      "ai": {
        "title": "Intelligence Artificielle",
        "description": "Génération de contenu avec Claude et GPT-4"
      },
      "infra": {
        "title": "Infrastructure",
        "description": "Docker, bases de données et monitoring"
      }
    },
    "metrics": {
      "goFiles": "Fichiers Go (Backend)",
      "reactComponents": "Composants React",
      "backendTests": "Tests Backend",
      "frontendTests": "Tests Frontend",
      "passingTests": "Tests Passants",
      "apiEndpoints": "Endpoints API",
      "documentation": "Documentation",
      "implementationGuides": "Guides Implémentation"
    },
    "securityItems": {
      "injection": {
        "title": "Injection Prevention",
        "description": "GORM ORM, requêtes paramétrées"
      },
      "xss": {
        "title": "XSS Protection",
        "description": "Input sanitization, bluemonday"
      },
      "rateLimit": {
        "title": "Rate Limiting",
        "description": "Redis sliding window"
      },
      "headers": {
        "title": "Security Headers",
        "description": "CSP, X-Frame-Options, HSTS"
      },
      "gdpr": {
        "title": "GDPR Compliance",
        "description": "IP hashing SHA256, pas de PII"
      },
      "cookies": {
        "title": "Secure Cookies",
        "description": "HttpOnly, Secure, SameSite"
      }
    }
  },
  "errors": {
    "generic": "Une erreur est survenue",
    "tryAgain": "Désolé, quelque chose s'est mal passé. Veuillez réessayer.",
    "notFound": "Page non trouvée",
    "serverError": "Erreur serveur"
  },
  "validation": {
    "minLength": "Doit contenir au moins {min} caractères",
    "maxLength": "Ne peut pas dépasser {max} caractères",
    "invalidChars": "Caractères invalides détectés",
    "required": "Ce champ est requis",
    "invalidEmail": "Adresse email invalide",
    "invalidUrl": "URL invalide"
  },
  "footer": {
    "description": "CV interactif intelligent avec génération de lettres par IA",
    "navigation": "Navigation",
    "contact": "Contact",
    "rights": "Tous droits réservés."
  }
}
```

#### 2.2 Créer `frontend/messages/en.json`
```json
{
  "metadata": {
    "title": "maicivy - Interactive Intelligent CV",
    "description": "Interactive CV with AI-powered cover letter generation"
  },
  "common": {
    "loading": "Loading...",
    "error": "Error",
    "retry": "Retry",
    "close": "Close",
    "backToHome": "Back to Home",
    "download": "Download",
    "copy": "Copy",
    "copied": "Copied!",
    "viewMore": "View more",
    "toggleTheme": "Toggle theme"
  },
  "nav": {
    "home": "Home",
    "cv": "CV",
    "letters": "Letters",
    "analytics": "Analytics",
    "architecture": "Architecture"
  },
  "home": {
    "title": "Interactive Intelligent CV",
    "subtitle": "Discover my adaptive professional journey and generate personalized cover letters powered by artificial intelligence.",
    "cta": {
      "viewCV": "View my CV",
      "generateLetter": "Generate a letter"
    },
    "features": {
      "cv": {
        "title": "Dynamic CV",
        "description": "A CV that automatically adapts to context: backend, frontend, DevOps...",
        "action": "Explore"
      },
      "letters": {
        "title": "AI Letters",
        "description": "Generation of personalized motivation and anti-motivation letters by AI.",
        "action": "Try it"
      },
      "analytics": {
        "title": "Public Analytics",
        "description": "Real-time dashboard of visit and usage statistics.",
        "action": "View stats"
      },
      "architecture": {
        "title": "Architecture",
        "description": "Complete tech stack: Go, Next.js, PostgreSQL, Redis, Claude AI...",
        "action": "Discover"
      }
    }
  },
  "cv": {
    "title": "Dynamic CV",
    "adaptedTo": "CV adapted to profile:",
    "selectTheme": "Select a theme",
    "sections": {
      "experiences": "Professional Experiences",
      "skills": "Skills",
      "projects": "Projects"
    },
    "duration": {
      "present": "Present",
      "year": "year",
      "years": "years",
      "month": "month",
      "months": "months",
      "unknownDate": "Unknown date",
      "invalidDate": "Invalid date",
      "unknownDuration": "Unknown duration"
    },
    "skills": {
      "all": "All",
      "tooltip": "{name} - Level {level}/5 - {years} years of experience",
      "legend": "Size represents skill level and relevance to selected theme"
    },
    "projects": {
      "featured": "Featured Project",
      "code": "Code",
      "demo": "Demo",
      "relevance": "Relevance"
    },
    "export": {
      "downloading": "Exporting...",
      "download": "Download PDF",
      "error": "PDF export failed"
    },
    "themes": {
      "backend": "Backend",
      "fullstack": "Full-Stack",
      "devops": "DevOps",
      "cpp": "C++",
      "artistic": "Artistic"
    }
  },
  "letters": {
    "title": "AI Letter Generator",
    "subtitle": "Instantly generate a professional cover letter and its humorous \"anti-motivation\" version",
    "form": {
      "companyName": "Company name",
      "placeholder": "E.g.: Google, Microsoft, Innovative Startup...",
      "submit": "Generate letters",
      "generating": "Generating...",
      "info": "The AI will generate two letters: a professional motivation letter and a humorous anti-motivation letter. Generation takes approximately 30-60 seconds."
    },
    "progress": {
      "analyzing": "Analyzing the company...",
      "writingMotivation": "Writing the cover letter...",
      "writingAnti": "Creating the anti-motivation...",
      "finalizing": "Finalizing..."
    },
    "preview": {
      "title": "Letters for {company}",
      "sector": "Sector:",
      "motivation": "Cover Letter",
      "antiMotivation": "Anti-Motivation Letter",
      "downloadDual": "Dual PDF",
      "newGeneration": "New generation",
      "copyText": "Copy text",
      "downloadPDF": "Download PDF",
      "warning": "Note: The anti-motivation letter is generated for humorous and creative purposes. It should NOT be sent to the company. Only use the professional cover letter for real applications.",
      "downloadError": "Error downloading PDF"
    },
    "accessGate": {
      "title": "Premium Feature",
      "description": "The AI letter generator is accessible from your 3rd visit.",
      "progress": "Your progress",
      "visits": "{count} / 3 visits",
      "remaining": "{count} more visit(s) before unlocking",
      "encourage": "Come back to explore my CV to unlock this feature!",
      "unlockTitle": "You will unlock:",
      "features": {
        "motivation": "Personalized cover letter generation",
        "anti": "Unique humorous anti-motivation letter",
        "pdf": "Professional PDF export of both letters",
        "analysis": "AI analysis of target company"
      },
      "exploreCV": "Explore my CV"
    },
    "errors": {
      "accessDenied": "Access denied. You must make 3 visits to unlock this feature.",
      "rateLimit": "Rate limit reached. Please try again in a few minutes.",
      "serverError": "Server error. Our AIs are taking a coffee break. Please try again shortly.",
      "generic": "An error occurred during generation."
    }
  },
  "analytics": {
    "title": "Analytics Dashboard",
    "subtitle": "Real-time public statistics",
    "description": "Discover how visitors interact with the site",
    "privacy": "This dashboard is public and updated in real-time. Data is collected anonymously in respect of visitor privacy.",
    "widgets": {
      "visitors": {
        "title": "Current Visitors",
        "online": "Online",
        "offline": "Disconnected",
        "person": "person",
        "people": "people",
        "rightNow": "right now",
        "realtime": "Real-time updates via WebSocket"
      },
      "themes": {
        "title": "Top CV Themes",
        "views": "views",
        "update": "Updates every 30 seconds"
      },
      "letters": {
        "title": "AI Letters",
        "total": "letters generated in total",
        "noData": "No data available",
        "evolution": "Evolution over {period}"
      },
      "heatmap": {
        "title": "Interaction Heatmap",
        "low": "Low",
        "high": "High",
        "noData": "No interaction data available",
        "interactions": "{count} interactions",
        "based": "Based on {count} interaction points"
      },
      "stats": {
        "visitors": "Visitors",
        "pageViews": "Page Views",
        "letters": "Letters",
        "conversion": "Conversion"
      }
    },
    "periods": {
      "today": "Today",
      "last7days": "Last 7 days",
      "last30days": "Last 30 days",
      "all": "All",
      "day": "Day",
      "week": "Week",
      "month": "Month",
      "select": "Select a period",
      "label": "Period:"
    }
  },
  "architecture": {
    "title": "Architecture & Tech Stack",
    "subtitle": "Discover the technologies and architecture behind maicivy",
    "overview": "Overview",
    "fullStack": "Complete Tech Stack",
    "scraper": "Multi-Source Company Scraper",
    "features": "Technical Features",
    "security": "Security (OWASP Top 10)",
    "codeExample": "Example: Multi-Source Scraper",
    "sourceCode": "Source Code",
    "sourceCodeDesc": "Explore the complete code on GitHub - 100+ Go files, 60+ React components, 882 tests",
    "viewOnGithub": "View on GitHub",
    "backToCV": "Back to CV",
    "layers": {
      "client": "CLIENT",
      "backend": "API BACKEND",
      "services": "SERVICES",
      "database": "DATABASE",
      "cache": "CACHE",
      "aiProviders": "AI PROVIDERS"
    },
    "stacks": {
      "backend": {
        "title": "Backend (Go)",
        "description": "High-performance API with Go and Fiber"
      },
      "frontend": {
        "title": "Frontend (Next.js)",
        "description": "Modern interface with React and TypeScript"
      },
      "ai": {
        "title": "Artificial Intelligence",
        "description": "Content generation with Claude and GPT-4"
      },
      "infra": {
        "title": "Infrastructure",
        "description": "Docker, databases and monitoring"
      }
    },
    "metrics": {
      "goFiles": "Go Files (Backend)",
      "reactComponents": "React Components",
      "backendTests": "Backend Tests",
      "frontendTests": "Frontend Tests",
      "passingTests": "Passing Tests",
      "apiEndpoints": "API Endpoints",
      "documentation": "Documentation",
      "implementationGuides": "Implementation Guides"
    },
    "securityItems": {
      "injection": {
        "title": "Injection Prevention",
        "description": "GORM ORM, parameterized queries"
      },
      "xss": {
        "title": "XSS Protection",
        "description": "Input sanitization, bluemonday"
      },
      "rateLimit": {
        "title": "Rate Limiting",
        "description": "Redis sliding window"
      },
      "headers": {
        "title": "Security Headers",
        "description": "CSP, X-Frame-Options, HSTS"
      },
      "gdpr": {
        "title": "GDPR Compliance",
        "description": "IP hashing SHA256, no PII"
      },
      "cookies": {
        "title": "Secure Cookies",
        "description": "HttpOnly, Secure, SameSite"
      }
    }
  },
  "errors": {
    "generic": "An error occurred",
    "tryAgain": "Sorry, something went wrong. Please try again.",
    "notFound": "Page not found",
    "serverError": "Server error"
  },
  "validation": {
    "minLength": "Must contain at least {min} characters",
    "maxLength": "Cannot exceed {max} characters",
    "invalidChars": "Invalid characters detected",
    "required": "This field is required",
    "invalidEmail": "Invalid email address",
    "invalidUrl": "Invalid URL"
  },
  "footer": {
    "description": "Interactive intelligent CV with AI-powered letter generation",
    "navigation": "Navigation",
    "contact": "Contact",
    "rights": "All rights reserved."
  }
}
```

---

### PHASE 3: Restructurer app/ avec [locale]
**Status:** [✅] COMPLÉTÉE
**Actions:**

#### 3.1 Déplacer toutes les pages sous [locale]
```
AVANT:
app/
├── page.tsx
├── layout.tsx
├── cv/page.tsx
├── letters/page.tsx
├── analytics/page.tsx
├── architecture/page.tsx
├── error.tsx
├── not-found.tsx
└── loading.tsx

APRÈS:
app/
└── [locale]/
    ├── page.tsx
    ├── layout.tsx
    ├── cv/page.tsx
    ├── letters/page.tsx
    ├── analytics/page.tsx
    ├── architecture/page.tsx
    ├── error.tsx
    ├── not-found.tsx
    └── loading.tsx
```

#### 3.2 Modifier `app/[locale]/layout.tsx`
```typescript
import { NextIntlClientProvider } from 'next-intl';
import { getMessages } from 'next-intl/server';
import { locales } from '@/i18n/config';

export function generateStaticParams() {
  return locales.map((locale) => ({ locale }));
}

export default async function LocaleLayout({
  children,
  params: { locale }
}: {
  children: React.ReactNode;
  params: { locale: string };
}) {
  const messages = await getMessages();

  return (
    <html lang={locale}>
      <body>
        <NextIntlClientProvider messages={messages}>
          {children}
        </NextIntlClientProvider>
      </body>
    </html>
  );
}
```

---

### PHASE 4: Créer le Language Switcher
**Status:** [✅] COMPLÉTÉE
**Fichier:** `components/shared/LanguageSwitcher.tsx`

```typescript
'use client';

import { useLocale } from 'next-intl';
import { useRouter, usePathname } from 'next/navigation';
import { Button } from '@/components/ui/button';

export function LanguageSwitcher() {
  const locale = useLocale();
  const router = useRouter();
  const pathname = usePathname();

  const switchLocale = (newLocale: string) => {
    // Remove current locale from pathname
    const segments = pathname.split('/');
    if (segments[1] === 'fr' || segments[1] === 'en') {
      segments[1] = newLocale;
    } else {
      segments.splice(1, 0, newLocale);
    }
    router.push(segments.join('/'));
  };

  return (
    <Button
      variant="ghost"
      size="sm"
      onClick={() => switchLocale(locale === 'fr' ? 'en' : 'fr')}
      className="gap-2"
    >
      {locale === 'fr' ? (
        <>🇬🇧 EN</>
      ) : (
        <>🇫🇷 FR</>
      )}
    </Button>
  );
}
```

**Intégrer dans Header.tsx** après le theme toggle.

---

### PHASE 5: Migrer les Composants
**Status:** [ ] NOT_STARTED

Pour chaque composant, remplacer le texte hardcodé par `useTranslations()`:

#### Pattern de migration:
```typescript
// AVANT
export function MyComponent() {
  return <h1>Bonjour le monde</h1>;
}

// APRÈS
import { useTranslations } from 'next-intl';

export function MyComponent() {
  const t = useTranslations('mySection');
  return <h1>{t('greeting')}</h1>;
}
```

#### Composants à migrer (dans l'ordre):

**Layout:**
- [ ] `components/layout/Header.tsx`
- [ ] `components/layout/Footer.tsx`

**Pages:**
- [ ] `app/[locale]/page.tsx` (homepage)
- [ ] `app/[locale]/cv/page.tsx`
- [ ] `app/[locale]/letters/page.tsx`
- [ ] `app/[locale]/analytics/page.tsx`
- [ ] `app/[locale]/architecture/page.tsx`
- [ ] `app/[locale]/error.tsx`
- [ ] `app/[locale]/not-found.tsx`
- [ ] `app/[locale]/loading.tsx`

**CV Components:**
- [ ] `components/cv/CVThemeSelector.tsx`
- [ ] `components/cv/ExperienceTimeline.tsx`
- [ ] `components/cv/SkillsCloud.tsx`
- [ ] `components/cv/ProjectsGrid.tsx`
- [ ] `components/cv/ExportPDFButton.tsx`

**Letters Components:**
- [ ] `components/letters/LetterGenerator.tsx`
- [ ] `components/letters/LetterPreview.tsx`
- [ ] `components/letters/AccessGate.tsx`

**Analytics Components:**
- [ ] `components/analytics/RealtimeVisitors.tsx`
- [ ] `components/analytics/ThemeStats.tsx`
- [ ] `components/analytics/LettersGenerated.tsx`
- [ ] `components/analytics/Heatmap.tsx`
- [ ] `components/analytics/DateFilter.tsx`
- [ ] `components/analytics/StatsOverview.tsx`

**Shared:**
- [ ] `components/shared/LoadingSpinner.tsx`

**Validations:**
- [ ] `lib/validations.ts`

---

### PHASE 6: Gestion des dates avec locale
**Status:** [ ] NOT_STARTED

Modifier `ExperienceTimeline.tsx` et autres composants utilisant `date-fns`:

```typescript
import { useLocale } from 'next-intl';
import { fr, enUS } from 'date-fns/locale';

const localeMap = {
  fr: fr,
  en: enUS
};

// Dans le composant:
const locale = useLocale();
const dateLocale = localeMap[locale as keyof typeof localeMap];

// Utiliser dateLocale dans format()
format(date, 'MMMM yyyy', { locale: dateLocale });
```

---

### PHASE 7: Tests
**Status:** [ ] NOT_STARTED

#### 7.1 Vérifier que toutes les pages chargent en FR et EN
```bash
# FR (défaut)
curl http://localhost:3000/
curl http://localhost:3000/cv
curl http://localhost:3000/letters

# EN
curl http://localhost:3000/en
curl http://localhost:3000/en/cv
curl http://localhost:3000/en/letters
```

#### 7.2 Vérifier le switch de langue
- Cliquer sur le switcher doit changer la langue
- L'URL doit refléter la locale
- Le cookie NEXT_LOCALE doit être set

#### 7.3 Vérifier les traductions
- Pas de texte français visible en mode EN
- Pas de clés de traduction visibles (ex: `nav.home`)

---

## Checklist Finale

- [ ] Phase 1: Setup next-intl complet
- [ ] Phase 2: Fichiers fr.json et en.json créés
- [ ] Phase 3: Structure [locale] en place
- [ ] Phase 4: LanguageSwitcher fonctionnel
- [ ] Phase 5: Tous les composants migrés
- [ ] Phase 6: Dates localisées
- [ ] Phase 7: Tests passés

---

## Notes pour l'Agent

1. **Ne pas traduire:**
   - Noms de technologies (Go, Next.js, etc.)
   - Nom de marque "maicivy"
   - Noms propres
   - Contenu dynamique de l'API

2. **Variables dans les traductions:**
   Utiliser `{variable}` pour les interpolations:
   ```json
   "greeting": "Hello {name}!"
   ```
   ```typescript
   t('greeting', { name: 'John' })
   ```

3. **Pluralisation:**
   ```json
   "items": "{count, plural, =0 {No items} =1 {1 item} other {# items}}"
   ```

4. **Fallback:**
   Si une clé manque en EN, next-intl affichera la clé. Toujours vérifier les deux langues.

---

**Dernière mise à jour:** 2025-12-30
