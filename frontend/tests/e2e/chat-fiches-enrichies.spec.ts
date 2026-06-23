import { test, expect, Page } from '@playwright/test';

// E2E des fiches ENRICHIES (skills avec niveau/années, expériences avec technos/accroche).
// POURQUOI ce test existe : on a changé le shape des données poussées dans le panel droit —
// SkillCategory.Skills est passé de string[] à SkillDetail{Name,Level,Years}, et ExperienceItem a
// gagné Technologies/Catchphrase. Un `tsc` vert ne prouve PAS que ça rend : le vrai risque est un
// crash React (ou un "[object Object]") si le composant lit mal le nouveau shape. On rend donc la
// fiche pour de vrai dans un navigateur (SSE mocké) et on vérifie le contenu + zéro exception JS.
//
// SSE mocké (comme chat-i18n.spec.ts) : l'endpoint /chat/stream appelle Claude (non déterministe,
// coûteux). On intercepte et on renvoie un flux fixe au format exact (PascalCase comme le backend Go).
// Tourne contre la prod (PLAYWRIGHT_TEST_BASE_URL).

// Construit un flux SSE : texte + tool_call + tool_result + done.
function sse(toolName: string, data: unknown): string {
  return [
    `data: ${JSON.stringify({ type: 'text', delta: 'Voici les informations demandées.' })}`,
    '',
    `data: ${JSON.stringify({ type: 'tool_call', name: toolName, input: { language: 'fr' } })}`,
    '',
    `data: ${JSON.stringify({ type: 'tool_result', name: toolName, data })}`,
    '',
    `data: ${JSON.stringify({ type: 'done' })}`,
    '',
    '',
  ].join('\n');
}

function trackPageErrors(page: Page): string[] {
  const errors: string[] = [];
  page.on('pageerror', (e) => errors.push(e.message));
  return errors;
}

async function mockChatStream(page: Page, body: string) {
  await page.route('**/api/v1/chat/stream', (route) =>
    route.fulfill({ status: 200, headers: { 'Content-Type': 'text/event-stream' }, body }),
  );
}

// Remplit le textarea PUIS attend que le bouton d'envoi soit actif avant de cliquer (cf. chat-i18n).
async function sendMessage(page: Page, text: string) {
  const ta = page.locator('textarea');
  await ta.fill(text);
  const sendBtn = page.locator('textarea ~ button').first();
  await expect(sendBtn).toBeEnabled({ timeout: 5000 });
  await sendBtn.click();
}

test.describe('Fiches enrichies (prod, SSE mocké)', () => {
  test('show_skills → skills avec niveau rendus, noms corrects (pas de [object Object])', async ({ page }) => {
    const pageErrors = trackPageErrors(page);

    // Shape RÉEL renvoyé par le backend après l'enrichissement : SkillDetail{Name, Level, Years}.
    const skills = [
      {
        Name: 'Languages',
        Skills: [
          { Name: 'Go', Level: 'advanced', Years: 5 },
          { Name: 'Rust', Level: 'intermediate', Years: 2 },
        ],
      },
      {
        Name: 'AI',
        Skills: [{ Name: 'AI / LLM Integration', Level: 'expert', Years: 4 }],
      },
    ];
    await mockChatStream(page, sse('show_skills', skills));
    await page.goto('/fr/chat', { waitUntil: 'load' });

    await sendMessage(page, 'Montre-moi ses compétences');

    // Les NOMS de skills doivent s'afficher correctement (le bug redouté = objet rendu en texte).
    await expect(page.getByText('AI / LLM Integration')).toBeVisible({ timeout: 10000 });
    await expect(page.getByText('Rust', { exact: true })).toBeVisible();
    // Les catégories réelles maiProFiles.
    await expect(page.getByText('Languages', { exact: true })).toBeVisible();
    // Régression critique : aucun "[object Object]" (= shape lu de travers).
    await expect(page.getByText('[object Object]')).toHaveCount(0);

    expect(pageErrors, 'exceptions JS:\n' + pageErrors.join('\n')).toHaveLength(0);
  });

  test('show_experience → poste avec technos (chips) + accroche rendus', async ({ page }) => {
    const pageErrors = trackPageErrors(page);

    // Shape RÉEL après enrichissement : ExperienceItem avec Technologies + Catchphrase.
    const experience = {
      Bio: 'bio courte',
      BioFull: 'Développeur full-stack indépendant.',
      Headline: 'Développeur Full-Stack',
      TJM: 'Sur demande',
      Dispo: 'Disponible',
      ExperienceYears: 9,
      Domains: ['Backend', 'IA'],
      Experience: [
        {
          Role: 'Stagiaire Développeur VBA',
          Company: 'Cogesco',
          Period: '2015-06 → 2015-10',
          Summary: 'Création d’outils RH.',
          Technologies: ['VBA', 'Access', 'SQL'],
          Catchphrase: 'Outils de gestion RH et automatisation',
          Category: 'fullstack',
        },
      ],
    };
    await mockChatStream(page, sse('show_experience', experience));
    await page.goto('/fr/chat', { waitUntil: 'load' });

    await sendMessage(page, 'Détaille son parcours');

    // Poste + employeur.
    await expect(page.getByText('Cogesco', { exact: true })).toBeVisible({ timeout: 10000 });
    await expect(page.getByText('Stagiaire Développeur VBA')).toBeVisible();
    // Accroche (nouveau champ).
    await expect(page.getByText('Outils de gestion RH et automatisation')).toBeVisible();
    // Chips technos (nouveau champ) — preuve que Technologies est rendu.
    await expect(page.getByText('VBA', { exact: true })).toBeVisible();
    await expect(page.getByText('Access', { exact: true })).toBeVisible();

    expect(pageErrors, 'exceptions JS:\n' + pageErrors.join('\n')).toHaveLength(0);
  });
});
