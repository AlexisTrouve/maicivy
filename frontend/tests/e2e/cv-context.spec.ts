import { test, expect, type Page } from '@playwright/test';

// ============================================================================
// /cv — CV adapté au besoin déjà exprimé (chat public, dossier vivant).
// ----------------------------------------------------------------------------
// QUOI : deux visiteurs, de vrais clics.
//   1. Sans contexte : la page montre le CV standard, sans bandeau ni question.
//   2. Avec contexte (simulé AU RÉSEAU par route() — aucun appel IA) : le bandeau apparaît
//      « en préparation », le CV adapté remplace le standard quand il est prêt, la bascule revient
//      au standard et repart sur l'adapté, le PDF adapté se télécharge.
// COMMENT : seuls les endpoints du contexte sont interceptés ; le CV standard vient du vrai rendu
// serveur. Le CV adapté simulé porte une expérience au nom introuvable ailleurs (« Contexte Simulé
// SARL ») : sa présence prouve que c'est bien LUI qui est affiché, pas le standard.
// ============================================================================

const ADAPTED_COMPANY = 'Contexte Simulé SARL';
const NEED = 'Industrialiser une plateforme d’agents IA en Go, équipe de six personnes.';

const adaptedCV = {
  theme: { id: 'dynamic', name: 'Lead Go — agents IA' },
  experiences: [
    {
      id: 'exp-ctx',
      title: 'Lead développeur Go',
      company: ADAPTED_COMPANY,
      description: 'Expérience mise en avant pour le besoin exprimé.',
      startDate: '2024-01-01T00:00:00Z',
      endDate: null,
      technologies: ['Go', 'Redis'],
      tags: [],
      category: 'backend',
      featured: true,
    },
  ],
  skills: [],
  projects: [],
  generated_at: '2026-11-09T00:00:00Z',
};

/** Simule le contexte : `pending` à la première lecture, puis `ready`. Compte les lectures. */
async function mockContext(page: Page) {
  const state = { reads: 0, pdfRequests: 0 };
  await page.route('**/api/v1/cv/context/pdf**', async (route) => {
    state.pdfRequests += 1;
    await route.fulfill({
      status: 200,
      contentType: 'application/pdf',
      headers: { 'Content-Disposition': 'attachment; filename="cv_adapte_fr.pdf"' },
      body: '%PDF-1.4\n% maicivy e2e\n',
    });
  });
  await page.route(/\/api\/v1\/cv\/context\?/, async (route) => {
    state.reads += 1;
    const base = { need: NEED, source: 'chat', lang: 'fr' };
    const body = state.reads === 1 ? { status: 'pending', ...base } : { status: 'ready', ...base, cv: adaptedCV };
    await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify(body) });
  });
  return state;
}

test.describe('CV adapté au besoin du visiteur', () => {
  test('sans contexte : CV standard, aucune question', async ({ page }) => {
    await page.route(/\/api\/v1\/cv\/context\?/, (route) =>
      route.fulfill({ status: 200, contentType: 'application/json', body: '{"status":"none"}' })
    );
    const contextRead = page.waitForRequest(/\/api\/v1\/cv\/context\?lang=fr/);
    await page.goto('/fr/cv');
    await contextRead;

    await expect(page.locator('#experiences').first()).toBeVisible();
    await expect(page.getByTestId('cv-standard')).toBeVisible();
    await expect(page.getByTestId('cv-context-banner')).toHaveCount(0);
    await expect(page.getByTestId('cv-adapted')).toHaveCount(0);
    // Aucune question : pas de champ de saisie tant que le visiteur ne le demande pas.
    await expect(page.locator('textarea')).toHaveCount(0);
    await expect(page.getByTestId('cv-offer-toggle')).toHaveText("L'adapter à une offre");
  });

  test('avec contexte : bandeau, préparation, bascule standard/adapté, téléchargement', async ({ page }) => {
    const state = await mockContext(page);
    await page.goto('/fr/cv');

    // 1. Bandeau en préparation — le standard reste affiché.
    const banner = page.getByTestId('cv-context-banner');
    await expect(banner).toBeVisible();
    await expect(banner.getByRole('heading', { name: "Adapté à ce que vous m'avez dit" })).toBeVisible();
    await expect(banner).toContainText("D'après notre conversation");
    await expect(page.getByTestId('cv-context-need')).toHaveText(NEED);
    await expect(page.getByTestId('cv-context-pending')).toBeVisible();
    await expect(page.getByTestId('cv-standard')).toBeVisible();

    // 2. Prêt (relecture du polling, 4 s) : l'adapté remplace le standard.
    await expect(page.getByTestId('cv-adapted')).toBeVisible({ timeout: 15_000 });
    await expect(page.getByTestId('cv-adapted')).toContainText(ADAPTED_COMPANY);
    await expect(page.getByTestId('cv-standard')).toBeHidden();
    await expect(page.getByTestId('cv-view-adapted')).toHaveAttribute('aria-pressed', 'true');
    expect(state.reads).toBeGreaterThanOrEqual(2);

    // 3. Retour au standard, puis de nouveau l'adapté — vrais clics.
    await page.getByTestId('cv-view-standard').click();
    await expect(page.getByTestId('cv-view-standard')).toHaveAttribute('aria-pressed', 'true');
    await expect(page.getByTestId('cv-adapted')).toHaveCount(0);
    await expect(page.getByTestId('cv-standard')).toBeVisible();
    await expect(page.getByText(ADAPTED_COMPANY)).toHaveCount(0);

    await page.getByTestId('cv-view-adapted').click();
    await expect(page.getByTestId('cv-adapted')).toContainText(ADAPTED_COMPANY);

    // 4. Le PDF adapté se télécharge (fichier réellement reçu par le navigateur).
    const downloadPromise = page.waitForEvent('download');
    await page.getByTestId('cv-context-download').click();
    const download = await downloadPromise;
    expect(download.suggestedFilename()).toBe('cv_adapte_fr.pdf');
    expect(state.pdfRequests).toBe(1);
    // Borné au bandeau : Next.js monte son propre `role=alert` (annonceur de route) sur toute page.
    await expect(banner.getByRole('alert')).toHaveCount(0);
  });

  test("l'adapter à une offre : le lien ouvre le champ, l'envoi réutilise /cv/generate", async ({ page }) => {
    await page.route(/\/api\/v1\/cv\/context\?/, (route) =>
      route.fulfill({ status: 200, contentType: 'application/json', body: '{"status":"none"}' })
    );
    let sent: Record<string, unknown> | null = null;
    await page.route('**/api/v1/cv/generate', async (route) => {
      sent = route.request().postDataJSON();
      await route.fulfill({
        status: 200,
        contentType: 'application/pdf',
        headers: { 'Content-Disposition': 'attachment; filename="CV_offre.pdf"' },
        body: '%PDF-1.4\n',
      });
    });
    await page.goto('/fr/cv');

    await page.getByTestId('cv-offer-toggle').click();
    await page.getByTestId('cv-offer-input').fill('Développeur Go senior, plateforme d’agents IA, Paris');
    const downloadPromise = page.waitForEvent('download');
    await page.getByTestId('cv-offer-submit').click();
    expect((await downloadPromise).suggestedFilename()).toBe('CV_offre.pdf');
    expect(sent).toEqual({ offer: 'Développeur Go senior, plateforme d’agents IA, Paris', lang: 'fr', format: 'pdf' });
  });
});
