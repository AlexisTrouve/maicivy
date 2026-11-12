import {
  buildDays,
  groupWeeks,
  repoWeekActivity,
  longestActiveStreak,
  rankRepos,
  languageBreakdown,
  niceMax,
  sixMonthsBefore,
} from '../gitstatsReleve';
import { GitRepoStat, LangStatsResponse } from '../types';

const repo = (name: string, over: Partial<GitRepoStat>): GitRepoStat => ({
  name, description: '', language: 'Go', stars: 0, updatedAt: '2026-01-01', commits: 0, commits30d: 0, commitDays: [], ...over,
});

describe('sixMonthsBefore', () => {
  it('recule de six mois, débordement de fin de mois normalisé', () => {
    expect(sixMonthsBefore('2026-11-09')).toBe('2026-05-09');
    expect(sixMonthsBefore('2026-08-31')).toBe('2026-03-03');
    expect(sixMonthsBefore('2026-02-15')).toBe('2025-08-15');
  });
});

describe('buildDays', () => {
  it('bouche les jours absents de `daily` à zéro et couvre exactement six mois', () => {
    const days = buildDays(
      [
        { date: '2026-11-07', commits: 3, additions: 10, deletions: 2 },
        { date: '2026-11-09', commits: 1, additions: 4, deletions: 0 },
      ],
      [{ date: '2026-11-09', commits: 1, additions: 0, deletions: 0 }],
      '2026-11-09',
    );
    // 9 mai → 9 novembre inclus : 185 jours, comme le relevé de la maquette.
    expect(days).toHaveLength(185);
    expect(days[0].date).toBe('2026-05-09');
    const byDate = Object.fromEntries(days.map((d) => [d.date, d]));
    expect(byDate['2026-11-08']).toEqual({ date: '2026-11-08', commits: 0, additions: 0, deletions: 0, gitlab: 0 });
    expect(byDate['2026-11-07'].commits).toBe(3);
    expect(byDate['2026-11-09'].gitlab).toBe(1);
    // Aucun jour dédoublé ni sauté au passage à l'heure d'hiver (25 octobre).
    expect(new Set(days.map((d) => d.date)).size).toBe(days.length);
  });

  it("prolonge la série jusqu'au dernier jour servi s'il dépasse « aujourd'hui »", () => {
    const days = buildDays([{ date: '2026-11-10', commits: 2, additions: 0, deletions: 0 }], undefined, '2026-11-09');
    expect(days[days.length - 1]).toMatchObject({ date: '2026-11-10', commits: 2 });
  });

  it('écarte les jours servis hors de la fenêtre', () => {
    const days = buildDays([{ date: '2026-01-01', commits: 9, additions: 0, deletions: 0 }], undefined, '2026-11-09');
    expect(days.reduce((s, d) => s + d.commits, 0)).toBe(0);
  });
});

describe('groupWeeks', () => {
  it('coupe au lundi, première semaine incomplète, sommes conservées', () => {
    const days = buildDays(
      [
        { date: '2026-11-01', commits: 2, additions: 5, deletions: 1 }, // dimanche
        { date: '2026-11-02', commits: 4, additions: 7, deletions: 3 }, // lundi
      ],
      undefined,
      '2026-11-09',
    );
    const weeks = groupWeeks(days);
    // 9 mai 2026 est un samedi : la première semaine n'a que deux jours.
    expect(weeks[0].days).toEqual(['2026-05-09', '2026-05-10']);
    expect(weeks.slice(1).every((w) => new Date(`${w.date}T00:00:00Z`).getUTCDay() === 1)).toBe(true);
    const w1 = weeks.find((w) => w.days.includes('2026-11-01'))!;
    const w2 = weeks.find((w) => w.days.includes('2026-11-02'))!;
    expect(w1).not.toBe(w2);
    expect(w1.commits).toBe(2);
    expect(w2).toMatchObject({ date: '2026-11-02', commits: 4, additions: 7, deletions: 3 });
    expect(weeks.reduce((s, w) => s + w.days.length, 0)).toBe(days.length);
  });
});

describe('repoWeekActivity', () => {
  it('compte les jours de commit de CE dépôt par semaine', () => {
    const weeks = groupWeeks(buildDays([], undefined, '2026-11-09'));
    const activity = repoWeekActivity(['2026-11-02', '2026-11-03', '2026-11-09', '2025-01-01'], weeks);
    expect(activity).toHaveLength(weeks.length);
    expect(activity[activity.length - 1]).toBe(1); // lundi 9
    expect(activity[activity.length - 2]).toBe(2); // 2 et 3
    expect(activity.reduce((s, n) => s + n, 0)).toBe(3); // 2025-01-01 hors fenêtre
  });
});

describe('longestActiveStreak', () => {
  it('lit la plus longue série de jours avec commit, jours bouchés compris', () => {
    const days = buildDays(
      ['2026-11-01', '2026-11-02', '2026-11-03', '2026-11-05'].map((date) => ({ date, commits: 1, additions: 0, deletions: 0 })),
      undefined,
      '2026-11-09',
    );
    expect(longestActiveStreak(days)).toBe(3);
  });
});

describe('rankRepos', () => {
  it('classe par commits 30 j, puis 6 mois, puis dernier commit ; dormants à part', () => {
    const { active, dormant } = rankRepos([
      repo('vieux-gros', { commits: 900, commits30d: 1, commitDays: ['2026-11-01'], updatedAt: '2026-11-09' }),
      repo('dormant', { commits: 0, commits30d: 0, commitDays: [], updatedAt: '2026-11-10' }),
      repo('chaud', { commits: 10, commits30d: 9, commitDays: ['2026-10-01'] }),
      repo('egal-recent', { commits: 5, commits30d: 3, commitDays: ['2026-11-08'] }),
      repo('egal-ancien', { commits: 5, commits30d: 3, commitDays: ['2026-10-08'] }),
      repo('egal-plus-gros', { commits: 6, commits30d: 3, commitDays: ['2026-09-08'] }),
    ]);
    expect(active.map((r) => r.name)).toEqual(['chaud', 'egal-plus-gros', 'egal-recent', 'egal-ancien', 'vieux-gros']);
    expect(dormant.map((r) => r.name)).toEqual(['dormant']);
  });
});

describe('languageBreakdown', () => {
  it('garde les N premiers, agrège le reste, parts sommant à 1', () => {
    const lang: LangStatsResponse = {
      languages: {
        go: { language: 'go', bytes: 0, loc: 50 },
        'c++': { language: 'c++', bytes: 0, loc: 30 },
        zig: { language: 'zig', bytes: 0, loc: 15 },
        lua: { language: 'lua', bytes: 0, loc: 5 },
      },
      totalLoc: 999, // champ agrégé ignoré : on somme le détail
      totalBytes: 0,
      period: 'all-time',
      repos: {},
    };
    const b = languageBreakdown(lang, 2);
    expect(b.head).toEqual([
      { name: 'Go', loc: 50, share: 0.5 },
      { name: 'C++', loc: 30, share: 0.3 },
    ]);
    expect(b.rest).toEqual({ count: 2, loc: 20, share: 0.2 });
    expect(b.total).toBe(4);
  });
});

describe('niceMax', () => {
  it('arrondit au palier rond supérieur', () => {
    expect(niceMax(453)).toBe(500);
    expect(niceMax(2033)).toBe(2500);
    expect(niceMax(120)).toBe(200);
    expect(niceMax(1000)).toBe(1000);
    expect(niceMax(0)).toBe(1);
  });
});
