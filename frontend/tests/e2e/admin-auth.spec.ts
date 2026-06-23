import { test, expect } from '@playwright/test';

// E2E de l'auth du panneau /admin : clique RÉELLEMENT le form de login, vérifie l'entrée dans le
// panneau, puis le logout. Gated sur PLAYWRIGHT_ADMIN_PASSWORD pour ne pas hardcoder le secret
// (skip proprement si absent). Doctrine UI : pas de clic réel = non vérifié.
//
// Run : PLAYWRIGHT_TEST_BASE_URL=https://maicivy.etheryale.com PLAYWRIGHT_ADMIN_PASSWORD=*** \
//       npx playwright test admin-auth --config=playwright.prod.config.ts

const ADMIN_PW = process.env.PLAYWRIGHT_ADMIN_PASSWORD;

test.describe('Admin auth (prod)', () => {
  // Serial : tests d'auth stateful (cookie) + évite de brûler le quota 100 req/min en parallèle.
  test.describe.configure({ mode: 'serial' });
  test.skip(!ADMIN_PW, 'PLAYWRIGHT_ADMIN_PASSWORD non défini — auth admin sautée');

  test('login → panneau → logout', async ({ page }) => {
    await page.goto('/admin/login', { waitUntil: 'load' });

    await page.getByTestId('admin-password').fill(ADMIN_PW!);
    await page.getByTestId('admin-submit').click();

    // Entré dans le panneau gardé
    await expect(page.getByTestId('admin-home')).toBeVisible({ timeout: 12000 });
    await expect(page).toHaveURL(/\/admin\/?$/);

    // Déconnexion → retour au login
    await page.getByTestId('admin-logout').click();
    await expect(page.getByTestId('admin-login-form')).toBeVisible({ timeout: 12000 });
  });

  test('mauvais mot de passe → reste sur le login avec erreur', async ({ page }) => {
    await page.goto('/admin/login', { waitUntil: 'load' });
    await page.getByTestId('admin-password').fill('definitely-wrong-password');
    await page.getByTestId('admin-submit').click();
    await expect(page.getByTestId('admin-error')).toBeVisible({ timeout: 12000 });
  });

  test('/admin sans auth → redirige vers le login', async ({ page }) => {
    await page.goto('/admin', { waitUntil: 'load' });
    await expect(page).toHaveURL(/\/admin\/login$/);
  });
});
