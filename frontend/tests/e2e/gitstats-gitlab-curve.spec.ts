import { test, expect } from '@playwright/test';
import { trackPageErrors } from './helpers/pageErrors';

// E2E de la part « GitLab » du graphe de cadence (page Git Stats, maquette A). Doctrine : une UI sans
// test qui la voit RÉELLEMENT rendue = non vérifiée. On prouve ici que le graphe superpose bien DEUX
// séries : « Tous les dépôts » (Gitea + GitLab mergés) et, en cuivre, la part du projet client GitLab.
// La légende GitLab et les barres cuivre n'apparaissent QUE si le backend renvoie `gitlabDaily` non
// vide (cf. `hasGitlab` dans GitStatsPanel).
// Tourne contre la prod : PLAYWRIGHT_TEST_BASE_URL=https://maicivy.etheryale.com

// Teinte cuivre de la série GitLab (DATA_TINTS.copper) : seule la part GitLab l'emploie.
const COPPER = '#d69b81';

test.describe('Git Stats — part GitLab du graphe de cadence (prod)', () => {
  test('le graphe de cadence superpose « Tous les dépôts » et la part GitLab', async ({ page }) => {
    const pageErrors = trackPageErrors(page);

    // ── 1. Charger la page (contexte Playwright neuf → sessionStorage vide → fetch backend frais) ────
    await page.goto('/fr/gitstats', { waitUntil: 'load' });
    const cadence = page.locator('section', { has: page.getByRole('heading', { name: 'Cadence' }) });
    await expect(cadence).toBeVisible({ timeout: 20000 });

    // ── 2. La légende de la cadence porte les DEUX séries, et des barres cuivre sont dessinées ───────
    const legendItems = cadence.getByTestId('gitstats-legend-item');
    await expect(legendItems).toHaveCount(2, { timeout: 10000 });
    await expect(legendItems.filter({ hasText: 'Tous les dépôts' })).toBeVisible();
    await expect(legendItems.filter({ hasText: 'GitLab' })).toBeVisible();
    // Barres de la part GitLab dans le graphe visible (quotidien sur PC, hebdomadaire sur téléphone).
    const copperBars = await cadence.locator(`[role="img"]:visible rect[fill="${COPPER}"]`).count();
    expect(copperBars, 'barres cuivre (part GitLab) dans le graphe de cadence').toBeGreaterThan(0);

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
