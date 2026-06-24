import { test, expect, Page } from '@playwright/test';
import { isEnvironmentalPageError } from './helpers/pageErrors';

// E2E i18n du CV : charge réellement /en/cv et /fr/cv dans un navigateur, vérifie que le contenu
// s'affiche dans la bonne langue ET qu'aucune exception JS ne survient (hydratation incluse).
// Tourne contre la prod : PLAYWRIGHT_TEST_BASE_URL=https://maicivy.etheryale.com

function trackErrors(page: Page) {
  const pageErrors: string[] = [];
  const consoleErrors: string[] = [];
  // Bruit cross-origin WebKit (« access control checks ») filtré — voir helpers/pageErrors.
  page.on('pageerror', (e) => {
    if (!isEnvironmentalPageError(e.message)) pageErrors.push(e.message);
  });
  page.on('console', (m) => {
    if (m.type() === 'error') consoleErrors.push(m.text());
  });
  return { pageErrors, consoleErrors };
}

test.describe('CV i18n (prod)', () => {
  test('/en/cv → anglais, zéro exception JS', async ({ page }) => {
    const { pageErrors, consoleErrors } = trackErrors(page);
    await page.goto('/en/cv?theme=fullstack', { waitUntil: 'load' });
    await page.waitForTimeout(3000); // laisse l'hydratation + fetchs client se faire

    const body = await page.locator('body').innerText();
    // Titre d'expérience en anglais présent, titre français absent
    expect(body).toContain('VBA Developer Intern');
    expect(body).not.toContain('Stagiaire Développeur');
    // Projet en anglais (name + grille bilingue)
    expect(body).toContain('AI-Powered Interactive CV');
    // Pas de fuite FR connue sur /en (veracite short, noms de skills français)
    expect(body).not.toContain('propulsé');
    expect(body).not.toContain('Architecture Logicielle');

    if (consoleErrors.length) console.log('[/en/cv] console errors (non-fatal):', consoleErrors);
    expect(pageErrors, 'exceptions JS:\n' + pageErrors.join('\n')).toHaveLength(0);
  });

  test('/fr/cv → français, zéro exception JS', async ({ page }) => {
    const { pageErrors, consoleErrors } = trackErrors(page);
    await page.goto('/fr/cv?theme=fullstack', { waitUntil: 'load' });
    await page.waitForTimeout(3000);

    const body = await page.locator('body').innerText();
    // Titre d'expérience en français présent, titre anglais absent
    expect(body).toContain('Stagiaire Développeur');
    expect(body).not.toContain('VBA Developer Intern');
    // Projet en français (name + grille bilingue)
    expect(body).toContain('CV interactif');

    if (consoleErrors.length) console.log('[/fr/cv] console errors (non-fatal):', consoleErrors);
    expect(pageErrors, 'exceptions JS:\n' + pageErrors.join('\n')).toHaveLength(0);
  });
});
