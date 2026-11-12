import { test, expect, Page } from '@playwright/test';
import { trackPageErrors } from './helpers/pageErrors';

// ============================================================================
// E2E : toutes les sections de /gitstats sont PEINTES sans que le visiteur ait à défiler.
// ----------------------------------------------------------------------------
// Panne mesurée en prod (2026-11-09) : sous le premier écran, « Lignes de code par jour » et
// « Dépôts (159) » — 5 690 px à 1440 de large — restaient à opacity 0 tant qu'un
// IntersectionObserver (framer-motion `whileInView`) ne s'était pas déclenché. La place était
// réservée mais vide : une capture pleine page, un aperçu, un défilement trop rapide ou un
// navigateur qui ne livre pas l'observateur montraient un grand trou sombre jusqu'au pied de page.
// Playwright `toBeVisible` ne voit pas ce défaut (opacity 0 compte pour « visible ») : on lit donc
// l'opacité EFFECTIVE, produit de celles de l'élément et de tous ses ancêtres.
//
// Tourne contre un front local pointé sur l'API de prod (lecture seule) : la vraie liste de dépôts
// est longue, c'est elle qui produisait le trou.
// ============================================================================

// Opacité effective de chaque élément ciblé, sans aucun défilement préalable.
async function effectiveOpacities(page: Page, selector: string): Promise<number[]> {
  return page.evaluate((sel) => {
    return Array.from(document.querySelectorAll(sel)).map((el) => {
      let opacity = 1;
      for (let node: Element | null = el; node; node = node.parentElement) {
        opacity *= Number(getComputedStyle(node).opacity);
      }
      return opacity;
    });
  }, selector);
}

test.describe('Git Stats — sections peintes sans défilement', () => {
  test('/fr/gitstats : graphes, dépôts et lignes par dépôt sont opaques dès le chargement', async ({ page }) => {
    const pageErrors = trackPageErrors(page);
    await page.goto('/fr/gitstats', { waitUntil: 'load' });
    await expect(page.getByTestId('gitstats-kpi-value').first()).toBeVisible({ timeout: 20000 });
    await expect(page.getByTestId('gitstats-repo').first()).toBeAttached({ timeout: 10000 });
    // Le LOC par dépôt vient de /cv/loc, chargé à part : on attend qu'il soit arrivé.
    await expect(page.getByTestId('gitstats-repo').getByText(/lignes$/).first()).toBeAttached({ timeout: 15000 });
    // Au-delà de toute animation d'entrée qui aurait pu rester : 0,6 s + délais.
    await page.waitForTimeout(1500);

    // Aucune section (graphes, dépôts) ne doit attendre un défilement pour exister à l'écran.
    // Maquette A : cadence, lignes écrites, langages, dépôts.
    const sections = await effectiveOpacities(page, 'section');
    expect(sections.length, 'sections attendues : 2 graphes + langages + dépôts').toBe(4);
    expect(sections, 'opacité effective des sections').toEqual(sections.map(() => 1));

    // Ni aucune rangée de dépôt — y compris la dernière de la table.
    const repos = await effectiveOpacities(page, '[data-testid="gitstats-repo"]');
    expect(repos.length).toBeGreaterThan(0);
    const hidden = repos.filter((o) => o < 1).length;
    expect(hidden, `${hidden} cartes de dépôt sur ${repos.length} non peintes sans défilement`).toBe(0);

    expect(pageErrors, 'exceptions JS:\n' + pageErrors.join('\n')).toHaveLength(0);
  });
});
