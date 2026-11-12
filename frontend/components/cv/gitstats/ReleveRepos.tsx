'use client';

import { useEffect, useRef, useState, type ReactNode } from 'react';
import { useTranslations } from 'next-intl';
import type { GitRepoStat } from '@/lib/types';
import { computeStreaks } from '@/lib/devStats';
import { lastCommitDay, repoWeekActivity, type WeekPoint } from '@/lib/gitstatsReleve';
import { DATA_TINTS } from '../dataPalette';
import type { Formats } from './format';

// ============================================================================
// Dépôts : une table de registre classée par commits des 30 derniers jours.
// ----------------------------------------------------------------------------
// Les douze premiers sont visibles ; le reste est derrière l'action principale de la page (l'or :
// UNE par écran). Les dépôts sans commit depuis six mois ne sont pas listés, seulement comptés.
// UN SEUL balisage pour les deux tailles : sur PC c'est une vraie table (en-têtes de colonnes) ;
// sur téléphone chaque rangée devient un bloc — nom et chiffre 30 j, la bande, puis une ligne
// « langage · lignes · commits en six mois ». D'où les unités (« en 30 j », « lignes ») : écrites
// sur téléphone, réservées aux lecteurs d'écran sur PC où l'en-tête de colonne les porte déjà.
// Deux listes (une par taille) dédoubleraient chaque dépôt dans le DOM et pour les tests.
// ============================================================================

export const REPOS_SHOWN = 12;

// Opacité d'une case de la bande selon le nombre de jours avec commit dans la semaine (0..7).
const WEEK_ALPHA = [0.07, 0.3, 0.5, 0.68, 0.84, 0.95, 1, 1];

// Cellule : padding de registre sur PC ; sur téléphone la rangée est un bloc flex, sans padding.
const CELL = 'py-3.5 pr-3 align-middle max-md:p-0';
// Point médian entre deux éléments de la ligne méta du téléphone.
const META_SEP = "max-md:before:mx-1.5 max-md:before:content-['·']";

