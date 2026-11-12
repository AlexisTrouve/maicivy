import { test, expect } from '@playwright/test';
import { loginAsOwner } from './helpers/adminLogin';

// E2E du rôle prospecteur (apporteur d'affaires), de bout en bout et avec de VRAIS clics :
// invitation (owner) → choix du mot de passe → login prospecteur → nav réduite à 3 outils (CV, Composer,
// Agent) → URL owner refusée → atelier Composer créé → conversation Agent créée, visible du seul
// prospecteur → révocation par l'owner → session suivante refusée.
//
// ⚠️ LOCAL UNIQUEMENT : ce test crée de vrais comptes en base. Il refuse de tourner contre autre chose
// que localhost. Il faut un backend local (Postgres + Redis, SESSION_SECRET et ADMIN_PASSWORD posés)
// sur :8080 et le front sur :3000.
//
// Run : PLAYWRIGHT_ADMIN_PASSWORD=*** npx playwright test admin-prospects --project=chromium
//
// L'agent est simulé au niveau réseau (même raison que admin-compose.spec.ts) : ce qui se vérifie ici
// est l'autorisation et le câblage, pas le contenu produit par le modèle.

const ADMIN_PW = process.env.PLAYWRIGHT_ADMIN_PASSWORD;
const BASE = process.env.PLAYWRIGHT_TEST_BASE_URL || 'http://localhost:3000';
const API = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080';
const IS_LOCAL = /^https?:\/\/(localhost|127\.0\.0\.1)(:\d+)?/.test(BASE);

// Réponse simulée de l'onglet Agent : du texte seul (aucun tool de rédaction hors Composer).
const AGENT_CHAT_SSE = [
  `data: ${JSON.stringify({ type: 'text', delta: 'Bonjour.' })}`,
  '',
  `data: ${JSON.stringify({ type: 'done' })}`,
  '',
].join('\n');

const AGENT_SSE = [
  `data: ${JSON.stringify({ type: 'tool_call', name: 'set_draft', input: { message: 'Bonjour,\n\nTest.\n\nNoé' } })}`,
  '',
  `data: ${JSON.stringify({ type: 'done' })}`,
  '',
].join('\n');

