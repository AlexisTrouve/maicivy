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
    // Projet en anglais : titre de la carte maicivy (première carte de la grille). L'ancienne ancre,
    // « AI-Powered Interactive CV », venait en réalité du PIED DE PAGE, remplacé par la refonte.
    expect(body).toContain('AI agent console');
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
    // Projet en français : titre de la carte maicivy. (« CV interactif » venait du pied de page.)
    expect(body).toContain("console d'agents IA");
    expect(body).not.toContain('AI agent console');

    if (consoleErrors.length) console.log('[/fr/cv] console errors (non-fatal):', consoleErrors);
    expect(pageErrors, 'exceptions JS:\n' + pageErrors.join('\n')).toHaveLength(0);
  });

  // zh est natif côté maiProFiles (?lang=zh) : le CONTENU (pas seulement l'UI next-intl) doit être
  // en chinois. Avant : UI chinoise + fiches anglaises.
  test('/zh/cv → contenu chinois, zéro exception JS', async ({ page }) => {
    const { pageErrors, consoleErrors } = trackErrors(page);
    await page.goto('/zh/cv?theme=fullstack', { waitUntil: 'load' });
    await page.waitForTimeout(3000);

    const body = await page.locator('body').innerText();
    // Titre d'expérience chinois (mesuré sur maiProFiles ?lang=zh), ni anglais ni français.
    // (Le poste « 全栈自由开发者 » a disparu des données ; ancre sur un intitulé toujours présent.)
    expect(body).toContain('VBA开发实习生');
    expect(body).not.toContain('VBA Developer Intern');
    expect(body).not.toContain('Freelance Full-Stack Developer');
    expect(body).not.toContain('Stagiaire Développeur');
    // Le projet maicivy est servi avec son nom chinois.
    expect(body).toContain('自由职业者网站');

    if (consoleErrors.length) console.log('[/zh/cv] console errors (non-fatal):', consoleErrors);
    expect(pageErrors, 'exceptions JS:\n' + pageErrors.join('\n')).toHaveLength(0);
  });
});
