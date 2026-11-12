import { expect, type Page } from '@playwright/test';

// ============================================================================
// Helpers E2E partagés — bascule de mode sur /admin/login.
// ----------------------------------------------------------------------------
// QUOI : la page de login s'ouvre en mode PROSPECTEUR (nom/email + mot de passe). L'owner doit
// cliquer `login-mode-switch` pour retrouver son champ unique.
// POURQUOI un helper : en dev, le clic peut précéder l'hydratation et rester sans effet. On le rejoue
// jusqu'à ce que l'état attendu soit visible (`toPass`), en un seul endroit plutôt que copié par spec.
// COMMENT : le mode owner se reconnaît à l'ABSENCE du champ `prospect-login`.
// ============================================================================

export async function switchToOwnerLogin(page: Page): Promise<void> {
  await expect(page.getByTestId('admin-login-form')).toBeVisible({ timeout: 20000 });
  await expect(async () => {
    if (await page.getByTestId('prospect-login').isVisible()) await page.getByTestId('login-mode-switch').click();
    await expect(page.getByTestId('prospect-login')).toHaveCount(0, { timeout: 1000 });
  }).toPass({ timeout: 20000 });
}

// Connexion owner complète par l'UI (mode owner, mot de passe, envoi).
export async function loginAsOwner(page: Page, password: string): Promise<void> {
  await page.goto('/admin/login', { waitUntil: 'load' });
  await switchToOwnerLogin(page);
  await page.getByTestId('admin-password').fill(password);
  await page.getByTestId('admin-submit').click();
}
