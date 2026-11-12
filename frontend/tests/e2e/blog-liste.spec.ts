import { test, expect, type Page } from '@playwright/test';
import { trackPageErrors } from './helpers/pageErrors';

// E2E de la liste du blog — pagination, filtre par thème, miniatures légères.
//
// Régressions du 2026-10-11 :
//  - la liste ne chargeait que les 50 premiers articles (88 sur 138 invisibles, compteurs compris) ;
//  - elle chargeait 16,4 MB de PNG pleine définition (LCP 16,6 s).
// Robuste aux données : les nombres attendus sont LUS sur la page (compteur, boutons de thème),
// jamais codés en dur. Tourne contre la prod : PLAYWRIGHT_TEST_BASE_URL=https://maicivy.etheryale.com

const PAR_PAGE = 12;
// La page est rendue côté serveur : le contenu est là au DOMContentLoaded. Attendre `load` (toutes
// les images, à travers l'edge Cloudflare) coûtait ~20 s par test depuis l'Asie, sans rien prouver.
test.describe.configure({ timeout: 60_000 });
const cartes = (page: Page) => page.locator('[data-testid="blog-grid"] [data-testid="blog-card"]');
const total = async (page: Page) =>
  Number(await page.locator('[data-testid="blog-count"]').getAttribute('data-count'));
const premierLien = async (page: Page) =>
  (await cartes(page).first().locator('xpath=ancestor::a').getAttribute('href')) ?? '';

test.describe('Blog — liste paginée', () => {
  test('tous les articles sont atteignables, page par page', async ({ page }) => {
    const erreurs = trackPageErrors(page);
    await page.goto('/fr/blog', { waitUntil: 'domcontentloaded' });

    const n = await total(page);
    expect(n).toBeGreaterThan(PAR_PAGE); // sinon il n'y a rien à paginer
    await expect(cartes(page)).toHaveCount(PAR_PAGE);
    const pages = Math.ceil(n / PAR_PAGE);
    const lienPage1 = await premierLien(page);

    // Clic réel sur « Suivant » → page 2, d'autres articles.
    await page.locator('[data-testid="blog-page-next"]').click();
    await expect(page).toHaveURL(/[?&]page=2\b/);
    await expect(cartes(page)).toHaveCount(PAR_PAGE);
    expect(await premierLien(page)).not.toBe(lienPage1);

    // Clic réel sur le numéro de la dernière page → elle porte exactement le reste.
    await page.locator(`[data-testid="blog-page-${pages}"]`).click();
    await expect(page).toHaveURL(new RegExp(`[?&]page=${pages}\\b`));
    await expect(cartes(page)).toHaveCount(n - PAR_PAGE * (pages - 1));
    await expect(page.locator('[data-testid="blog-page-next"]')).toHaveCount(0);

    expect(erreurs, 'exceptions JS:\n' + erreurs.join('\n')).toHaveLength(0);
  });

  // QUOI : une page au-delà de la dernière répond un VRAI 404 (statut HTTP), en plus de la page 404.
  // POURQUOI : avant le 2026-10-12, `[locale]/loading.tsx` faisait diffuser la page en flux ;
  // l'en-tête 200 était parti quand notFound() survenait (« Soft 404 » vu par Google).
  // COMMENT : loading.tsx racine retiré → le statut est fixé après le rendu serveur complet.
  test('une page au-delà de la dernière rend la page 404, non indexable', async ({ page }) => {
    const response = await page.goto('/fr/blog?page=999', { waitUntil: 'domcontentloaded' });
    expect(response?.status()).toBe(404);
    // Au moins une : Next en émet parfois deux (tête initiale + frontière not-found).
    expect(await page.locator('meta[name="robots"][content="noindex"]').count()).toBeGreaterThan(0);
    await expect(page.locator('[data-testid="blog-grid"]')).toHaveCount(0);
  });

  test('filtre par thème : compteur sur tout le catalogue, « Tous » réinitialise', async ({ page }) => {
    await page.goto('/fr/blog', { waitUntil: 'domcontentloaded' });
    const n = await total(page);

    const filtre = page.locator('[data-testid="tag-filter"]');
    const theme = filtre.locator('[data-testid^="tag-"]:not([data-testid="tag-all"])').first();
    test.skip((await theme.count()) === 0, 'aucun thème présent sur le blog');
    const attendu = parseInt((await theme.locator('span').last().innerText()).trim(), 10);
    expect(attendu).toBeGreaterThan(0);
    expect(attendu).toBeLessThanOrEqual(n);

    await theme.click();
    await expect(page).toHaveURL(/[?&]theme=/);
    await expect(page.locator('[data-testid="blog-count"]')).toHaveAttribute('data-count', String(attendu));
    await expect(cartes(page)).toHaveCount(Math.min(PAR_PAGE, attendu));
    await expect(page.locator('[data-testid="active-filter"]')).toBeVisible();

    await page.locator('[data-testid="tag-all"]').click();
    await expect(page.locator('[data-testid="blog-count"]')).toHaveAttribute('data-count', String(n));
    await expect(page.locator('[data-testid="active-filter"]')).toHaveCount(0);
  });

  test('les miniatures sont des variantes optimisées, pas les PNG d’origine', async ({ page }) => {
    await page.goto('/fr/blog', { waitUntil: 'domcontentloaded' });
    const img = cartes(page).first().locator('img');
    await expect(img).toBeVisible();
    const src = (await img.evaluate((el: HTMLImageElement) => el.currentSrc || el.src)) ?? '';
    expect(src).toContain('/_next/image');

    const res = await page.request.get(src, { headers: { Accept: 'image/avif,image/webp,*/*' } });
    expect(res.status()).toBe(200);
    expect(res.headers()['content-type']).toMatch(/image\/(avif|webp)/);
    // Une couverture d'origine pèse ~2 MB ; la variante d'une carte doit rester bien en dessous.
    expect((await res.body()).length).toBeLessThan(200_000);
  });

  test('libellés dans la langue de la page', async ({ page }) => {
    await page.goto('/en/blog', { waitUntil: 'domcontentloaded' });
    await expect(page.locator('[data-testid="tag-filter"] h2')).toHaveText(/Filter by theme/i);
    await expect(page.locator('[data-testid="blog-count"]')).toHaveText(/articles?/);
  });
});
