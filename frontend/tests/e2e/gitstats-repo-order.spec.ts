import { test, expect } from '@playwright/test';
import { trackPageErrors } from './helpers/pageErrors';

// E2E : la table « Dépôts » de /gitstats (maquette A, « Le relevé ») est classée par commits des
// 30 DERNIERS JOURS — l'activité de développement en ce moment — et jamais par l'ordre brut de
// l'API (qui n'a jamais été un ordre d'activité), par `updatedAt` (métadonnée Gitea qui bouge sur
// un simple push de tag, cf. CLAUDE.md › Git Stats), par le total six mois ni par la date du
// dernier commit. Les dépôts sans aucun commit sur six mois ne sont PAS listés : ils sont comptés
// sous la table.
//
// Le fixture est construit pour que CHACUN des mauvais critères donne un ordre différent de
// l'ordre attendu : si le composant triait sur le mauvais champ, le test serait rouge.

const REPOS_FIXTURE = [
  // Dernier commit le plus récent et `updatedAt` récent, mais peu de commits sur 30 jours.
  { name: 'repo-recent', description: '', language: 'Go', stars: 0, updatedAt: '2026-06-24', commits: 40, commits30d: 2, commitDays: ['2026-01-10', '2026-06-24'] },
  // Aucun commit dans la fenêtre mais `updatedAt` le plus récent de tous (ex : tag poussé) : hors liste.
  { name: 'repo-no-commits', description: '', language: 'Go', stars: 0, updatedAt: '2026-07-05', commits: 0, commits30d: 0, commitDays: [] },
  // Le plus gros total six mois, rien sur 30 jours.
  { name: 'repo-big-old', description: '', language: 'C++', stars: 0, updatedAt: '2026-03-01', commits: 50, commits30d: 0, commitDays: ['2026-02-01'] },
  { name: 'repo-hot', description: '', language: 'TypeScript', stars: 0, updatedAt: '2026-01-01', commits: 10, commits30d: 9, commitDays: ['2026-03-01', '2026-06-10'] },
  { name: 'repo-warm', description: '', language: 'Rust', stars: 0, updatedAt: '2026-04-01', commits: 5, commits30d: 5, commitDays: ['2026-02-15', '2026-06-01'] },
];

const MOCK_RESPONSE = {
  daily: [],
  repos: REPOS_FIXTURE,
  totalCommits: 105,
  totalAdded: 100,
  totalDeleted: 50,
  activeRepos: 4,
  period: '6mo',
};

// LOC mockées pour 2 des dépôts seulement — vérifie à la fois le merge par nom (repo-hot,
// repo-warm affichent bien LEUR LOC) et l'absence de chiffre bidon pour les autres (absents de la
// map, PAS un « 0 ligne » trompeur).
const LOC_MOCK_RESPONSE = {
  languages: {},
  totalLoc: 0,
  totalBytes: 0,
  period: 'all-time',
  repos: { 'repo-hot': 120, 'repo-warm': 45 },
};

const rowOf = (page: import('@playwright/test').Page, name: string) =>
  page.getByTestId('gitstats-repo').filter({ has: page.getByTestId('gitstats-repo-name').getByText(name, { exact: true }) });

