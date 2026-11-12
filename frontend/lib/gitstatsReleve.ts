// ============================================================================
// gitstatsReleve — données dérivées de la page /gitstats (maquette A, « Le relevé »).
// ----------------------------------------------------------------------------
// QUOI : la série journalière complète, son regroupement par semaine, l'activité hebdomadaire de
// chaque dépôt, le classement des dépôts et la répartition par langage.
// POURQUOI un module pur : le composant ne fait qu'afficher. Tout ce qui peut être faux (un jour
// manquant, une semaine coupée, un tri) se teste ici sans rendu React.
// Les dates sont des chaînes "YYYY-MM-DD" manipulées en UTC : on ne compare que des jours nus, un
// changement d'heure ne doit pas faire sauter ni dédoubler un jour.
// ============================================================================

import { computeStreaks, totalLoc } from './devStats';
import { GitDayStat, GitRepoStat, LangStatsResponse } from './types';

const DAY_MS = 86_400_000;

function dayMs(iso: string): number {
  const [y, m, d] = iso.split('-').map(Number);
  return Date.UTC(y, m - 1, d);
}

function isoOf(ms: number): string {
  return new Date(ms).toISOString().slice(0, 10);
}

/** Même jour, six mois plus tôt (31 août → 3 mars : débordement normalisé par Date.UTC). */
export function sixMonthsBefore(iso: string): string {
  const [y, m, d] = iso.split('-').map(Number);
  return isoOf(Date.UTC(y, m - 1 - 6, d));
}

export interface DayPoint {
  date: string;
  commits: number;
  additions: number;
  deletions: number;
  /** Commits GitLab seuls (sous-ensemble de `commits`). */
  gitlab: number;
}

export interface WeekPoint extends DayPoint {
  /** Lundi de la semaine — ou premier jour de la série si la première semaine est incomplète. */
  date: string;
  days: string[];
}

/**
 * Série journalière COMPLÈTE, de `today` − 6 mois à `today` (ou au dernier jour servi s'il est
 * postérieur : l'API compte en UTC, le visiteur peut avoir un jour de retard).
 * L'API omet les jours sans commit : sans ce bouchage, les barres glisseraient vers la gauche et
 * « jours avec un commit » vaudrait toujours 100 %.
 */
export function buildDays(daily: GitDayStat[], gitlabDaily: GitDayStat[] | undefined, today: string): DayPoint[] {
  const lastServed = daily.reduce((max, d) => (d.date > max ? d.date : max), today);
  const start = dayMs(sixMonthsBefore(today));
  const end = dayMs(lastServed);
  const byDate = new Map(daily.map((d) => [d.date, d]));
  const gitlab = new Map((gitlabDaily ?? []).map((d) => [d.date, d.commits]));
  const days: DayPoint[] = [];
  for (let t = start; t <= end; t += DAY_MS) {
    const date = isoOf(t);
    const d = byDate.get(date);
    days.push({
      date,
      commits: d?.commits ?? 0,
      additions: d?.additions ?? 0,
      deletions: d?.deletions ?? 0,
      gitlab: gitlab.get(date) ?? 0,
    });
  }
  return days;
}

/** Semaines du lundi au dimanche ; la première peut être incomplète. */
export function groupWeeks(days: DayPoint[]): WeekPoint[] {
  const weeks: WeekPoint[] = [];
  for (const d of days) {
    const monday = new Date(dayMs(d.date)).getUTCDay() === 1;
    if (!weeks.length || monday) {
      weeks.push({ date: d.date, days: [], commits: 0, additions: 0, deletions: 0, gitlab: 0 });
    }
    const w = weeks[weeks.length - 1];
    w.days.push(d.date);
    w.commits += d.commits;
    w.additions += d.additions;
    w.deletions += d.deletions;
    w.gitlab += d.gitlab;
  }
  return weeks;
}