export function ReleveRepos({
  active,
  dormant,
  weeks,
  locByName,
  today,
  formats,
}: {
  active: GitRepoStat[];
  dormant: GitRepoStat[];
  weeks: WeekPoint[];
  /** LOC par dépôt (/cv/loc, chargé à part) : `null` tant qu'il n'est pas arrivé. */
  locByName: Record<string, number> | null;
  today: string;
  formats: Formats;
}) {
  const t = useTranslations('gitstats.repos');
  const [expanded, setExpanded] = useState(false);
  const firstRevealed = useRef<HTMLTableRowElement>(null);

  // À l'ouverture, le focus passe au premier dépôt révélé : le bouton disparaît, le focus clavier
  // ne doit pas retomber en haut de la page.
  useEffect(() => {
    if (expanded) firstRevealed.current?.focus();
  }, [expanded]);

  const visible = expanded ? active : active.slice(0, REPOS_SHOWN);
  const hidden = active.length - REPOS_SHOWN;
  // Unité écrite sur téléphone, lue seulement par les lecteurs d'écran sur PC.
  const unit = (chunks: ReactNode) => <span className="md:sr-only">{chunks}</span>;

  return (
    <div>
      <table className="w-full border-collapse text-[14.5px] max-md:block">
        <thead className="max-md:hidden">
          <tr className="border-b border-border text-left text-xs font-medium uppercase tracking-[0.08em] text-muted-foreground">
            <th className="w-7 pb-3"><span className="sr-only">#</span></th>
            <th className="pb-3 pr-3 font-medium">{t('colRepo')}</th>
            <th className="pb-3 pr-3 font-medium">{t('colWeeks')}</th>
            <th className="pb-3 pr-3 text-right font-medium">{t('col30d')}</th>
            <th className="pb-3 pr-3 text-right font-medium">{t('col6mo')}</th>
            <th className="pb-3 pr-3 text-right font-medium">{t('colLines')}</th>
            <th className="pb-3 text-right font-medium">{t('colLast')}</th>
          </tr>
        </thead>
        <tbody id="gitstats-repos" className="max-md:block">
          {visible.map((repo, i) => {
            const loc = locByName?.[repo.name];
            const last = lastCommitDay(repo);
            const activity = repoWeekActivity(repo.commitDays, weeks);
            // Série PAR DÉPÔT, depuis SES jours de commit (pas l'agrégat de tous les dépôts).
            const streak = computeStreaks(repo.commitDays ?? [], today);
            const activityLabel = t('activity', {
              days: activity.reduce((s, n) => s + n, 0),
              current: streak.current,
              longest: streak.longest,
            });
            return (
              <tr
                key={repo.name}
                ref={i === REPOS_SHOWN ? firstRevealed : undefined}
                tabIndex={i === REPOS_SHOWN ? -1 : undefined}
                data-testid="gitstats-repo"
                className="border-b border-border/60 outline-none focus-visible:outline-2 focus-visible:outline-gold max-md:flex max-md:flex-wrap max-md:items-baseline max-md:gap-y-2 max-md:py-4"
              >
                <td className={`${CELL} w-7 tabular-nums text-muted-foreground/70 max-md:hidden`}>{i + 1}</td>
                {/* Sur téléphone, la cellule s'efface (`contents`) : le nom va en tête de bloc, le
                    langage ouvre la ligne méta. */}
                <td className={`${CELL} max-md:contents`}>
                  <span data-testid="gitstats-repo-name" className="font-medium text-foreground max-md:order-1 max-md:min-w-0 max-md:flex-1 max-md:truncate max-md:text-base">
                    {repo.name}
                  </span>
                  {repo.language && (
                    <span className="block text-[12.5px] text-muted-foreground max-md:order-4 max-md:text-[13px]">{repo.language}</span>
                  )}
                </td>
                <td className={`${CELL} max-md:order-3 max-md:basis-full`}>
                  <span
                    role="img"
                    aria-label={activityLabel}
                    title={activityLabel}
                    className="grid h-[18px] auto-cols-[7px] grid-flow-col gap-[2px] max-md:h-[14px] max-md:auto-cols-fr"
                  >
                    {activity.map((days, w) => (
                      <i key={w} className="rounded-[1.5px]" style={{ background: DATA_TINTS.ivory, opacity: WEEK_ALPHA[days] }} />
                    ))}
                  </span>
                </td>
                <td className={`${CELL} text-right max-md:order-2 max-md:ml-3`}>
                  <span className="font-heading text-[22px] font-semibold leading-none tabular-nums lining-nums text-foreground max-md:text-2xl">
                    {t.rich('hot', {
                      count: formats.number(repo.commits30d || 0),
                      unit: (chunks) => <small className="ml-1 font-sans text-xs font-normal text-muted-foreground md:sr-only">{chunks}</small>,
                    })}
                  </span>
                </td>
                <td className={`${CELL} text-right tabular-nums text-foreground/90 max-md:order-6 max-md:text-[13px] max-md:text-muted-foreground ${repo.language || loc !== undefined ? META_SEP : ''}`}>
                  {t.rich('commits6mo', { count: formats.number(repo.commits || 0), unit })}
                </td>
                <td className={`${CELL} text-right tabular-nums text-foreground/90 max-md:order-5 max-md:text-[13px] max-md:text-muted-foreground ${loc === undefined ? 'max-md:hidden' : ''} ${repo.language ? META_SEP : ''}`}>
                  {loc !== undefined ? t.rich('loc', { loc: formats.number(loc), unit }) : '—'}
                </td>
                <td className={`${CELL} whitespace-nowrap pr-0 text-right text-foreground/90 max-md:hidden`}>{last ? formats.day(last) : '—'}</td>
              </tr>
            );
          })}
        </tbody>
      </table>

      {(hidden > 0 && !expanded) || dormant.length > 0 ? (
        <div className="mt-10 flex flex-wrap items-center gap-6">
          {hidden > 0 && !expanded && (
            <button
              type="button"
              aria-controls="gitstats-repos"
              onClick={() => setExpanded(true)}
              className="inline-flex min-h-[46px] items-center justify-center rounded-lg border border-gold bg-gold px-6 text-[15px] font-semibold text-primary-foreground transition-colors hover:bg-gold-bright focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 focus-visible:ring-offset-background max-md:w-full"
            >
              {t('more', { count: hidden })}
            </button>
          )}
          {dormant.length > 0 && (
            <p data-testid="gitstats-dormant" className="text-sm text-muted-foreground">{t('dormant', { count: dormant.length })}</p>
          )}
        </div>
      ) : null}
    </div>
  );
}

/** Réserve de la table pendant le chargement : douze rangées à la hauteur des vraies. */
export function ReleveReposPlaceholder() {
  return (
    <div aria-hidden="true">
      <div className="h-8 border-b border-border max-md:hidden" />
      {Array.from({ length: REPOS_SHOWN }, (_, i) => (
        <div key={i} className="flex h-[70px] items-center border-b border-border/60 max-md:h-[106px]">
          <div className="h-4 w-2/3 animate-pulse rounded bg-muted" />
        </div>
      ))}
    </div>
  );
}
