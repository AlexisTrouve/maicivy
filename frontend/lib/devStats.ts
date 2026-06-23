// ============================================================================
// devStats — dérive les "stats pro" du Portrait Dev à partir des endpoints EXISTANTS
// (/cv/loc + /cv/gitstats). Aucune nouvelle source : on agrège ce qui est déjà servi.
// ----------------------------------------------------------------------------
// POURQUOI un module pur séparé du composant : toute la logique de calcul (totaux,
// répartition par langage, tendance 90j vs année, momentum) est testable unitairement
// sans rendu React. Les fenêtres temporelles sont ancrées sur la DATE MAX du dataset
// (pas sur `new Date()`) → fonctions déterministes : même input ⇒ même output, donc
// des tests stables avec une fixture figée.
// ============================================================================

import { GitStatsResponse, GitRepoStat, GitDayStat, LangStatsResponse } from './types';

// --- Helpers date (parse "YYYY-MM-DD" → nombre de jours, sans dépendance externe) ---

// Convertit "YYYY-MM-DD" en index de jour absolu (jours depuis l'époque). Sert uniquement
// à calculer des écarts en jours entre deux dates du dataset → robuste aux mois/années.
function dayIndex(iso: string): number {
  const [y, m, d] = iso.split('-').map(Number);
  // Date.UTC évite tout décalage de fuseau (on ne compare que des dates nues).
  return Math.floor(Date.UTC(y, m - 1, d) / 86_400_000);
}

// Date max présente dans une liste de jours-commit (toutes dates confondues). C'est notre
// "aujourd'hui" de référence : la fenêtre "90 derniers jours" part de la dernière activité réelle.
function maxDay(repos: GitRepoStat[]): number | null {
  let max: number | null = null;
  for (const r of repos) {
    for (const cd of r.commitDays ?? []) {
      const idx = dayIndex(cd);
      if (max === null || idx > max) max = idx;
    }
  }
  return max;
}

// --- LOC ---

export interface TopLang {
  language: string;
  loc: number;
  pct: number; // part du LOC total, 0..100
}

// Top N langages par LOC, avec leur part du total. Source : /cv/loc (déjà cache Redis backend).
export function topLanguages(lang: LangStatsResponse, n: number): TopLang[] {
  const entries = Object.values(lang.languages);
  const total = entries.reduce((s, e) => s + e.loc, 0) || 1;
  return entries
    .sort((a, b) => b.loc - a.loc)
    .slice(0, n)
    .map((e) => ({ language: e.language, loc: e.loc, pct: (100 * e.loc) / total }));
}

// LOC total tous langages confondus. On recalcule depuis le détail plutôt que de faire
// confiance à un champ agrégé (qui peut diverger si le backend filtre certaines clés).
export function totalLoc(lang: LangStatsResponse): number {
  return Object.values(lang.languages).reduce((s, e) => s + e.loc, 0);
}

// --- Genre récent (tendance par langage) ---

export type Trend = 'up' | 'down' | 'flat';

export interface GenreEntry {
  language: string;
  recentPct: number;   // part des jours-commit dans la fenêtre récente (ex: 90j)
  baselinePct: number; // part sur la fenêtre baseline (ex: 365j) — pour comparer
  trend: Trend;        // récent vs baseline : up si +3pts, down si −3pts, flat sinon
}

// Agrège les JOURS DE COMMIT par langage dominant du repo sur une fenêtre, et renvoie une
// répartition en %. POURQUOI jours-commit (et non commits) : `commitDays` est la donnée fiable
// par repo ; le langage est celui dominant du repo (linguist). C'est une tendance DIRECTIONNELLE
// (pas une analyse par ligne) — assumé et labellisé comme tel dans l'UI.
function langShare(repos: GitRepoStat[], anchor: number, windowDays: number): Map<string, number> {
  const counts = new Map<string, number>();
  for (const r of repos) {
    const lang = r.language || 'Autre';
    for (const cd of r.commitDays ?? []) {
      if (anchor - dayIndex(cd) <= windowDays) {
        counts.set(lang, (counts.get(lang) ?? 0) + 1);
      }
    }
  }
  return counts;
}

