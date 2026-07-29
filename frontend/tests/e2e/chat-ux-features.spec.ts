import { test, expect, Page } from '@playwright/test';
import { trackPageErrors } from './helpers/pageErrors';

// E2E des features UX ajoutées au chat cette session — parcours réels dans un navigateur.
// POURQUOI ces tests spécifiquement : l'auto-resize du textarea dépend du layout réel (scrollHeight),
// JSDOM (jest) le simule toujours à 0 → seul un vrai navigateur peut le prouver. Le bouton stop et le
// badge de quota touchent au vrai fetch/AbortController, pas juste à la logique déjà testée en jest.

async function sendMessage(page: Page, text: string) {
  const ta = page.locator('textarea');
  await ta.click();
  await ta.pressSequentially(text);
  const sendBtn = page.locator('textarea ~ button').first();
  await expect(sendBtn).toBeEnabled({ timeout: 5000 });
  await sendBtn.click();
}

test.describe('Chat — auto-resize du textarea (layout réel, pas simulable en jest)', () => {
  test('la hauteur du textarea grandit avec un contenu multi-lignes', async ({ page }) => {
    const pageErrors = trackPageErrors(page);
    await page.goto('/fr/chat', { waitUntil: 'load' });

    const ta = page.locator('textarea');
    const initialHeight = await ta.evaluate((el) => el.getBoundingClientRect().height);

    await ta.click();
    // Shift+Enter insère un saut de ligne (pas d'envoi) — 5 lignes doivent dépasser la hauteur initiale.
    for (let i = 0; i < 5; i++) {
      await ta.pressSequentially(`ligne ${i}`);
      await page.keyboard.down('Shift');
      await page.keyboard.press('Enter');
      await page.keyboard.up('Shift');
    }

    const grownHeight = await ta.evaluate((el) => el.getBoundingClientRect().height);
    expect(grownHeight, 'le textarea doit avoir grandi avec le contenu multi-lignes').toBeGreaterThan(initialHeight);

    expect(pageErrors, 'exceptions JS:\n' + pageErrors.join('\n')).toHaveLength(0);
  });
});

test.describe('Chat — bouton stop (vrai AbortController, pas juste la logique chat-api.ts)', () => {
  test('le bouton stop apparaît pendant le streaming, et l\'arrêt ne montre PAS d\'erreur réseau', async ({ page }) => {
    const pageErrors = trackPageErrors(page);

    // Délai avant de répondre → la fenêtre "isStreaming=true, avant tout byte reçu" est observable.
    await page.route('**/api/v1/chat/stream', async (route) => {
      await new Promise((r) => setTimeout(r, 3000));
      await route.fulfill({
        status: 200,
        headers: { 'Content-Type': 'text/event-stream' },
        body: `data: ${JSON.stringify({ type: 'done' })}\n\n`,
      });
    });

    await page.goto('/fr/chat', { waitUntil: 'load' });
    await sendMessage(page, 'Bonjour');

    const stopButton = page.getByRole('button', { name: 'Arrêter la génération' });
    await expect(stopButton).toBeVisible({ timeout: 2000 });
    await stopButton.click();

    // Pas de message d'erreur réseau affiché — l'arrêt volontaire est silencieux (onDone, pas onError).
    await expect(page.getByText('Connexion perdue')).not.toBeVisible();
    // Le bouton d'envoi redevient disponible (isStreaming repasse à false).
    await expect(page.getByRole('button', { name: 'Arrêter la génération' })).toHaveCount(0);

    expect(pageErrors, 'exceptions JS:\n' + pageErrors.join('\n')).toHaveLength(0);
  });
});

test.describe('Chat — indicateur de quota (X-RateLimit-* réels, pas juste chat-api.ts en isolation)', () => {
  test('affiche le quota restant après une réponse réussie', async ({ page }) => {
    const pageErrors = trackPageErrors(page);

    // Access-Control-Expose-Headers : le backend réel l'envoie déjà (middleware.CORS, cors.go) — sans
    // ça, le navigateur masque les headers custom à fetch() sur une requête cross-origin (dev local :
    // frontend :3000, API :8080). Sans cette ligne, ce test échouait en local (pas un bug produit,
    // juste un mock qui ne reproduisait pas le vrai comportement CORS du backend).
    await page.route('**/api/v1/chat/stream', (route) =>
      route.fulfill({
        status: 200,
        headers: {
          'Content-Type': 'text/event-stream',
          'X-RateLimit-Limit': '20',
          'X-RateLimit-Remaining': '14',
          'Access-Control-Expose-Headers': 'X-RateLimit-Limit, X-RateLimit-Remaining',
        },
        body: `data: ${JSON.stringify({ type: 'done' })}\n\n`,
      }),
    );

    await page.goto('/fr/chat', { waitUntil: 'load' });
    await sendMessage(page, 'Bonjour');

    await expect(page.getByText("14/20 messages aujourd'hui")).toBeVisible({ timeout: 10000 });
    expect(pageErrors, 'exceptions JS:\n' + pageErrors.join('\n')).toHaveLength(0);
  });
});

test.describe('Chat — suggest_followups remplace les hints statiques dans le LeftPanel', () => {
  test('les relances contextuelles s\'affichent et sont cliquables (pas les hints par défaut)', async ({ page }) => {
    const pageErrors = trackPageErrors(page);
    const sse = [
      `data: ${JSON.stringify({ type: 'tool_call', name: 'suggest_followups', input: { questions: ['Quels sont tes projets en Rust ?'], language: 'fr' } })}`,
      '',
      `data: ${JSON.stringify({ type: 'tool_result', name: 'suggest_followups', data: { questions: ['Quels sont tes projets en Rust ?'] } })}`,
      '',
      `data: ${JSON.stringify({ type: 'done' })}`,
      '',
      '',
    ].join('\n');

    await page.route('**/api/v1/chat/stream', (route) =>
      route.fulfill({ status: 200, headers: { 'Content-Type': 'text/event-stream' }, body: sse }),
    );

    await page.goto('/fr/chat', { waitUntil: 'load' });
    await sendMessage(page, 'Parle-moi de tes projets');

    await expect(page.getByText('Pour continuer')).toBeVisible({ timeout: 10000 });
    const followupButton = page.getByRole('button', { name: 'Quels sont tes projets en Rust ?' });
    await expect(followupButton).toBeVisible();

    expect(pageErrors, 'exceptions JS:\n' + pageErrors.join('\n')).toHaveLength(0);
  });
});
