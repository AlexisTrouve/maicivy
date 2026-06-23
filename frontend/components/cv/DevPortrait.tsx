import type { ReactNode } from 'react';
import { getTranslations } from 'next-intl/server';
import { Link } from '@/i18n/navigation';
import { Code2, GitCommit, FolderGit2, TrendingUp, TrendingDown, Minus, Flame, ArrowRight } from 'lucide-react';
import { buildDevPortrait } from '@/lib/devStats';
import type { Trend } from '@/lib/devStats';
import { GitStatsResponse, LangStatsResponse } from '@/lib/types';

// ============================================================================
// DevPortrait — bande "Portrait Dev", en-tête de la page /cv.
// ----------------------------------------------------------------------------
// QUOI : agrège des stats pro (LOC, commits, genre récent par langage, momentum, repos chauds)
//        à partir des endpoints DÉJÀ servis (/cv/loc + /cv/gitstats). Les stats de TESTS, elles,
//        vivent sur /analytics (composant TestStatsCard) — séparation voulue : ici = portrait dev.
// POURQUOI server component : les chiffres sont SSR (visibles par un recruteur sans JS, bons pour
//        le SEO) et ne nécessitent aucune interactivité ni refresh temps réel.
// COMMENT : 1. fetch parallèle loc+gitstats côté serveur (URL interne docker), 2. buildDevPortrait
//        (module pur testé), 3. rendu. Enrichissement NON critique → si un fetch échoue, on rend
//        null (la bande disparaît, l'analytics visiteurs en dessous reste). Ce n'est pas un fallback
//        masquant un bug : c'est une dégradation gracieuse d'un bloc d'agrément.
// ============================================================================

// URL backend : interne (réseau docker) côté serveur. Identique au pattern de cv/page.tsx.
function getApiUrl(): string {
  return process.env.API_URL || 'http://maicivy-backend:8080';
}

// Récupère un JSON, ou null si indispo (timeout, 5xx). On ne throw pas : la bande est optionnelle.
async function fetchJson<T>(path: string): Promise<T | null> {
  try {
    const res = await fetch(`${getApiUrl()}${path}`, { cache: 'no-store' });
    if (!res.ok) return null;
    return (await res.json()) as T;
  } catch {
    return null;
  }
}

// Icône de tendance par langage (récent vs baseline annuelle).
function TrendIcon({ trend }: { trend: Trend }) {
  if (trend === 'up') return <TrendingUp className="h-4 w-4 text-green-500" aria-label="en hausse" />;
  if (trend === 'down') return <TrendingDown className="h-4 w-4 text-orange-500" aria-label="en baisse" />;
  return <Minus className="h-4 w-4 text-muted-foreground" aria-label="stable" />;
}

