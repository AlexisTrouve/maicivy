import { test, expect } from '@playwright/test';
import { trackPageErrors } from './helpers/pageErrors';

// E2E : la liste "Repos actifs" de /gitstats doit être triée par date du DERNIER COMMIT RÉEL
// (dernier élément de `commitDays`), pas par l'ordre brut renvoyé par l'API (qui n'a jamais été un
// ordre d'activité) ni par `updatedAt` (métadonnée Gitea qui bouge sur un simple push de tag/settings,
// piège déjà rencontré et corrigé pour le badge "Repos chauds" du CV, cf. CLAUDE.md).
//
// On mock la réponse `/cv/gitstats` avec 4 repos dans un ordre VOLONTAIREMENT mélangé (ni alphabétique,
// ni déjà trié par date) + un repo SANS commit sur la fenêtre (commitDays vide) qui doit finir en
// dernier. Si le composant ne triait pas (ou triait sur le mauvais champ), l'ordre observé dans le DOM
// serait celui du fixture, pas l'ordre attendu → test rouge.

const REPOS_FIXTURE = [
  { name: 'repo-old', description: '', language: 'Go', stars: 0, updatedAt: '2026-07-01', commits: 3, commits30d: 0, commitDays: ['2026-01-10', '2026-02-01'] },
  { name: 'repo-newest', description: '', language: 'TypeScript', stars: 0, updatedAt: '2026-01-01', commits: 5, commits30d: 5, commitDays: ['2026-03-01', '2026-06-20'] },
  // Aucun commit dans la fenêtre mais `updatedAt` très récent (ex: tag pushé) — doit rester en DERNIER,
  // preuve que le tri n'utilise pas `updatedAt` comme critère primaire.
  { name: 'repo-no-commits', description: '', language: 'Go', stars: 0, updatedAt: '2026-07-05', commits: 0, commits30d: 0, commitDays: [] },
  { name: 'repo-middle', description: '', language: 'Rust', stars: 0, updatedAt: '2026-04-01', commits: 2, commits30d: 0, commitDays: ['2026-02-15', '2026-04-15'] },
];

const MOCK_RESPONSE = {
  daily: [],
  repos: REPOS_FIXTURE,
  totalCommits: 10,
  totalAdded: 100,
  totalDeleted: 50,
  activeRepos: 3,
  period: '6mo',
};

// LOC mockées pour 2 des 4 repos seulement — vérifie à la fois le merge par nom (repo-newest,
// repo-middle affichent bien LEUR LOC) et l'absence de badge bidon pour repo-old/repo-no-commits
// (absents de la map, PAS un "0 ligne" trompeur).
const LOC_MOCK_RESPONSE = {
  languages: {},
  totalLoc: 0,
  totalBytes: 0,
  period: 'all-time',
  repos: { 'repo-newest': 120, 'repo-middle': 45 },
};

