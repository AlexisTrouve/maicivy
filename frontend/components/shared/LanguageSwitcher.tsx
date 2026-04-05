'use client';

import { useRouter } from '@/i18n/navigation';
import { useParams, usePathname } from 'next/navigation';
import { useState, useRef, useEffect } from 'react';

const LANGUAGES = [
  { code: 'fr', label: 'Français', flag: '🇫🇷' },
  { code: 'en', label: 'English',  flag: '🇬🇧' },
  { code: 'de', label: 'Deutsch',  flag: '🇩🇪' },
  { code: 'it', label: 'Italiano', flag: '🇮🇹' },
  { code: 'zh', label: '中文',     flag: '🇨🇳' },
] as const;

export function LanguageSwitcher() {
  const router = useRouter();
  const pathname = usePathname();
  const params = useParams();
  const [open, setOpen] = useState(false);
  const ref = useRef<HTMLDivElement>(null);

  const rawPathname = usePathname(); // ex: '/en', '/fr/cv', '/de/letters'
  const currentLocale = (params.locale as string) || 'fr';
  const current = LANGUAGES.find(l => l.code === currentLocale) ?? LANGUAGES[0];

  // Chemin sans le préfixe locale — ex: '/en/cv' → '/cv', '/fr' → '/'
  const pathWithoutLocale = rawPathname.replace(/^\/(fr|en|de|it|zh)/, '') || '/';

  // Fermer au clic extérieur
  useEffect(() => {
    const handler = (e: MouseEvent) => {
      if (ref.current && !ref.current.contains(e.target as Node)) {
        setOpen(false);
      }
    };
    document.addEventListener('mousedown', handler);
    return () => document.removeEventListener('mousedown', handler);
  }, []);

  const switchTo = (code: string) => {
    setOpen(false);
    router.replace(pathWithoutLocale, { locale: code });
  };

  return (
    <div ref={ref} className="relative">
      {/* Bouton principal — affiche la langue actuelle */}
      <button
        onClick={() => setOpen(o => !o)}
        className="flex items-center gap-1.5 px-3 py-1.5 rounded-md text-sm font-medium
                   bg-transparent hover:bg-accent hover:text-accent-foreground
                   border border-transparent hover:border-border
                   transition-colors"
        aria-haspopup="listbox"
        aria-expanded={open}
      >
        <span>{current.flag}</span>
        <span>{current.code.toUpperCase()}</span>
        <svg className={`w-3 h-3 transition-transform ${open ? 'rotate-180' : ''}`} fill="none" viewBox="0 0 24 24" stroke="currentColor">
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 9l-7 7-7-7" />
        </svg>
      </button>

      {/* Dropdown */}
      {open && (
        <ul
          role="listbox"
          className="absolute right-0 mt-1 w-36 rounded-md border border-border bg-popover shadow-md z-50 py-1 text-sm"
        >
          {LANGUAGES.map(lang => (
            <li key={lang.code}>
              <button
                role="option"
                aria-selected={lang.code === currentLocale}
                onClick={() => switchTo(lang.code)}
                className={`w-full flex items-center gap-2 px-3 py-2 hover:bg-accent hover:text-accent-foreground transition-colors
                  ${lang.code === currentLocale ? 'font-semibold text-primary' : ''}`}
              >
                <span>{lang.flag}</span>
                <span>{lang.label}</span>
              </button>
            </li>
          ))}
        </ul>
      )}
    </div>
  );
}
