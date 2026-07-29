import { test, expect } from '@playwright/test';
import { trackPageErrors } from './helpers/pageErrors';

// E2E de la 2e courbe « GitLab » du chart commits/jour (page Git Stats). Doctrine : une UI sans test qui
// la voit RÉELLEMENT rendue = non vérifiée. On prouve ici que le chart superpose bien DEUX séries :
//   - « Total » (Gitea + GitLab mergés)  - « GitLab » (contribution du repo client, seule)
// La légende n'apparaît QUE si le backend renvoie `gitlabDaily` non vide ET que la 2e <Area> est montée
// (cf. `hasGitlab` dans GitStatsPanel). Il n'existe qu'UNE seule <Legend> sur la page (le chart lignes
// n'en a pas) → les items de légende sont donc ceux du chart commits, source directe de la preuve.
// Tourne contre la prod : PLAYWRIGHT_TEST_BASE_URL=https://maicivy.etheryale.com

test.describe('Git Stats — 2e courbe GitLab (prod)', () => {
  test('le chart commits superpose les séries « Total » et « GitLab »', async ({ page }) => {
    const pageErrors = trackPageErrors(page);

    // ── 1. Charger la page (contexte Playwright neuf → sessionStorage vide → fetch backend frais) ────
    await page.goto('/fr/gitstats', { waitUntil: 'load' });

    // Le chart commits doit être rendu (SVG recharts monté sous le titre « Commits par jour »).
    await expect(page.getByRole('heading', { name: 'Commits par jour' })).toBeVisible({ timeout: 20000 });

    // ── 2. La légende (unique sur la page) porte les DEUX séries ─────────────────────────────────────
    // Preuve que la 2e <Area> « GitLab » est bien montée : sans elle, `hasGitlab` serait faux → aucune
    // légende, et le test échoue.
    const legendItems = page.locator('.recharts-legend-item-text');
    await expect(legendItems).toHaveCount(2, { timeout: 10000 });
    await expect(page.locator('.recharts-legend-item-text', { hasText: 'Total' })).toBeVisible();
    await expect(page.locator('.recharts-legend-item-text', { hasText: 'GitLab' })).toBeVisible();

    // ── 3. La série GitLab porte de VRAIES données récentes (pas une courbe à plat) ──────────────────
    // Preuve que les commits GitLab (branches de feature incluses) remontent jusqu'au front : on relit
    // l'API comme le fait la page et on vérifie qu'au moins un jour des 14 derniers a des commits GitLab.
    // Verrouille le fix « 0 commit GitLab » (branche par défaut seule → travail non mergé invisible).
    const recentGitlab = await page.evaluate(async () => {
      const base = (window as any).process?.env?.NEXT_PUBLIC_API_URL || '';
      const res = await fetch(`${base}/api/v1/cv/gitstats`, { credentials: 'include' });
      const d = await res.json();
      const cutoff = new Date(Date.now() - 14 * 864e5).toISOString().slice(0, 10);
      return (d.gitlabDaily || [])
        .filter((x: any) => x.date >= cutoff)
        .reduce((s: number, x: any) => s + (x.commits || 0), 0);
    });
    expect(recentGitlab, 'commits GitLab sur les 14 derniers jours (doit être > 0)').toBeGreaterThan(0);

    // ── 4. Aucune exception JS applicative ──────────────────────────────────────────────────────────
    expect(pageErrors, 'exceptions JS:\n' + pageErrors.join('\n')).toHaveLength(0);
  });
});
