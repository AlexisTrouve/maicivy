import { test, expect } from '@playwright/test';
import { trackPageErrors } from './helpers/pageErrors';

// ============================================================================
// E2E du LAZY-LOAD des cartes projet de /cv.
// ----------------------------------------------------------------------------
// Doctrine du projet : une UI qu'on n'a pas réellement pilotée n'est pas vérifiée. Ce test charge
// la page dans un vrai navigateur, compte les cartes MONTÉES, scrolle, et vérifie qu'il en arrive.
//
// CE QU'IL VERROUILLE — les deux moitiés du compromis, qui se contredisent si on en oublie une :
//   1. SEO : le HTML SERVEUR contient TOUS les projets. /cv est dans le sitemap et ces cartes sont
//      sa matière indexable ; tronquer le rendu serveur les ferait disparaître de l'index.
//   2. Perf : après hydratation, le navigateur n'en MONTE qu'un premier palier. C'est le montage
//      de 79 cartes animées (framer-motion) qui coûtait, pas le HTML.
// Un test qui ne regarderait que (2) laisserait passer une « optimisation » qui casse le référencement.
//
// Tourne contre la prod : PLAYWRIGHT_TEST_BASE_URL=https://maicivy.etheryale.com
// ============================================================================

const PALIER = 12; // doit rester aligné sur CARTES_INITIALES dans ProjectsGrid.tsx

test.describe('CV — lazy-load des projets', () => {
  test('le HTML serveur porte tous les projets (indexable)', async ({ request }) => {
    // Requête HTTP BRUTE, sans exécution de JS : c'est exactement ce que voit un crawler.
    const res = await request.get('/fr/cv');
    expect(res.status()).toBe(200);
    const html = await res.text();

    const cartes = (html.match(/data-testid="project-card"/g) || []).length;
    // Bien plus que le premier palier : le serveur ne tronque pas.
    expect(cartes).toBeGreaterThan(PALIER);
  });

  test('le navigateur ne monte que le premier palier, puis charge au scroll', async ({ page }) => {
    const pageErrors = trackPageErrors(page);
    await page.goto('/fr/cv', { waitUntil: 'load' });

    const cartes = page.locator('[data-testid="project-card"]');
    await cartes.first().waitFor({ state: 'attached', timeout: 20000 });

    // Après hydratation, l'effet d'élagage a ramené la liste au premier palier.
    await expect.poll(() => cartes.count(), { timeout: 20000 }).toBe(PALIER);

    // On descend jusqu'à la section projets : la sentinelle entre dans le viewport.
    await page.locator('#projects').scrollIntoViewIfNeeded();
    await page.mouse.wheel(0, 4000);

    // De nouvelles cartes sont montées — c'est la révélation progressive.
    await expect.poll(() => cartes.count(), { timeout: 20000 }).toBeGreaterThan(PALIER);

    // Aucune erreur JS pendant tout le parcours (un observer mal câblé se verrait ici).
    expect(pageErrors, 'exceptions JS:\n' + pageErrors.join('\n')).toHaveLength(0);
  });

  test('une carte révélée reste cliquable et ouvre sa modale', async ({ page }) => {
    // Le lazy-load ne doit pas casser l'interaction : une carte arrivée APRÈS coup se comporte
    // comme les autres (handlers bien attachés, pas seulement du HTML inerte).
    await page.goto('/fr/cv', { waitUntil: 'load' });

    const cartes = page.locator('[data-testid="project-card"]');
    await expect.poll(() => cartes.count(), { timeout: 20000 }).toBe(PALIER);

    await page.locator('#projects').scrollIntoViewIfNeeded();
    await page.mouse.wheel(0, 4000);
    await expect.poll(() => cartes.count(), { timeout: 20000 }).toBeGreaterThan(PALIER);

    // On clique une carte du SECOND palier (index au-delà du premier), pas une carte initiale.
    const carteRevelee = cartes.nth(PALIER);
    await carteRevelee.scrollIntoViewIfNeeded();
    await carteRevelee.click();

    // La modale de détail s'ouvre réellement.
    await expect(page.getByRole('dialog')).toBeVisible({ timeout: 10000 });
  });
});
