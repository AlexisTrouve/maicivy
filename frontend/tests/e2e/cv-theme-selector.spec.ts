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

    // État initial : profil fullstack.
    await expect(page).toHaveURL(/theme=fullstack/);

    // Ouvre le dropdown et clique réellement sur "Backend Developer" (theme=backend).
    await selector.click();
    await page.getByRole('option', { name: /Backend Developer/i }).click();

    // ATTENDU (échoue tant que la navigation est cassée → test ROUGE) :
    //  1. l'URL reflète le profil choisi
    await expect(page).toHaveURL(/[?&]theme=backend/, { timeout: 8000 });
    //  2. le sélecteur affiche désormais le profil choisi (re-render après navigation)
    await expect(selector).toContainText(/Backend Developer/i, { timeout: 8000 });

    expect(pageErrors, 'exceptions JS:\n' + pageErrors.join('\n')).toHaveLength(0);
  });
});
