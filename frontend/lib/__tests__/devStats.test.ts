import {
  totalLoc,
  topLanguages,
  recentGenre,
  momentum,
  hotRepos,
  buildDevPortrait,
} from '../devStats';
import { GitStatsResponse, LangStatsResponse } from '../types';

// --- Fixtures déterministes (ancre = date max présente = "2026-06-20") ---

const lang: LangStatsResponse = {
  languages: {
    go: { language: 'go', bytes: 3800, loc: 100 },
    dart: { language: 'dart', bytes: 11400, loc: 300 },
    rust: { language: 'rust', bytes: 3800, loc: 100 },
  },
  totalLoc: 500,
  totalBytes: 19000,
  period: '6months',
};

const git: GitStatsResponse = {
  totalCommits: 95,
  totalAdded: 1000,
  totalDeleted: 200,
  activeRepos: 3,
  period: '6months',
  // daily : seulement des jours actifs (pas de zéros) — momentum doit borner par DATE, pas par index.
  daily: [
    { date: '2026-04-01', commits: 99, additions: 0, deletions: 0 }, // age 80 → ni recent ni previous
    { date: '2026-05-05', commits: 2, additions: 0, deletions: 0 },  // age 46 → previous
    { date: '2026-05-20', commits: 3, additions: 0, deletions: 0 },  // age 31 → previous
    { date: '2026-06-05', commits: 2, additions: 0, deletions: 0 },  // age 15 → recent
    { date: '2026-06-20', commits: 8, additions: 0, deletions: 0 },  // age 0  → recent
  ],
  repos: [
    { name: 'goproj', description: '', language: 'Go', stars: 0, updatedAt: '2026-06-20', commits: 50, commits30d: 8,
      commitDays: ['2026-06-20', '2026-06-19', '2026-06-18'] }, // 3 jours, tous dans 90j
    { name: 'dartproj', description: '', language: 'Dart', stars: 0, updatedAt: '2026-06-18', commits: 40, commits30d: 20,
      commitDays: ['2026-06-15', '2026-06-10'] },               // 2 jours dans 90j ; plus chaud sur 30j
    { name: 'rustproj', description: '', language: 'Rust', stars: 0, updatedAt: '2026-01-02', commits: 5, commits30d: 0,
      commitDays: ['2026-01-01'] },                             // hors 90j, dans 365j
  ],
};

describe('devStats', () => {
  it('totalLoc somme les LOC de tous les langages', () => {
    expect(totalLoc(lang)).toBe(500);
  });

  it('topLanguages trie par LOC avec le pourcentage du total', () => {
    const top = topLanguages(lang, 3);
    expect(top[0]).toEqual({ language: 'dart', loc: 300, pct: 60 });
    expect(top.map((l) => l.language)).toEqual(['dart', 'go', 'rust']);
  });

  it('recentGenre : fenêtre 90j exclut ce qui est plus vieux', () => {
    const genre = recentGenre(git.repos, 90, 365, 6);
    // Récent (90j) = Go 3 jours, Dart 2 jours → Go 60%, Dart 40%. Rust (170j) absent.
    expect(genre.map((g) => g.language)).toEqual(['Go', 'Dart']);
    expect(genre[0].recentPct).toBeCloseTo(60);
    expect(genre[1].recentPct).toBeCloseTo(40);
  });

  it('recentGenre : tendance up quand récent dépasse la baseline de ≥3pts', () => {
    const genre = recentGenre(git.repos, 90, 365, 6);
    // Baseline 365j : Go 3/6=50%, Dart 2/6=33.3%. Récent : Go 60 (+10), Dart 40 (+6.7) → up tous deux.
    const go = genre.find((g) => g.language === 'Go')!;
    expect(go.baselinePct).toBeCloseTo(50);
    expect(go.trend).toBe('up');
    expect(genre.find((g) => g.language === 'Dart')!.trend).toBe('up');
  });

  it('momentum borne par jours calendaires (pas par nombre d’entrées)', () => {
    const m = momentum(git.daily, 30);
    // recent (age<30) = 06-20(8) + 06-05(2) = 10 ; previous (30..59) = 05-20(3) + 05-05(2) = 5.
    expect(m.recent).toBe(10);
    expect(m.previous).toBe(5);
    expect(m.ratio).toBe(2);
  });

  it('momentum : ratio 0 si pas de période précédente', () => {
    const m = momentum([{ date: '2026-06-20', commits: 4, additions: 0, deletions: 0 }], 30);
    expect(m.previous).toBe(0);
    expect(m.ratio).toBe(0);
  });

  it('hotRepos trie par commits des 30 derniers jours (pas par updatedAt)', () => {
    const hot = hotRepos(git.repos, 2);
    // dartproj (20 commits/30j) passe devant goproj (8) malgré un updatedAt plus ancien → tri par commits30d.
    expect(hot.map((r) => r.name)).toEqual(['dartproj', 'goproj']);
  });

  it('buildDevPortrait agrège tout sans crasher', () => {
    const d = buildDevPortrait(lang, git);
    expect(d.totalLoc).toBe(500);
    expect(d.langCount).toBe(3);
    expect(d.totalCommits).toBe(95);
    expect(d.activeRepos).toBe(3);
    expect(d.genre.length).toBe(2);
    expect(d.momentum.ratio).toBe(2);
    expect(d.hot.length).toBe(3);
  });
});
