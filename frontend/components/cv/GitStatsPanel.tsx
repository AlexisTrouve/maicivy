'use client';

import { useState, useEffect } from 'react';
import {
  XAxis, YAxis, Tooltip, ResponsiveContainer,
  CartesianGrid, Area, AreaChart, Legend,
} from 'recharts';
import { GitDayStat, GitStatsResponse, LangStatsResponse } from '@/lib/types';
import { computeStreaks } from '@/lib/devStats';
import { motion } from 'framer-motion';
import { useTranslations, useLocale } from 'next-intl';

const API_URL = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080';

// Cache de session : peinture INSTANTANÉE des stats au re-visit de la page (navigation interne), au
// lieu de re-fetch + skeleton à chaque montage. Couplé au stale-while-revalidate du backend, ça tue le
// « repull à chaque fois » des deux côtés. Clé versionnée → invalidation si le shape de la réponse change.
// sessionStorage (pas localStorage) : cache éphémère par onglet, pas une persistance longue qui
// afficherait des chiffres datés à la prochaine ouverture du navigateur.
// Générique sur la clé : sert à la fois gitstats (v1) et loc (v1, LOC par repo — cf. fetchLocStats).
// v2 : la réponse a gagné le champ `gitlabDaily` (2e courbe GitLab). Bump de clé → les onglets qui ont
// un cache v1 d'avant ce changement le laissent tomber et refetchent, au lieu de repeindre une réponse
// périmée SANS gitlabDaily (→ courbe GitLab manquante / commits récents « à 0 » jusqu'au refetch).
const CACHE_KEY = 'maicivy:gitstats:v2';
const LOC_CACHE_KEY = 'maicivy:gitloc:v1';

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

// Filtre les 6 derniers mois et formate les dates pour l'affichage.
// gitlabByDate : commits GitLab seuls par jour (map date→commits). Chaque point du chart reçoit un
// champ `gitlabCommits` (0 si aucun commit GitLab ce jour-là) → source de la 2e courbe superposée.
function filterLast6Months(daily: GitDayStat[], locale: string, gitlabByDate: Map<string, number>) {
  const sixMonthsAgo = new Date();
  sixMonthsAgo.setMonth(sixMonthsAgo.getMonth() - 6);

  return daily
    .filter((d) => new Date(d.date) >= sixMonthsAgo)
    .map((d) => ({
      ...d,
      // Contribution GitLab de ce jour (sous-ensemble du total `commits`) — 0 si absente.
      gitlabCommits: gitlabByDate.get(d.date) ?? 0,
      // Label de l'axe X formaté selon la locale courante (jour + mois court)
      label: new Date(d.date).toLocaleDateString(locale, { day: '2-digit', month: 'short' }),
    }));
}

// Tri des repos par date du dernier commit RÉEL (dernier élément de commitDays, trié croissant côté
// backend), PAS par `updatedAt` (métadonnée Gitea qui bouge sur un simple push de tag/settings, sans
// code — piège déjà rencontré et corrigé pour le badge "Repos chauds" du CV, cf. CLAUDE.md). Repos sans
// commit sur la fenêtre 6 mois (commitDays vide) → relégués en fin de liste.
function sortReposByLastCommit(repos: GitStatsResponse['repos']) {
  return [...repos].sort((a, b) => {
    const lastA = a.commitDays?.length ? a.commitDays[a.commitDays.length - 1] : '';
    const lastB = b.commitDays?.length ? b.commitDays[b.commitDays.length - 1] : '';
    return lastB.localeCompare(lastA);
  });
}

// Formate la date du dernier commit réel d'un repo (dernier élément de commitDays) selon la locale
// courante. Même source que le tri — délibérément PAS `updatedAt` (cf. sortReposByLastCommit).
function lastCommitLabel(repo: GitStatsResponse['repos'][number], locale: string): string | null {
  if (!repo.commitDays?.length) return null;
  const last = repo.commitDays[repo.commitDays.length - 1];
  return new Date(last).toLocaleDateString(locale, { day: '2-digit', month: 'short', year: 'numeric' });
}

// Date du jour au format "YYYY-MM-DD" en heure LOCALE (pas UTC — cohérent avec le reste du composant,
// cf. filterLast6Months/lastCommitLabel qui raisonnent déjà en Date locale). Injectée dans
// computeStreaks (fonction pure côté lib) plutôt que lue en interne, pour rester testable sans mock de
// l'horloge côté lib — c'est le composant, au moment du rendu, qui fournit le "aujourd'hui" réel.
function todayIso(): string {
  const d = new Date();
  return `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}-${String(d.getDate()).padStart(2, '0')}`;
}

