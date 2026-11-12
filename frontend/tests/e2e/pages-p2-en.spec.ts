import { test, expect, type Page } from '@playwright/test';

/**
 * Refonte passe 2 — pages de contenu en ANGLAIS : aucun libellé français ne doit rester visible.
 *
 * QUOI : charge chaque page du lot (architecture, analytics, chat, portfolio + fiche, mentions
 *   légales, confidentialité, 404 localisée) en /en, vérifie qu'un libellé anglais réécrit est là
 *   ET qu'aucun des libellés français correspondants n'apparaît.
 * POURQUOI : ces pages avaient des textes en dur dans le schéma (« Profile Builder », « GORM
 *   ORM »…) et un « 7 jours » codé dans un composant ; un oubli de clé ne casse rien à l'écran
 *   français, il ne se voit QUE dans une autre langue. On teste donc la langue où il se verrait.
 * COMMENT : contre le front qui sert la page (PLAYWRIGHT_TEST_BASE_URL), données de l'API de prod
 *   en lecture. Le chat ne poste AUCUN message (aucun appel IA) ; son flux est néanmoins intercepté
 *   par sécurité. Les contenus maiProFiles (récit, fiche technique) ne sont pas testés ici : leur
 *   langue est celle de la curation, pas de l'interface.
 */

// Libellés français de l'interface que la refonte a touchés : aucun ne doit survivre en /en.
const FRANCAIS_INTERDIT: Record<string, RegExp[]> = {
  architecture: [
    /Comment ce site est construit/,
    /En chiffres/,
    /Vue d'ensemble/,
    /Notation du CV/,
    /Construction du profil/,
    /Bord & sécurité/,
    /Recherche d'entreprise multi-sources/,
    /Le code est public/,
    /Voir sur GitHub/,
  ],
  analytics: [
    /Le site, en direct/,
    /Tableau de bord public/,
    /Visiteurs actuels/,
    /en ce moment/,
    /Thèmes du CV les plus consultés/,
    /7 jours|30 jours/,
  ],
  chat: [/Fiches/, /Posez-moi/],
  portfolio: [/Projets livrés/, /Voir le projet/],
  legal: [/Informations légales/, /Mentions légales/],
  privacy: [/Données personnelles/, /Politique de confidentialité/],
};

/** Texte visible de <main> (l'en-tête et le pied ont leurs propres tests). */
async function texteMain(page: Page): Promise<string> {
  return page.getByRole('main').innerText();
}

function sansFrancais(texte: string, page: keyof typeof FRANCAIS_INTERDIT) {
  for (const motif of FRANCAIS_INTERDIT[page]) {
    expect(texte, `libellé français resté visible en /en (${page}) : ${motif}`).not.toMatch(motif);
  }
}

test.describe('Pages passe 2 en anglais', () => {
  // Rendu serveur contre l'API de prod : le premier octet attend un aller-retour réseau.
  test.slow();

  test('architecture', async ({ page }) => {
    await page.goto('/en/architecture');
    await expect(page.getByRole('heading', { level: 1, name: 'How this site is built' })).toBeVisible();
    await expect(page.getByTestId('architecture-diagram')).toContainText('CV scoring');
    await expect(page.getByTestId('architecture-diagram')).toContainText('Profile builder');
    sansFrancais(await texteMain(page), 'architecture');
  });

  test('analytics', async ({ page }) => {
    await page.goto('/en/analytics');
    await expect(page.getByRole('heading', { level: 1, name: 'The site, live' })).toBeVisible();
    await expect(page.getByTestId('kpi-card')).toHaveCount(5, { timeout: 20000 });
    await expect(page.getByText('Current visitors')).toBeVisible();
    sansFrancais(await texteMain(page), 'analytics');
  });

  test('chat — vues mobiles et champ, sans envoyer de message', async ({ page }) => {
    await page.route('**/api/v1/chat/stream', (route) =>
      route.fulfill({ status: 200, headers: { 'Content-Type': 'text/event-stream' }, body: 'data: {"type":"done"}\n\n' }),
    );
    await page.setViewportSize({ width: 390, height: 844 });
    await page.goto('/en/chat');
    const vues = page.getByRole('tablist', { name: 'Chat view' });
    await expect(vues.getByRole('tab', { name: 'Conversation' })).toHaveAttribute('aria-selected', 'true');
    await expect(vues.getByRole('tab', { name: 'Cards' })).toBeVisible();
    await expect(page.getByRole('textbox', { name: 'Your question' })).toBeVisible();
    // Bascule de vue : les questions suggérées s'affichent, la conversation se masque.
    await vues.getByRole('tab', { name: 'Questions' }).click();
    await expect(page.locator('#chat-panneau-questions')).toBeVisible();
    await expect(page.locator('#chat-panneau-conversation')).toBeHidden();
    sansFrancais(await texteMain(page), 'chat');
  });

  test('portfolio — index puis fiche', async ({ page }) => {
    await page.goto('/en/portfolio');
    await expect(page.getByRole('main').getByText('Shipped projects', { exact: true })).toBeVisible({ timeout: 20000 });
    await expect(page.getByRole('link', { name: /View project/ }).first()).toBeVisible();
    sansFrancais(await texteMain(page), 'portfolio');

    await page.goto('/en/portfolio/melodicode');
    await expect(page.getByTestId('portfolio-back')).toHaveText(/Back to portfolio/i);
    // La vidéo de démo garde preload="metadata" (JAMAIS "none" : première image noire).
    await expect(page.getByTestId('portfolio-video-player')).toHaveAttribute('preload', 'metadata');
  });

  test('mentions légales et confidentialité', async ({ page }) => {
    await page.goto('/en/legal');
    await expect(page.getByRole('heading', { level: 1, name: 'Legal notice' })).toBeVisible();
    await expect(page.getByRole('main').getByText('Legal information', { exact: true })).toBeVisible();
    sansFrancais(await texteMain(page), 'legal');

    await page.goto('/en/privacy');
    await expect(page.getByRole('heading', { level: 1, name: 'Privacy policy' })).toBeVisible();
    await expect(page.getByRole('main').getByText('Personal data', { exact: true })).toBeVisible();
    sansFrancais(await texteMain(page), 'privacy');
  });

  test('404 localisée', async ({ page }) => {
    const reponse = await page.goto('/en/portfolio/ce_projet_nexiste_pas');
    expect(reponse?.status()).toBe(404);
    await expect(page.locator('h1')).toContainText('Page not found');
    await expect(page.getByRole('link', { name: 'See the portfolio' })).toBeVisible();
    await expect(page.locator('body')).not.toContainText('introuvable');
  });
});
