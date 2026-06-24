import { test, expect } from '@playwright/test';
import { trackPageErrors } from './helpers/pageErrors';

// E2E du badge "Repos chauds en ce moment" du DevPortrait (/cv) après le fix du compteur de commits.
// CONTEXTE : l'ancien backend faisait `r.Commits += c.Commits` à chaque fetch de 30 min → empilement
// non borné (Melodicode affiché à 64459 commits, blender-mcp à 9300 pour 0 jour d'activité). Le fix
// recalcule par repo depuis un set de SHA (union dédupliquée + rolloff) et affiche les commits sur 30j.
// Ce test verrouille les DEUX faces : l'API (source) et le badge rendu (UI réellement observée).
// Cible la prod : PLAYWRIGHT_TEST_BASE_URL=https://maicivy.etheryale.com

// Borne de cohérence : aucun repo ne peut afficher autant de commits sur 6 mois. L'ancien bug crachait
// 64459 / 25392 / 18808… → un plafond à 5000 attrape la régression sans être flaky sur un mois actif.
const SANE_MAX = 5000;

test.describe('Git Stats — compteur de commits par repo (prod)', () => {
  test('API /cv/gitstats : commits par-repo bornés + commits30d présent et ≤ 6 mois', async ({ request }) => {
    const res = await request.get('/api/v1/cv/gitstats');
    expect(res.ok()).toBeTruthy();
    const data = await res.json();

    expect(Array.isArray(data.repos)).toBeTruthy();
    expect(data.repos.length).toBeGreaterThan(0);

    for (const r of data.repos) {
      // commits (6 mois) : recalculé, plus l'accumulateur cassé → doit rester sous le plafond de cohérence.
      expect(typeof r.commits).toBe('number');
      expect(r.commits, `repo ${r.name} : commits 6 mois=${r.commits} (régression de l'accumulateur ?)`)
        .toBeLessThan(SANE_MAX);
      // commits30d : nouveau champ servi → présent, ≥ 0, et JAMAIS supérieur au total 6 mois (sous-ensemble).
      expect(typeof r.commits30d, `repo ${r.name} : commits30d manquant dans l'API`).toBe('number');
      expect(r.commits30d).toBeGreaterThanOrEqual(0);
      expect(r.commits30d, `repo ${r.name} : commits30d (${r.commits30d}) > commits 6 mois (${r.commits})`)
        .toBeLessThanOrEqual(r.commits);
    }
  });

  test('UI /cv : le badge "Repos chauds" affiche des commits 30j sains (· 30j)', async ({ page }) => {
    const errors = trackPageErrors(page);

    await page.goto('/fr/cv', { waitUntil: 'load' });

    // La carte "Repos chauds en ce moment" doit être rendue (DevPortrait monté).
    await expect(page.getByText('Repos chauds en ce moment')).toBeVisible({ timeout: 20000 });

    // Les badges par-repo portent la fenêtre explicite "· 30j" (preuve que le label 30j a shippé).
    const badges = page.getByText(/commits · 30j/);
    await expect(badges.first()).toBeVisible({ timeout: 10000 });

    // Aucun badge ne doit afficher un nombre aberrant (l'ancien bug : "64459 commits").
    const texts = await badges.allInnerTexts();
    expect(texts.length).toBeGreaterThan(0);
    for (const t of texts) {
      // Capturer le COMPTE (avant "commits"), pas le "30" de "· 30j". Gère le groupement de milliers
      // fr (espace insécable) : on isole la tranche [chiffres + séparateurs] devant "commits", puis on
      // ne garde que les chiffres.
      const m = t.match(/([\d\s., ]+?)\s*commits/);
      expect(m, `badge "${t}" : format inattendu`).not.toBeNull();
      const n = parseInt(m![1].replace(/[^\d]/g, ''), 10);
      expect(Number.isNaN(n)).toBeFalsy();
      expect(n, `badge "${t}" : nombre aberrant (régression de l'accumulateur ?)`).toBeLessThan(SANE_MAX);
    }

    // Le rendu ne doit lever aucune exception JS.
    expect(errors, 'exceptions JS:\n' + errors.join('\n')).toHaveLength(0);
  });
});