// Animation variants pour les sections qui apparaissent au scroll
const sectionVariants = {
  hidden: { opacity: 0, y: 40 },
  visible: { opacity: 1, y: 0 },
};

// Stat card avec stagger animation
function StatCard({ label, value, color, index }: { label: string; value: string | number; color: string; index: number }) {
  return (
    <motion.div
      variants={sectionVariants}
      initial="hidden"
      whileInView="visible"
      viewport={{ once: true, margin: '-50px' }}
      transition={{ duration: 0.5, delay: index * 0.1, ease: 'easeOut' }}
      className="bg-white dark:bg-gray-800 rounded-xl p-5 shadow-sm border border-gray-200 dark:border-gray-700"
    >
      <div className={`text-3xl font-bold ${color}`}>{value.toLocaleString()}</div>
      <div className="text-sm text-gray-500 dark:text-gray-400 mt-1">{label}</div>
    </motion.div>
  );
}

// Wrapper pour les charts — apparaît au scroll avec slide-up
function ChartSection({ children, delay = 0 }: { children: React.ReactNode; delay?: number }) {
  return (
    <motion.div
      variants={sectionVariants}
      initial="hidden"
      whileInView="visible"
      viewport={{ once: true, margin: '-80px' }}
      transition={{ duration: 0.6, delay, ease: 'easeOut' }}
      className="bg-white dark:bg-gray-800 rounded-xl p-6 shadow-sm border border-gray-200 dark:border-gray-700"
    >
      {children}
    </motion.div>
  );
}

