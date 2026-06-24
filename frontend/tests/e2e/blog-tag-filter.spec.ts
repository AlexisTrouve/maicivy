import { test, expect } from '@playwright/test';
import { trackPageErrors } from './helpers/pageErrors';

// E2E du filtre de tags du blog : charge réellement /fr/blog dans un navigateur, CLIQUE un
// tag dans la sidebar, et vérifie que la grille d'articles se restreint au bon nombre, puis
// que "Tous" remet tout. Doctrine : une UI cliquable sans test qui clique pour de vrai = non
// vérifiée. Robuste aux données changeantes : on LIT le compte attendu depuis le bouton du
// tag (rendu par l'UI) plutôt que de coder en dur un nombre d'articles.
//
// Tourne contre la prod : PLAYWRIGHT_TEST_BASE_URL=https://maicivy.etheryale.com

test.describe('Blog — filtre par tag', () => {
  test('clic sur un tag → la grille se filtre au bon nombre, "Tous" réinitialise', async ({ page }) => {
    const pageErrors = trackPageErrors(page);
    await page.goto('/fr/blog', { waitUntil: 'load' });

    // La sidebar de filtrage doit être présente.
    const filter = page.locator('[data-testid="tag-filter"]');
    await filter.waitFor({ state: 'visible', timeout: 15000 });

    // Cartes d'articles dans la grille.
    const cards = page.locator('[data-testid="blog-grid"] [data-testid="blog-card"]');
    await expect(cards.first()).toBeVisible({ timeout: 15000 });
    const totalCards = await cards.count();
    expect(totalCards).toBeGreaterThan(0);

    // Premier bouton de tag (hors "Tous"). Si aucun tag (données vides), on neutralise.
    const firstTag = filter
      .locator('[data-testid^="tag-"]:not([data-testid="tag-all"])')
      .first();
    test.skip((await firstTag.count()) === 0, 'aucun tag présent sur le blog');
    await firstTag.waitFor({ state: 'visible', timeout: 5000 });

    // Le compte attendu = le nombre affiché dans le bouton (2e span). Robuste aux données.
    const countText = (await firstTag.locator('span').last().innerText()).trim();
    const expectedCount = parseInt(countText, 10);
    const tagTestId = await firstTag.getAttribute('data-testid');
    expect(Number.isFinite(expectedCount)).toBe(true);
    expect(expectedCount).toBeGreaterThan(0);
    // Un tag ne peut pas avoir plus d'articles que le total — sinon le compteur est faux.
    expect(expectedCount).toBeLessThanOrEqual(totalCards);

    // Clic réel sur le tag → la grille doit se restreindre EXACTEMENT à expectedCount cartes.
    await firstTag.click();
    await expect(cards).toHaveCount(expectedCount, { timeout: 5000 });

    // La pastille de filtre actif apparaît.
    await expect(page.locator('[data-testid="active-filter"]')).toBeVisible();

    // "Tous" remet la grille au total initial.
    await page.locator('[data-testid="tag-all"]').click();
    await expect(cards).toHaveCount(totalCards, { timeout: 5000 });
    await expect(page.locator('[data-testid="active-filter"]')).toHaveCount(0);

    // Aucune exception JS pendant le parcours.
    expect(pageErrors, 'exceptions JS:\n' + pageErrors.join('\n')).toHaveLength(0);

    // Sanity : on a bien testé un vrai tag identifiable.
    expect(tagTestId).toMatch(/^tag-/);
  });
});
