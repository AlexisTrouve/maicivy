'use client';

import { useRouter } from '@/i18n/navigation';
import { useParams, usePathname } from 'next/navigation';
import { useState, useRef, useEffect } from 'react';
import { useTranslations } from 'next-intl';

// Chaque langue est nommée DANS ELLE-MÊME (autonyme) : c'est le nom que cherche quelqu'un qui ne
// lit pas la langue courante. Pas de drapeau, ni ici ni dans le menu (cf. le bouton plus bas).
const LANGUAGES = [
  { code: 'fr', label: 'Français' },
  { code: 'en', label: 'English' },
  { code: 'de', label: 'Deutsch' },
  { code: 'it', label: 'Italiano' },
  { code: 'zh', label: '中文' },
] as const;

/**
 * LanguageSwitcher — le sélecteur de langue du site (en-tête et pied de page).
 * `placement` : le pied de page est au bas du document, son menu s'ouvre donc VERS LE HAUT —
 * ouvert vers le bas, il tomberait sous la dernière ligne de la page.
 */
export function LanguageSwitcher({ placement = 'down' }: { placement?: 'down' | 'up' } = {}) {
  const t = useTranslations('nav');
  const router = useRouter();
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
      {/* Bouton principal — affiche la langue actuelle par son CODE seul.
          Pas de drapeau emoji ici : un drapeau est une paire de « regional indicators », et
          Windows n'a aucun glyphe pour eux — Chromium y dessine les deux lettres, si bien que
          « 🇫🇷 FR » s'affichait « FR FR » (deux fois le code, en deux tailles). Le code suffit et
          se lit partout. Verrou : LanguageSwitcher.test.tsx + montagne.spec.ts. */}
      <button
        onClick={() => setOpen(o => !o)}
        // min-h-[44px] : cible tactile minimale (WCAG 2.5.8 renforcé), le sélecteur vit aussi sur téléphone.
        className="flex min-h-[44px] items-center gap-1.5 px-3 py-1.5 rounded-md text-sm font-medium
                   bg-transparent hover:bg-accent hover:text-accent-foreground
                   border border-transparent hover:border-border
                   transition-colors"
        aria-haspopup="listbox"
        aria-expanded={open}
        // Le code seul (« FR ») ne dit rien à un lecteur d'écran : le nom accessible dit ce que fait
        // le bouton et la langue courante, dans son autonyme.
        aria-label={t('language', { current: current.label })}
      >
        <span>{current.code.toUpperCase()}</span>
        <svg className={`w-3 h-3 transition-transform ${open ? 'rotate-180' : ''}`} fill="none" viewBox="0 0 24 24" stroke="currentColor">
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 9l-7 7-7-7" />
        </svg>
      </button>

      {/* Dropdown */}
      {open && (
        <ul
          role="listbox"
          aria-label={t('languageList')}
          className={`absolute right-0 w-36 rounded-md border border-border bg-popover shadow-md z-50 py-1 text-sm ${placement === 'up' ? 'bottom-full mb-1' : 'mt-1'}`}
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
                <span lang={lang.code}>{lang.label}</span>
              </button>
            </li>
          ))}
        </ul>
      )}
    </div>
  );
}
