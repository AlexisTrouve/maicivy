import { test, expect, type Page } from '@playwright/test';

import { trackPageErrors } from './helpers/pageErrors';

/**
 * Chrome du site — refonte « montagne » (2026-11) : l'accueil EST la montagne, une seule chrome
 * (en-tête + pied de page) mène partout, le site est sombre en permanence.
 *
 * Remplace theme-dark-default.spec.ts (le thème sombre n'est plus un DÉFAUT qu'un choix stocké
 * pouvait renverser : il n'y a plus de choix) et background-switcher.spec.ts (le fond animé et son
 * sélecteur ont été retirés au profit du ciel de la montagne).
 *
 * Tout est fait par de VRAIS clics, en grand écran (1920×1080, nav à plat) et sur téléphone
 * (390×844, menu téléphone). L'URL d'arrivée ET un repère de la page d'arrivée sont vérifiés : une
 * URL juste sur une page cassée ne passerait pas.
 *
 * Données : la montagne est lue sur le backend (MONTAGNE_API_URL, comme montagne.spec.ts).
 */

/** Destination → repère qui prouve que la BONNE page est rendue. */
const DESTINATIONS = [
  { id: 'dossierVivant', url: /\/fr\/dossier-vivant$/, landmark: (p: Page) => p.getByTestId('dv-root') },
  { id: 'portfolio', url: /\/fr\/portfolio$/, landmark: (p: Page) => p.getByRole('heading', { level: 1 }) },
  { id: 'cv', url: /\/fr\/cv(\?.*)?$/, landmark: (p: Page) => p.getByRole('heading', { level: 1 }).first() },
  { id: 'blog', url: /\/fr\/blog$/, landmark: (p: Page) => p.getByRole('heading', { level: 1 }) },
  { id: 'chat', url: /\/fr\/chat$/, landmark: (p: Page) => p.locator('textarea, input[type="text"]').first() },
] as const;

const DESKTOP = { width: 1920, height: 1080 };
const PHONE = { width: 390, height: 844 };

async function gotoHome(page: Page) {
  await page.goto('/fr');
  await expect(page.getByTestId('montagne-voice')).toBeVisible({ timeout: 60_000 });
}

test.describe('Accueil = la montagne', () => {
  test.slow();

  test('/fr rend la montagne sous l’en-tête du site, avec le pied de page', async ({ page }) => {
    const errors = trackPageErrors(page);
    await gotoHome(page);
    await expect(page.getByTestId('montagne-voice-title')).toBeVisible();
    await expect(page.getByTestId('montagne-summit-card').first()).toBeVisible({ timeout: 30_000 });
    await expect(page.getByTestId('site-header')).toBeVisible();
    await expect(page.getByTestId('site-footer')).toHaveCount(1);
    // L'entrée « Accueil » est celle où l'on est.
    await expect(page.getByTestId('site-nav-home')).toHaveAttribute('aria-current', 'page');
    // JSON-LD Person au nouveau positionnement, dans le HTML serveur.
    const ld = JSON.parse((await page.locator('script[type="application/ld+json"]').first().textContent())!);
    expect(ld.jobTitle).toBe('Agent Orchestrator — Technomancer');
    expect(errors, 'exceptions JS:\n' + errors.join('\n')).toHaveLength(0);
  });

  test('/fr/montagne → 308 permanent vers /fr', async ({ request, page }) => {
    const res = await request.get('/fr/montagne', { maxRedirects: 0 });
    expect(res.status()).toBe(308);
    expect(new URL(res.headers()['location'], 'http://x').pathname).toBe('/fr');
    // Et un navigateur qui suit l'ancien lien arrive sur la montagne.
    await page.goto('/en/montagne');
    await expect(page).toHaveURL(/\/en$/);
    await expect(page.getByTestId('montagne-voice')).toBeVisible({ timeout: 60_000 });
  });
});

test.describe('Navigation depuis l’accueil — grand écran (1920)', () => {
  test.use({ viewport: DESKTOP });
  test.slow();

  for (const d of DESTINATIONS) {
    test(`clic sur « ${d.id} » dans l’en-tête`, async ({ page }) => {
      await gotoHome(page);
      // Grand écran : la nav est à plat, pas de bouton de menu.
      await expect(page.getByTestId('site-menu-toggle')).toBeHidden();
      await page.getByTestId(`site-nav-${d.id}`).click();
      await expect(page).toHaveURL(d.url, { timeout: 30_000 });
      await expect(d.landmark(page)).toBeVisible({ timeout: 30_000 });
    });
  }
});

