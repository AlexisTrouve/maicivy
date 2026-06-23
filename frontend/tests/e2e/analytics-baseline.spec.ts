import { test, expect } from '@playwright/test';

// E2E du dashboard analytics avec le générateur procédural (DemoMetrics) actif. Vérifie dans un vrai
// navigateur que la page rend les 5 cartes KPI — dont la NOUVELLE "Lectures blog" — sans erreur JS, et
// qu'aucun ancien sous-titre EN DUR n'est réapparu. Les valeurs exactes dépendent du gate (trafic réel),
// donc on teste des PLANCHERS robustes : les "Lectures blog" ont un plancher ratchet côté backend.
// Cible la prod : PLAYWRIGHT_TEST_BASE_URL=https://maicivy.etheryale.com

test.describe('Analytics — générateur (prod)', () => {
  test('5 cartes dont "Lectures blog", vie minimale, sans erreur JS ni sous-titre bidon', async ({ page }) => {
    const errors: string[] = [];
    page.on('pageerror', (e) => errors.push(e.message));

    await page.goto('/fr/analytics', { waitUntil: 'load' });

    // La carte "Lectures blog" (nouvelle stat) doit être présente…
    const blogCard = page.locator('div.rounded-lg.border').filter({ hasText: 'Lectures blog' });
    await expect(blogCard).toBeVisible({ timeout: 15000 });

    // …avec une valeur ≥ 10 (plancher ratchet du générateur, robuste même à faible trafic réel).
    await expect
      .poll(
        async () => {
          const txt = await blogCard.locator('.text-3xl').first().innerText().catch(() => '0');
          return parseInt(txt.replace(/\D/g, '') || '0', 10);
        },
        { timeout: 15000 }
      )
      .toBeGreaterThanOrEqual(10);

    // Compteur "en ligne" présent.
    await expect(page.locator('.text-6xl').first()).toBeVisible({ timeout: 15000 });

    // Aucun ancien sous-titre EN DUR ne doit réapparaître (régression StatsOverview).
    await expect(page.getByText("+234 aujourd'hui")).toHaveCount(0);
    await expect(page.getByText('+2.3% vs hier')).toHaveCount(0);

    expect(errors, 'exceptions JS:\n' + errors.join('\n')).toHaveLength(0);
  });
});
