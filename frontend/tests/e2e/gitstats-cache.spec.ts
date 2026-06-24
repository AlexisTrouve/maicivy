import { test, expect } from '@playwright/test';
import { trackPageErrors } from './helpers/pageErrors';

// E2E du cache sessionStorage de la page Git Stats. Doctrine : une UI sans test qui clique/navigue
// pour de vrai = non vérifiée. On prouve ici le « stale-while-revalidate » CÔTÉ CLIENT :
//   1. 1ère visite → les stats se chargent et sont mises en cache (sessionStorage).
//   2. On RALENTIT volontairement l'API à 4s.
//   3. On revient sur la page : les KPI doivent réapparaître en < 2.5s — donc PEINTS DEPUIS LE CACHE,
//      pas depuis le réseau (qui est bloqué 4s). Sans cache, la page attendrait les 4s → test rouge.
// Tourne contre la prod : PLAYWRIGHT_TEST_BASE_URL=https://maicivy.etheryale.com

const CACHE_KEY = 'maicivy:gitstats:v1';
// .text-3xl.font-bold = les 4 valeurs KPI (le h1 "Git Stats" est en text-4xl → pas matché).
const KPI = '.text-3xl.font-bold';

test.describe('Git Stats — cache client (prod)', () => {
  test('revisite : peinture instantanée depuis sessionStorage (API ralentie)', async ({ page }) => {
    const pageErrors = trackPageErrors(page);

    // ── 1. Première visite (cache froid : contexte Playwright neuf = sessionStorage vide) ───────────
    await page.goto('/fr/gitstats', { waitUntil: 'load' });
    const firstKpi = page.locator(KPI).first();
    await firstKpi.waitFor({ state: 'visible', timeout: 20000 });
    await expect(firstKpi).toHaveText(/\d/, { timeout: 5000 }); // un vrai chiffre, pas un placeholder

    // Le cache de session a bien été écrit.
    const cached = await page.evaluate((k) => sessionStorage.getItem(k), CACHE_KEY);
    expect(cached, 'le cache sessionStorage doit être peuplé après la 1ère visite').toBeTruthy();

    // ── 2. On ralentit l'API gitstats à ~4s (la revalidation de fond sera lente) ────────────────────
    await page.route('**/cv/gitstats**', async (route) => {
      await new Promise((r) => setTimeout(r, 4000));
      await route.continue();
    });

    // ── 3. On quitte la page puis on y revient ──────────────────────────────────────────────────────
    await page.goto('/fr', { waitUntil: 'load' });

    const start = Date.now();
    await page.goto('/fr/gitstats', { waitUntil: 'commit' }); // ne pas attendre 'load' (réseau lent)
    // Les KPI doivent apparaître AVANT que l'API (4s) ne réponde → preuve qu'ils viennent du cache.
    await expect(page.locator(KPI).first()).toBeVisible({ timeout: 2500 });
    await expect(page.locator(KPI).first()).toHaveText(/\d/);
    const elapsed = Date.now() - start;
    expect(elapsed, `peinture en ${elapsed}ms — doit être < 4s (cache, pas réseau)`).toBeLessThan(3500);

    expect(pageErrors, 'exceptions JS:\n' + pageErrors.join('\n')).toHaveLength(0);
  });
});
