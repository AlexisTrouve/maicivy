import { test, expect } from '@playwright/test';

// E2E du tracker de clics qui alimente la heatmap. Doctrine : une UI/feature sans test qui clique pour
// de vrai = non vérifiée. On prouve la chaîne réelle : un clic sur un élément interactif déclenche un
// POST /api/v1/analytics/event (button_click|link_click) avec des coordonnées, et le backend le persiste
// (201 → visitor_id présent). C'est ce qui rend la heatmap non vide. 100% réel, aucune donnée fabriquée.
// Tourne contre la prod : PLAYWRIGHT_TEST_BASE_URL=https://maicivy.etheryale.com

test.describe('Heatmap tracking (prod)', () => {
  test('un clic sur un élément interactif POST un event de heatmap (201)', async ({ page }) => {
    // 1. Première visite → le backend pose le cookie de session signé (maicivy_session).
    await page.goto('/fr', { waitUntil: 'load' });
    // 2. Deuxième navigation → le cookie signé revient → le middleware tracking attribue un visitor_id
    //    (sans lui, TrackEvent renvoie 400 ; c'est le cas du tout premier hit, ignoré côté tracker).
    await page.goto('/fr/cv?theme=fullstack', { waitUntil: 'load' });

    // On capte le POST déclenché par le clic (le tracker écoute en capture sur document).
    const eventResponse = page.waitForResponse(
      (r) => r.url().includes('/api/v1/analytics/event') && r.request().method() === 'POST',
      { timeout: 15000 }
    );

    // 3. Clic réel sur un élément interactif qui NE navigue PAS (pastille skill → ouvre une fiche).
    const badge = page.locator('[data-testid^="skill-badge-"]').first();
    await badge.waitFor({ state: 'visible', timeout: 15000 });
    await badge.click();

    const resp = await eventResponse;
    // visitor_id présent (cookie signé) → event persisté → 201 Created.
    expect(resp.status(), 'le POST event doit être persisté (201)').toBe(201);

    // Le payload a la forme attendue par la heatmap (type + coordonnées numériques + page).
    const payload = JSON.parse(resp.request().postData() || '{}');
    expect(['button_click', 'link_click']).toContain(payload.event_type);
    expect(typeof payload.event_data?.x).toBe('number');
    expect(typeof payload.event_data?.y).toBe('number');
    expect(payload.page_url).toContain('/cv');
  });
});