/** Nombre de jours avec au moins un commit de CE dépôt, semaine par semaine (0..7). */
export function repoWeekActivity(commitDays: string[] | undefined, weeks: WeekPoint[]): number[] {
  const set = new Set(commitDays ?? []);
  return weeks.map((w) => w.days.filter((d) => set.has(d)).length);
}

/** Plus longue série de jours consécutifs avec au moins un commit, sur toute la série. */
export function longestActiveStreak(days: DayPoint[]): number {
  const active = days.filter((d) => d.commits > 0).map((d) => d.date);
  return computeStreaks(active, days.length ? days[days.length - 1].date : '1970-01-01').longest;
}

export function lastCommitDay(repo: GitRepoStat): string | null {
  return repo.commitDays?.length ? repo.commitDays[repo.commitDays.length - 1] : null;
}

/**
 * Dépôts actifs (au moins un jour de commit sur six mois) classés par commits des 30 derniers
 * jours, puis par commits sur six mois, puis par dernier commit réel. Les dépôts dormants sont
 * rendus à part : ils ne sont pas listés, seulement comptés.
 * Jamais `updatedAt` : il bouge sur un simple push de tag (cf. CLAUDE.md › Git Stats).
 */
export function rankRepos(repos: GitRepoStat[]): { active: GitRepoStat[]; dormant: GitRepoStat[] } {
  const active = repos
    .filter((r) => r.commitDays?.length)
    .sort(
      (a, b) =>
        (b.commits30d || 0) - (a.commits30d || 0) ||
        (b.commits || 0) - (a.commits || 0) ||
        (lastCommitDay(b) ?? '').localeCompare(lastCommitDay(a) ?? '') ||
        a.name.localeCompare(b.name),
    );
  const dormant = repos.filter((r) => !r.commitDays?.length);
  return { active, dormant };
}

// Noms d'affichage des langages : des noms propres techniques, identiques dans toutes les langues.
const LANGUAGE_NAMES: Record<string, string> = {
  python: 'Python', 'c++': 'C++', dart: 'Dart', typescript: 'TypeScript', c: 'C', javascript: 'JavaScript',
  go: 'Go', html: 'HTML', shell: 'Shell', rust: 'Rust', cmake: 'CMake', powershell: 'PowerShell', css: 'CSS',
  'c#': 'C#', kotlin: 'Kotlin', swift: 'Swift', java: 'Java', lua: 'Lua', glsl: 'GLSL', hlsl: 'HLSL',
};

export function languageName(key: string): string {
  return LANGUAGE_NAMES[key] ?? key.charAt(0).toUpperCase() + key.slice(1);
}

export interface LanguageShare {
  name: string;
  loc: number;
  /** Part du total, 0..1 */
  share: number;
}

/**
 * Les `top` premiers langages par lignes, puis l'agrégat des suivants. Les parts sont rapportées à
 * la SOMME du détail (comme `totalLoc`) : la bande fait exactement 100 %.
 */
export function languageBreakdown(lang: LangStatsResponse, top: number) {
  const total = totalLoc(lang) || 1;
  const all = Object.values(lang.languages).sort((a, b) => b.loc - a.loc);
  const head: LanguageShare[] = all.slice(0, top).map((l) => ({ name: languageName(l.language), loc: l.loc, share: l.loc / total }));
  const rest = all.slice(top);
  const restLoc = rest.reduce((s, l) => s + l.loc, 0);
  return { head, rest: { count: rest.length, loc: restLoc, share: restLoc / total }, total: all.length };
}

/** Plafond « rond » d'un axe : 1, 2, 2,5 ou 5 × 10^k au-dessus du maximum (1 pour une série nulle). */
export function niceMax(v: number): number {
  if (v <= 0) return 1;
  const p = 10 ** Math.floor(Math.log10(v));
  return [1, 2, 2.5, 5, 10].map((m) => m * p).find((m) => m >= v) as number;
}
