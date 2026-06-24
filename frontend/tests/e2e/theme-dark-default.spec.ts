import { test, expect } from '@playwright/test';
import { trackPageErrors } from './helpers/pageErrors';

// E2E du thème par défaut : un visiteur SANS choix stocké doit voir le site en DARK dès le 1er paint
// (esthétique dark-first). Le script inline du <head> pose la classe `dark` sur <html> avant l'hydratation.
// CONTEXTE : on a basculé le défaut de "système → light" vers "dark forcé" ; le choix explicite de
// l'utilisateur (localStorage `theme`, posé par le toggle) garde TOUJOURS la priorité.
// Doctrine : une UI sans test qui l'observe pour de vrai = non vérifiée. On charge la page dans un
// navigateur réel et on lit la classe de <html>. Cible la prod : PLAYWRIGHT_TEST_BASE_URL.
//
// NOTE Playwright : chaque test a un contexte vierge (pas de localStorage hérité) → "1re visite" gratuite.

test.describe('Thème — dark par défaut (prod)', () => {
  test('1re visite sur la home → <html class="dark"> sans choix stocké', async ({ page }) => {
    const errors = trackPageErrors(page);

    await page.goto('/fr', { waitUntil: 'load' });

    // Le script anti-flash du <head> a posé `dark` sur <html> avant le paint.
    await expect(page.locator('html')).toHaveClass(/dark/);
    // Preuve qu'aucun choix n'a "fuité" : le localStorage est vierge à la 1re visite.
    const stored = await page.evaluate(() => localStorage.getItem('theme'));
    expect(stored).toBeNull();

    expect(errors, 'exceptions JS:\n' + errors.join('\n')).toHaveLength(0);
  });

  test('1re visite sur /cv → dark par défaut (la page vedette)', async ({ page }) => {
    const errors = trackPageErrors(page);

    await page.goto('/fr/cv', { waitUntil: 'load' });

    await expect(page.locator('html')).toHaveClass(/dark/);

    expect(errors, 'exceptions JS:\n' + errors.join('\n')).toHaveLength(0);
  });

  test('un choix stocké "light" gagne sur le défaut dark', async ({ page }) => {
    // addInitScript s'exécute AVANT tout script de page → on simule un utilisateur ayant déjà choisi
    // light via le toggle. Le défaut dark ne doit PAS réécraser ce choix.
    await page.addInitScript(() => localStorage.setItem('theme', 'light'));

    await page.goto('/fr/cv', { waitUntil: 'load' });

    await expect(page.locator('html')).not.toHaveClass(/dark/);
  });
});
