import { test, expect, type Page, type Route } from '@playwright/test';
import { trackPageErrors } from './helpers/pageErrors';

// E2E de la refonte « montagne » du blog (passe 2) — liste et article.
//
// QUOI :
//  1. en /en, aucun des libellés français retouchés ne reste visible (liste ET article) ;
//  2. l'article se lit à une mesure de 55 à 80 caractères par ligne sur grand écran (cible 65–72) ;
//  3. l'abonnement fonctionne de bout en bout côté interface — réseau SIMULÉ (page.route) : aucun
//     vrai abonné n'est jamais créé par ce test ;
//  4. la couverture s'agrandit et se referme au clavier.
// Robuste aux données : l'article testé est le premier de la liste, lu sur la page.

test.describe.configure({ timeout: 60_000 });

// Libellés FR touchés par la refonte : aucun ne doit apparaître sur une page /en.
const LIBELLES_FR = [
  'Carnet de bord',
  'Écrit à partir de mes commits',
  'Filtrer par thème',
  'Flux RSS',
  'Précédent',
  'Retour au blog',
  'min de lecture',
  'Suivre par email',
  "S'abonner",
  'Généré depuis ces commits',
  'Sujets à suivre',
];

async function sansFrancais(page: Page) {
  const texte = await page.locator('main').innerText();
  for (const fr of LIBELLES_FR) expect(texte, `libellé FR visible : « ${fr} »`).not.toContain(fr);
}

async function premierArticle(page: Page, locale: string): Promise<string> {
  await page.goto(`/${locale}/blog`, { waitUntil: 'domcontentloaded' });
  const href = await page.locator('[data-testid="blog-grid"] a').first().getAttribute('href');
  expect(href).toBeTruthy();
  return href as string;
}

test.describe('Blog — refonte montagne', () => {
  test('liste en /en : libellés anglais, aucun français', async ({ page }) => {
    const erreurs = trackPageErrors(page);
    await page.goto('/en/blog', { waitUntil: 'domcontentloaded' });
    await expect(page.getByRole('heading', { level: 1 })).toHaveText('Blog');
    await expect(page.getByText('Logbook', { exact: true })).toBeVisible();
    await expect(page.locator('[data-testid="blog-rss"]')).toHaveText(/RSS feed/);
    await expect(page.locator('[data-testid="blog-page-next"]')).toHaveText(/Next/);
    await sansFrancais(page);
    expect(erreurs, 'exceptions JS:\n' + erreurs.join('\n')).toHaveLength(0);
  });

  test('article en /en : libellés anglais, aucun français', async ({ page }) => {
    const erreurs = trackPageErrors(page);
    await page.goto(await premierArticle(page, 'en'), { waitUntil: 'domcontentloaded' });
    await expect(page.locator('[data-testid="blog-back"]')).toHaveText(/Back to blog/);
    await expect(page.locator('[data-testid="subscribe-form"] h2')).toHaveText('Follow by email');
    await expect(page.locator('[data-testid="blog-article"]')).toContainText('min read');
    await sansFrancais(page);
    expect(erreurs, 'exceptions JS:\n' + erreurs.join('\n')).toHaveLength(0);
  });

  test('mesure de lecture : 55 à 80 caractères par ligne sur grand écran', async ({ page }) => {
    await page.setViewportSize({ width: 1920, height: 1080 });
    await page.goto(await premierArticle(page, 'en'), { waitUntil: 'domcontentloaded' });

    // Le plus long paragraphe de texte : on regroupe ses caractères par ligne (haut du rectangle de
    // chaque caractère) et on prend la médiane des lignes pleines (la dernière est exclue).
    const parLigne = await page.locator('[data-testid="blog-article"]').evaluate((article) => {
      const ps = Array.from(article.querySelectorAll('p')).filter((p) => p.children.length === 0);
      const p = ps.sort((a, b) => (b.textContent ?? '').length - (a.textContent ?? '').length)[0];
      const node = p.firstChild as Text;
      const range = document.createRange();
      const lignes = new Map<number, number>();
      for (let i = 0; i < node.length; i++) {
        range.setStart(node, i);
        range.setEnd(node, i + 1);
        const top = Math.round(range.getBoundingClientRect().top);
        lignes.set(top, (lignes.get(top) ?? 0) + 1);
      }
      const comptes = Array.from(lignes.values()).slice(0, -1).sort((a, b) => a - b);
      return comptes[Math.floor(comptes.length / 2)];
    });
    expect(parLigne).toBeGreaterThanOrEqual(55);
    expect(parLigne).toBeLessThanOrEqual(80);
  });

  test('abonnement : réseau simulé, sujets cochés transmis, succès annoncé', async ({ page }) => {
    const envois: unknown[] = [];
    // AUCUN vrai abonnement : les deux appels du formulaire sont interceptés. En local, le front
    // (localhost) appelle l'API de prod : la réponse simulée doit porter les en-têtes CORS, et le
    // préalable OPTIONS du POST JSON doit être servi aussi.
    const simuler = (corps: unknown, status = 200) => async (route: Route) => {
      const origin = route.request().headers()['origin'] ?? '*';
      const headers = {
        'Access-Control-Allow-Origin': origin,
        'Access-Control-Allow-Credentials': 'true',
        'Access-Control-Allow-Headers': 'Content-Type',
        'Access-Control-Allow-Methods': 'GET, POST, OPTIONS',
      };
      if (route.request().method() === 'OPTIONS') return route.fulfill({ status: 204, headers });
      if (route.request().method() === 'POST') envois.push(route.request().postDataJSON());
      return route.fulfill({ status, headers, json: corps });
    };
    await page.route('**/api/v1/blog/topics', simuler({ topics: ['Drifterra', 'tech'] }));
    await page.route('**/api/v1/blog/subscribe', simuler({ success: true, message: 'ok' }, 201));

    await page.goto(await premierArticle(page, 'en'), { waitUntil: 'domcontentloaded' });
    const form = page.locator('[data-testid="subscribe-form"]');
    const sujet = form.locator('[data-testid="subscribe-topic-Drifterra"]');
    await sujet.click();
    await expect(sujet).toHaveAttribute('aria-pressed', 'true');
    await form.getByLabel('Email address').fill('e2e@example.invalid');
    await form.locator('[data-testid="subscribe-submit"]').click();

    await expect(page.locator('[data-testid="subscribe-success"]')).toBeVisible();
    expect(envois).toHaveLength(1);
    expect(envois[0]).toMatchObject({ email: 'e2e@example.invalid', topics: ['Drifterra'] });
  });

  test('couverture : s’agrandit au clic, se referme avec Échap', async ({ page }) => {
    await page.goto(await premierArticle(page, 'fr'), { waitUntil: 'domcontentloaded' });
    const cover = page.locator('[data-testid="blog-cover"]');
    test.skip((await cover.count()) === 0, 'article sans couverture');
    await cover.click();
    const dialog = page.getByRole('dialog');
    await expect(dialog).toBeVisible();
    await page.keyboard.press('Escape');
    await expect(dialog).toHaveCount(0);
  });
});
