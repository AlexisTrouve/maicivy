'use client';

import { usePathname } from 'next/navigation';
import { useTranslations } from 'next-intl';
import { BarChart3, Bot, FileText, Inbox, Mail, PenLine, Send, Users } from 'lucide-react';
import { PROSPECT_PATHS, type AdminRole } from './RoleGate';

// Nav de la sidebar admin, avec mise en évidence de l'outil actif. Client (usePathname). Ajouter un
// outil = une entrée ici (clé i18n admin.nav.* + une icône) + la page app/admin/(panel)/<outil>/page.tsx.
//
// POURQUOI une icône par entrée : les libellés sont traduits en cinq langues et l'opératrice navigue en
// anglais dans un panneau dont elle ne lit pas les contenus. Une forme constante d'une langue à
// l'autre se mémorise et se vise, là où un mot doit être relu à chaque fois.
const LINKS = [
  { href: '/admin', key: 'cv', Icon: FileText },
  { href: '/admin/compose', key: 'compose', Icon: PenLine },
  { href: '/admin/messages', key: 'messages', Icon: Send },
  { href: '/admin/letters', key: 'letters', Icon: Mail },
  { href: '/admin/stats', key: 'stats', Icon: BarChart3 },
  { href: '/admin/chat', key: 'chat', Icon: Bot },
  { href: '/admin/mailbox', key: 'mailbox', Icon: Inbox },
  // Clé hors de admin.nav : les libellés des prospecteurs vivent sous admin.prospects (cf. labelOf).
  { href: '/admin/prospects', key: 'prospects', Icon: Users },
];

// Un prospecteur ne voit QUE ses outils (CV, Composer, Agent). Filtrer la nav est du confort : la barrière réelle est
// le garde serveur (403 sur tout le reste de /admin).
export default function AdminNav({ role }: { role: AdminRole }) {
  const pathname = usePathname();
  const t = useTranslations('admin.nav');
  const tp = useTranslations('admin.prospects');
  const labelOf = (key: string) => (key === 'prospects' ? tp('nav') : t(key));
  const links = role === 'prospect' ? LINKS.filter((l) => PROSPECT_PATHS.includes(l.href)) : LINKS;
  return (
    <nav className="space-y-0.5 text-sm" data-testid="admin-nav">
      {links.map((l) => {
        const active = pathname === l.href;
        return (
          <a
            key={l.href}
            href={l.href}
            aria-current={active ? 'page' : undefined}
            // L'entrée active porte une barre dorée à gauche EN PLUS du fond et de la couleur.
            // POURQUOI trois signaux : un simple fond légèrement plus clair disparaît sur une dalle
            // mal calibrée ou en plein jour, et « où suis-je » est l'information qu'on consulte le
            // plus souvent. La barre survit à tout.
            className={`group relative flex items-center gap-2.5 rounded-lg py-2 pl-3.5 pr-3 transition-colors ${
              active
                ? 'bg-gradient-to-r from-gold/[0.14] to-transparent font-medium text-slate-50'
                : 'text-slate-400 hover:bg-slate-800/50 hover:text-slate-200'
            }`}
          >
            {active && (
              <span
                aria-hidden="true"
                className="absolute inset-y-1 left-0 w-[3px] rounded-full bg-gold shadow-[0_0_10px_rgba(217,180,95,0.7)]"
              />
            )}
            <l.Icon
              aria-hidden="true"
              className={`h-4 w-4 shrink-0 ${active ? 'text-gold' : 'text-slate-500 group-hover:text-slate-400'}`}
            />
            {labelOf(l.key)}
          </a>
        );
      })}
    </nav>
  );
}
