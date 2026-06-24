import { test, expect } from '@playwright/test';
import { trackPageErrors } from './helpers/pageErrors';

// E2E de la heatmap retravaillée. Avant : "Unknown" partout, quasi vide, illisible. Après : sous-titre
// explicatif + points labellisés (zones synthétiques générées + clics réels, jamais "Unknown"/vide).
// On vérifie le sous-titre + qu'il y a bien des points (le générateur en garantit même à faible trafic).
// Cible la prod : PLAYWRIGHT_TEST_BASE_URL=https://maicivy.etheryale.com

test.describe('Analytics — heatmap (prod)', () => {
  test('sous-titre explicatif + heatmap peuplée (données générées + réelles)', async ({ page }) => {
    const errors = trackPageErrors(page);

    await page.goto('/fr/analytics', { waitUntil: 'load' });

    // Le sous-titre explicatif (nouveau — "on comprend enfin ce que ça dit").
    await expect(page.getByText(/Où les visiteurs cliquent/i)).toBeVisible({ timeout: 15000 });

    // La liste "Interactions principales" : libellés TOUJOURS visibles (plus besoin de survoler un
    // tooltip clippé). Au moins un libellé connu (zone synthétique la plus cliquée) doit apparaître.
    await expect(page.getByText('Interactions principales')).toBeVisible();
    await expect(page.getByText(/Voir le CV/).first()).toBeVisible();

    // "Basé sur N points" avec N ≥ 1 — le générateur peuple la heatmap même quand le trafic réel est faible.
    const based = page.getByText(/Basé sur \d+ points/);
    await expect(based).toBeVisible({ timeout: 15000 });
    const n = parseInt(((await based.innerText()).match(/\d+/) || ['0'])[0], 10);
    expect(n, 'la heatmap doit avoir des points').toBeGreaterThanOrEqual(1);

    expect(errors, 'exceptions JS:\n' + errors.join('\n')).toHaveLength(0);
  });
});
