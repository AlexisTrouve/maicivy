import { test, expect, Page } from '@playwright/test';
import { trackPageErrors } from './helpers/pageErrors';

// E2E des messages d'erreur du chat public — parcours réel : on envoie un message, le backend/réseau
// échoue de 4 façons différentes, et on vérifie que le message affiché est LOCALISÉ (next-intl) et
// distingue bien la cause. AVANT ce fix, toute erreur affichait un texte FR en dur ("❌ Erreur: ...")
// peu importe la langue du visiteur, et ne distinguait pas rate-limit / réseau / serveur / contexte
// trop long. Tourne contre la prod (PLAYWRIGHT_TEST_BASE_URL) comme les autres specs chat-*.

async function sendMessage(page: Page, text: string) {
  const ta = page.locator('textarea');
  await ta.click();
  await ta.pressSequentially(text);
  const sendBtn = page.locator('textarea ~ button').first();
  await expect(sendBtn).toBeEnabled({ timeout: 5000 });
  await sendBtn.click();
}

test.describe('Chat — messages d\'erreur localisés et distingués par cause', () => {
  test('429 avec retry_after → message de rate-limit localisé AVEC le délai (FR)', async ({ page }) => {
    const pageErrors = trackPageErrors(page);
    await page.route('**/api/v1/chat/stream', (route) =>
      route.fulfill({
        status: 429,
        contentType: 'application/json',
        body: JSON.stringify({ error: 'Cooldown actif', code: 'COOLDOWN_ACTIVE', retry_after: 42 }),
      }),
    );
    await page.goto('/fr/chat', { waitUntil: 'load' });
    await sendMessage(page, 'Bonjour');

    await expect(page.getByText('Merci de patienter 42s avant le prochain message.')).toBeVisible({ timeout: 10000 });
    expect(pageErrors, 'exceptions JS:\n' + pageErrors.join('\n')).toHaveLength(0);
  });

  test('429 SANS retry_after (corps non-JSON) → message de rate-limit générique (EN)', async ({ page }) => {
    const pageErrors = trackPageErrors(page);
    await page.route('**/api/v1/chat/stream', (route) =>
      route.fulfill({ status: 429, contentType: 'text/plain', body: 'Too Many Requests' }),
    );
    await page.goto('/en/chat', { waitUntil: 'load' });
    await sendMessage(page, 'Hello');

    await expect(page.getByText('Too many messages right now. Try again shortly.')).toBeVisible({ timeout: 10000 });
    expect(pageErrors, 'exceptions JS:\n' + pageErrors.join('\n')).toHaveLength(0);
  });

  test('coupure réseau (requête avortée) → message réseau, PAS un texte d\'erreur JS brut (EN)', async ({ page }) => {
    const pageErrors = trackPageErrors(page);
    await page.route('**/api/v1/chat/stream', (route) => route.abort('failed'));
    await page.goto('/en/chat', { waitUntil: 'load' });
    await sendMessage(page, 'Hello');

    await expect(page.getByText('Connection lost. Check your network and try again.')).toBeVisible({ timeout: 10000 });
    expect(pageErrors, 'exceptions JS:\n' + pageErrors.join('\n')).toHaveLength(0);
  });

  test('erreur serveur 500 → message serveur générique (DE)', async ({ page }) => {
    const pageErrors = trackPageErrors(page);
    await page.route('**/api/v1/chat/stream', (route) =>
      route.fulfill({ status: 500, contentType: 'application/json', body: JSON.stringify({ error: 'boom' }) }),
    );
    await page.goto('/de/chat', { waitUntil: 'load' });
    await sendMessage(page, 'Hallo');

    await expect(page.getByText('Der Dienst hatte ein Problem. Bitte versuche es gleich noch einmal.')).toBeVisible({ timeout: 10000 });
    expect(pageErrors, 'exceptions JS:\n' + pageErrors.join('\n')).toHaveLength(0);
  });

  test('event SSE error code=context_too_long → invite à démarrer une nouvelle conversation (IT)', async ({ page }) => {
    const pageErrors = trackPageErrors(page);
    const sse = [
      `data: ${JSON.stringify({ type: 'error', code: 'context_too_long' })}`,
      '',
      '',
    ].join('\n');
    await page.route('**/api/v1/chat/stream', (route) =>
      route.fulfill({ status: 200, headers: { 'Content-Type': 'text/event-stream' }, body: sse }),
    );
    await page.goto('/it/chat', { waitUntil: 'load' });
    await sendMessage(page, 'Ciao');

    await expect(
      page.getByText('Questa conversazione è diventata troppo lunga. Avvia una nuova conversazione per continuare.'),
    ).toBeVisible({ timeout: 10000 });
    expect(pageErrors, 'exceptions JS:\n' + pageErrors.join('\n')).toHaveLength(0);
  });

  test('event SSE error avec code inconnu → repli générique, pas de crash (ZH)', async ({ page }) => {
    const pageErrors = trackPageErrors(page);
    const sse = [
      `data: ${JSON.stringify({ type: 'error', code: 'something_new_the_frontend_doesnt_know' })}`,
      '',
      '',
    ].join('\n');
    await page.route('**/api/v1/chat/stream', (route) =>
      route.fulfill({ status: 200, headers: { 'Content-Type': 'text/event-stream' }, body: sse }),
    );
    await page.goto('/zh/chat', { waitUntil: 'load' });
    await sendMessage(page, '你好');

    await expect(page.getByText('抱歉，出现了问题，请重试。')).toBeVisible({ timeout: 10000 });
    expect(pageErrors, 'exceptions JS:\n' + pageErrors.join('\n')).toHaveLength(0);
  });
});
