import { test, expect, type Page } from '@playwright/test';
import { trackPageErrors } from './helpers/pageErrors';

// ============================================================================
// E2E : la valeur d'une barre de /gitstats s'affiche TOUT DE SUITE, à la souris comme au doigt.
// ----------------------------------------------------------------------------
// QUOI : survoler (PC) ou toucher (téléphone) une barre des graphes « Cadence » et « Lignes
// écrites » fait apparaître un relevé avec la date et le nombre exact de CETTE barre.
// POURQUOI : la valeur ne passait que par un <title> SVG — infobulle native qui attend ~1 s et
// n'existe pas du tout sur téléphone, l'usage principal du site. Plainte d'Alexis : « dans les
// graph faut pas attendre avant d'indiquer le nombre ».
// COMMENT : l'API est simulée (deux jours seulement portent des commits : le premier jour de la
// série et aujourd'hui), donc la première et la dernière barre ont des valeurs connues, sur la
// série quotidienne (PC) comme hebdomadaire (téléphone). On vise ces barres aux bords du tracé —
// pas besoin de connaître le nombre de barres. Délai d'assertion court (200 ms) : un relevé qui
// attend, c'est le défaut corrigé.
// ============================================================================

// « Aujourd'hui » en heure locale, comme le calcule la page (même machine que le navigateur).
function isoLocal(d: Date): string {
  return `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}-${String(d.getDate()).padStart(2, '0')}`;
}
const now = new Date();
const TODAY = isoLocal(now);
// Premier jour de la série : même jour six mois plus tôt (même calcul UTC que buildDays).
const FIRST = new Date(Date.UTC(now.getFullYear(), now.getMonth() - 6, now.getDate())).toISOString().slice(0, 10);

const MOCK_RESPONSE = {
  daily: [
    { date: FIRST, commits: 12, additions: 300, deletions: 40 },
    { date: TODAY, commits: 37, additions: 4321, deletions: 123 },
  ],
  repos: [{ name: 'repo-a', description: '', language: 'Go', stars: 0, updatedAt: TODAY, commits: 49, commits30d: 37, commitDays: [TODAY] }],
  totalCommits: 49,
  totalAdded: 4621,
  totalDeleted: 163,
  activeRepos: 1,
  period: '6mo',
};

// Le relevé doit être là avant qu'un humain ne remarque une attente.
const INSTANT = { timeout: 200 };

async function openMocked(page: Page) {
  await page.route('**/cv/gitstats**', (route) =>
    route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify(MOCK_RESPONSE) }));
  // Le LOC n'est pas l'objet du test : un refus le laisse en réserve (chemin non bloquant).
  await page.route('**/cv/loc**', (route) => route.fulfill({ status: 503, body: '' }));
  await page.goto('/fr/gitstats', { waitUntil: 'load' });
}

// Le tracé VISIBLE d'une section (PC et téléphone ont chacun leur graphe, l'autre est masqué).
function plotOf(page: Page, heading: string) {
  const section = page.locator('section').filter({ has: page.getByRole('heading', { name: heading, exact: true }) });
  return {
    section,
    // Le <svg> du tracé (sans la colonne des graduations) : ses bords sont la première et la dernière barre.
    plot: section.locator('[role="img"]:visible > svg'),
    readout: section.locator('[data-testid="gitstats-bar-readout"]:visible'),
  };
}

test.describe('Git Stats — relevé immédiat de la valeur d\'une barre', () => {
  test('souris : survoler une barre affiche aussitôt sa valeur exacte, quitter le graphe la retire', async ({ page }) => {
    const pageErrors = trackPageErrors(page);
    await openMocked(page);

    const cadence = plotOf(page, 'Cadence');
    await expect(cadence.plot).toBeVisible({ timeout: 10000 });
    const box = (await cadence.plot.boundingBox())!;

    // Dernière barre (aujourd'hui, 37 commits).
    await page.mouse.move(box.x + box.width - 1, box.y + box.height / 2);
    await expect(cadence.readout).toBeVisible(INSTANT);
    await expect(cadence.readout).toHaveText(/^37 commits le /, INSTANT);
    // La colonne survolée est éclairée.
    await expect(cadence.plot.getByTestId('gitstats-bar-highlight')).toHaveCount(1, INSTANT);

    // Première barre (12 commits) : le relevé suit le pointeur.
    await page.mouse.move(box.x + 1, box.y + box.height / 2);
    await expect(cadence.readout).toHaveText(/^12 commits le /, INSTANT);

    // Hors du graphe : le relevé disparaît.
    await page.mouse.move(box.x + box.width / 2, box.y - 200);
    await expect(cadence.readout).toHaveCount(0, INSTANT);

    // Graphe des lignes : ajoutées ET supprimées de la barre survolée.
    const lines = plotOf(page, 'Lignes écrites');
    await lines.plot.scrollIntoViewIfNeeded();
    const lbox = (await lines.plot.boundingBox())!;
    await page.mouse.move(lbox.x + lbox.width - 1, lbox.y + lbox.height / 2);
    await expect(lines.readout).toHaveText(/\+4\s?321 · −123$/, INSTANT);

    expect(pageErrors, 'exceptions JS:\n' + pageErrors.join('\n')).toHaveLength(0);
  });

  test.describe('téléphone', () => {
    test.use({ viewport: { width: 390, height: 844 }, hasTouch: true, isMobile: true });

    test('toucher une barre affiche aussitôt la valeur de sa semaine', async ({ page }) => {
      const pageErrors = trackPageErrors(page);
      await openMocked(page);

      const cadence = plotOf(page, 'Cadence');
      await expect(cadence.plot).toBeVisible({ timeout: 10000 });
      await cadence.plot.scrollIntoViewIfNeeded();
      const box = (await cadence.plot.boundingBox())!;

      // Dernière semaine (celle d'aujourd'hui : 37 commits), puis la première (12).
      await page.touchscreen.tap(box.x + box.width - 1, box.y + box.height / 2);
      await expect(cadence.readout).toBeVisible(INSTANT);
      await expect(cadence.readout).toHaveText(/^37 commits la semaine du /, INSTANT);

      await page.touchscreen.tap(box.x + 1, box.y + box.height / 2);
      await expect(cadence.readout).toHaveText(/^12 commits la semaine du /, INSTANT);

      expect(pageErrors, 'exceptions JS:\n' + pageErrors.join('\n')).toHaveLength(0);
    });
  });
});
