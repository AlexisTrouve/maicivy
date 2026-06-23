'use client';

import { usePathname } from 'next/navigation';
import { useTranslations } from 'next-intl';

// Nav de la sidebar admin, avec mise en évidence de l'outil actif. Client (usePathname). Ajouter un
// outil = une entrée ici (clé i18n admin.nav.*) + la page app/admin/(panel)/<outil>/page.tsx.
const LINKS = [
  { href: '/admin', key: 'cv' },
  { href: '/admin/messages', key: 'messages' },
  { href: '/admin/letters', key: 'letters' },
  { href: '/admin/stats', key: 'stats' },
  { href: '/admin/chat', key: 'chat' },
];

export default function AdminNav() {
  const pathname = usePathname();
  const t = useTranslations('admin.nav');
  return (
    <nav className="space-y-1 text-sm" data-testid="admin-nav">
      {LINKS.map((l) => {
        const active = pathname === l.href;
        return (
          <a
            key={l.href}
            href={l.href}
            aria-current={active ? 'page' : undefined}
            className={`block rounded-md px-3 py-2 ${
              active ? 'bg-slate-800 text-white' : 'text-slate-400 hover:bg-slate-800/50'
            }`}
          >
            {t(l.key)}
          </a>
        );
      })}
    </nav>
  );
}
