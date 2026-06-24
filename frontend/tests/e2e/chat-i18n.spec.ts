import { test, expect, Page } from '@playwright/test';
import { trackPageErrors } from './helpers/pageErrors';

// E2E du chat — parcours réel dans un navigateur : on tape un message, on l'envoie, et on vérifie
// que la fiche projet s'ouvre dans le panel droit avec les LABELS dans la bonne langue (i18n chat.*).
//
// POURQUOI on mocke le SSE : l'endpoint /chat/stream appelle Claude (non déterministe, coûteux,
// rate-limité). On intercepte la requête et on renvoie un flux SSE fixe → le test exerce TOUT le
// câblage frontend (parsing SSE → onToolResult → ouverture d'onglet → rendu ProjectFiche) sans LLM.
// Le CONTENU de la fiche (Title/ShortDesc) vient du mock ; les LABELS viennent de l'i18n → c'est eux
// qu'on vérifie par locale. Tourne contre la prod (PLAYWRIGHT_TEST_BASE_URL).

// Fixture SSE : un texte assistant + un tool_result show_project, au format exact attendu
// (data: {json}\n\n). Données en PascalCase comme le backend Go réel.
function chatSSE(): string {
  const project = {
    Name: 'maicivy',
    Title: 'maicivy - AI-Powered Interactive CV',
    Category: 'Web Application',
    ShortDesc: 'Production-grade interactive CV platform with AI features.',
    TechStack: ['Go', 'Next.js', 'PostgreSQL'],
    KeyFeatures: ['Adaptive CV themes', 'AI cover letters'],
    Stats: [],
    SkillsTags: ['Go', 'React'],
  };
  return [
    `data: ${JSON.stringify({ type: 'text', delta: 'Here is the maicivy project.' })}`,
    '',
    `data: ${JSON.stringify({ type: 'tool_call', name: 'show_project', input: { name: 'maicivy', language: 'en' } })}`,
    '',
    `data: ${JSON.stringify({ type: 'tool_result', name: 'show_project', data: project })}`,
    '',
    `data: ${JSON.stringify({ type: 'done' })}`,
    '',
    '',
  ].join('\n');
}

async function mockChatStream(page: Page) {
  await page.route('**/api/v1/chat/stream', (route) =>
    route.fulfill({
      status: 200,
      headers: { 'Content-Type': 'text/event-stream' },
      body: chatSSE(),
    }),
  );
}

// Envoie un message dans le chat. On remplit le textarea PUIS on attend que le bouton d'envoi
// devienne actif (= l'état React `input` est commité) avant de cliquer — sinon course entre fill()
// et le handler qui lit `input` (le bouton est disabled tant que input.trim() est vide).
async function sendMessage(page: Page, text: string) {
  const ta = page.locator('textarea');
  await ta.fill(text);
  const sendBtn = page.locator('textarea ~ button').first();
  await expect(sendBtn).toBeEnabled({ timeout: 5000 });
  await sendBtn.click();
}

test.describe('Chat i18n (prod, SSE mocké)', () => {
  test('/en/chat → fiche projet rendue avec labels EN, zéro exception JS', async ({ page }) => {
    const pageErrors = trackPageErrors(page);
    await mockChatStream(page);
    await page.goto('/en/chat', { waitUntil: 'load' });

    // UI statique localisée (placeholder de l'input)
    await expect(page.getByPlaceholder(/Ask your question/i)).toBeVisible();

    await sendMessage(page, 'Show me the maicivy project');

    // La fiche projet (contenu du mock) doit apparaître dans le panel droit.
    await expect(page.getByText(/AI-Powered Interactive CV/)).toBeVisible({ timeout: 10000 });

    // Labels de la fiche en ANGLAIS (i18n), pas de fuite FR.
    await expect(page.getByText('Tech stack')).toBeVisible();
    await expect(page.getByText('Key features')).toBeVisible();
    await expect(page.getByText('Stack technique')).toHaveCount(0);
    await expect(page.getByText('Fonctionnalités clés')).toHaveCount(0);

    expect(pageErrors, 'exceptions JS:\n' + pageErrors.join('\n')).toHaveLength(0);
  });

  test('/fr/chat → fiche projet rendue avec labels FR, zéro exception JS', async ({ page }) => {
    const pageErrors = trackPageErrors(page);
    await mockChatStream(page);
    await page.goto('/fr/chat', { waitUntil: 'load' });

    await expect(page.getByPlaceholder(/Posez votre question/i)).toBeVisible();

    await sendMessage(page, 'Montre-moi le projet maicivy');

    await expect(page.getByText(/AI-Powered Interactive CV/)).toBeVisible({ timeout: 10000 });

    // Labels de la fiche en FRANÇAIS (i18n), pas de fuite EN.
    await expect(page.getByText('Stack technique')).toBeVisible();
    await expect(page.getByText('Fonctionnalités clés')).toBeVisible();
    await expect(page.getByText('Tech stack')).toHaveCount(0);

    expect(pageErrors, 'exceptions JS:\n' + pageErrors.join('\n')).toHaveLength(0);
  });
});
