'use client';

import { useRouter } from 'next/navigation';
import { useLocale } from 'next-intl';

// Sélecteur de langue de l'admin. Pose le cookie NEXT_LOCALE (le même que le site public) puis
// router.refresh() → le layout serveur relit le cookie et re-rend tout le panneau dans la langue
// choisie. Présent sur le login ET dans la sidebar.
const LANGS = [
  { code: 'fr', label: 'FR' },
  { code: 'en', label: 'EN' },
  { code: 'de', label: 'DE' },
  { code: 'it', label: 'IT' },
  { code: 'zh', label: '中文' },
];

export default function AdminLangSwitcher() {
  const router = useRouter();
  const locale = useLocale();

  function setLang(code: string) {
    document.cookie = `NEXT_LOCALE=${code}; path=/; max-age=31536000; samesite=lax`;
    router.refresh();
  }

  return (
    <select
      value={locale}
      onChange={(e) => setLang(e.target.value)}
      data-testid="admin-lang"
      aria-label="Language"
      className="cursor-pointer rounded-lg border border-slate-800 bg-slate-950/70 px-2 py-1.5 text-xs text-slate-300 outline-none transition-colors hover:border-slate-700 focus:border-gold/50"
    >
      {LANGS.map((l) => (
        <option key={l.code} value={l.code}>
          {l.label}
        </option>
      ))}
    </select>
  );
}