test.describe('Git Stats — classement des dépôts (commits sur 30 jours)', () => {
  test('la table est classée par commits 30 j décroissants, dépôts sans commit hors liste et comptés', async ({ page }) => {
    const pageErrors = trackPageErrors(page);

    await page.route('**/cv/gitstats**', async (route) => {
      await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify(MOCK_RESPONSE) });
    });

    await page.goto('/fr/gitstats', { waitUntil: 'load' });

    const repoNames = page.getByTestId('gitstats-repo-name');
    await expect(repoNames).toHaveCount(REPOS_FIXTURE.length - 1, { timeout: 10000 });

    // 9 > 5 > 2 > 0. Par dernier commit on aurait recent, hot, warm, big-old ; par total six mois
    // big-old, recent, hot, warm ; par `updatedAt` recent, warm, big-old, hot.
    expect(await repoNames.allTextContents()).toEqual(['repo-hot', 'repo-warm', 'repo-recent', 'repo-big-old']);

    // Le dépôt sans commit n'est pas listé, malgré son `updatedAt` le plus récent ; il est compté.
    await expect(page.getByTestId('gitstats-repo-name').getByText('repo-no-commits')).toHaveCount(0);
    await expect(page.getByTestId('gitstats-dormant')).toHaveText('1 dépôt sans commit depuis six mois reste hors de la liste.');

    expect(pageErrors, 'exceptions JS:\n' + pageErrors.join('\n')).toHaveLength(0);
  });

  test('LOC et date de dernier commit affichées par dépôt, merge par nom avec /cv/loc', async ({ page }) => {
    const pageErrors = trackPageErrors(page);

    await page.route('**/cv/gitstats**', async (route) => {
      await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify(MOCK_RESPONSE) });
    });
    await page.route('**/cv/loc**', async (route) => {
      await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify(LOC_MOCK_RESPONSE) });
    });

    await page.goto('/fr/gitstats', { waitUntil: 'load' });
    await expect(page.getByTestId('gitstats-repo-name')).toHaveCount(REPOS_FIXTURE.length - 1, { timeout: 10000 });

    const hotRow = rowOf(page, 'repo-hot');
    await expect(hotRow.getByText('120 lignes')).toBeVisible();
    await expect(hotRow.getByText('10 juin')).toBeVisible(); // dernier commit réel (2026-06-10), pas updatedAt

    await expect(rowOf(page, 'repo-warm').getByText('45 lignes')).toBeVisible();

    // repo-recent n'a PAS de LOC mockée → aucun « X lignes » pour lui.
    await expect(rowOf(page, 'repo-recent').getByText(/lignes/)).toHaveCount(0);

    expect(pageErrors, 'exceptions JS:\n' + pageErrors.join('\n')).toHaveLength(0);
  });

  test('série PAR DÉPÔT (pas globale) : chaque rangée calcule sa propre série depuis SON commitDays', async ({ page }) => {
    const pageErrors = trackPageErrors(page);

    // "Aujourd'hui" fixé à 2026-06-25 (horloge navigateur figée AVANT navigation, cf. Playwright Clock)
    // pour rendre la série « en cours » déterministe — sinon la vraie date du run du test s'appliquerait.
    await page.clock.setFixedTime(new Date('2026-06-25T12:00:00'));

    // Deux dépôts avec des historiques DISTINCTS → si la série était calculée globalement (agrégat
    // de tous les dépôts) au lieu de par dépôt, les deux rangées diraient la MÊME chose.
    const reposFixture = [
      {
        name: 'repo-streak-a', description: '', language: 'Go', stars: 0, updatedAt: '2026-06-24', commits: 5, commits30d: 5,
        // Série de 5 jours se terminant HIER (06-24) → série EN COURS (grâce d'1 jour), record = 5.
        commitDays: ['2026-06-20', '2026-06-21', '2026-06-22', '2026-06-23', '2026-06-24'],
      },
      {
        name: 'repo-streak-b', description: '', language: 'Rust', stars: 0, updatedAt: '2026-05-02', commits: 2, commits30d: 0,
        // Série de 2 jours, ANCIENNE (loin du 25/06) → série en cours CASSÉE (0), mais record = 2.
        commitDays: ['2026-05-01', '2026-05-02'],
      },
    ];
    const mockResponse = { ...MOCK_RESPONSE, repos: reposFixture };

    await page.route('**/cv/gitstats**', async (route) => {
      await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify(mockResponse) });
    });
    await page.route('**/cv/loc**', async (route) => {
      await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify(LOC_MOCK_RESPONSE) });
    });

    await page.goto('/fr/gitstats', { waitUntil: 'load' });
    await expect(page.getByTestId('gitstats-repo-name')).toHaveCount(reposFixture.length, { timeout: 10000 });

    // Aucun chiffre global « Streak »/« Record » ne doit exister (retiré au profit du par-dépôt).
    await expect(page.getByText('Streak actuel (jours)')).toHaveCount(0);
    await expect(page.getByText('Record (jours)', { exact: true })).toHaveCount(0);

    // La série se lit sur la bande « six mois » de chaque rangée : nom accessible + infobulle.
    // Le nombre de jours compte les jours de commit de CE dépôt dans la fenêtre.
    const stripA = rowOf(page, 'repo-streak-a').getByRole('img');
    await expect(stripA).toHaveAttribute('aria-label', '5 jours avec commit sur six mois · série en cours 5 j · record 5 j');
    await expect(stripA).toHaveAttribute('title', '5 jours avec commit sur six mois · série en cours 5 j · record 5 j');

    const stripB = rowOf(page, 'repo-streak-b').getByRole('img');
    await expect(stripB).toHaveAttribute('aria-label', '2 jours avec commit sur six mois · série en cours 0 j · record 2 j');

    expect(pageErrors, 'exceptions JS:\n' + pageErrors.join('\n')).toHaveLength(0);
  });
});