test.describe('Git Stats — ordre des repos (dernier commit réel)', () => {
  test('la liste "Repos actifs" est triée par dernier commit décroissant, repos sans commit en dernier', async ({ page }) => {
    const pageErrors = trackPageErrors(page);

    await page.route('**/cv/gitstats**', async (route) => {
      await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify(MOCK_RESPONSE) });
    });

    await page.goto('/fr/gitstats', { waitUntil: 'load' });

    const repoNames = page.locator('.font-medium.text-sm.truncate');
    await expect(repoNames).toHaveCount(REPOS_FIXTURE.length, { timeout: 10000 });

    const renderedOrder = await repoNames.allTextContents();
    // Ordre attendu : dernier commit décroissant (2026-06-20 > 2026-04-15 > 2026-02-01), puis le
    // repo sans commit en dernier malgré son `updatedAt` le plus récent des quatre.
    expect(renderedOrder).toEqual(['repo-newest', 'repo-middle', 'repo-old', 'repo-no-commits']);

    expect(pageErrors, 'exceptions JS:\n' + pageErrors.join('\n')).toHaveLength(0);
  });

  test('LOC et date de dernier commit affichées par repo, merge par nom avec /cv/loc', async ({ page }) => {
    const pageErrors = trackPageErrors(page);

    await page.route('**/cv/gitstats**', async (route) => {
      await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify(MOCK_RESPONSE) });
    });
    await page.route('**/cv/loc**', async (route) => {
      await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify(LOC_MOCK_RESPONSE) });
    });

    await page.goto('/fr/gitstats', { waitUntil: 'load' });
    await expect(page.locator('.font-medium.text-sm.truncate')).toHaveCount(REPOS_FIXTURE.length, { timeout: 10000 });

    const newestCard = page.locator('text=repo-newest').locator('xpath=ancestor::*[contains(@class,"rounded-lg")][1]');
    await expect(newestCard.getByText('120 lignes')).toBeVisible();
    await expect(newestCard.getByText(/2026/)).toBeVisible(); // date du dernier commit (2026-06-20) rendue

    const middleCard = page.locator('text=repo-middle').locator('xpath=ancestor::*[contains(@class,"rounded-lg")][1]');
    await expect(middleCard.getByText('45 lignes')).toBeVisible();

    // repo-old n'a PAS de LOC mockée → pas de badge "X lignes" affiché pour lui.
    const oldCard = page.locator('text=repo-old').locator('xpath=ancestor::*[contains(@class,"rounded-lg")][1]');
    await expect(oldCard.getByText(/lignes/)).toHaveCount(0);

    expect(pageErrors, 'exceptions JS:\n' + pageErrors.join('\n')).toHaveLength(0);
  });

  test('streak PAR REPO (pas global) : chaque carte calcule sa propre série depuis SON commitDays', async ({ page }) => {
    const pageErrors = trackPageErrors(page);

    // "Aujourd'hui" fixé à 2026-06-25 (horloge navigateur figée AVANT navigation, cf. Playwright Clock)
    // pour rendre le streak "en cours" déterministe — sinon la vraie date du run du test s'appliquerait.
    await page.clock.setFixedTime(new Date('2026-06-25T12:00:00'));

    // Deux repos avec des historiques de commit DISTINCTS → si le streak était calculé globalement
    // (agrégat tous repos confondus) au lieu de par repo, les deux cartes afficheraient la MÊME chose.
    const reposFixture = [
      {
        name: 'repo-streak-a', description: '', language: 'Go', stars: 0, updatedAt: '2026-06-24', commits: 5, commits30d: 5,
        // Run de 5 jours se terminant HIER (06-24) → streak EN COURS (grâce d'1 jour), record = 5.
        commitDays: ['2026-06-20', '2026-06-21', '2026-06-22', '2026-06-23', '2026-06-24'],
      },
      {
        name: 'repo-streak-b', description: '', language: 'Rust', stars: 0, updatedAt: '2026-05-02', commits: 2, commits30d: 0,
        // Run de 2 jours, ANCIEN (loin du 25/06) → streak en cours CASSÉE (0), mais record = 2.
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
    await expect(page.locator('.font-medium.text-sm.truncate')).toHaveCount(reposFixture.length, { timeout: 10000 });

    // Aucune carte KPI globale "Streak"/"Record" ne doit exister (retiré au profit du per-repo).
    await expect(page.getByText('Streak actuel (jours)')).toHaveCount(0);
    await expect(page.getByText('Record (jours)', { exact: true })).toHaveCount(0);

    const cardA = page.locator('text=repo-streak-a').locator('xpath=ancestor::*[contains(@class,"rounded-lg")][1]');
    await expect(cardA.getByText('🔥 5j · record 5j')).toBeVisible();

    const cardB = page.locator('text=repo-streak-b').locator('xpath=ancestor::*[contains(@class,"rounded-lg")][1]');
    await expect(cardB.getByText('🔥 0j · record 2j')).toBeVisible();

    expect(pageErrors, 'exceptions JS:\n' + pageErrors.join('\n')).toHaveLength(0);
  });
});
