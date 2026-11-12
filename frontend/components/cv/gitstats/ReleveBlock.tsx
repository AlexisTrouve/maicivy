import type { ReactNode } from 'react';

// ============================================================================
// Section de registre de la maquette A : titre, note de lecture et chiffre-clé dans une marge à
// gauche, la donnée à droite, un filet dessous. Sur téléphone, la marge passe au-dessus.
// Peinte d'emblée, sans apparition au défilement : le contenu d'une page ne doit pas dépendre
// d'une animation pour exister (cf. gitstats-sections-visible.spec.ts).
// ============================================================================

export function ReleveBlock({
  id,
  title,
  note,
  figure,
  children,
}: {
  id: string;
  title: string;
  note: ReactNode;
  /** `null` pendant le chargement : la place du chiffre est réservée, le titre et la note sont déjà là. */
  figure: { value: ReactNode; sub: ReactNode } | null;
  children: ReactNode;
}) {
  return (
    <section
      aria-labelledby={id}
      className="grid grid-cols-1 gap-6 border-b border-border/60 py-10 md:grid-cols-[260px_minmax(0,1fr)] md:gap-10 md:py-16"
    >
      <div>
        <h2 id={id} className="text-[30px] font-medium leading-[1.1] text-foreground">{title}</h2>
        <div className="mt-3 text-sm text-muted-foreground md:max-w-[32ch]">{note}</div>
        <p className="mt-4 font-heading text-[26px] font-semibold leading-[1.15] tabular-nums lining-nums text-foreground">
          {figure ? figure.value : <span aria-hidden="true" className="block h-[30px] w-40 animate-pulse rounded bg-muted" />}
          <small className="block font-sans text-[13px] font-normal leading-snug text-muted-foreground">
            {figure ? figure.sub : '\u00a0'}
          </small>
        </p>
      </div>
      <div className="min-w-0">{children}</div>
    </section>
  );
}

/** Légende d'un graphe : une pastille de la teinte de la série, puis son nom. */
export function Legend({ items }: { items: { color: string; label: string }[] }) {
  return (
    <div className="mt-4 flex flex-wrap gap-x-6 gap-y-2 text-[13px] text-muted-foreground">
      {items.map((item) => (
        <span key={item.label} data-testid="gitstats-legend-item" className="inline-flex items-center gap-1.5">
          <i aria-hidden="true" className="inline-block h-2.5 w-2.5 rounded-sm" style={{ background: item.color }} />
          {item.label}
        </span>
      ))}
    </div>
  );
}
