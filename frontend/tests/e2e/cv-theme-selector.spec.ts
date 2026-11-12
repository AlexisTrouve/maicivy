import { test, expect } from '@playwright/test';
import { trackPageErrors } from './helpers/pageErrors';

// E2E du SÉLECTEUR DE PROFIL (thème) du CV. Doctrine : une UI cliquable sans test qui clique pour
// de vrai = non vérifiée. Ce test ouvre le dropdown, choisit un autre profil, et vérifie que le CV
// CHANGE réellement (URL ?theme= + libellé du profil sélectionné).
//
// RÉGRESSION CIBLÉE : sous next-intl v4, CVThemeSelector faisait `router.push(`/cv?${params}`)` —
// un href STRING avec query passé au router localisé, qui ne navigue plus en v4. Résultat : le
// dropdown se peuple mais sélectionner un profil ne change rien. Ce test reproduit ça.
//
// Tourne contre la prod : PLAYWRIGHT_TEST_BASE_URL=https://maicivy.etheryale.com

test.describe('CV profile selector (prod)', () => {
  test('sélectionner un profil change le CV (URL + libellé)', async ({ page }) => {
    const pageErrors = trackPageErrors(page);
    await page.goto('/fr/cv?theme=fullstack', { waitUntil: 'load' });

    // Le sélecteur de profil est le seul Radix Select (role=combobox) de la page.
    // (LanguageSwitcher / BackgroundSwitcher sont des listbox custom, pas des combobox.)
    const selector = page.getByRole('combobox');
    await selector.waitFor({ state: 'visible', timeout: 15000 });

    // État initial : profil fullstack, affiché sous son nom traduit.
    await expect(page).toHaveURL(/theme=fullstack/);
    await expect(selector).toHaveText('Full-stack');

    // Ouvre le dropdown et clique réellement sur « Backend » (theme=backend). Le nom accessible de
    // l'option contient aussi sa description : on vise le début.
    await selector.click();
    await page.getByRole('option', { name: /^Backend/ }).click();

    // ATTENDU (échoue tant que la navigation est cassée → test ROUGE) :
    //  1. l'URL reflète le profil choisi
    await expect(page).toHaveURL(/[?&]theme=backend/, { timeout: 8000 });
    //  2. le sélecteur affiche désormais le profil choisi (re-render après navigation)
    await expect(selector).toHaveText('Backend', { timeout: 8000 });

    expect(pageErrors, 'exceptions JS:\n' + pageErrors.join('\n')).toHaveLength(0);
  });

  // RÉGRESSION du libellé VIDE (passe 1) : chaque option porte nom + description ; Radix recopiait
  // les deux blocs dans le déclencheur, et le `line-clamp-1` du SelectTrigger n'en laissait AUCUNE
  // ligne visible. Le texte était dans le DOM : seul un test de RENDU (hauteur réelle) l'attrape.
  test('sans ?theme : profil agents IA par défaut, libellé réellement visible', async ({ page }) => {
    const pageErrors = trackPageErrors(page);
    await page.goto('/fr/cv', { waitUntil: 'load' });

    const selector = page.getByRole('combobox');
    await selector.waitFor({ state: 'visible', timeout: 15000 });
    await expect(selector).toHaveText('Agents IA — Technomancer');

    // La valeur affichée occupe au moins une ligne de texte (> 12 px de haut) dans le déclencheur.
    const valueHeight = await selector.evaluate((el) => {
      const span = el.querySelector('span');
      return span ? span.getBoundingClientRect().height : 0;
    });
    expect(valueHeight, 'le libellé du profil doit être rendu, pas rogné à 0 px').toBeGreaterThan(12);

    // Le profil par défaut est aussi en tête du menu.
    await selector.click();
    await expect(page.getByRole('option').first()).toContainText('Agents IA — Technomancer');

    expect(pageErrors, 'exceptions JS:\n' + pageErrors.join('\n')).toHaveLength(0);
  });
});
