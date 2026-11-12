import { test, expect } from '@playwright/test';
import http from 'node:http';

// E2E — un prospecteur au plafond quotidien de vérifications voit que la checklist est PÉRIMÉE.
//
// QUOI : premier tour → vérification servie (checklist affichée) ; second tour → le backend répond
// 429 PROSPECT_VERIFY_LIMIT ; l'écran doit alors dire, avec un glyphe ✕, que la checklist affichée
// est celle du brouillon précédent. Sans cet avertissement, l'opérateur prendrait l'ancienne
// checklist pour la vérification du message qu'il s'apprête à envoyer.
//
// POURQUOI un faux backend : le panneau est gardé CÔTÉ SERVEUR (layout.tsx appelle /admin/me via
// API_URL). page.route n'intercepte que le navigateur ; il faut donc un vrai serveur HTTP pour le
// garde. Le test le démarre lui-même : il ne répond qu'à /admin/me (rôle prospecteur). Tout le
// reste (ateliers, agent, vérification) est simulé dans le navigateur, où le test l'observe.
//
// Run (le front de dev lancé par Playwright hérite de ces variables) :
//   E2E_FAKE_API_PORT=8098 API_URL=http://localhost:8098 NEXT_PUBLIC_API_URL=http://localhost:8098 \
//   npx playwright test admin-compose-quota --project=chromium

const FAKE_PORT = Number(process.env.E2E_FAKE_API_PORT || 0);
const BASE = process.env.PLAYWRIGHT_TEST_BASE_URL || 'http://localhost:3000';
const IS_LOCAL = /^https?:\/\/(localhost|127\.0\.0\.1)(:\d+)?/.test(BASE);

const sse = (message: string) =>
  [
    `data: ${JSON.stringify({ type: 'tool_call', name: 'set_draft', input: { message } })}`,
    '',
    `data: ${JSON.stringify({ type: 'done' })}`,
    '',
  ].join('\n');

const VERIFY_OK = {
  translation: 'Hello,\n\nFirst version.\n\nJeanne',
  mechanical: [{ id: 'remote', status: 'pass' }],
  semantic: [],
};

test.describe('Composer — plafond de vérifications du prospecteur', () => {
  test.describe.configure({ timeout: 120_000 });
  test.skip(!FAKE_PORT, 'E2E_FAKE_API_PORT non défini (cf. en-tête du fichier)');
  test.skip(!IS_LOCAL, 'front local uniquement : le garde serveur doit viser le faux backend');

  let server: http.Server;
  test.beforeAll(async () => {
    server = http.createServer((req, res) => {
      if (req.url?.startsWith('/api/v1/admin/me')) {
        res.writeHead(200, { 'Content-Type': 'application/json' });
        res.end(JSON.stringify({ role: 'prospect', name: 'Jeanne' }));
        return;
      }
      res.writeHead(404);
      res.end();
    });
    await new Promise<void>((ok) => server.listen(FAKE_PORT, ok));
  });
  test.afterAll(async () => {
    await new Promise<void>((ok) => server.close(() => ok()));
  });

  test('429 sur la vérification → avertissement ✕, checklist précédente conservée', async ({ page, context }) => {
    await context.addCookies([{ name: 'maicivy_prospect', value: 'e2e', url: BASE }]);

    // Ateliers : aucun à restaurer ; les sauvegardes réussissent.
    await page.route('**/api/v1/admin/compose/drafts**', async (route) => {
      const m = route.request().method();
      const body = m === 'GET' ? { drafts: [] } : { id: 'd-e2e' };
      await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify(body) });
    });

    // Agent : un brouillon différent à chaque tour.
    let tour = 0;
    await page.route('**/api/v1/chat/stream', async (route) => {
      tour += 1;
      await route.fulfill({
        status: 200,
        headers: { 'Content-Type': 'text/event-stream', 'Cache-Control': 'no-cache' },
        body: sse(tour === 1 ? 'Bonjour,\n\nPremière version.\n\nJeanne' : 'Bonjour,\n\nSeconde version.\n\nJeanne'),
      });
    });

    // Vérification : servie au premier appel, refusée pour quota ensuite.
    let verifs = 0;
    await page.route('**/api/v1/admin/compose/verify', async (route) => {
      verifs += 1;
      if (verifs === 1) {
        await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify(VERIFY_OK) });
      } else {
        await route.fulfill({
          status: 429,
          contentType: 'application/json',
          body: JSON.stringify({ error: 'daily verification limit reached', code: 'PROSPECT_VERIFY_LIMIT' }),
        });
      }
    });

    await page.goto('/admin/compose', { waitUntil: 'domcontentloaded' });
    await expect(page.getByTestId('compose-root')).toBeVisible({ timeout: 60_000 });

    // Tour 1 : vérification servie, aucun avertissement.
    await page.getByTestId('compose-announce').fill('Mission agents IA en production.');
    await page.getByTestId('compose-start').click();
    await expect(page.getByTestId('compose-draft')).toContainText('Première version', { timeout: 20_000 });
    await expect(page.getByTestId('compose-translation')).toContainText('First version', { timeout: 20_000 });
    await expect(page.getByTestId('compose-verify-limit')).toHaveCount(0);

    // Tour 2 : le backend refuse la vérification pour quota.
    await page.getByTestId('compose-chat-input').fill('Plus court.');
    await page.getByTestId('compose-chat-send').click();
    await expect(page.getByTestId('compose-draft')).toContainText('Seconde version', { timeout: 20_000 });
    const avert = page.getByTestId('compose-verify-limit');
    await expect(avert).toBeVisible({ timeout: 20_000 });
    await expect(avert).toContainText('✕');
    expect(verifs).toBe(2);
    // La checklist précédente reste affichée (jamais vidée : vide se lirait « rien à signaler »).
    await expect(page.getByTestId('compose-translation')).toContainText('First version');
  });
});