test.describe('Navigation depuis l’accueil — téléphone (390×844, menu)', () => {
  test.use({ viewport: PHONE, hasTouch: true, isMobile: true });
  test.slow();

  test('le menu téléphone : fermé par défaut, cibles ≥ 44 px, Échap le referme', async ({ page }) => {
    await gotoHome(page);
    const toggle = page.getByTestId('site-menu-toggle');
    await expect(toggle).toBeVisible();
    // La nav à plat n'est pas affichée sur téléphone.
    await expect(page.getByTestId('site-nav-portfolio')).toBeHidden();
    await expect(toggle).toHaveAttribute('aria-expanded', 'false');
    const tb = (await toggle.boundingBox())!;
    expect(Math.min(tb.width, tb.height)).toBeGreaterThanOrEqual(44);

    await toggle.tap();
    await expect(toggle).toHaveAttribute('aria-expanded', 'true');
    const menu = page.getByTestId('site-menu');
    await expect(menu).toBeVisible();
    for (const id of ['home', 'dossierVivant', 'portfolio', 'cv', 'blog', 'chat']) {
      const link = page.getByTestId(`site-menu-${id}`);
      await expect(link).toBeVisible();
      expect((await link.boundingBox())!.height, `${id} : cible tactile`).toBeGreaterThanOrEqual(44);
    }
    // Contact et langue restent dans la barre, cibles tactiles aussi.
    for (const id of ['site-header-contact', 'site-header-language']) {
      const box = (await page.getByTestId(id).boundingBox())!;
      expect(box.height, id).toBeGreaterThanOrEqual(44);
    }
    // Aucun défilement horizontal, menu ouvert.
    expect(await page.evaluate(() => document.documentElement.scrollWidth)).toBeLessThanOrEqual(PHONE.width);

    await page.keyboard.press('Escape');
    await expect(menu).toBeHidden();
    await expect(toggle).toBeFocused();
  });

  for (const d of DESTINATIONS) {
    test(`menu → « ${d.id} »`, async ({ page }) => {
      await gotoHome(page);
      await page.getByTestId('site-menu-toggle').tap();
      await page.getByTestId(`site-menu-${d.id}`).tap();
      await expect(page).toHaveURL(d.url, { timeout: 30_000 });
      await expect(d.landmark(page)).toBeVisible({ timeout: 30_000 });
      // Le menu s'est refermé derrière la navigation (sauf sur le dossier vivant, sans en-tête).
      await expect(page.getByTestId('site-menu')).toHaveCount(0);
    });
  }
});

test.describe('Le reste du site', () => {
  test.slow();

  test('le dossier vivant garde sa barre, et son pied de page mène au reste du site', async ({ page }) => {
    await page.goto('/fr/dossier-vivant');
    await expect(page.getByTestId('dv-root')).toBeVisible({ timeout: 60_000 });
    // Sa propre barre, pas l'en-tête du site.
    await expect(page.getByTestId('site-header')).toHaveCount(0);
    const footer = page.getByTestId('site-footer');
    await footer.scrollIntoViewIfNeeded();
    await page.getByTestId('site-footer-gitstats').click();
    await expect(page).toHaveURL(/\/fr\/gitstats$/);
    await expect(page.getByTestId('site-header')).toBeVisible();
  });

  test('le pied de page : coulisses, légal, contact — vrais clics', async ({ page }) => {
    await page.goto('/fr/blog');
    await page.getByTestId('site-footer-legalNotice').click();
    await expect(page).toHaveURL(/\/fr\/legal$/, { timeout: 30_000 });
    await expect(page.getByRole('heading', { level: 1 })).toBeVisible();
    await page.getByTestId('site-footer-architecture').click();
    await expect(page).toHaveURL(/\/fr\/architecture$/, { timeout: 30_000 });
    await expect(page.getByTestId('site-footer-email')).toHaveAttribute('href', /^mailto:.+@.+/);
  });

  test('le sélecteur de langue du pied s’ouvre vers le haut et change la langue de la page', async ({ page }) => {
    await page.goto('/fr/portfolio');
    const trigger = page.getByTestId('site-footer-language').locator('button[aria-haspopup]');
    await trigger.scrollIntoViewIfNeeded();
    await trigger.click();
    const list = page.getByTestId('site-footer-language').getByRole('listbox');
    await expect(list).toBeVisible();
    expect((await list.boundingBox())!.y).toBeLessThan((await trigger.boundingBox())!.y);
    await list.getByRole('option', { name: 'English' }).click();
    await expect(page).toHaveURL(/\/en\/portfolio$/);
    await expect(page.getByTestId('site-nav-home')).toHaveText('Home');
  });
});

test.describe('Sombre uniquement', () => {
  test('aucune bascule clair / sombre ni sélecteur de fond, sur l’accueil comme sur une page interne', async ({ page }) => {
    for (const path of ['/fr', '/fr/cv']) {
      await page.goto(path);
      await expect(page.getByTestId('site-header')).toBeVisible({ timeout: 60_000 });
      await expect(page.getByRole('button', { name: /thème|theme/i })).toHaveCount(0);
      await expect(page.getByTestId('background-switcher')).toHaveCount(0);
      await expect(page.locator('[data-testid="background-host"]')).toHaveCount(0);
    }
  });

  test('un ancien choix « light » stocké ne rallume pas le clair', async ({ page }) => {
    // Un visiteur qui avait basculé en clair avant la refonte garde ce localStorage.
    await page.addInitScript(() => localStorage.setItem('theme', 'light'));
    await page.goto('/fr/cv');
    await expect(page.locator('html')).toHaveClass(/\bdark\b/);
    // Le fond rendu est la nuit de la montagne (#04060b), pas un fond clair.
    const bg = await page.evaluate(() => getComputedStyle(document.body).backgroundColor);
    const [r, g, b] = bg.match(/\d+/g)!.map(Number);
    expect(Math.max(r, g, b), `fond du body : ${bg}`).toBeLessThan(20);
  });
});

/*
 * Une URL inconnue SOUS une locale doit tomber sur le 404 localisé, dans la chrome du site — pas sur
 * le 404 racine brut et bilingue. Rendu possible par app/[locale]/[...rest]/page.tsx.
 */
test.describe('404 localisé', () => {
  test('/en/nexiste-pas → statut 404, page en anglais avec en-tête et pied du site', async ({ page }) => {
    const res = await page.goto('/en/nexiste-pas-du-tout');
    expect(res?.status()).toBe(404);
    await expect(page.getByRole('heading', { level: 1 })).toContainText('Page not found');
    await expect(page.locator('header').first()).toBeVisible();
    await expect(page.locator('footer').first()).toBeVisible();
    // Le 404 racine disait « Page non trouvee / Page not found » : il ne doit plus apparaître.
    await expect(page.getByText('Page non trouvee')).toHaveCount(0);
  });
});
