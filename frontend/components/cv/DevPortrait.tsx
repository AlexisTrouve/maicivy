import type { ReactNode } from 'react';
import { getTranslations } from 'next-intl/server';
import { Link } from '@/i18n/navigation';
import { Code2, GitCommit, FolderGit2, TrendingUp, TrendingDown, Minus, Flame, ArrowRight } from 'lucide-react';
import { buildDevPortrait } from '@/lib/devStats';
import type { Trend } from '@/lib/devStats';
import { GitStatsResponse, LangStatsResponse } from '@/lib/types';
import { LANGUAGE_TINTS } from './dataPalette';

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
    // revalidate 300 : ces stats (LOC, commits sur 6 mois) bougent à l'échelle de l'heure, pas de
    // la seconde. En no-store, chaque rendu de /cv repayait l'aller-retour — sur le chemin qui
    // bloquait l'affichage. Le backend sert déjà du stale-while-revalidate derrière.
    const res = await fetch(`${getApiUrl()}${path}`, { next: { revalidate: 300 } });
    if (!res.ok) return null;
    return (await res.json()) as T; // await dans le try : un corps illisible doit tomber dans le catch
  } catch {
    return null;
  }
}

// Icône de tendance par langage (récent vs baseline annuelle). C'est le GLYPHE qui porte l'état
// (flèche montante / descendante / tiret), la teinte ne fait que l'appuyer : ivoire pour la hausse,
// gris pour le reste. Libellé lu par les lecteurs d'écran, traduit.
function TrendIcon({ trend, labels }: { trend: Trend; labels: Record<Trend, string> }) {
  if (trend === 'up') return <TrendingUp className="h-4 w-4 text-foreground" role="img" aria-label={labels.up} />;
  if (trend === 'down') return <TrendingDown className="h-4 w-4 text-muted-foreground" role="img" aria-label={labels.down} />;
  return <Minus className="h-4 w-4 text-muted-foreground" role="img" aria-label={labels.flat} />;
}

