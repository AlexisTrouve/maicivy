import { test, expect } from '@playwright/test';
import { loginAsOwner } from './helpers/adminLogin';

// E2E de l'écran /admin/compose : clique RÉELLEMENT dessus, avec un agent simulé au niveau réseau.
//
// POURQUOI un agent simulé plutôt que le vrai : le vrai tour coûte un appel Opus, dure plusieurs
// secondes et son contenu varie — on ne peut rien y assérer de stable. Ce qui doit être vérifié ici
// est le CÂBLAGE, et lui est déterministe : le draft affiché vient-il bien de l'input du tool_call,
// la vérification est-elle demandée après le tour, le livrable est-il réellement non éditable. Le
// contenu produit par l'agent, lui, est couvert côté backend (contrôles mécaniques, contexte vierge).
//
// POURQUOI ce test existe malgré ça : doctrine du projet — une UI sans clic réel n'est pas vérifiée,
// donc elle n'existe pas. Lire le code du composant ne prouve rien.
//
// Run : PLAYWRIGHT_TEST_BASE_URL=https://maicivy.etheryale.com PLAYWRIGHT_ADMIN_PASSWORD=*** \
//       npx playwright test admin-compose --config=playwright.prod.config.ts

const ADMIN_PW = process.env.PLAYWRIGHT_ADMIN_PASSWORD;

// Draft volontairement PROPRE (aucun markdown) : c'est ce qu'un bon tour d'agent produit.
const DRAFT = 'Bonjour,\n\nJe travaille exclusivement en remote.\n\nAlexis';

// Flux SSE d'un tour d'agent qui appelle set_rate puis set_draft, au format exact du backend
// (cf. ChatEvent : type/name/input/delta).
const AGENT_SSE = [
  // Le raisonnement arrive AVANT les tools et le texte — c'est l'ordre réel du modèle, et ce que l'UI
  // doit refléter à l'écran.
  `data: ${JSON.stringify({ type: 'thinking', delta: 'Leur vrai problème est la mise en production.' })}`,
  '',
  `data: ${JSON.stringify({ type: 'tool_call', name: 'set_rate', input: { amount: 300, rationale: 'stack agentique' } })}`,
  '',
  `data: ${JSON.stringify({ type: 'tool_call', name: 'set_draft', input: { message: DRAFT } })}`,
  '',
  `data: ${JSON.stringify({ type: 'text', delta: 'Voilà une première version.' })}`,
  '',
  `data: ${JSON.stringify({ type: 'done' })}`,
  '',
].join('\n');

const VERIFY_BODY = {
  translation: 'Hello,\n\nI work exclusively remote.\n\nAlexis',
  mechanical: [
    { id: 'remote', status: 'pass' },
    { id: 'no_markdown', status: 'pass' },
    { id: 'signature', status: 'pass' },
    { id: 'rate_coherence', status: 'na' },
    { id: 'length', status: 'pass', detail: '62/1500' },
  ],
  semantic: [{ id: 'terraform', label: 'Mentions Terraform experience', status: 'fail' }],
};

