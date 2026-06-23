import { test, expect } from '@playwright/test';

// E2E du filtre de période. AVANT : les boutons (DateFilter) ne pilotaient RIEN — état local sans
// consommateur → cliquer ne changeait aucune donnée. APRÈS : le filtre écrit ?period= dans l'URL et
// StatsOverview refetch. On prouve qu'un clic déclenche un fetch /analytics/stats avec la bonne période
// (si le filtre était toujours décoratif, ce fetch n'arriverait jamais → timeout = échec).
// Cible la prod : PLAYWRIGHT_TEST_BASE_URL=https://maicivy.etheryale.com

test.describe('Analytics — filtre de période (prod)', () => {
  test('cliquer un preset refetch /analytics/stats avec la bonne période + met à jour l’URL', async ({ page }) => {
    await page.goto('/fr/analytics', { waitUntil: 'load' });

    // 1er fetch au montage (période par défaut).
    await page.waitForResponse(
      (r) => r.url().includes('/analytics/stats') && r.url().includes('period='),
      { timeout: 15000 }
    );

    // Clic sur "30 derniers jours" → doit déclencher un fetch period=month.
    const monthRequest = page.waitForResponse(
      (r) => r.url().includes('/analytics/stats') && r.url().includes('period=month'),
      { timeout: 10000 }
    );
    await page.getByRole('button', { name: '30 derniers jours' }).click();
    await monthRequest; // décoratif → jamais émis → timeout ; câblé → OK

    // L'URL reflète la sélection (partageable / bookmarkable).
    await expect(page).toHaveURL(/period=30d/);
  });
});
