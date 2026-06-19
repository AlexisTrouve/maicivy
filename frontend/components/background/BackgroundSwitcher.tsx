'use client';

import { useEffect, useRef, useState } from 'react';
import { useLocale } from 'next-intl';
import { useBackground } from './BackgroundProvider';
import { BACKGROUNDS, BG_NONE, BG_RANDOM } from '@/lib/backgrounds/registry';

/**
 * BackgroundSwitcher — sélecteur de fond, top-right (dans le Header, à côté du LanguageSwitcher).
 *
 * QUOI : dropdown listant 🎲 Aléatoire + chaque plugin + Aucun. Coche la `preference` courante.
 * POURQUOI : donne au visiteur le contrôle du fond ; par défaut chaque visite est aléatoire.
 * COMMENT : lit/écrit la préférence via useBackground(). Style aligné sur LanguageSwitcher.
 *   Masqué sous reduced-motion (aucun fond ne tourne → le contrôle n'aurait aucun effet).
 */

// Libellés inline (3 chaînes) plutôt que dans messages/*.json : les noms de plugins restent
// techniques/non traduits, donc seuls Aléatoire/Aucun/label nécessitent une traduction. Évite
// d'éditer 5 fichiers de locale pour si peu. Fallback 'en' si la locale n'est pas couverte.
const LABELS: Record<string, { label: string; random: string; none: string }> = {
  fr: { label: 'Fond animé', random: 'Aléatoire', none: 'Aucun' },
  en: { label: 'Background', random: 'Random', none: 'None' },
  de: { label: 'Hintergrund', random: 'Zufällig', none: 'Keiner' },
  it: { label: 'Sfondo', random: 'Casuale', none: 'Nessuno' },
  zh: { label: '背景', random: '随机', none: '无' },
};

export function BackgroundSwitcher() {
  const locale = useLocale();
  const L = LABELS[locale] ?? LABELS.en;
  const { preference, setPreference } = useBackground();
  const [open, setOpen] = useState(false);
  const ref = useRef<HTMLDivElement>(null);

  // Fermer au clic extérieur
  useEffect(() => {
    const handler = (e: MouseEvent) => {
      if (ref.current && !ref.current.contains(e.target as Node)) setOpen(false);
    };
    document.addEventListener('mousedown', handler);
    return () => document.removeEventListener('mousedown', handler);
  }, []);

  // Le sélecteur reste TOUJOURS visible (même sous reduced-motion) : c'est le seul moyen pour
  // l'utilisateur de réactiver un fond s'il le souhaite. Le masquer = lui retirer le contrôle.

  // Options : 🎲 Aléatoire, puis chaque plugin (nom technique), puis Aucun.
  const options = [
    { id: BG_RANDOM, label: `🎲 ${L.random}` },
    ...BACKGROUNDS.map((b) => ({ id: b.id, label: b.name })),
    { id: BG_NONE, label: L.none },
  ];
  // Libellé de l'option courante affiché sur le bouton → le contrôle est explicitement un dropdown.
  const current = options.find((o) => o.id === preference) ?? options[0];

  return (
    <div ref={ref} className="relative">
      {/* Bouton principal — icône sparkles + chevron, compact comme les autres contrôles top-right */}
      <button
        onClick={() => setOpen((o) => !o)}
        data-testid="background-switcher"
        className="flex items-center gap-1.5 px-3 py-1.5 rounded-md text-sm font-medium
                   bg-transparent hover:bg-accent hover:text-accent-foreground
                   border border-transparent hover:border-border
                   transition-colors"
        aria-haspopup="listbox"
        aria-expanded={open}
        aria-label={L.label}
        title={L.label}
      >
        <span aria-hidden>✨</span>
        <span className="max-w-[8rem] truncate">{current.label}</span>
        <svg
          className={`w-3 h-3 transition-transform ${open ? 'rotate-180' : ''}`}
          fill="none"
          viewBox="0 0 24 24"
          stroke="currentColor"
        >
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 9l-7 7-7-7" />
        </svg>
      </button>

      {/* Dropdown */}
      {open && (
        <ul
          role="listbox"
          className="absolute right-0 mt-1 w-44 rounded-md border border-border bg-popover shadow-md z-50 py-1 text-sm"
        >
          {options.map((o) => (
            <li key={o.id}>
              <button
                role="option"
                data-testid={`bg-option-${o.id}`}
                aria-selected={o.id === preference}
                onClick={() => {
                  setPreference(o.id);
                  setOpen(false);
                }}
                className={`w-full flex items-center gap-2 px-3 py-2 text-left hover:bg-accent hover:text-accent-foreground transition-colors
                  ${o.id === preference ? 'font-semibold text-primary' : ''}`}
              >
                {o.label}
              </button>
            </li>
          ))}
        </ul>
      )}
    </div>
  );
}
