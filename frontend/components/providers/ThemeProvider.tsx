'use client';

import { createContext, useContext, useEffect, useState } from 'react';

type Theme = 'light' | 'dark';

interface ThemeContextType {
  theme: Theme;
  setTheme: (theme: Theme) => void;
  toggleTheme: () => void;
}

const ThemeContext = createContext<ThemeContextType | undefined>(undefined);

export function useTheme() {
  const context = useContext(ThemeContext);
  if (!context) {
    throw new Error('useTheme must be used within ThemeProvider');
  }
  return context;
}

interface ThemeProviderProps {
  children: React.ReactNode;
  defaultTheme?: Theme;
}

export function ThemeProvider({ children, defaultTheme = 'dark' }: ThemeProviderProps) {
  const [theme, setThemeState] = useState<Theme>(defaultTheme);
  const [mounted, setMounted] = useState(false);

  useEffect(() => {
    setMounted(true);
    // QUOI : détermine le thème initial au montage. Le choix explicite de l'utilisateur (localStorage,
    // posé par le toggle) gagne ; à défaut, on tombe sur `dark`.
    // POURQUOI : dark est le défaut produit (esthétique dark-first). On n'écoute PLUS
    // `prefers-color-scheme` — sinon un visiteur en OS clair verrait le site en clair et le « dark par
    // défaut » ne s'afficherait jamais pour lui. localStorage reste prioritaire → on ne piétine pas un
    // choix déjà fait.
    // COMMENT : stored (null si jamais togglé) || 'dark'.
    const stored = localStorage.getItem('theme') as Theme | null;
    const initialTheme = stored || 'dark';

    if (initialTheme !== defaultTheme) {
      setThemeState(initialTheme);
    }

    // Update document class on mount
    if (initialTheme === 'dark') {
      document.documentElement.classList.add('dark');
    } else {
      document.documentElement.classList.remove('dark');
    }
  }, [defaultTheme]);

  const setTheme = (newTheme: Theme) => {
    setThemeState(newTheme);
    localStorage.setItem('theme', newTheme);

    // Update document class
    if (newTheme === 'dark') {
      document.documentElement.classList.add('dark');
    } else {
      document.documentElement.classList.remove('dark');
    }
  };

  const toggleTheme = () => {
    setTheme(theme === 'light' ? 'dark' : 'light');
  };

  return (
    <ThemeContext.Provider value={{ theme, setTheme, toggleTheme }}>
      {children}
    </ThemeContext.Provider>
  );
}

// Script anti-flash à injecter dans le <head> : pose la classe `dark` avant le paint.
// ⚠️ DEAD CODE actuellement : le layout (app/[locale]/layout.tsx) a son propre <Script> inline et
// n'importe PAS ce composant. On garde la logique alignée sur ThemeProvider (dark par défaut,
// localStorage prioritaire, pas de prefers-color-scheme) pour éviter un piège si on le rebranche un jour.
export function ThemeScript() {
  const themeScript = `
    (function() {
      try {
        const theme = localStorage.getItem('theme');
        const initialTheme = theme || 'dark';

        if (initialTheme === 'dark') {
          document.documentElement.classList.add('dark');
        } else {
          document.documentElement.classList.remove('dark');
        }
      } catch (e) {}
    })();
  `;

  return (
    <script
      dangerouslySetInnerHTML={{ __html: themeScript }}
      suppressHydrationWarning
    />
  );
}
