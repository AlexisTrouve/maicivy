import { test, expect, Page } from '@playwright/test';
import { trackPageErrors } from './helpers/pageErrors';

// E2E de la fiche détail d'un skill : charge réellement /fr/cv dans un navigateur, CLIQUE une pastille
// de compétence, et vérifie que la fiche s'ouvre avec les stats puis se ferme. Doctrine : une UI
// cliquable sans test qui clique pour de vrai = non vérifiée.
// Tourne contre la prod : PLAYWRIGHT_TEST_BASE_URL=https://maicivy.etheryale.com

test.describe('Skill detail (prod)', () => {
  test('clic sur une pastille → fiche détail ouverte avec stats, puis fermeture', async ({ page }) => {
    const pageErrors = trackPageErrors(page);
    await page.goto('/fr/cv?theme=fullstack', { waitUntil: 'load' });

    // La page CV a une section #skills avec les pastilles (boutons data-testid="skill-badge-*").
    const firstBadge = page.locator('[data-testid^="skill-badge-"]').first();
    await firstBadge.waitFor({ state: 'visible', timeout: 15000 });
    const skillName = (await firstBadge.innerText()).trim();

    // Clic réel sur la pastille
    await firstBadge.click();

    // La fiche s'ouvre
    const modal = page.locator('[data-testid="skill-detail-modal"]');
    await expect(modal).toBeVisible({ timeout: 5000 });

    // Elle contient le nom du skill cliqué (titre) + le libellé "projets" (toujours présent)
    await expect(modal).toContainText(skillName);
    await expect(modal).toContainText(/projets|projects|Projekte|progetti|项目/i);

    // Fermeture via Échap → la fiche disparaît
    await page.keyboard.press('Escape');
    await expect(modal).toBeHidden({ timeout: 5000 });

    expect(pageErrors, 'exceptions JS:\n' + pageErrors.join('\n')).toHaveLength(0);
  });

  test('une pastille de langage affiche un compteur de LOC', async ({ page }) => {
    await page.goto('/fr/cv?theme=fullstack', { waitUntil: 'load' });

    // Cherche une pastille de langage probable (présent dans le portfolio + matché par /cv/loc).
    // On essaie plusieurs noms ; le premier visible sert de cible. Si aucun (données changeantes),
    // le test est neutralisé (skip) plutôt que faussement rouge.
    const candidates = ['Go', 'TypeScript', 'Python', 'JavaScript', 'C++'];
    let target = null as ReturnType<Page['getByRole']> | null;
    for (const name of candidates) {
      const btn = page.getByRole('button', { name: new RegExp(`^${name}\\b`, 'i') }).first();
      if (await btn.count()) {
        target = btn;
        break;
      }
    }
    test.skip(!target, 'aucun skill-langage connu présent sur la page');

    await target!.click();
    const modal = page.locator('[data-testid="skill-detail-modal"]');
    await expect(modal).toBeVisible({ timeout: 5000 });

    // Le bloc LOC affiche "~<nombre> lignes de code" (libellé i18n FR). Présent uniquement si le
    // backend /cv/loc a renvoyé des octets pour ce langage — preuve que la chaîne LOC fonctionne.
    await expect(modal).toContainText(/lignes de code/i, { timeout: 5000 });
  });
});