// Props : `langStats` est OPTIONNEL et sert à la DÉDUPLICATION.
// POURQUOI : la page /cv fetche déjà /cv/loc pour la fiche détail des skills. Sans ce passage de
// prop, ce composant refaisait le MÊME appel — deux allers-retours pour une donnée identique, sur
// le chemin le plus lent de la page. Quand l'appelant a déjà la donnée, il la passe ; sinon
// (montage isolé, tests) on retombe sur le fetch autonome et le composant reste utilisable seul.
export default async function DevPortrait({
  locale,
  langStats,
}: {
  locale: string;
  langStats?: LangStatsResponse | null;
}) {
  const t = await getTranslations({ locale, namespace: 'cv.devPortrait' });

  // Données réelles. `langStats` fourni → on ne redemande QUE les stats git ; sinon les deux en
  // parallèle. L'une ou l'autre absente → bande masquée (enrichissement non critique).
  const [lang, git] = await Promise.all([
    langStats !== undefined ? Promise.resolve(langStats) : fetchJson<LangStatsResponse>('/api/v1/cv/loc'),
    fetchJson<GitStatsResponse>('/api/v1/cv/gitstats'),
  ]);
  if (!lang || !git) return null;

  const d = buildDevPortrait(lang, git);
  const nf = (n: number) => n.toLocaleString(locale);
  const ratioStr = d.momentum.ratio > 0 ? d.momentum.ratio.toFixed(1) : null;

  const trendLabels: Record<Trend, string> = { up: t('trendUp'), down: t('trendDown'), flat: t('trendFlat') };

  return (
    <section className="rounded-xl border border-border bg-card p-6 md:p-8" aria-labelledby="dev-portrait-title">
      {/* En-tête de la bande */}
      <div className="mb-8">
        <h2 id="dev-portrait-title" className="text-2xl font-medium text-foreground md:text-3xl">{t('title')}</h2>
        <p className="mt-1 text-sm text-muted-foreground">{t('subtitle')}</p>
      </div>

      {/* Rangée hero : 3 chiffres clés (les tests sont sur /analytics, pas ici). Une colonne sur
          téléphone : à 390 px, trois cartes côte à côte coupaient « 6 921 634 ». */}
      <div className="mb-8 grid grid-cols-1 gap-4 sm:grid-cols-3">
        <HeroCard icon={<Code2 className="h-4 w-4" aria-hidden="true" />} value={nf(d.totalLoc)} label={t('loc')} sub={t('langs', { count: d.langCount })} />
        <HeroCard icon={<GitCommit className="h-4 w-4" aria-hidden="true" />} value={nf(d.totalCommits)} label={t('commits')} sub={t('commitsSub')} />
        <HeroCard icon={<FolderGit2 className="h-4 w-4" aria-hidden="true" />} value={nf(d.activeRepos)} label={t('repos')} sub="" />
      </div>

      {/* Barre de répartition du code par langage (volume total LOC). Teintes de DONNÉES (une par
          rang), cf. dataPalette.ts ; la légende répète chaque langage en clair. */}
      <div className="mb-10">
        <div className="flex h-2 w-full overflow-hidden rounded-full bg-muted">
          {d.topLangs.map((l, i) => (
            <div
              key={l.language}
              style={{ width: `${l.pct}%`, backgroundColor: LANGUAGE_TINTS[i % LANGUAGE_TINTS.length] }}
              title={`${l.language} ${Math.round(l.pct)}%`}
            />
          ))}
        </div>
        <div className="mt-3 flex flex-wrap gap-x-5 gap-y-1.5 text-xs text-muted-foreground">
          {d.topLangs.map((l, i) => (
            <span key={l.language} className="inline-flex items-center gap-1.5">
              <span className="h-2 w-2 rounded-full" style={{ backgroundColor: LANGUAGE_TINTS[i % LANGUAGE_TINTS.length] }} />
              {l.language} <span className="tabular-nums">{Math.round(l.pct)}%</span>
            </span>
          ))}
        </div>
      </div>

      {/* Genre récent : répartition des jours-commit par langage, avec tendance */}
      <div className="mb-10">
        <h3 className="text-xl font-medium text-foreground">{t('genreTitle')}</h3>
        <p className="mb-5 mt-1 text-xs text-muted-foreground">{t('genreSub')}</p>
        <div className="space-y-3">
          {d.genre.map((g) => (
            <div key={g.language} className="flex items-center gap-3">
              <div className="w-20 shrink-0 truncate text-sm font-medium text-foreground sm:w-24">{g.language}</div>
              <div className="h-1.5 flex-1 overflow-hidden rounded-full bg-muted">
                <div className="h-full rounded-full bg-primary/80" style={{ width: `${g.recentPct}%` }} />
              </div>
              <div className="flex w-16 shrink-0 items-center justify-end gap-1.5 text-sm tabular-nums text-foreground">
                {Math.round(g.recentPct)}%
                <TrendIcon trend={g.trend} labels={trendLabels} />
              </div>
            </div>
          ))}
        </div>
      </div>

      {/* Cadence + dépôts les plus actifs */}
      <div className="grid gap-4 lg:grid-cols-2">
        <div className="rounded-lg border border-border bg-background p-5">
          <div className="mb-3 flex items-center gap-2 text-muted-foreground">
            <TrendingUp className="h-4 w-4" aria-hidden="true" />
            <h3 className="text-lg font-medium text-foreground">{t('momentumTitle')}</h3>
          </div>
          <div className="text-3xl font-semibold tabular-nums text-foreground">{nf(d.momentum.recent)}</div>
          <p className="mt-1 text-sm text-muted-foreground">{t('momentumValue', { recent: nf(d.momentum.recent) })}</p>
          {ratioStr && <p className="mt-1 text-sm font-medium text-foreground">{t('momentumRatio', { ratio: ratioStr })}</p>}
        </div>

        <div className="rounded-lg border border-border bg-background p-5">
          <div className="mb-3 flex items-center gap-2 text-muted-foreground">
            <Flame className="h-4 w-4" aria-hidden="true" />
            <h3 className="text-lg font-medium text-foreground">{t('hotTitle')}</h3>
          </div>
          <ul className="divide-y divide-border">
            {d.hot.map((r) => (
              <li key={r.name} className="flex items-center justify-between gap-3 py-2 text-sm">
                <span className="truncate font-medium text-foreground">{r.name}</span>
                <span className="flex shrink-0 items-center gap-2 text-xs text-muted-foreground">
                  {r.language && <span className="rounded bg-secondary px-1.5 py-0.5">{r.language}</span>}
                  <span className="tabular-nums">{t('hotCommits', { count: r.commits30d })}</span>
                </span>
              </li>
            ))}
          </ul>
        </div>
      </div>

      {/* Réserve honnête + lien vers le détail */}
      <div className="mt-8 flex flex-col gap-3 border-t border-border pt-5 sm:flex-row sm:items-center sm:justify-between">
        <p className="text-xs text-muted-foreground">{t('caveat')}</p>
        <Link
          href="/gitstats"
          className="inline-flex shrink-0 items-center gap-1.5 text-sm font-medium text-foreground underline-offset-4 hover:underline focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
        >
          {t('detailLink')}
          <ArrowRight className="h-3.5 w-3.5" aria-hidden="true" />
        </Link>
      </div>
    </section>
  );
}

// Petite carte chiffre-clé réutilisée dans la rangée hero.
function HeroCard({ icon, value, label, sub }: { icon: ReactNode; value: string; label: string; sub: string }) {
  return (
    <div className="rounded-lg border border-border bg-background p-4">
      <div className="mb-1 flex items-center justify-between text-muted-foreground">
        <span className="text-xs font-medium">{label}</span>
        {icon}
      </div>
      <div className="text-2xl font-semibold tabular-nums text-foreground">{value}</div>
      {sub && <p className="mt-0.5 text-xs text-muted-foreground">{sub}</p>}
    </div>
  );
}
