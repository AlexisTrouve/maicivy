import { test, expect } from '@playwright/test';
import testStats from '../../lib/test-stats.json';
import { trackPageErrors } from './helpers/pageErrors';

// E2E de la page /architecture après update : les métriques de tests sont LIVE (test-stats.json) et
// l'ancien chiffre figé "882" a disparu. Vérifie aussi le rendu sans erreur JS.
// Cible la prod : PLAYWRIGHT_TEST_BASE_URL=https://maicivy.etheryale.com

test.describe('Architecture (prod)', () => {
  test('métriques de tests live + plus de chiffre figé périmé', async ({ page }) => {
    const errors = trackPageErrors(page);

    await page.goto('/fr/architecture', { waitUntil: 'load' });

    // Titre visible (page rendue, gate `mounted` passé).
    await expect(page.locator('h1')).toBeVisible({ timeout: 15000 });

    // Diagramme système à jour : la couche Edge & Sécurité (construite cette session) doit apparaître.
    await expect(page.getByText('Frontdoor')).toBeVisible({ timeout: 15000 });
    await expect(page.getByText('Cloudflare')).toBeVisible();
    await expect(page.getByText('GitStats')).toBeVisible(); // API backend à jour

    const body = await page.locator('body').innerText();
    // Le total de tests LIVE (test-stats.json) apparaît (en ignorant les séparateurs de milliers).
    const digits = body.replace(/[\s,  .]/g, '');
    expect(digits).toContain(String(testStats.total));
    // L'ancienne valeur figée périmée ne doit plus apparaître nulle part.
    expect(body).not.toContain('882');

    // Le bouton "Voir sur GitHub" pointe vers le vrai repo public (était commenté / TODO USERNAME).
    await expect(page.getByRole('link', { name: /GitHub/i })).toHaveAttribute(
      'href',
      'https://github.com/AlexisTrouve/maicivy'
    );

    expect(errors, 'exceptions JS:\n' + errors.join('\n')).toHaveLength(0);
  });
});
