import { test, expect } from '@playwright/test';
import { trackPageErrors } from './helpers/pageErrors';

// ============================================================================
// E2E : l'action principale de /gitstats (maquette A), « Voir les N autres dépôts actifs ».
// ----------------------------------------------------------------------------
// La table montre les douze premiers dépôts actifs ; le bouton d'or révèle les suivants. On clique
// pour de vrai et on vérifie que la liste S'ALLONGE jusqu'au total des dépôts actifs, que le focus
// passe au premier dépôt révélé, et que les dépôts dormants n'y entrent jamais.
// Fixture : 15 dépôts actifs + 2 dormants — le décompte attendu ne dépend pas de la prod.
// ============================================================================

const ACTIVE = Array.from({ length: 15 }, (_, i) => ({
  name: `repo-${String(i + 1).padStart(2, '0')}`,
  description: '',
  language: 'Go',
  stars: 0,
  updatedAt: '2026-06-01',
  commits: 100 - i,
  // Décroissant : repo-01 est le plus actif, l'ordre attendu est l'ordre des noms.
  commits30d: 30 - i,
  commitDays: ['2026-06-01'],
}));
const DORMANT = ['dormant-a', 'dormant-b'].map((name) => ({
  name, description: '', language: 'Go', stars: 0, updatedAt: '2026-06-20', commits: 0, commits30d: 0, commitDays: [],
}));

const MOCK_RESPONSE = {
  daily: [],
  repos: [...DORMANT, ...ACTIVE],
  totalCommits: 1000,
  totalAdded: 100,
  totalDeleted: 50,
  activeRepos: ACTIVE.length,
  period: '6mo',
};

test.describe('Git Stats — voir les autres dépôts', () => {
  test('le bouton révèle les dépôts actifs au-delà des douze premiers', async ({ page }) => {
    const pageErrors = trackPageErrors(page);
    await page.route('**/cv/gitstats**', async (route) => {
      await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify(MOCK_RESPONSE) });
    });

    await page.goto('/fr/gitstats', { waitUntil: 'load' });
    const names = page.getByTestId('gitstats-repo-name');
    await expect(names).toHaveCount(12, { timeout: 10000 });
    expect(await names.allTextContents()).toEqual(ACTIVE.slice(0, 12).map((r) => r.name));

    const more = page.getByRole('button', { name: 'Voir les 3 autres dépôts actifs' });
    await expect(more).toBeVisible();
    await more.click();

    // La liste s'allonge jusqu'au total des dépôts actifs, dans le même ordre ; aucun dormant.
    await expect(names).toHaveCount(ACTIVE.length);
    expect(await names.allTextContents()).toEqual(ACTIVE.map((r) => r.name));
    await expect(names.getByText(/^dormant-/)).toHaveCount(0);
    await expect(page.getByTestId('gitstats-dormant')).toHaveText('2 dépôts sans commit depuis six mois restent hors de la liste.');

    // L'action est faite : le bouton disparaît et le focus passe au premier dépôt révélé.
    await expect(more).toHaveCount(0);
    await expect(page.getByTestId('gitstats-repo').nth(12)).toBeFocused();

    expect(pageErrors, 'exceptions JS:\n' + pageErrors.join('\n')).toHaveLength(0);
  });

  test('prod : la liste réelle s\'allonge au clic', async ({ page }) => {
    const pageErrors = trackPageErrors(page);
    await page.goto('/fr/gitstats', { waitUntil: 'load' });
    const rows = page.getByTestId('gitstats-repo');
    await expect(rows).toHaveCount(12, { timeout: 20000 });

    const more = page.getByRole('button', { name: /^Voir les \d+ autres dépôts actifs$/ });
    const label = await more.textContent();
    const announced = Number((label ?? '').replace(/\D/g, ''));
    expect(announced).toBeGreaterThan(0);

    await more.click();
    await expect(rows).toHaveCount(12 + announced);

    expect(pageErrors, 'exceptions JS:\n' + pageErrors.join('\n')).toHaveLength(0);
  });
});
