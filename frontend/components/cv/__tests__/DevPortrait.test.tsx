import { render, screen } from '@testing-library/react';
import DevPortrait from '../DevPortrait';
import { GitStatsResponse, LangStatsResponse } from '@/lib/types';

// Fixtures minimales mais réalistes (ancre = date max = "2026-06-20").
const LOC: LangStatsResponse = {
  languages: {
    go: { language: 'go', bytes: 22800, loc: 600 },
    dart: { language: 'dart', bytes: 11400, loc: 300 },
  },
  totalLoc: 900,
  totalBytes: 34200,
  period: '6months',
};

const GIT: GitStatsResponse = {
  totalCommits: 95,
  totalAdded: 1000,
  totalDeleted: 200,
  activeRepos: 3,
  period: '6months',
  daily: [
    { date: '2026-05-10', commits: 3, additions: 0, deletions: 0 },  // previous
    { date: '2026-05-15', commits: 7, additions: 0, deletions: 0 },  // previous → 10
    { date: '2026-06-10', commits: 2, additions: 0, deletions: 0 },  // recent
    { date: '2026-06-20', commits: 40, additions: 0, deletions: 0 }, // recent → 42
  ],
  repos: [
    { name: 'goproj', description: '', language: 'Go', stars: 0, updatedAt: '2026-06-20', commits: 50,
      commitDays: ['2026-06-20', '2026-06-19'] },
    { name: 'dartproj', description: '', language: 'Dart', stars: 0, updatedAt: '2026-06-18', commits: 30,
      commitDays: ['2026-06-15'] },
  ],
};

// Stub global.fetch : route /cv/loc et /cv/gitstats vers les fixtures.
function mockFetchOk() {
  global.fetch = jest.fn((url: RequestInfo | URL) => {
    const u = String(url);
    if (u.includes('/cv/loc')) return Promise.resolve({ ok: true, json: () => Promise.resolve(LOC) } as Response);
    if (u.includes('/cv/gitstats')) return Promise.resolve({ ok: true, json: () => Promise.resolve(GIT) } as Response);
    return Promise.resolve({ ok: false } as Response);
  }) as jest.Mock;
}

describe('DevPortrait', () => {
  afterEach(() => {
    jest.restoreAllMocks();
  });

  it('rend les chiffres clés + le genre récent + momentum + repos chauds', async () => {
    mockFetchOk();
    render(await DevPortrait({ locale: 'fr' }));

    // Titre de la bande (i18n FR via mock)
    expect(screen.getByText('Portrait Dev')).toBeInTheDocument();
    // Hero : LOC total (900) et commits (95)
    expect(screen.getByText('900')).toBeInTheDocument();
    expect(screen.getByText('95')).toBeInTheDocument();
    // Momentum : 42 commits ces 30 jours, ×4.2 vs le mois précédent (42/10)
    expect(screen.getByText(/42 commits ces 30 jours/)).toBeInTheDocument();
    expect(screen.getByText('×4.2 vs le mois précédent')).toBeInTheDocument();
    // Repos chauds : le plus récent
    expect(screen.getByText('goproj')).toBeInTheDocument();
    // Lien vers le détail
    expect(screen.getByText('Voir la heatmap détaillée')).toBeInTheDocument();
  });

  it('rend null (bande masquée) si un endpoint est indisponible', async () => {
    global.fetch = jest.fn(() => Promise.reject(new Error('backend down'))) as jest.Mock;
    const result = await DevPortrait({ locale: 'fr' });
    expect(result).toBeNull();
  });
});
