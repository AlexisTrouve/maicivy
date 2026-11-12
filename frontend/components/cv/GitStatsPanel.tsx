'use client';

import { useEffect, useMemo, useState, type ReactNode } from 'react';
import { RefreshCw } from 'lucide-react';
import { useLocale, useTranslations } from 'next-intl';
import { GitStatsResponse, LangStatsResponse } from '@/lib/types';
import { totalLoc } from '@/lib/devStats';
import {
  buildDays,
  groupWeeks,
  languageBreakdown,
  longestActiveStreak,
  rankRepos,
  type DayPoint,
} from '@/lib/gitstatsReleve';
import { DATA_TINTS } from './dataPalette';
import { createFormats } from './gitstats/format';
import { ReleveBars, type BarPoint } from './gitstats/ReleveBars';
import { Legend, ReleveBlock } from './gitstats/ReleveBlock';
import { LANGUAGE_ROWS, ReleveLanguages } from './gitstats/ReleveLanguages';
import { ReleveRepos, ReleveReposPlaceholder } from './gitstats/ReleveRepos';

const API_URL = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080';

// Cache de session : peinture INSTANTANÉE des stats au re-visit de la page (navigation interne), au
// lieu de re-fetch + skeleton à chaque montage. Couplé au stale-while-revalidate du backend, ça tue le
// « repull à chaque fois » des deux côtés. Clé versionnée → invalidation si le shape de la réponse change.
// sessionStorage (pas localStorage) : cache éphémère par onglet, pas une persistance longue qui
// afficherait des chiffres datés à la prochaine ouverture du navigateur.
// v2 : la réponse a gagné le champ `gitlabDaily`. Bump de clé → les onglets qui ont un cache v1
// le laissent tomber et refetchent, au lieu de repeindre une réponse périmée SANS gitlabDaily.
const CACHE_KEY = 'maicivy:gitstats:v2';
const LOC_CACHE_KEY = 'maicivy:gitloc:v1';

// Hauteurs des graphes : hebdomadaires sur téléphone (182 barres sur 350 px ne se lisent pas),
// quotidiens sur PC. Partagées par le graphe et sa réserve de chargement.
const CADENCE_H = { phone: 190, pc: 230 };
const LINES_H = { phone: 210, pc: 250 };

// Lit le cache de session. try/catch car sessionStorage lève en navigation privée / quota plein →
// on dégrade sans cache (ce n'est pas masquer une erreur métier, juste un accès navigateur faillible).
function readSessionCache<T>(key: string): T | null {
  try {
    const raw = sessionStorage.getItem(key);
    return raw ? (JSON.parse(raw) as T) : null;
  } catch {
    return null;
  }
}

function writeSessionCache<T>(key: string, data: T): void {
  try {
    sessionStorage.setItem(key, JSON.stringify(data));
  } catch {
    /* quota / mode privé → on ignore : le cache n'est qu'une optimisation, pas une source de vérité */
  }
}

// Date du jour "YYYY-MM-DD" en heure LOCALE : c'est le « aujourd'hui » du visiteur qui borne la
// série et la série en cours de chaque dépôt.
function todayIso(): string {
  const d = new Date();
  return `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}-${String(d.getDate()).padStart(2, '0')}`;
}

/** Réserve d'un graphe : la boîte exacte du graphe réel, à chaque taille d'écran. */
function ChartReserve({ height }: { height: { phone: number; pc: number } }) {
  return (
    <>
      <div aria-hidden="true" className="animate-pulse rounded-lg bg-muted md:hidden" style={{ height: height.phone }} />
      <div aria-hidden="true" className="hidden animate-pulse rounded-lg bg-muted md:block" style={{ height: height.pc }} />
    </>
  );
}