test.describe('Rôle prospecteur (local)', () => {
  test.describe.configure({ mode: 'serial', timeout: 120_000 });
  test.skip(!ADMIN_PW, 'PLAYWRIGHT_ADMIN_PASSWORD non défini');
  test.skip(!IS_LOCAL, 'jamais contre la prod : ce test crée des comptes');

  test('invitation → login → 3 outils → refus → atelier → révocation', async ({ browser }) => {
    const name = `e2e-${Date.now()}`;
    const password = 'un-mot-de-passe-e2e';

    // --- Owner : crée l'invitation ---
    const ownerCtx = await browser.newContext();
    const owner = await ownerCtx.newPage();
    await loginAsOwner(owner, ADMIN_PW!);
    await expect(owner.getByTestId('admin-cv-tool')).toBeVisible({ timeout: 20000 });
    await owner.goto('/admin/prospects');
    await owner.getByTestId('prospect-invite-name').fill(name);
    await owner.getByTestId('prospect-invite-submit').click();
    const link = await owner.getByTestId('prospect-invite-link').inputValue();
    expect(link).toContain('/admin/invite/');

    // --- Prospecteur : choisit son mot de passe ---
    const pCtx = await browser.newContext();
    const p = await pCtx.newPage();
    await p.goto(link);
    await p.getByTestId('invite-password').fill(password);
    // Confirmation fausse → message ✕ et bouton bloqué ; corrigée → on peut valider.
    await p.getByTestId('invite-password-confirm').fill(password + 'x');
    await expect(p.getByTestId('invite-mismatch')).toContainText('✕');
    await expect(p.getByTestId('invite-submit')).toBeDisabled();
    await p.getByTestId('invite-password-confirm').fill(password);
    await expect(p.getByTestId('invite-mismatch')).toHaveCount(0);
    await p.getByTestId('invite-submit').click();
    await expect(p.getByTestId('invite-done')).toBeVisible({ timeout: 15000 });

    // Usage unique : le même lien ne sert plus.
    const p2 = await pCtx.newPage();
    await p2.goto(link);
    await expect(p2.getByTestId('invite-invalid')).toBeVisible({ timeout: 15000 });
    await p2.close();

    // --- Login prospecteur ---
    await p.getByTestId('invite-go-login').click();
    await expect(p.getByTestId('admin-login-form')).toBeVisible({ timeout: 20000 });
    // Mode prospecteur PAR DÉFAUT : le champ nom/email est là, sans clic, et a le focus.
    await expect(p.getByTestId('prospect-login')).toBeVisible({ timeout: 20000 });
    await p.getByTestId('prospect-login').fill(name);
    await p.getByTestId('admin-password').fill(password);
    await p.getByTestId('admin-submit').click();
    await expect(p.getByTestId('admin-cv-tool')).toBeVisible({ timeout: 20000 });

    // --- Nav : exactement les trois outils ---
    const navLinks = p.getByTestId('admin-nav').locator('a');
    await expect(navLinks).toHaveCount(3);
    await expect(navLinks.nth(0)).toHaveAttribute('href', '/admin');
    await expect(navLinks.nth(1)).toHaveAttribute('href', '/admin/compose');
    await expect(navLinks.nth(2)).toHaveAttribute('href', '/admin/chat');

    // --- URL owner interdite : écran ET API ---
    await p.goto('/admin/mailbox');
    await expect(p.getByTestId('admin-forbidden')).toBeVisible({ timeout: 15000 });
    const mailboxApi = await p.request.get(`${API}/api/v1/admin/mailbox`);
    expect(mailboxApi.status()).toBe(403);

    // --- Composer : un atelier est créé à son nom ---
    await p.route('**/api/v1/chat/stream', (route) =>
      route.fulfill({ status: 200, headers: { 'Content-Type': 'text/event-stream' }, body: AGENT_SSE }),
    );
    await p.route('**/api/v1/admin/compose/verify', (route) =>
      route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ translation: 't', mechanical: [], semantic: [] }) }),
    );
    await p.goto('/admin/compose');
    await expect(p.getByTestId('compose-announce')).toBeVisible({ timeout: 20000 });
    await p.getByTestId('compose-announce').fill('Recherche freelance Go pour une API.');
    const created = p.waitForResponse(
      (r) => r.url().endsWith('/admin/compose/drafts') && r.request().method() === 'POST',
      { timeout: 20000 },
    );
    await p.getByTestId('compose-start').click();
    expect((await created).status()).toBe(200);
    const list = await (await p.request.get(`${API}/api/v1/admin/compose/drafts`)).json();
    expect(list.drafts).toHaveLength(1);

    // --- Agent : une conversation créée à son nom, invisible de l'owner ---
    await p.unroute('**/api/v1/chat/stream');
    await p.route('**/api/v1/chat/stream', (route) =>
      route.fulfill({ status: 200, headers: { 'Content-Type': 'text/event-stream' }, body: AGENT_CHAT_SSE }),
    );
    await p.goto('/admin/chat');
    await expect(p.getByTestId('admin-forbidden')).toHaveCount(0);
    await expect(p.getByTestId('chat-input')).toBeVisible({ timeout: 20000 });
    await p.getByTestId('chat-input').fill('Quels projets en Go ?');
    const saved = p.waitForResponse(
      (r) => /\/admin\/chat\/conversations\/[^/]+$/.test(r.url()) && r.request().method() === 'PUT',
      { timeout: 20000 },
    );
    await p.getByTestId('chat-send').click();
    expect((await saved).status()).toBe(200);
    const mine = (await (await p.request.get(`${API}/api/v1/admin/chat/conversations`)).json()).conversations;
    expect(mine).toHaveLength(1);
    const convId = mine[0].id;
    const ownerConvs = (await (await owner.request.get(`${API}/api/v1/admin/chat/conversations`)).json()).conversations;
    expect(ownerConvs.map((c: { id: string }) => c.id)).not.toContain(convId);
    expect((await owner.request.get(`${API}/api/v1/admin/chat/conversations/${convId}`)).status()).toBe(404);

    // --- Owner révoque ---
    await owner.reload();
    await owner.getByTestId(`prospect-toggle-${name}`).click();
    await expect(owner.getByTestId(`prospect-row-${name}`)).toContainText('✕', { timeout: 15000 });

    // --- La session suivante du prospecteur est refusée ---
    expect((await p.request.get(`${API}/api/v1/admin/compose/drafts`)).status()).toBe(401);
    await p.goto('/admin/compose');
    await expect(p).toHaveURL(/\/admin\/login$/, { timeout: 15000 });

    await ownerCtx.close();
    await pCtx.close();
  });
});
