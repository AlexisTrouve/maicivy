import { test, expect, type Locator } from '@playwright/test';

/**
 * Portfolio — parcours réel : l'index, un clic sur une carte, la page projet.
 *
 * Le contenu vient de la curation maiProFiles (`data/book.yaml`), la même source qui produit le
 * PDF du portfolio book. Ces tests ne vérifient donc pas un projet nommé en dur — la curation
 * évolue — mais la STRUCTURE : des cartes, un clic qui mène à la bonne fiche, et les sections
 * qui font qu'une page projet est une page projet.
 */

test.describe('Portfolio', () => {
  // Les pages sont rendues côté serveur contre maiProFiles : le premier octet attend un aller-retour
  // API. Le défaut de 5 s d'expect suffit en local mais pas contre la prod, surtout avec les tests
  // en parallèle — mesuré, la page s'affiche bien, elle arrive juste après le délai.
  test.slow();
  test("l'index affiche les projets curés avec leurs captures", async ({ page }) => {
    await page.goto('/fr/portfolio');

    const cartes = page.getByTestId('portfolio-card');
    await expect(cartes.first()).toBeVisible({ timeout: 20000 });
    const nombre = await cartes.count();
    expect(nombre).toBeGreaterThan(5);

    // La capture est l'argument principal : une carte sans image trahirait une image non servie.
    // `toBeVisible` ne prouve que la présence dans le DOM — les images sont en `loading="lazy"`,
    // donc on attend que le navigateur l'ait RÉELLEMENT décodée (naturalWidth > 0).
    const image = cartes.first().locator('img');
    await expect(image).toBeVisible();
    await expect
      .poll(async () => image.evaluate((el: HTMLImageElement) => el.naturalWidth), { timeout: 20000 })
      .toBeGreaterThan(0);
  });

  test('un clic sur une carte mène à la page de CE projet', async ({ page }) => {
    await page.goto('/fr/portfolio');

    const carte = page.getByTestId('portfolio-card').first();
    await expect(carte).toBeVisible({ timeout: 20000 });
    const id = await carte.getAttribute('data-project-id');
    await carte.click();

    // Vérifier qu'il s'est passé la BONNE chose : la bonne URL, pas juste une navigation.
    // Timeout explicite : en routage App Router, l'URL ne change qu'une fois la charge RSC
    // revenue — donc après le rendu serveur, qui interroge maiProFiles. Le défaut de 5 s a
    // échoué une fois sur deux exécutions en parallèle ; ce n'est pas un aléa, c'est la latence
    // d'un rendu serveur qu'on mesure de bout en bout.
    await expect(page).toHaveURL(new RegExp(`/fr/portfolio/${id}$`), { timeout: 20000 });
    await expect(page.getByTestId('portfolio-project')).toBeVisible();
    await expect(page.getByTestId('portfolio-title')).not.toBeEmpty();
  });

  test('la page projet porte le récit, les chiffres et la fiche technique', async ({ page }) => {
    await page.goto('/fr/portfolio');
    await expect(page.getByTestId('portfolio-card').first()).toBeVisible({ timeout: 20000 });
    const id = await page.getByTestId('portfolio-card').first().getAttribute('data-project-id');
    await page.goto(`/fr/portfolio/${id}`);

    await expect(page.getByTestId('portfolio-hero-image')).toBeVisible({ timeout: 20000 });
    await expect(page.getByTestId('portfolio-pitch')).toBeVisible();
    await expect(page.getByTestId('portfolio-badges')).toBeVisible();

    // La fiche technique est rendue en HTML côté Go : on vérifie qu'elle a bien la structure
    // d'un document, pas une bouillie de markdown non converti.
    const fiche = page.getByTestId('portfolio-technical');
    await expect(fiche).toBeVisible();
    await expect(fiche.locator('h1, h2, h3').first()).toBeVisible();
    await expect(fiche).not.toContainText('## ');
  });

  test('un projet a plusieurs visuels affiche sa galerie', async ({ page }) => {
    // confluent_language porte 5 images curées : la page montre la première en tête et les
    // suivantes en galerie. Un projet à une seule image n'affiche pas de section vide.
    await page.goto('/fr/portfolio/confluent_language');

    const galerie = page.getByTestId('portfolio-gallery');
    await expect(galerie).toBeVisible({ timeout: 20000 });

    const visuels = galerie.locator('img');
    expect(await visuels.count()).toBeGreaterThan(1);
    // Chargés pour de vrai, pas seulement présents dans le DOM.
    await expect
      .poll(async () => visuels.first().evaluate((el: HTMLImageElement) => el.naturalWidth), {
        timeout: 20000,
      })
      .toBeGreaterThan(0);
  });

  test('la démo vidéo se lance au clic, avec sa légende dans la langue de la page', async ({ page }) => {
    // melodicode déclare une démo dans book.yaml (fichiers dans le store maiProFiles). Le lecteur
    // ne charge que les métadonnées avant le clic (preload="metadata" — avec "none", Chrome ignore le clic) : on vérifie donc que la vidéo JOUE pour de
    // vrai après un clic réel, pas seulement qu'une balise <video> existe.
    await page.goto('/fr/portfolio/melodicode');
    const section = page.getByTestId('portfolio-video');
    await expect(section).toBeVisible({ timeout: 20000 });
    await expect(page.getByTestId('portfolio-video-caption')).toContainText('Quatre chats MelodiCode');

    const lecteur = page.getByTestId('portfolio-video-player');
    expect(await lecteur.getAttribute('src')).toMatch(/^https:\/\/maiprofiles\.etheryale\.com\/images\//);
    expect(await lecteur.getAttribute('poster')).toMatch(/\/images\//);
    // Régression du 20/10 : preload="none" rendait le clic sans effet dans Chrome.
    expect(await lecteur.getAttribute('preload')).toBe('metadata');

    // Clic réel au centre du lecteur — sur Chromium, il lance la lecture d'une vidéo à contrôles.
    await lecteur.scrollIntoViewIfNeeded();
    await lecteur.click();
    await expect
      .poll(async () => lecteur.evaluate((v: HTMLVideoElement) => v.currentTime), { timeout: 30000 })
      .toBeGreaterThan(0.5);
    expect(await lecteur.evaluate((v: HTMLVideoElement) => v.videoWidth)).toBeGreaterThan(0);

    // L'édition anglaise sert la légende anglaise, le même fichier.
    await page.goto('/en/portfolio/melodicode');
    await expect(page.getByTestId('portfolio-video-caption')).toContainText('Four MelodiCode chats', {
      timeout: 20000,
    });
  });

  test("un projet sans démo n'affiche pas de lecteur vide", async ({ page }) => {
    await page.goto('/fr/portfolio/confluent_language');
    await expect(page.getByTestId('portfolio-pitch')).toBeVisible({ timeout: 20000 });
    await expect(page.getByTestId('portfolio-video')).toHaveCount(0);
  });

  test('les visuels localisés changent avec la langue', async ({ page }) => {
    // aurelm_gm_toolkit porte 4 visuels, tous avec une variante chinoise : les diagrammes ont
    // leurs libellés traduits. L'édition zh doit donc servir les fichiers de zh/, pas les mêmes.
    await page.goto('/zh/portfolio/aurelm_gm_toolkit');
    const galerie = page.getByTestId('portfolio-gallery');
    await expect(galerie).toBeVisible({ timeout: 20000 });

    const sources = await galerie.locator('img').evaluateAll((imgs) =>
      imgs.map((i) => (i as HTMLImageElement).getAttribute('src') || ''),
    );
    expect(sources.length).toBeGreaterThan(0);
    // Les images passent par /_next/image : le chemin d'origine est dans le paramètre `url`,
    // encodé — on le décode avant de chercher le dossier de langue.
    expect(sources.every((s) => decodeURIComponent(s).includes('/portfolio-images/zh/'))).toBe(true);
  });

  test('les captures sont des variantes optimisées, pas la pleine définition', async ({ page }) => {
    // QUOI : toutes les images de l'index et d'une fiche passent par /_next/image, et la variante
    // d'une carte est légère et en AVIF/WebP.
    // POURQUOI : régression du 2026-10-12 — captures servies en pleine définition, LCP ~12 s.
    // COMMENT : sur le modèle du test « miniatures » de blog-liste.spec.ts.
    const sources = (loc: Locator) =>
      loc.evaluateAll((imgs) => imgs.map((i) => (i as HTMLImageElement).getAttribute('src') || ''));
    const directes = (liste: string[]) =>
      liste.filter((s) => s.includes('portfolio-images') && !s.includes('/_next/image'));

    await page.goto('/fr/portfolio', { waitUntil: 'domcontentloaded' });
    const cartes = page.getByTestId('portfolio-card');
    await expect(cartes.first()).toBeVisible({ timeout: 20000 });
    const srcIndex = await sources(cartes.locator('img'));
    expect(srcIndex.length).toBeGreaterThan(0);
    expect(directes(srcIndex)).toEqual([]);

    const img = cartes.first().locator('img');
    await expect(img).toBeVisible();
    const src = (await img.evaluate((el: HTMLImageElement) => el.currentSrc || el.src)) ?? '';
    expect(src).toContain('/_next/image');
    const res = await page.request.get(src, { headers: { Accept: 'image/avif,image/webp,*/*' } });
    expect(res.status()).toBe(200);
    expect(res.headers()['content-type']).toMatch(/image\/(avif|webp)/);
    expect((await res.body()).length).toBeLessThan(200_000);

    // Une fiche projet : image de tête et galerie, aucune en src direct.
    const id = await cartes.first().getAttribute('data-project-id');
    await page.goto(`/fr/portfolio/${id}`, { waitUntil: 'domcontentloaded' });
    const fiche = page.getByTestId('portfolio-project');
    await expect(fiche).toBeVisible({ timeout: 20000 });
    const srcFiche = await sources(fiche.locator('img'));
    expect(srcFiche.length).toBeGreaterThan(0);
    expect(directes(srcFiche)).toEqual([]);
  });

  test('le retour ramène à l\'index', async ({ page }) => {
    await page.goto('/fr/portfolio');
    await expect(page.getByTestId('portfolio-card').first()).toBeVisible({ timeout: 20000 });
    const id = await page.getByTestId('portfolio-card').first().getAttribute('data-project-id');
    await page.goto(`/fr/portfolio/${id}`);

    await page.getByTestId('portfolio-back').click();
    await expect(page).toHaveURL(/\/fr\/portfolio$/);
    await expect(page.getByTestId('portfolio-card').first()).toBeVisible();
  });

  test('un projet hors curation affiche une page introuvable, pas une erreur serveur', async ({ page }) => {
    const reponse = await page.goto('/fr/portfolio/ce_projet_nexiste_pas');

    // Ce que voit le visiteur : la page 404 standard du site, pas une erreur serveur.
    expect(reponse?.status()).toBeLessThan(500);
    // Le h1 porte le code ET la phrase (« 404 · Page introuvable ») depuis la refonte passe 2.
    await expect(page.locator('h1')).toContainText('404');
    await expect(page.locator('h1')).toContainText('Page introuvable');

    // Vrai 404 HTTP depuis le 2026-10-12 : le `[locale]/loading.tsx` qui faisait diffuser la page
    // en flux (en-têtes 200 partis avant notFound()) a été retiré.
    expect(reponse?.status()).toBe(404);
  });

  test("l'interface suit la locale, y compris pour une langue non curée", async ({ page }) => {
    // Le livre est curé en en/fr/zh. En allemand, le CONTENU retombe sur l'anglais, mais
    // l'INTERFACE doit rester en allemand — deux publics, deux règles.
    await page.goto('/de/portfolio');
    await expect(page.getByTestId('portfolio-card').first()).toBeVisible({ timeout: 20000 });
    await expect(page.getByRole('link', { name: /Projekt ansehen/ }).first()).toBeVisible();
  });
});