// ============================================================================
// GitStatsPanel — corps de la page /gitstats, maquette A « Le relevé ».
// ----------------------------------------------------------------------------
// Un registre éditorial sans cartes : quatre chiffres en ligne, puis quatre sections (cadence,
// lignes, langages, dépôts) avec titre et chiffre-clé en marge, la donnée à droite.
// MISE EN PAGE STABLE (CLS) : la page sautait au chargement (CLS mesuré 0,65). La COQUILLE est la
// même dans les deux états — en-tête et « Actualiser », les quatre chiffres (libellés réels,
// valeur réservée à la hauteur de sa ligne), les titres et notes des sections, les graphes à
// hauteur fixe, la table des langages et la liste des dépôts réservées. Seul le CONTENU de ces
// boîtes change quand les données arrivent.
// ============================================================================
export default function GitStatsPanel({ heading }: { heading: ReactNode }) {
  const t = useTranslations('gitstats');
  const locale = useLocale();
  const formats = useMemo(() => createFormats(locale), [locale]);
  const [stats, setStats] = useState<GitStatsResponse | null>(null);
  const [locStats, setLocStats] = useState<LangStatsResponse | null>(null);
  const [loading, setLoading] = useState(true);
  const [refreshing, setRefreshing] = useState(false);

  // Fetch stats — force=true pour ignorer le cache Redis
  const fetchStats = (force = false) => {
    const query = force ? '?force=true' : '';
    if (force) setRefreshing(true);

    fetch(`${API_URL}/api/v1/cv/gitstats${query}`)
      .then((res) => {
        if (!res.ok) throw new Error(`gitstats ${res.status}`);
        return res.json();
      })
      .then((data: GitStatsResponse) => {
        setStats(data);
        writeSessionCache(CACHE_KEY, data); // mémorise pour la peinture instantanée du prochain montage
      })
      // Le détail technique va à la console ; le visiteur lit le message traduit de l'état vide.
      .catch((err) => console.error('Git stats unavailable:', err))
      .finally(() => {
        setLoading(false);
        setRefreshing(false);
      });
  };

  // Fetch LOC (par langage et par dépôt) — endpoint séparé (cache Redis à refresh lent, 6h) : s'il
  // échoue, la section Langages garde sa réserve et les dépôts s'affichent sans leurs lignes.
  const fetchLocStats = (force = false) => {
    const query = force ? '?force=true' : '';
    fetch(`${API_URL}/api/v1/cv/loc${query}`)
      .then((res) => {
        if (!res.ok) throw new Error('Failed to fetch LOC stats');
        return res.json();
      })
      .then((data: LangStatsResponse) => {
        setLocStats(data);
        writeSessionCache(LOC_CACHE_KEY, data);
      })
      .catch(() => { /* non-bloquant — la page reste utilisable sans LOC */ });
  };

  useEffect(() => {
    // 1. Si on revient sur la page dans la même session, on peint le cache TOUT DE SUITE (zéro skeleton).
    const cached = readSessionCache<GitStatsResponse>(CACHE_KEY);
    if (cached) {
      setStats(cached);
      setLoading(false);
    }
    const cachedLoc = readSessionCache<LangStatsResponse>(LOC_CACHE_KEY);
    if (cachedLoc) setLocStats(cachedLoc);
    // 2. On revalide en fond dans tous les cas : le backend répond instantanément (stale-while-revalidate),
    //    donc ce fetch est quasi gratuit et rafraîchit l'affichage si les chiffres ont bougé.
    fetchStats();
    fetchLocStats();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  // Données dérivées : série complète (jours absents = 0), semaines, classement des dépôts.
  // `stats` n'existe que côté client (fetch après montage) : « aujourd'hui » n'est jamais lu au
  // rendu serveur, aucun écart d'hydratation possible.
  const derived = useMemo(() => {
    if (!stats) return null;
    const today = todayIso();
    const days = buildDays(stats.daily, stats.gitlabDaily, today);
    return { today, days, weeks: groupWeeks(days), ...rankRepos(stats.repos) };
  }, [stats]);

  const languages = useMemo(() => (locStats ? languageBreakdown(locStats, LANGUAGE_ROWS) : null), [locStats]);

  // État vide SEULEMENT si on n'a aucune donnée à montrer. Si du cache est déjà peint, une
  // revalidation de fond qui échoue ne doit PAS remplacer de bons chiffres par une page d'erreur.
  // L'état se lit au glyphe ✕ autant qu'au texte.
  if (!loading && !stats) {
    return (
      <>
        <div className="py-10 md:py-16">{heading}</div>
        <div role="status" className="border-y border-border px-6 py-16 text-center text-muted-foreground">
          <span aria-hidden="true">✕ </span>
          {t('unavailable')}
        </div>
      </>
    );
  }

  const hasGitlab = (stats?.gitlabDaily?.length ?? 0) > 0;
  const gitlabTotal = (stats?.gitlabDaily ?? []).reduce((s, d) => s + d.commits, 0);
  const lastDay = derived?.days[derived.days.length - 1]?.date;

  const kpis = [
    { label: t('commits6mo'), value: stats && formats.number(stats.totalCommits), sub: hasGitlab ? t('ledger.gitlab', { count: formats.number(gitlabTotal) }) : null },
    { label: t('linesAdded'), value: stats && formats.short(stats.totalAdded), sub: stats && t('ledger.exactly', { value: formats.number(stats.totalAdded) }) },
    { label: t('linesDeleted'), value: stats && formats.short(stats.totalDeleted), sub: stats && t('ledger.exactly', { value: formats.number(stats.totalDeleted) }) },
    { label: t('activeRepos'), value: stats && formats.number(stats.activeRepos), sub: stats && t('ledger.reposTotal', { total: formats.number(stats.repos.length) }) },
  ];

  // Points des graphes. Le relevé de chaque barre porte sa valeur exacte (survol ou toucher).
  const cadencePoints = (series: DayPoint[], weekly: boolean): BarPoint[] =>
    series.map((d) => ({
      date: d.date,
      up: d.commits,
      overlay: d.gitlab,
      title: t(weekly ? 'cadence.barWeek' : 'cadence.barDay', { date: formats.day(d.date), value: formats.number(d.commits) }),
    }));
  const linePoints = (series: DayPoint[], weekly: boolean): BarPoint[] =>
    series.map((d) => ({
      date: d.date,
      up: d.additions,
      down: d.deletions,
      title: t(weekly ? 'lines.barWeek' : 'lines.barDay', {
        date: formats.day(d.date),
        added: formats.number(d.additions),
        deleted: formats.number(d.deletions),
      }),
    }));
  const peak = (key: 'cadence' | 'lines', weekly: boolean) => (p: BarPoint) =>
    t(`${key}.${weekly ? 'peakWeek' : 'peakDay'}`, { value: formats.short(p.up), date: formats.day(p.date) });

  const chartFormats = { formatShort: formats.short, formatMonth: formats.month };

  return (
    <div aria-busy={loading}>
      {/* En-tête : titre de page à gauche, date du relevé et « Actualiser » à droite (dessous sur
          téléphone). Le bouton existe dès le premier rendu (désactivé tant qu'il n'y a rien à
          rafraîchir) et la ligne du relevé a sa hauteur : rien ne s'insère à l'arrivée des données. */}
      <div className="flex flex-col items-start gap-6 pb-6 pt-10 md:flex-row md:items-end md:justify-between md:gap-10 md:pb-10 md:pt-16">
        {heading}
        <div className="grid shrink-0 justify-items-start gap-2 text-[13px] text-muted-foreground md:justify-items-end">
          <span className="h-5 leading-5">{lastDay ? t('surveyedOn', { date: formats.dayLong(lastDay) }) : '\u00a0'}</span>
          <button
            type="button"
            onClick={() => { fetchStats(true); fetchLocStats(true); }}
            disabled={refreshing || !stats}
            className="inline-flex min-h-10 items-center gap-2 whitespace-nowrap rounded-md border border-muted-foreground/40 bg-secondary px-3 text-sm text-foreground/80 transition-colors hover:border-muted-foreground hover:text-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring disabled:opacity-50"
          >
            <RefreshCw className={`h-4 w-4 ${refreshing ? 'animate-spin' : ''}`} aria-hidden="true" />
            {refreshing ? t('refreshing') : t('refresh')}
          </button>
        </div>
      </div>

      {/* Les quatre chiffres : une ligne de registre, séparée par des filets (2 × 2 sur téléphone).
          Le libellé est un <dt> placé SOUS le chiffre par `order` : l'ordre de lecture reste
          « libellé, valeur ». */}
      <dl data-testid="gitstats-kpis" className="grid grid-cols-2 border-y border-border md:grid-cols-4">
        {kpis.map((kpi, i) => (
          <div
            key={kpi.label}
            className={[
              'flex flex-col border-border/60 py-4 md:py-6 md:pr-6',
              i % 2 === 1 ? 'border-l pl-4' : '',
              i === 2 ? 'md:border-l' : '',
              i > 0 ? 'md:pl-6' : '',
              i < 2 ? 'border-b md:border-b-0' : '',
            ].join(' ')}
          >
            <dt className="order-2 mt-1.5 text-[13px] text-muted-foreground">{kpi.label}</dt>
            {/* Même police, même interligne (1) dans les deux états : la réserve a la hauteur du chiffre. */}
            <dd
              data-testid={kpi.value ? 'gitstats-kpi-value' : undefined}
              aria-hidden={kpi.value ? undefined : true}
              className="order-1 font-heading text-[clamp(2.2rem,3.6vw,3.2rem)] font-semibold leading-none tabular-nums lining-nums text-foreground"
            >
              {kpi.value ?? <span className="inline-block h-[0.8em] w-24 animate-pulse rounded bg-muted align-middle" />}
            </dd>
            <dd className="order-3 mt-2 text-[13px] tabular-nums text-muted-foreground/70">{kpi.sub ?? '\u00a0'}</dd>
          </div>
        ))}
      </dl>

      <ReleveBlock
        id="gitstats-cadence"
        title={t('cadence.title')}
        note={
          <>
            <span className="md:hidden">{t('cadence.noteWeek')}</span>
            <span className="hidden md:inline">{t('cadence.noteDay')}</span>
            {hasGitlab && ` ${t('cadence.noteGitlab')}`}
          </>
        }
        figure={
          derived && {
            value: t('cadence.figure', {
              active: derived.days.filter((d) => d.commits > 0).length,
              total: formats.number(derived.days.length),
            }),
            sub: t('cadence.figureSub', { streak: longestActiveStreak(derived.days) }),
          }
        }
      >
        {derived ? (
          <>
            <ReleveBars className="md:hidden" points={cadencePoints(derived.weeks, true)} height={CADENCE_H.phone}
              upColor={DATA_TINTS.ivory} overlayColor={hasGitlab ? DATA_TINTS.copper : undefined}
              label={t('cadence.noteWeek')} peakLabel={peak('cadence', true)} {...chartFormats} />
            <ReleveBars className="hidden md:block" points={cadencePoints(derived.days, false)} height={CADENCE_H.pc}
              upColor={DATA_TINTS.ivory} overlayColor={hasGitlab ? DATA_TINTS.copper : undefined}
              label={t('cadence.noteDay')} peakLabel={peak('cadence', false)} {...chartFormats} />
          </>
        ) : (
          <ChartReserve height={CADENCE_H} />
        )}
        <Legend
          items={[
            { color: DATA_TINTS.ivory, label: t('cadence.legendAll') },
            ...(hasGitlab ? [{ color: DATA_TINTS.copper, label: t('cadence.legendGitlab') }] : []),
          ]}
        />
      </ReleveBlock>

      <ReleveBlock
        id="gitstats-lines"
        title={t('lines.title')}
        note={t('lines.note')}
        figure={
          stats && {
            value: t('lines.figure', { net: formats.short(stats.totalAdded - stats.totalDeleted) }),
            sub: t('lines.figureSub', { added: formats.short(stats.totalAdded), deleted: formats.short(stats.totalDeleted) }),
          }
        }
      >
        {derived ? (
          <>
            <ReleveBars className="md:hidden" points={linePoints(derived.weeks, true)} height={LINES_H.phone}
              upColor={DATA_TINTS.sage} downColor={DATA_TINTS.terra}
              label={t('lines.note')} peakLabel={peak('lines', true)} {...chartFormats} />
            <ReleveBars className="hidden md:block" points={linePoints(derived.days, false)} height={LINES_H.pc}
              upColor={DATA_TINTS.sage} downColor={DATA_TINTS.terra}
              label={t('lines.note')} peakLabel={peak('lines', false)} {...chartFormats} />
          </>
        ) : (
          <ChartReserve height={LINES_H} />
        )}
        <Legend
          items={[
            { color: DATA_TINTS.sage, label: t('lines.legendAdded') },
            { color: DATA_TINTS.terra, label: t('lines.legendDeleted') },
          ]}
        />
      </ReleveBlock>

      <ReleveBlock
        id="gitstats-langs"
        title={t('langs.title')}
        note={t('langs.note')}
        figure={
          locStats && languages && {
            value: t('langs.figure', { loc: formats.short(totalLoc(locStats)) }),
            sub: t('langs.figureSub', { count: languages.total }),
          }
        }
      >
        <ReleveLanguages
          head={languages?.head ?? null}
          rest={languages?.rest ?? null}
          othersLabel={(count) => t('langs.others', { count })}
          formats={formats}
        />
      </ReleveBlock>

      <ReleveBlock
        id="gitstats-repos-title"
        title={t('repos.title')}
        note={t('repos.note')}
        figure={
          derived && {
            value: t('repos.figure', { count: derived.active.filter((r) => r.commits30d > 0).length }),
            sub: t('repos.figureSub', { count: derived.active.filter((r) => r.commits30d > 0).length }),
          }
        }
      >
        {derived ? (
          <ReleveRepos
            active={derived.active}
            dormant={derived.dormant}
            weeks={derived.weeks}
            locByName={locStats?.repos ?? null}
            today={derived.today}
            formats={formats}
          />
        ) : (
          <ReleveReposPlaceholder />
        )}
      </ReleveBlock>
    </div>
  );
}
