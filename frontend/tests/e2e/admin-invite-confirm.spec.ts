import { test, expect } from '@playwright/test';

// E2E — page d'invitation prospecteur : le mot de passe se saisit DEUX fois.
//
// POURQUOI : le lien est à usage unique et l'owner ne connaît jamais le mot de passe. Une faute de
// frappe non détectée rendait le compte inaccessible (arrivé le 15/10 au premier test réel) : il
// fallait une nouvelle invitation.
// COMMENT : la page est un composant client, hors du groupe (panel) — aucun garde serveur. L'API est
// donc simulée dans le navigateur (page.route) ; ce qui se vérifie ici est le comportement du
// formulaire : bouton bloqué et ✕ tant que les deux saisies diffèrent, et le POST n'envoie que le
// mot de passe confirmé. Le parcours réel (backend + base) reste couvert par admin-prospects.spec.ts.
// Front local uniquement (npm run dev, lancé par Playwright).

const BASE = process.env.PLAYWRIGHT_TEST_BASE_URL || 'http://localhost:3000';
const IS_LOCAL = /^https?:\/\/(localhost|127\.0\.0\.1)(:\d+)?/.test(BASE);

test.describe('Invitation prospecteur — confirmation du mot de passe', () => {
  test.describe.configure({ timeout: 120_000 });
  test.skip(!IS_LOCAL, 'API simulée dans le navigateur : front local uniquement');

  test('deux saisies différentes → ✕ et bouton bloqué ; identiques → compte activé', async ({ page }) => {
    const posts: string[] = [];
    await page.route('**/api/v1/admin/invite/tok-e2e', async (route) => {
      if (route.request().method() === 'POST') {
        posts.push(route.request().postData() ?? '');
        return route.fulfill({ status: 200, contentType: 'application/json', body: '{"ok":true}' });
      }
      return route.fulfill({ status: 200, contentType: 'application/json', body: '{"name":"Jeanne"}' });
    });

    await page.goto('/admin/invite/tok-e2e', { waitUntil: 'domcontentloaded' });
    await expect(page.getByTestId('invite-form')).toBeVisible({ timeout: 60_000 });

    const pw = 'un-mot-de-passe-sur';
    await page.getByTestId('invite-password').fill(pw);
    // Sans confirmation : bouton bloqué, aucun message (rien n'est encore « faux »).
    await expect(page.getByTestId('invite-submit')).toBeDisabled();
    await expect(page.getByTestId('invite-mismatch')).toHaveCount(0);

    // Faute de frappe dans la confirmation.
    await page.getByTestId('invite-password-confirm').fill(pw + 'x');
    await expect(page.getByTestId('invite-mismatch')).toContainText('✕');
    await expect(page.getByTestId('invite-submit')).toBeDisabled();
    // Entrée dans le champ ne doit pas contourner le bouton bloqué.
    await page.getByTestId('invite-password-confirm').press('Enter');
    await page.waitForTimeout(300);
    expect(posts).toHaveLength(0);

    // Corrigée : le message disparaît, on valide, le POST porte le mot de passe confirmé.
    await page.getByTestId('invite-password-confirm').fill(pw);
    await expect(page.getByTestId('invite-mismatch')).toHaveCount(0);
    await page.getByTestId('invite-submit').click();
    await expect(page.getByTestId('invite-done')).toBeVisible({ timeout: 15_000 });
    expect(posts).toHaveLength(1);
    expect(JSON.parse(posts[0])).toEqual({ password: pw });
  });
});