test.describe('Admin compose', () => {
  // 90 s : le parcours complet enchaîne connexion, deux rechargements et l'aller-retour d'envoi,
  // à travers l'edge Cloudflare (~20 s de `load` par page depuis l'Asie). 30 s ne suffisait plus.
  test.describe.configure({ mode: 'serial', timeout: 90_000 });
  test.skip(!ADMIN_PW, 'PLAYWRIGHT_ADMIN_PASSWORD non défini — écran compose sauté');

  test('annonce → draft → traduction → checklist, et le livrable reste non éditable', async ({ page }) => {
    // Interception AVANT navigation : le premier tour part dès le clic sur « rédiger ».
    let composeFlagSent = false;
    await page.route('**/api/v1/chat/stream', async (route) => {
      const body = route.request().postDataJSON() as { compose?: boolean };
      composeFlagSent = body?.compose === true;
      await route.fulfill({
        status: 200,
        headers: { 'Content-Type': 'text/event-stream', 'Cache-Control': 'no-cache' },
        body: AGENT_SSE,
      });
    });

    // Conteneur plutôt qu'une variable nue : TypeScript rétrécit une `let` assignée uniquement dans
    // une closure de route à `never`, ce qui casse la lecture des champs plus bas.
    const captured: { verify?: { draft?: string; retained_rate?: number } } = {};
    await page.route('**/api/v1/admin/compose/verify', async (route) => {
      captured.verify = route.request().postDataJSON();
      await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify(VERIFY_BODY) });
    });

    // Login réel — l'écran est derrière le garde owner du panneau.
    await loginAsOwner(page, ADMIN_PW!);
    await expect(page.getByTestId('admin-cv-tool')).toBeVisible({ timeout: 12000 });

    // Navigation par la nav réelle : vérifie au passage que l'entrée existe et pointe au bon endroit.
    // On attend la réponse de restauration AVANT de décider quoi que ce soit : sans ça le test court
    // contre elle et l'état de l'écran dépend d'une course réseau.
    const restored = page.waitForResponse(
      (r) => r.url().includes('/admin/compose/drafts') && r.request().method() === 'GET',
      { timeout: 20000 },
    );
    await page.goto('/admin/compose', { waitUntil: 'load' });
    await expect(page.getByTestId('compose-root')).toBeVisible({ timeout: 12000 });
    await restored;

    // La base peut contenir un atelier d'une exécution précédente, que l'écran restaure au montage.
    // On repart d'un état connu — c'est aussi ce que fait l'opérateur pour traiter une nouvelle
    // annonce, donc le chemin est celui de la vraie vie et pas un artifice de test.
    // Attendre que l'écran SE STABILISE, pas seulement que la requête réponde : la restauration
    // s'applique au rendu suivant, et décider avant ferait disparaître l'élément sous le clic.
    await page.waitForFunction(
      () =>
        document.querySelector('[data-testid="compose-announce"]') !== null ||
        document.querySelector('[data-testid="compose-draft"]') !== null,
      undefined,
      { timeout: 15000 },
    );
    const newDraftButton = page.getByTestId('compose-new');
    if (await newDraftButton.isVisible().catch(() => false)) {
      await newDraftButton.click();
    }
    await expect(page.getByTestId('compose-announce')).toBeVisible({ timeout: 12000 });

    // Saisie de l'annonce + lancement.
    await page.getByTestId('compose-announce').fill('Recherche freelance IA, Terraform apprécié.');
    await page.getByTestId('compose-start').click();

    // 1. Le draft vient de l'input du tool_call, affiché sans attendre la fin du tour.
    await expect(page.getByTestId('compose-draft')).toContainText('exclusivement en remote', { timeout: 15000 });
    expect(composeFlagSent).toBe(true);

    // 2. Le tool call est visible : le message ne doit pas changer sans explication. Et son contenu
    //    est consultable une fois déplié — savoir qu'il a appelé set_draft n'apprend rien.
    const toolLine = page.getByTestId('compose-toolcall-set_draft');
    await expect(toolLine).toBeVisible();
    await expect(toolLine).toContainText('exclusivement en remote');

    // 2b. Le raisonnement est exposé, pas seulement le résultat.
    await expect(page.getByText('Leur vrai problème est la mise en production.')).toBeAttached();

    // 2c. Le fil de conversation DÉFILE dans son cadre au lieu de pousser la page : sans ça, quelques
    //     tours suffisent à transformer l'écran en colonne interminable.
    const debordePage = await page.evaluate(
      () => document.documentElement.scrollHeight > document.documentElement.clientHeight + 4,
    );
    expect(debordePage, 'la page déborde verticalement : le fil n’est pas contenu').toBe(false);

    // 3. Le TJM fixé par l'agent est affiché et remonté à la vérification.
    await expect(page.getByTestId('compose-rate')).toContainText('300');
    await expect.poll(() => captured.verify?.retained_rate, { timeout: 10000 }).toBe(300);
    expect(String(captured.verify?.draft)).toContain('exclusivement en remote');

    // 4. La traduction opérateur et la checklist arrivent après le tour.
    await expect(page.getByTestId('compose-translation')).toContainText('I work exclusively remote');
    await expect(page.getByTestId('check-remote')).toHaveAttribute('data-status', 'pass');
    await expect(page.getByTestId('check-semantic-terraform')).toHaveAttribute('data-status', 'fail');

    // 5. Persistance : on RECHARGE la page pour de vrai. Les endpoints /drafts ne sont PAS interceptés
    //    — l'aller-retour passe par le vrai backend et la vraie base. C'est le seul moyen de prouver
    //    qu'un opérateur qui rafraîchit ne perd pas plusieurs tours de travail.
    await page.reload({ waitUntil: 'load' });
    await expect(page.getByTestId('compose-draft')).toContainText('exclusivement en remote', {
      timeout: 15000,
    });
    await expect(page.getByTestId('compose-translation')).toContainText('I work exclusively remote');
    await expect(page.getByTestId('compose-rate')).toContainText('300');
    await expect(page.getByTestId('check-remote')).toHaveAttribute('data-status', 'pass');

    // 6. LA garantie centrale : ni le draft ni la traduction ne sont saisissables. Une édition à la
    // main contournerait checklist et traduction, et l'opérateur ne relit pas le français.
    for (const id of ['compose-draft', 'compose-translation']) {
      const node = page.getByTestId(id);
      await expect(node).not.toHaveAttribute('contenteditable', 'true');
      expect(await node.evaluate((el) => el.tagName)).not.toBe('TEXTAREA');
      expect(await node.evaluate((el) => el.tagName)).not.toBe('INPUT');
    }

    // 7. « Envoyé » à côté de « Copier » (mesuré le 2026-10-12 : sent_at vide sur 25 ateliers sur 25,
    //    le geste était caché dans le menu de la liste). Clic réel, persistance sur le VRAI backend
    //    (rechargement), puis annulation : l'atelier de test ne doit pas compter comme un envoi réel.
    await page.context().grantPermissions(['clipboard-read', 'clipboard-write']).catch(() => {});
    const envoye = page.getByTestId('compose-sent');
    await expect(envoye).toHaveAttribute('data-sent', 'false');
    await page.getByTestId('compose-copy').click();
    await envoye.click();
    await expect(envoye).toHaveAttribute('data-sent', 'true', { timeout: 10000 });
    await expect(envoye).toContainText('✓');
    await page.reload({ waitUntil: 'load' });
    await expect(page.getByTestId('compose-sent')).toHaveAttribute('data-sent', 'true', { timeout: 15000 });
    await page.getByTestId('compose-sent').click();
    await expect(page.getByTestId('compose-sent')).toHaveAttribute('data-sent', 'false', { timeout: 10000 });
  });

  // L'HISTORIQUE : garder les ateliers, les retrouver, basculer de l'un à l'autre.
  //
  // POURQUOI ce test clique pour de vrai sur le VRAI backend : la bascule est précisément le geste
  // qui ne peut pas être prouvé par du rendu isolé — il faut qu'un atelier existe en base, qu'il
  // apparaisse dans la liste, et que l'ouvrir remplace réellement le contenu à l'écran.
  test('garde l’historique des ateliers et permet d’en changer', async ({ page }) => {
    await loginAsOwner(page, ADMIN_PW!);
    await expect(page.getByTestId('admin-cv-tool')).toBeVisible({ timeout: 12000 });

    await page.goto('/admin/compose', { waitUntil: 'load' });
    await expect(page.getByTestId('compose-root')).toBeVisible({ timeout: 12000 });

    // Le nom de l'atelier courant est lisible SANS rien ouvrir — c'était le manque principal :
    // rien n'indiquait sur quelle annonce on travaillait.
    const bouton = page.getByTestId('compose-workshop-button');
    await expect(bouton).toBeVisible({ timeout: 12000 });

    // La liste ne s'ouvre qu'au clic (elle n'est pas chargée au montage).
    await expect(page.getByTestId('compose-workshop-list')).toHaveCount(0);
    await bouton.click();
    const liste = page.getByTestId('compose-workshop-list');
    await expect(liste).toBeVisible({ timeout: 12000 });

    // Les deux onglets existent : on archive, on ne supprime jamais — les archives doivent donc
    // rester atteignables.
    await expect(page.getByTestId('compose-workshop-tab-active')).toBeVisible();
    await expect(page.getByTestId('compose-workshop-tab-archived')).toBeVisible();

    // Bascule réelle sur le premier atelier proposé, s'il y en a un en base.
    const premier = liste.locator('[data-testid^="compose-workshop-open-"]').first();
    if (await premier.isVisible().catch(() => false)) {
      const nom = (await premier.innerText()).split('\n')[0].trim();
      await premier.click();
      // La liste se referme : on a choisi, on retourne au travail.
      await expect(liste).toHaveCount(0);
      // Et le titre en tête reflète l'atelier ouvert.
      if (nom) await expect(bouton).toContainText(nom.slice(0, 12), { timeout: 12000 });
    }

    // Le panneau flottant ne doit pas faire déborder la page : c'est le contrat de confinement de
    // cet écran, et un panneau mal contenu le casserait sans qu'aucun test unitaire ne le voie.
    const debordePage = await page.evaluate(
      () => document.documentElement.scrollHeight > document.documentElement.clientHeight + 4,
    );
    expect(debordePage, 'la liste d’ateliers fait déborder la page').toBe(false);
  });

  test('l’écran est derrière la garde owner', async ({ page }) => {
    await page.context().clearCookies();
    await page.goto('/admin/compose', { waitUntil: 'load' });
    await expect(page).toHaveURL(/\/admin\/login$/);
  });
});
