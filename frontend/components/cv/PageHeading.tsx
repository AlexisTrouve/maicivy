import type { ReactNode } from 'react';

// ============================================================================
// En-têtes des pages /cv et /gitstats — le vocabulaire de la feuille du dossier vivant.
// ----------------------------------------------------------------------------
// QUOI : `PageHeading` = surtitre en petites capitales espacées (comme la signature du pied de
// page), titre h1 en serif ivoire, chapeau en gris doux, puis les actions de la page. Un filet
// sépare l'en-tête du contenu, comme `.dosHead` sépare l'en-tête du dossier de son corps.
// `SectionHeading` = titre h2 d'une section, avec la même ligne de base.
//
// POURQUOI un composant partagé : les deux pages partagent déjà GitStatsPanel et le lien de l'une
// vers l'autre ; leurs en-têtes doivent se lire comme deux feuilles du même dossier. Composant
// serveur (aucun hook) : utilisable depuis les deux pages sans franchir de frontière client.
// ============================================================================

export function PageHeading({
  eyebrow,
  title,
  lead,
  children,
}: {
  eyebrow: string;
  title: string;
  lead: string;
  children?: ReactNode;
}) {
  return (
    <header className="border-b border-border pb-10">
      <p className="text-[11px] font-medium uppercase tracking-[0.24em] text-muted-foreground">{eyebrow}</p>
      <h1 className="mt-4 text-4xl font-medium leading-tight text-foreground md:text-5xl">{title}</h1>
      <p className="mt-4 max-w-2xl text-base leading-relaxed text-muted-foreground md:text-lg">{lead}</p>
      {children && <div className="mt-8">{children}</div>}
    </header>
  );
}

export function SectionHeading({ children }: { children: ReactNode }) {
  return (
    <h2 className="mb-8 border-b border-border pb-3 text-3xl font-medium text-foreground">{children}</h2>
  );
}