export default async function DevPortrait({ locale }: { locale: string }) {
  const t = await getTranslations({ locale, namespace: 'cv.devPortrait' });

  // Données réelles, en parallèle. l'une ou l'autre absente → bande masquée.
  const [lang, git] = await Promise.all([
    fetchJson<LangStatsResponse>('/api/v1/cv/loc'),
    fetchJson<GitStatsResponse>('/api/v1/cv/gitstats'),
  ]);
  if (!lang || !git) return null;

  const d = buildDevPortrait(lang, git);
  const nf = (n: number) => n.toLocaleString(locale);
  const ratioStr = d.momentum.ratio > 0 ? d.momentum.ratio.toFixed(1) : null;

  // Palette des barres de répartition LOC (top langages) — couleurs stables par position.
  const barColors = ['bg-blue-500', 'bg-purple-500', 'bg-green-500', 'bg-orange-500', 'bg-pink-500', 'bg-cyan-500'];

  return (
    <section className="rounded-xl border bg-card p-6 shadow-sm">
      {/* En-tête de la bande */}
      <div className="mb-6">
        <h2 className="text-2xl font-bold">{t('title')}</h2>
        <p className="text-sm text-muted-foreground">{t('subtitle')}</p>
      </div>

      {/* Rangée hero : 3 chiffres clés (les tests sont sur /analytics, pas ici) */}
      <div className="grid grid-cols-3 gap-4 mb-6">
        <HeroCard icon={<Code2 className="h-4 w-4 text-blue-500" />} value={nf(d.totalLoc)} label={t('loc')} sub={t('langs', { count: d.langCount })} />
        <HeroCard icon={<GitCommit className="h-4 w-4 text-purple-500" />} value={nf(d.totalCommits)} label={t('commits')} sub={t('commitsSub')} />
        <HeroCard icon={<FolderGit2 className="h-4 w-4 text-green-500" />} value={nf(d.activeRepos)} label={t('repos')} sub="" />
      </div>

      {/* Barre de répartition du code par langage (volume total LOC) */}
      <div className="mb-8">
        <div className="flex h-2.5 w-full overflow-hidden rounded-full">
          {d.topLangs.map((l, i) => (
            <div key={l.language} className={barColors[i % barColors.length]} style={{ width: `${l.pct}%` }} title={`${l.language} ${Math.round(l.pct)}%`} />
          ))}
        </div>
        <div className="mt-2 flex flex-wrap gap-x-4 gap-y-1 text-xs text-muted-foreground">
          {d.topLangs.map((l, i) => (
            <span key={l.language} className="inline-flex items-center gap-1.5">
              <span className={`h-2 w-2 rounded-full ${barColors[i % barColors.length]}`} />
              {l.language} {Math.round(l.pct)}%
            </span>
          ))}
        </div>
      </div>

      {/* Genre récent : répartition des jours-commit par langage, avec tendance */}
      <div className="mb-8">
        <h3 className="font-semibold">{t('genreTitle')}</h3>
        <p className="mb-4 text-xs text-muted-foreground">{t('genreSub')}</p>
        <div className="space-y-2.5">
          {d.genre.map((g) => (
            <div key={g.language} className="flex items-center gap-3">
              <div className="w-24 shrink-0 text-sm font-medium">{g.language}</div>
              <div className="h-2.5 flex-1 overflow-hidden rounded-full bg-muted">
                <div className="h-full rounded-full bg-primary" style={{ width: `${g.recentPct}%` }} />
              </div>
              <div className="flex w-16 shrink-0 items-center justify-end gap-1.5 text-sm tabular-nums">
                {Math.round(g.recentPct)}%
                <TrendIcon trend={g.trend} />
              </div>
            </div>
          ))}
        </div>
      </div>

      {/* Momentum + repos chauds */}
      <div className="grid gap-4 lg:grid-cols-2">
        {/* Momentum */}
        <div className="rounded-lg border bg-background p-5">
          <div className="mb-2 flex items-center gap-2">
            <TrendingUp className="h-4 w-4 text-green-500" />
            <h3 className="font-semibold">{t('momentumTitle')}</h3>
          </div>
          <div className="text-3xl font-bold">{nf(d.momentum.recent)}</div>
          <p className="text-sm text-muted-foreground">{t('momentumValue', { recent: nf(d.momentum.recent) })}</p>
          {ratioStr && <p className="mt-1 text-sm font-medium text-green-600 dark:text-green-400">{t('momentumRatio', { ratio: ratioStr })}</p>}
        </div>

        {/* Repos chauds */}
        <div className="rounded-lg border bg-background p-5">
          <div className="mb-3 flex items-center gap-2">
            <Flame className="h-4 w-4 text-orange-500" />
            <h3 className="font-semibold">{t('hotTitle')}</h3>
          </div>
          <ul className="space-y-2">
            {d.hot.map((r) => (
              <li key={r.name} className="flex items-center justify-between gap-2 text-sm">
                <span className="truncate font-medium">{r.name}</span>
                <span className="flex shrink-0 items-center gap-2 text-xs text-muted-foreground">
                  {r.language && <span className="rounded bg-muted px-1.5 py-0.5">{r.language}</span>}
                  {t('hotCommits', { count: r.commits })}
                </span>
              </li>
            ))}
          </ul>
        </div>
      </div>

      {/* Caveat honnête + lien vers le détail */}
      <div className="mt-6 flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between">
        <p className="text-[11px] text-muted-foreground">{t('caveat')}</p>
        <Link href="/gitstats" className="inline-flex shrink-0 items-center gap-1 text-sm font-medium text-primary hover:underline">
          {t('detailLink')}
          <ArrowRight className="h-3.5 w-3.5" />
        </Link>
      </div>
    </section>
  );
}

// Petite carte chiffre-clé réutilisée dans la rangée hero.
function HeroCard({ icon, value, label, sub, subGreen }: { icon: ReactNode; value: string; label: string; sub: string; subGreen?: boolean }) {
  return (
    <div className="rounded-lg border bg-background p-4">
      <div className="mb-1 flex items-center justify-between">
        <span className="text-xs font-medium text-muted-foreground">{label}</span>
        {icon}
      </div>
      <div className="text-2xl font-bold">{value}</div>
      {sub && <p className={`mt-0.5 text-xs ${subGreen ? 'text-green-600 dark:text-green-400' : 'text-muted-foreground'}`}>{sub}</p>}
    </div>
  );
}