export default function GitStatsPanel() {
  const t = useTranslations('gitstats');
  const locale = useLocale();
  const [stats, setStats] = useState<GitStatsResponse | null>(null);
  const [locStats, setLocStats] = useState<LangStatsResponse | null>(null);
  const [loading, setLoading] = useState(true);
  const [refreshing, setRefreshing] = useState(false);
  const [error, setError] = useState<string | null>(null);

  // Fetch stats — force=true pour ignorer le cache Redis
  const fetchStats = (force = false) => {
    const query = force ? '?force=true' : '';
    if (force) setRefreshing(true);

    fetch(`${API_URL}/api/v1/cv/gitstats${query}`)
      .then(res => {
        if (!res.ok) throw new Error('Failed to fetch git stats');
        return res.json();
      })
      .then((data: GitStatsResponse) => {
        setStats(data);
        writeSessionCache(CACHE_KEY, data); // mémorise pour la peinture instantanée du prochain montage
      })
      .catch(err => setError(err.message))
      .finally(() => {
        setLoading(false);
        setRefreshing(false);
      });
  };

  // Fetch LOC par repo — endpoint séparé (cache Redis à refresh lent, 6h), n'affecte pas le skeleton
  // ni l'écran d'erreur principal : si ça échoue, on affiche juste les repos sans le champ LOC.
  const fetchLocStats = (force = false) => {
    const query = force ? '?force=true' : '';
    fetch(`${API_URL}/api/v1/cv/loc${query}`)
      .then(res => {
        if (!res.ok) throw new Error('Failed to fetch LOC stats');
        return res.json();
      })
      .then((data: LangStatsResponse) => {
        setLocStats(data);
        writeSessionCache(LOC_CACHE_KEY, data);
      })
      .catch(() => { /* non-bloquant — la liste repos reste utilisable sans LOC */ });
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

  // Skeleton uniquement au TOUT premier chargement (pas de cache à peindre). Si on a du cache, on a
  // déjà setLoading(false) → on saute directement au rendu, la revalidation se fait en silence.
  if (loading && !stats) {
    return (
      <div className="space-y-4">
        <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
          {[...Array(4)].map((_, i) => (
            <div key={i} className="h-24 bg-gray-200 dark:bg-gray-700 rounded-xl animate-pulse" />
          ))}
        </div>
        <div className="h-64 bg-gray-200 dark:bg-gray-700 rounded-xl animate-pulse" />
      </div>
    );
  }

  // Écran d'erreur SEULEMENT si on n'a aucune donnée à montrer. Si du cache est déjà peint, une
  // revalidation de fond qui échoue ne doit PAS remplacer de bons chiffres par une page d'erreur.
  if (!stats) {
    return (
      <div className="text-center text-gray-500 py-12">
        <p>{t('unavailable')}</p>
        {error && <p className="text-sm mt-2 text-red-400">{error}</p>}
      </div>
    );
  }

  // Map date→commits GitLab pour joindre la série GitLab au chart commits/jour (2e courbe). Vide si le
  // backend ne renvoie pas gitlabDaily (GitLab non configuré) → la 2e courbe ne sera pas rendue.
  const gitlabByDate = new Map((stats.gitlabDaily ?? []).map((d) => [d.date, d.commits]));
  const hasGitlab = (stats.gitlabDaily?.length ?? 0) > 0;
  const daily = filterLast6Months(stats.daily, locale, gitlabByDate);
  const sortedRepos = sortReposByLastCommit(stats.repos);

  // Tooltip custom
  const ChartTooltip = ({ active, payload, label }: any) => {
    if (!active || !payload?.length) return null;
    return (
      <div className="bg-gray-900 text-white px-3 py-2 rounded-lg text-sm shadow-lg">
        <div className="font-medium mb-1">{label}</div>
        {payload.map((p: any) => (
          <div key={p.dataKey} className="flex items-center gap-2">
            <span className="w-2 h-2 rounded-full" style={{ backgroundColor: p.color }} />
            <span>{p.name}: {p.value.toLocaleString()}</span>
          </div>
        ))}
      </div>
    );
  };

  const kpis = [
    { label: t('commits6mo'), value: stats.totalCommits, color: 'text-blue-600' },
    { label: t('linesAdded'), value: stats.totalAdded, color: 'text-green-600' },
    { label: t('linesDeleted'), value: stats.totalDeleted, color: 'text-red-500' },
    { label: t('activeRepos'), value: stats.activeRepos, color: 'text-purple-600' },
  ];

  return (
    <div className="space-y-8">
      {/* Header avec bouton refresh */}
      <div className="flex justify-end">
        <button
          onClick={() => { fetchStats(true); fetchLocStats(true); }}
          disabled={refreshing}
          className="flex items-center gap-2 px-3 py-1.5 text-sm rounded-lg bg-gray-100 dark:bg-gray-700 hover:bg-gray-200 dark:hover:bg-gray-600 text-gray-600 dark:text-gray-300 transition-colors disabled:opacity-50"
        >
          <svg
            className={`w-4 h-4 ${refreshing ? 'animate-spin' : ''}`}
            fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}
          >
            <path strokeLinecap="round" strokeLinejoin="round" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
          </svg>
          {refreshing ? 'Refresh...' : 'Refresh'}
        </button>
      </div>

      {/* KPI cards — stagger animation */}
      <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
        {kpis.map((kpi, i) => (
          <StatCard key={kpi.label} index={i} {...kpi} />
        ))}
      </div>

      {/* Commits par jour — area chart */}
      <ChartSection>
        <h3 className="text-lg font-semibold mb-4">{t('commitsPerDay')}</h3>
        <ResponsiveContainer width="100%" height={280}>
          <AreaChart data={daily}>
            <defs>
              <linearGradient id="commitGrad" x1="0" y1="0" x2="0" y2="1">
                <stop offset="0%" stopColor="#3b82f6" stopOpacity={0.3} />
                <stop offset="100%" stopColor="#3b82f6" stopOpacity={0} />
              </linearGradient>
              <linearGradient id="gitlabGrad" x1="0" y1="0" x2="0" y2="1">
                <stop offset="0%" stopColor="#f97316" stopOpacity={0.35} />
                <stop offset="100%" stopColor="#f97316" stopOpacity={0} />
              </linearGradient>
            </defs>
            <CartesianGrid strokeDasharray="3 3" stroke="#374151" opacity={0.3} />
            <XAxis dataKey="label" tick={{ fontSize: 11 }} stroke="#9ca3af" interval="preserveStartEnd" />
            <YAxis tick={{ fontSize: 12 }} stroke="#9ca3af" />
            <Tooltip content={<ChartTooltip />} />
            {/* Légende affichée seulement quand la 2e série existe — sinon une seule courbe, inutile. */}
            {hasGitlab && <Legend wrapperStyle={{ fontSize: 12 }} />}
            {/* Courbe TOTAL (Gitea + GitLab mergés) — dessinée en premier (dessous). */}
            <Area
              type="monotone"
              dataKey="commits"
              name={hasGitlab ? t('commitsTotal') : t('commits')}
              stroke="#3b82f6"
              strokeWidth={2}
              fill="url(#commitGrad)"
              dot={false}
              activeDot={{ r: 4 }}
            />
            {/* Courbe GitLab seule (sous-ensemble du total) — au-dessus, orange, uniquement si présente. */}
            {hasGitlab && (
              <Area
                type="monotone"
                dataKey="gitlabCommits"
                name={t('commitsGitlab')}
                stroke="#f97316"
                strokeWidth={2}
                fill="url(#gitlabGrad)"
                dot={false}
                activeDot={{ r: 4 }}
              />
            )}
          </AreaChart>
        </ResponsiveContainer>
      </ChartSection>

      {/* Lignes ajoutées/supprimées par jour */}
      <ChartSection delay={0.1}>
        <h3 className="text-lg font-semibold mb-4">{t('linesPerDay')}</h3>
        <ResponsiveContainer width="100%" height={280}>
          <AreaChart data={daily}>
            <defs>
              <linearGradient id="addGrad" x1="0" y1="0" x2="0" y2="1">
                <stop offset="0%" stopColor="#22c55e" stopOpacity={0.3} />
                <stop offset="100%" stopColor="#22c55e" stopOpacity={0} />
              </linearGradient>
              <linearGradient id="delGrad" x1="0" y1="0" x2="0" y2="1">
                <stop offset="0%" stopColor="#ef4444" stopOpacity={0.3} />
                <stop offset="100%" stopColor="#ef4444" stopOpacity={0} />
              </linearGradient>
            </defs>
            <CartesianGrid strokeDasharray="3 3" stroke="#374151" opacity={0.3} />
            <XAxis dataKey="label" tick={{ fontSize: 11 }} stroke="#9ca3af" interval="preserveStartEnd" />
            <YAxis tick={{ fontSize: 12 }} stroke="#9ca3af" />
            <Tooltip content={<ChartTooltip />} />
            <Area
              type="monotone"
              dataKey="additions"
              name={t('seriesAdded')}
              stroke="#22c55e"
              strokeWidth={2}
              fill="url(#addGrad)"
              dot={false}
              activeDot={{ r: 4 }}
            />
            <Area
              type="monotone"
              dataKey="deletions"
              name={t('seriesRemoved')}
              stroke="#ef4444"
              strokeWidth={2}
              fill="url(#delGrad)"
              dot={false}
              activeDot={{ r: 4 }}
            />
          </AreaChart>
        </ResponsiveContainer>
      </ChartSection>

      {/* Repos actifs — liste compacte */}
      <ChartSection delay={0.15}>
        <h3 className="text-lg font-semibold mb-4">
          {t('repositories', { count: stats.repos.length })}
        </h3>
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-3">
          {sortedRepos.map((repo, i) => {
            const loc = locStats?.repos[repo.name];
            const lastCommit = lastCommitLabel(repo, locale);
            // Streak PAR REPO (pas global) : dérivé de commitDays de CE repo uniquement.
            const repoStreak = computeStreaks(repo.commitDays ?? [], todayIso());
            return (
              <motion.div
                key={repo.name}
                initial={{ opacity: 0, scale: 0.95 }}
                whileInView={{ opacity: 1, scale: 1 }}
                viewport={{ once: true }}
                transition={{ duration: 0.3, delay: i * 0.03 }}
                className="flex items-center gap-3 p-3 rounded-lg bg-gray-50 dark:bg-gray-700/50 border border-gray-100 dark:border-gray-600"
              >
                <div className="flex-1 min-w-0">
                  <div className="font-medium text-sm truncate">{repo.name}</div>
                  <div className="flex items-center gap-2 mt-1 flex-wrap">
                    {repo.language && (
                      <span className="text-xs px-2 py-0.5 rounded-full bg-blue-100 dark:bg-blue-900/30 text-blue-700 dark:text-blue-300">
                        {repo.language}
                      </span>
                    )}
                    {loc !== undefined && (
                      <span className="text-xs text-gray-500 dark:text-gray-400">
                        {t('repoLoc', { loc: loc.toLocaleString(locale) })}
                      </span>
                    )}
                  </div>
                  {lastCommit && (
                    <div className="text-xs text-gray-400 dark:text-gray-500 mt-0.5">
                      {t('repoLastCommit', { date: lastCommit })}
                    </div>
                  )}
                  {repoStreak.longest > 0 && (
                    <div className="text-xs text-orange-500 dark:text-orange-400 mt-0.5">
                      {t('repoStreak', { current: repoStreak.current, longest: repoStreak.longest })}
                    </div>
                  )}
                </div>
                {repo.stars > 0 && (
                  <span className="text-xs text-yellow-500">★ {repo.stars}</span>
                )}
              </motion.div>
            );
          })}
        </div>
      </ChartSection>
    </div>
  );
}
