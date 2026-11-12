import { test, expect } from '@playwright/test';
import { switchToOwnerLogin } from './helpers/adminLogin';

// QUOI : la page de login du panneau s'ouvre en mode PROSPECTEUR, focus sur le champ nom/email ; le
// lien bascule en mode owner et place le focus sur le mot de passe.
// POURQUOI : en mode owner par défaut, les prospecteurs ne voyaient pas le lien « je suis
// prospecteur » et tapaient leur mot de passe dans le champ owner. Aucun backend requis : on ne
// soumet rien, seul le rendu et le focus sont vérifiés.

test.describe('Login admin — mode par défaut', () => {
  test('prospecteur par défaut, bascule owner, focus sur le premier champ', async ({ page }) => {
    await page.goto('/admin/login', { waitUntil: 'load' });
    await expect(page.getByTestId('prospect-login')).toBeVisible({ timeout: 20000 });
    await expect(page.getByTestId('prospect-login')).toBeFocused({ timeout: 20000 });

    await switchToOwnerLogin(page);
    await expect(page.getByTestId('admin-password')).toBeFocused();
  });
});