function toPct(counts: Map<string, number>): Map<string, number> {
  const total = Array.from(counts.values()).reduce((s, v) => s + v, 0) || 1;
  const pct = new Map<string, number>();
  counts.forEach((v, k) => pct.set(k, (100 * v) / total));
  return pct;
}

// "Mon genre récent" : top N langages sur la fenêtre récente, chacun annoté de sa tendance
// (comparaison récent vs baseline annuelle). Renvoie [] si aucune donnée de jours-commit.
export function recentGenre(
  repos: GitRepoStat[],
  recentDays: number,
  baselineDays: number,
  topN: number
): GenreEntry[] {
  const anchor = maxDay(repos);
  if (anchor === null) return [];

  const recent = toPct(langShare(repos, anchor, recentDays));
  const baseline = toPct(langShare(repos, anchor, baselineDays));

  return Array.from(recent.entries())
    .sort((a, b) => b[1] - a[1])
    .slice(0, topN)
    .map(([language, recentPct]) => {
      const baselinePct = baseline.get(language) ?? 0;
      const delta = recentPct - baselinePct;
      // Seuil de 3 points : en-deçà on considère que c'est du bruit (flat).
      const trend: Trend = delta >= 3 ? 'up' : delta <= -3 ? 'down' : 'flat';
      return { language, recentPct, baselinePct, trend };
    });
}

// --- Momentum (accélération du rythme de commits) ---

export interface Momentum {
  recent: number;   // commits sur la fenêtre récente (ex: 30 derniers jours calendaires)
  previous: number; // commits sur la fenêtre précédente de même durée
  ratio: number;    // recent / previous (0 si previous == 0)
}

// Compare les commits de la dernière fenêtre vs la précédente, en jours CALENDAIRES (le tableau
// `daily` ne contient que les jours actifs → on borne par date, pas par nombre d'entrées).
export function momentum(daily: GitDayStat[], windowDays: number): Momentum {
  if (daily.length === 0) return { recent: 0, previous: 0, ratio: 0 };
  const anchor = Math.max(...daily.map((d) => dayIndex(d.date)));
  let recent = 0;
  let previous = 0;
  for (const d of daily) {
    const age = anchor - dayIndex(d.date);
    if (age < windowDays) recent += d.commits;
    else if (age < 2 * windowDays) previous += d.commits;
  }
  return { recent, previous, ratio: previous > 0 ? recent / previous : 0 };
}

// --- Repos chauds ---

// Repos les plus récemment actifs (par updatedAt décroissant), tronqué à N. "Sur quoi je bosse
// en ce moment" — on garde name/language/commits pour l'affichage.
export function hotRepos(repos: GitRepoStat[], n: number): GitRepoStat[] {
  return [...repos].sort((a, b) => (b.updatedAt || '').localeCompare(a.updatedAt || '')).slice(0, n);
}

// --- Agrégat de haut niveau consommé par le composant ---

export interface DevPortraitData {
  totalLoc: number;
  langCount: number;
  topLangs: TopLang[];
  totalCommits: number;
  activeRepos: number;
  genre: GenreEntry[];
  momentum: Momentum;
  hot: GitRepoStat[];
}

// Construit toutes les stats du Portrait en une passe. Paramètres de fenêtre par défaut :
// 90j (récent) vs 365j (baseline) pour le genre, 30j pour le momentum, top 6 langages, 5 repos chauds.
export function buildDevPortrait(lang: LangStatsResponse, git: GitStatsResponse): DevPortraitData {
  return {
    totalLoc: totalLoc(lang),
    langCount: Object.keys(lang.languages).length,
    topLangs: topLanguages(lang, 6),
    totalCommits: git.totalCommits,
    activeRepos: git.activeRepos,
    genre: recentGenre(git.repos, 90, 365, 6),
    momentum: momentum(git.daily, 30),
    hot: hotRepos(git.repos, 5),
  };
}
