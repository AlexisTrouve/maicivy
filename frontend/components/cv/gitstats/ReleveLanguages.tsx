import type { LanguageShare } from '@/lib/gitstatsReleve';
import { DATA_TINTS, LANGUAGE_TINTS } from '../dataPalette';
import type { Formats } from './format';

// ============================================================================
// Langages : une bande proportionnelle, puis sa table (deux colonnes sur PC, une sur téléphone).
// Six teintes de données pour les six premiers, gris effacé pour les suivants, filet appuyé pour
// l'agrégat « autres » : la teinte suit le RANG, pas le langage.
// ============================================================================

// Rangées de la table : les huit premiers langages + « autres ». La réserve de chargement en
// dessine autant, pour que la table arrive dans une boîte déjà à sa taille.
export const LANGUAGE_ROWS = 8;

const ROW = 'grid grid-cols-[12px_minmax(0,1fr)_auto_64px] items-baseline gap-3 border-b border-border/60 py-2.5 text-[15px]';

export function ReleveLanguages({
  head,
  rest,
  othersLabel,
  formats,
}: {
  head: LanguageShare[] | null;
  rest: { count: number; loc: number; share: number } | null;
  othersLabel: (count: number) => string;
  formats: Formats;
}) {
  if (!head || !rest) {
    return (
      <div aria-hidden="true">
        <div className="h-3 animate-pulse rounded-[3px] bg-muted" />
        <ul className="mt-6 grid grid-cols-1 gap-x-10 md:grid-cols-2">
          {Array.from({ length: LANGUAGE_ROWS + 1 }, (_, i) => (
            <li key={i} className={ROW}>
              <span />
              <span className="h-3 w-24 animate-pulse self-center rounded bg-muted" />
              <span>{'\u00a0'}</span>
              <span />
            </li>
          ))}
        </ul>
      </div>
    );
  }

  const parts = [
    ...head.map((l, i) => ({ ...l, tint: LANGUAGE_TINTS[i] ?? DATA_TINTS.faint })),
    ...(rest.count ? [{ name: othersLabel(rest.count), loc: rest.loc, share: rest.share, tint: DATA_TINTS.rule }] : []),
  ];

  return (
    <div>
      <div
        role="img"
        aria-label={parts.map((p) => `${p.name} ${formats.percent(p.share)}`).join(', ')}
        className="flex h-3 gap-0.5 overflow-hidden rounded-[3px]"
      >
        {parts.map((p) => (
          <span key={p.name} style={{ flex: p.share, background: p.tint }} />
        ))}
      </div>
      <ul className="mt-6 grid grid-cols-1 gap-x-10 md:grid-cols-2">
        {parts.map((p) => (
          <li key={p.name} className={ROW}>
            <i aria-hidden="true" className="h-2.5 w-2.5 self-center rounded-sm" style={{ background: p.tint }} />
            <span className="truncate text-foreground">{p.name}</span>
            <span className="tabular-nums text-foreground/80">{formats.number(p.loc)}</span>
            <span className="text-right tabular-nums text-muted-foreground">{formats.percent(p.share)}</span>
          </li>
        ))}
      </ul>
    </div>
  );
}
