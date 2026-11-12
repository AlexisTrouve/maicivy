import { readFileSync } from 'fs';
import path from 'path';

import { test, expect, type Locator, type Page, type APIRequestContext } from '@playwright/test';

/**
 * Montagne — parcours réels sur l'accueil /[locale] (ex-/[locale]/montagne, 308) : la voix, le fond peint, le sommet (une
 * illustration par carte) et les PREUVES de chaque carte dans sa fleur (jeux bâtis sur GroveEngine,
 * démo vidéo de MelodiCode qui JOUE, chaîne Theomen → Emergika, pages réelles de maicivy), la
 * montagne à plat (HTML + SVG rendus côté serveur), la fleur du BON
 * projet après un vrai tap / clic, le bouton vers la BONNE page projet, la montagne présente sans
 * JavaScript, la vue réseau chargée au clic.
 *
 * À lancer sur les DEUX projets Playwright : `--project=chromium --project="Mobile Chrome"`
 * (Pixel 5, 393×851, tactile). Sur le projet tactile, chaque activation est un `tap()` réel.
 *
 * Données : rien n'est codé en dur sauf UNE tuile notoire, Drifterra (`drifterra`, redescendue du
 * sommet en v3), la liste du sommet — une décision éditoriale, verrouillée aussi côté backend
 * (montagne_sommet.go) — et les quatre pages du site qui prouvent maicivy.
 * Le reste est lu dans la réponse RÉELLE de l'endpoint (`GET /api/v1/montagne`), jamais simulée :
 * le test exige un backend qui tourne (MONTAGNE_API_URL, défaut http://localhost:8080).
 */

const API = process.env.MONTAGNE_API_URL || 'http://localhost:8080';
const NOTORIOUS = 'drifterra';
const SUMMIT = ['grove_engine', 'melodicode', 'theomen', 'maicivy'];
/** Pages de ce site qui prouvent maicivy (règle `site`), dans l'ordre. */
const SITE_PAGES = ['blog', 'gitstats', 'chat', 'cv'];
/** Le titre attendu vient du fichier de messages lui-même : Alexis le réécrira, le test suivra. */
const FR = JSON.parse(readFileSync(path.join(__dirname, '../../messages/fr.json'), 'utf-8'));

interface ApiNode {
  id: string;
  name: string;
  curated: boolean;
  visuals: { kind: string; url: string; poster_url?: string }[];
}
interface ApiProof {
  kind: 'image' | 'video' | 'page';
  url: string;
  poster_url?: string;
  project_id?: string;
  project_name?: string;
  stage?: string;
  page?: string;
  path?: string;
}
interface ApiCard {
  id: string;
  rule: string;
  proofs: ApiProof[];
  members: string[];
}
interface ApiMontagne {
  nodes: ApiNode[];
  links: { from: string; to: string }[];
  summit: ApiCard[];
}

/** Ids des cartes du sommet et projets qu'elles absorbent : aucun n'est une tuile. */
function notTiles(data: ApiMontagne): Set<string> {
  return new Set(data.summit.flatMap((c) => [c.id, ...c.members]));
}

/** Liens tels que dessinés : un membre absorbé se lit sur sa carte, les boucles disparaissent. */
function drawnLinks(data: ApiMontagne): Set<string> {
  const holder = new Map(data.summit.flatMap((c) => c.members.map((m) => [m, c.id] as const)));
  return new Set(data.links.filter((l) => (holder.get(l.from) ?? l.from) !== (holder.get(l.to) ?? l.to)).map((l) => `${l.from}>${l.to}`));
}

/** Ouvre la fleur d'une carte du sommet par un vrai tap / clic. */
async function openSummitFlower(page: Page, id: string, touch: boolean) {
  await activate(page, page.locator(`[data-testid="montagne-summit-card"][data-project-id="${id}"]`), touch);
  const flower = page.getByTestId('montagne-flower');
  await expect(flower).toHaveAttribute('data-project-id', id);
  await expect(flower).toHaveAttribute('data-variant', 'proofs');
  return flower;
}

async function fetchMontagne(request: APIRequestContext, lang: string): Promise<ApiMontagne> {
  const res = await request.get(`${API}/api/v1/montagne?lang=${lang}`, { timeout: 60_000 });
  expect(res.status(), 'le backend doit servir la montagne (aucune simulation)').toBe(200);
  return res.json();
}

/**
 * Couleur moyenne d'une bande de la PAGE (coordonnées du document), mesurée sur les pixels rendus.
 * POURQUOI les pixels et pas un style calculé : « le fond est là » se juge à l'écran — un calque
 * présent mais transparent, masqué ou hors champ passerait un test de style.
 * COMMENT : capture Playwright de la bande, décodée dans la page même (Image + canvas) : aucune
 * dépendance de décodage PNG côté Node.
 */
async function meanColor(page: Page, clip: { x: number; y: number; width: number; height: number }) {
  const png = await page.screenshot({ clip, fullPage: true });
  return page.evaluate(async (b64) => {
    const img = new Image();
    img.src = `data:image/png;base64,${b64}`;
    await img.decode();
    const canvas = document.createElement('canvas');
    canvas.width = img.width;
    canvas.height = img.height;
    const ctx = canvas.getContext('2d')!;
    ctx.drawImage(img, 0, 0);
    const d = ctx.getImageData(0, 0, canvas.width, canvas.height).data;
    let r = 0;
    let g = 0;
    let b = 0;
    for (let i = 0; i < d.length; i += 4) {
      r += d[i];
      g += d[i + 1];
      b += d[i + 2];
    }
    const n = d.length / 4;
    return { r: r / n, g: g / n, b: b / n };
  }, png.toString('base64'));
}

/** Opacité de trait CALCULÉE de chaque filet de la géométrie affichée, par lien `from>to`. */
async function linkOpacities(page: Page): Promise<Map<string, number>> {
  const entries = await page
    .locator('[data-testid="montagne-links"]:visible path[data-from]')
    .evaluateAll((els) => els.map((e) => [`${e.getAttribute('data-from')}>${e.getAttribute('data-to')}`, Number(getComputedStyle(e).strokeOpacity)] as const));
  return new Map(entries);
}

/** Active un élément comme le ferait l'utilisateur de ce projet : un doigt, ou une souris. */
async function activate(page: Page, target: Locator, touch: boolean) {
  await target.scrollIntoViewIfNeeded();
  if (touch) await target.tap();
  else await target.click();
}

/**
 * La page est hydratée : la délégation de clic de MontagneExplorer est en place. Pas
 * `networkidle` : la démo vidéo du sommet, en boucle, garde le réseau actif indéfiniment.
 */
async function waitHydrated(page: Page) {
  await expect(page.getByTestId('montagne-explorer')).toHaveAttribute('data-hydrated', 'true', { timeout: 60_000 });
}

test.describe('Montagne', () => {
  // Rendu serveur contre le backend local (et compilation à la première visite en dev).
  test.slow();

  test('la voix : titre, ligne dorée, CV PDF et chat — sous l’en-tête unique du site', async ({ page }) => {
    await page.goto('/fr');
    const voice = page.getByTestId('montagne-voice');
    await expect(voice).toBeVisible({ timeout: 30_000 });
    await expect(page.getByTestId('montagne-voice-title')).toHaveText(FR.montagne.voice.title);
    await expect(page.getByRole('heading', { level: 1 })).toHaveText(FR.montagne.voice.title);
    await expect(page.getByTestId('montagne-voice-tagline')).toHaveText(FR.montagne.voice.tagline);
    await expect(page.getByTestId('montagne-voice-cv')).toHaveAttribute('href', /\/api\/v1\/cv\/export\?.*lang=fr/);
    await expect(page.getByTestId('montagne-voice-chat')).toHaveAttribute('href', '/fr/chat');
    // Un seul en-tête : celui du site, qui a repris le langage de l'ancien bandeau de la montagne.
    await expect(page.locator('header')).toHaveCount(1);
    await expect(page.getByTestId('site-header-contact')).toHaveAttribute('href', /^mailto:.+@.+/);
    // La voix précède le sommet à l'écran : c'est elle qu'on lit d'abord.
    const titleBox = (await page.getByTestId('montagne-voice-title').boundingBox())!;
    const firstCard = (await page.getByTestId('montagne-summit-card').first().boundingBox())!;
    expect(titleBox.y + titleBox.height).toBeLessThan(firstCard.y);
  });

  test('le sommet : les 4 cartes choisies, dans l’ordre, chacune couverte par SON illustration, réellement décodée', async ({ page, request }) => {
    const data = await fetchMontagne(request, 'fr');
    expect(data.summit.map((c) => c.id)).toEqual(SUMMIT);
    await page.goto('/fr');

    const cards = page.getByTestId('montagne-summit-card');
    await expect(cards).toHaveCount(SUMMIT.length);
    expect(await cards.evaluateAll((els) => els.map((e) => e.getAttribute('data-project-id')))).toEqual(SUMMIT);

    const covers: string[] = [];
    for (const c of data.summit) {
      const node = data.nodes.find((n) => n.id === c.id)!;
      const card = page.locator(`[data-testid="montagne-summit-card"][data-project-id="${c.id}"]`);
      await expect(card).toBeVisible();
      await expect(card).toContainText(node.name);
      // La couverture est l'illustration de CETTE carte (lib/montagne/art.ts), plus une preuve :
      // aucune vidéo, aucune capture dans la carte — les preuves sont dans la fleur.
      await expect(card).toHaveAttribute('data-visual', 'art');
      await expect(card.locator('video')).toHaveCount(0);
      const cover = card.getByTestId('montagne-summit-cover');
      await expect(cover).toHaveCount(1);
      await expect(cover).toHaveAttribute('alt', FR.montagne.art.covers[c.id]);
      await cover.scrollIntoViewIfNeeded();
      // Décodée pour de vrai : un fichier absent ou cassé laisse naturalWidth à 0.
      await expect.poll(async () => cover.evaluate((i: HTMLImageElement) => (i.complete ? i.naturalWidth : 0)), { timeout: 30_000 }).toBeGreaterThan(0);
      const src = decodeURIComponent((await cover.evaluate((i: HTMLImageElement) => i.currentSrc)) ?? '');
      expect(src, `${c.id} : sa propre illustration`).toContain(`/montagne/art/cover-${c.id}.webp`);
      covers.push(src);
    }
    // Quatre illustrations distinctes (la v3 montrait la même capture sur deux cartes).
    expect(new Set(covers).size).toBe(SUMMIT.length);
  });

  test('le fond : la montagne peinte est décodée, dans le bon cadrage, derrière le sommet', async ({ page }) => {
    await page.goto('/fr');
    const backdrop = page.getByTestId('montagne-backdrop');
    await expect(backdrop).toHaveCount(1);
    const img = backdrop.locator('img');
    await expect(img).toHaveAttribute('alt', FR.montagne.art.backdrop);
    await expect.poll(async () => img.evaluate((i: HTMLImageElement) => (i.complete ? i.naturalWidth : 0)), { timeout: 30_000 }).toBeGreaterThan(0);
    // Cadrage portrait sous 768 px (téléphone), paysage au-delà.
    const viewport = page.viewportSize()!;
    const src = await img.evaluate((i: HTMLImageElement) => i.currentSrc);
    expect(src).toContain(viewport.width < 768 ? '/montagne/art/fond-mobile.webp' : '/montagne/art/fond-desktop.webp');
    // La pointe peinte naît au bas du sommet : le fond enjambe le bord bas de la DERNIÈRE carte
    // (sur téléphone, son haut — masqué, fondu dans la nuit — tombe dans la première rangée de
    // cartes : c'est voulu, cf. montagne.module.css › .backdrop), et il passe DERRIÈRE les cartes.
    // Carte à l'écran d'abord : elementFromPoint ne voit que la fenêtre, et les boîtes sont
    // mesurées dans le même état de défilement.
    const lastCard = page.getByTestId('montagne-summit-card').last();
    await lastCard.scrollIntoViewIfNeeded();
    const art = (await backdrop.boundingBox())!;
    const card = (await lastCard.boundingBox())!;
    const summitBottom = card.y + card.height;
    expect(art.y).toBeLessThan(summitBottom);
    expect(art.y + art.height).toBeGreaterThan(summitBottom);
    expect(await backdrop.evaluate((e) => getComputedStyle(e).zIndex)).toBe('-1');
    // Un tap au milieu d'une carte touche la carte, pas le fond.
    const hit = await page.evaluate(
      ([x, y]) => document.elementFromPoint(x, y)?.closest('[data-testid="montagne-summit-card"]') !== null,
      [card.x + card.width / 2, card.y + card.height / 2] as const,
    );
    expect(hit).toBe(true);
    // Plus large que l'écran, il est rogné : aucun défilement horizontal.
    expect(await page.evaluate(() => document.documentElement.scrollWidth)).toBeLessThanOrEqual(viewport.width);
  });

  test('la langue : le sélecteur affiche UN code, jamais un drapeau qui se lit en lettres', async ({ page }) => {
    await page.goto('/fr');
    const trigger = page.getByTestId('site-header-language').locator('button[aria-haspopup]');
    await expect(trigger).toBeVisible({ timeout: 30_000 });
    // Le drapeau emoji (deux « regional indicators ») n'a pas de glyphe sous Windows : Chromium y
    // dessine les deux lettres, et l'écran affichait « FR FR » (v4). Le bouton ne porte plus que
    // le code : tout le texte rendu, espaces retirés, est exactement « FR ».
    expect((await trigger.innerText()).replace(/\s+/g, '')).toBe('FR');
    // Le menu garde les cinq langues, et le choix mène à la même page dans l'autre langue.
    await waitHydrated(page);
    await trigger.click();
    await expect(page.getByRole('option')).toHaveCount(5);
  });

  test('les filets : discrets au repos, ceux d’une carte s’allument au survol / focus, les autres restent en retrait', async ({ page }, testInfo) => {
    await page.goto('/fr');
    await waitHydrated(page);
    // Repos : AUCUN filet n'est un trait plein — v4 les dessinait tous à 0,75 + halo, et la
    // vingtaine de traits partant de GroveEngine brouillait la composition.
    const rest = await linkOpacities(page);
    expect(rest.size).toBeGreaterThan(0);
    for (const [link, op] of rest) expect(op, `${link} au repos`).toBeLessThanOrEqual(0.3);

    const own = [...rest.keys()].filter((k) => k.split('>').includes('grove_engine'));
    const others = [...rest.keys()].filter((k) => !k.split('>').includes('grove_engine'));
    expect(own.length, 'GroveEngine a des liens réels').toBeGreaterThan(0);
    const card = page.locator('[data-testid="montagne-summit-card"][data-project-id="grove_engine"]');
    // Focus clavier sur les deux projets ; survol en plus à la souris (sur le tactile, le doigt
    // ouvre la fleur : pas d'état de survol à tester).
    const states: [string, () => Promise<void>][] = [['focus', () => card.focus()]];
    if (!testInfo.project.use.hasTouch) states.push(['survol', () => card.hover()]);
    for (const [label, enter] of states) {
      await enter();
      // Transition CSS : on attend la fin de l'allumage avant de juger.
      await expect
        .poll(async () => {
          const now = await linkOpacities(page);
          return Math.min(...own.map((k) => now.get(k)!));
        }, { message: `${label} : filets de GroveEngine allumés` })
        .toBeGreaterThanOrEqual(0.8);
      const lit = await linkOpacities(page);
      for (const k of own) expect(lit.get(k)!, `${label} : ${k}`).toBeGreaterThanOrEqual(0.8);
      for (const k of others) expect(lit.get(k)!, `${label} : ${k} reste en retrait`).toBeLessThanOrEqual(0.3);
      // Sortie : retour au repos.
      await card.blur();
      await page.mouse.move(0, 0);
      await expect.poll(async () => Math.max(...(await linkOpacities(page)).values())).toBeLessThanOrEqual(0.3);
    }
  });

  test('le bas de la montagne : le dernier rang repose sur la roche et la brume, pas sur la nuit', async ({ page }, testInfo) => {
    await page.goto('/fr');
    const backdrop = page.getByTestId('montagne-backdrop');
    await expect.poll(async () => backdrop.locator('img').evaluate((i: HTMLImageElement) => (i.complete ? i.naturalWidth : 0)), { timeout: 30_000 }).toBeGreaterThan(0);
    // On mesure le FOND derrière le dernier rang : tuiles, noms et filets masqués le temps de la
    // capture (sinon on mesurerait les tuiles).
    await page.addStyleTag({ content: '[data-testid="montagne-mountain"] ol, [data-testid="montagne-links"] { visibility: hidden !important; }' });
    const rows = await page.getByTestId('montagne-tile').evaluateAll((els) => els.map((e) => e.getBoundingClientRect().top + window.scrollY));
    const lastRow = Math.max(...rows);
    const tile = (await page.getByTestId('montagne-tile').first().boundingBox())!.height;
    const viewport = page.viewportSize()!;
    const bottom = await meanColor(page, { x: 0, y: lastRow, width: viewport.width, height: tile });
    // Repère : la nuit pure du ciel (#04060b ≈ 4, 6, 11) — bleutée, R < B.
    testInfo.annotations.push({ type: 'fond du dernier rang', description: `r=${bottom.r.toFixed(1)} g=${bottom.g.toFixed(1)} b=${bottom.b.toFixed(1)}` });
    // Brume dorée : chaude (R nettement au-dessus de B) et plus claire que la nuit.
    expect(bottom.r - bottom.b, 'fond chaud derrière le dernier rang').toBeGreaterThanOrEqual(4);
    expect(bottom.r, 'fond plus clair que la nuit derrière le dernier rang').toBeGreaterThanOrEqual(14);
    // La pointe reste visible : rien ne la recouvre — le fond prolongé commence sous les cartes.
    const lastCard = (await page.getByTestId('montagne-summit-card').last().boundingBox())!;
    const foot = page.getByTestId('montagne-foothills');
    if ((await foot.count()) > 0) {
      const top = await foot.evaluate((e) => e.getBoundingClientRect().top + window.scrollY);
      const cardsBottom = lastCard.y + lastCard.height + (await page.evaluate(() => window.scrollY));
      expect(top, 'le prolongement commence bien sous la pointe').toBeGreaterThan(cardsBottom + 200);
    }
  });

  test('MelodiCode : dans la fleur, la démo est lisible avec ses contrôles et se lance au tap / clic', async ({ page }, testInfo) => {
    const touch = !!testInfo.project.use.hasTouch;
    await page.goto('/fr');
    await waitHydrated(page);
    const flower = await openSummitFlower(page, 'melodicode', touch);
    const player = flower.getByTestId('montagne-proof-video');
    await expect(player).toHaveCount(1);
    expect(await player.evaluate((v: HTMLVideoElement) => v.controls)).toBe(true);
    expect(await player.getAttribute('preload')).toBe('metadata');
    // Les métadonnées chargées, la vidéo a une durée : le lecteur est prêt à être lancé.
    await expect.poll(async () => player.evaluate((v: HTMLVideoElement) => v.readyState), { timeout: 30_000 }).toBeGreaterThanOrEqual(1);
    await player.scrollIntoViewIfNeeded();
    if (touch) {
      // Sur téléphone, Chrome pose sa barre de contrôles AU MILIEU d'un lecteur compact : un tap au
      // centre tombe sur l'affichage du temps (sans effet). L'utilisateur touche le ▶, à gauche de
      // cette barre — mesuré sur Pixel 5 : ~26 px du bord, aux trois quarts de la hauteur.
      const box = (await player.boundingBox())!;
      await player.tap({ position: { x: 26, y: box.height * 0.75 } });
    } else {
      await player.click();
    }
    await expect.poll(async () => player.evaluate((v: HTMLVideoElement) => v.currentTime), { timeout: 45_000 }).toBeGreaterThan(0.5);
  });

  test('GroveEngine : sa fleur montre les jeux construits sur lui (liens réels), jamais sa propre image', async ({ page, request }, testInfo) => {
    const touch = !!testInfo.project.use.hasTouch;
    const data = await fetchMontagne(request, 'fr');
    const built = new Set(data.links.filter((l) => l.from === 'grove_engine').map((l) => l.to));
    const builtVisuals = data.nodes.filter((n) => built.has(n.id)).flatMap((n) => n.visuals.map((v) => v.url.split('?')[0]));
    const ownVisuals = data.nodes.find((n) => n.id === 'grove_engine')!.visuals.map((v) => v.url.split('?')[0]);

    await page.goto('/fr');
    await waitHydrated(page);
    const flower = await openSummitFlower(page, 'grove_engine', touch);
    const proofs = flower.getByTestId('montagne-proof');
    await expect(proofs.first()).toBeVisible();
    // Chaque preuve est portée par un projet relié à GroveEngine.
    for (const id of await proofs.evaluateAll((els) => els.map((e) => e.getAttribute('data-proof-project')))) {
      expect(built.has(id ?? ''), `${id} : relié à grove_engine`).toBe(true);
    }
    // Les images affichées : au moins une d'un jeu relié, aucune de GroveEngine lui-même.
    const srcs = await flower.locator('[data-testid="montagne-proof"] img').evaluateAll((els) => els.map((e) => decodeURIComponent(e.getAttribute('src') ?? '')));
    expect(srcs.length).toBeGreaterThan(0);
    expect(srcs.some((src) => builtVisuals.some((u) => src.includes(u)))).toBe(true);
    for (const src of srcs) for (const own of ownVisuals) expect(src).not.toContain(own);
    // Les images sont réellement chargées (pas un cadre cassé).
    await expect.poll(async () => flower.locator('[data-testid="montagne-proof"] img').first().evaluate((i: HTMLImageElement) => i.complete && i.naturalWidth), { timeout: 30_000 }).toBeGreaterThan(0);
  });

  test('Theomen : sa fleur montre la chaîne global → local, chaque étage légendé, sans cadre vide', async ({ page, request }, testInfo) => {
    const touch = !!testInfo.project.use.hasTouch;
    const data = await fetchMontagne(request, 'fr');
    const card = data.summit.find((c) => c.id === 'theomen')!;
    const emergika = data.nodes.find((n) => n.id === 'emergika')!;

    await page.goto('/fr');
    await waitHydrated(page);
    const flower = await openSummitFlower(page, 'theomen', touch);
    const global = flower.locator('[data-testid="montagne-proof"][data-stage="global"]');
    await expect(global.first()).toBeVisible();
    await expect(global.first()).toContainText(FR.montagne.proofs.stages.global.replace('{project}', 'Theomen'));
    const locals = flower.locator('[data-testid="montagne-proof"][data-stage="local"]');
    await expect(locals).toHaveCount(card.proofs.filter((p) => p.stage === 'local').length);
    if (emergika.visuals.length === 0) {
      // Emergika sans visuel : aucun étage local (pas de cadre vide) et elle reste une tuile.
      await expect(locals).toHaveCount(0);
      expect(card.members).toEqual([]);
      await expect(page.locator('[data-testid="montagne-tile"][data-project-id="emergika"]')).toHaveCount(1);
    } else {
      // Emergika illustrée : absorbée par la carte, plus aucune tuile en double.
      expect(card.members).toContain('emergika');
      await expect(page.locator('[data-testid="montagne-tile"][data-project-id="emergika"]')).toHaveCount(0);
    }
  });

  test('maicivy : sa fleur montre les vraies pages du site, et une capture touchée mène à la page', async ({ page }, testInfo) => {
    const touch = !!testInfo.project.use.hasTouch;
    await page.goto('/fr');
    await waitHydrated(page);
    const flower = await openSummitFlower(page, 'maicivy', touch);
    const pages = flower.getByTestId('montagne-proof-page');
    expect(await pages.evaluateAll((els) => els.map((e) => e.getAttribute('data-page')))).toEqual(SITE_PAGES);
    expect(await pages.evaluateAll((els) => els.map((e) => e.getAttribute('href')))).toEqual(SITE_PAGES.map((p) => `/fr/${p}`));
    // Les captures sont chargées (fichiers réels de public/montagne/maicivy).
    for (const p of SITE_PAGES) {
      const img = flower.locator(`[data-testid="montagne-proof-page"][data-page="${p}"] img`);
      await img.scrollIntoViewIfNeeded();
      await expect.poll(async () => img.evaluate((i: HTMLImageElement) => i.complete && i.naturalWidth), { timeout: 20_000 }).toBeGreaterThan(0);
    }
    await activate(page, flower.locator('[data-testid="montagne-proof-page"][data-page="blog"]'), touch);
    await expect(page).toHaveURL(/\/fr\/blog$/, { timeout: 60_000 });
  });

  test('la montagne : toutes les autres tuiles, touchables, dans l’écran, reliées par les liens réels', async ({ page, request }, testInfo) => {
    const data = await fetchMontagne(request, 'fr');
    await page.goto('/fr');

    const tiles = page.getByTestId('montagne-tile');
    const hidden = notTiles(data);
    const expected = data.nodes.filter((n) => !hidden.has(n.id)).map((n) => n.id);
    await expect(tiles).toHaveCount(expected.length);
    expect(new Set(await tiles.evaluateAll((els) => els.map((e) => e.getAttribute('data-project-id'))))).toEqual(new Set(expected));

    // Taille au pouce : 44 px au moins (sur téléphone comme ailleurs), et rien hors de l'écran
    // en largeur — la montagne s'allonge, elle ne déborde pas et ne rapetisse pas.
    const viewport = page.viewportSize()!;
    const boxes = await tiles.evaluateAll((els) =>
      els.map((e) => {
        const r = e.getBoundingClientRect();
        return { id: e.getAttribute('data-project-id'), w: r.width, h: r.height, left: r.left, right: r.right };
      }),
    );
    for (const b of boxes) {
      expect(b.w, `${b.id} : largeur`).toBeGreaterThanOrEqual(44);
      expect(b.h, `${b.id} : hauteur`).toBeGreaterThanOrEqual(44);
      expect(b.left, `${b.id} : coupé à gauche`).toBeGreaterThanOrEqual(0);
      expect(b.right, `${b.id} : coupé à droite`).toBeLessThanOrEqual(viewport.width);
    }
    expect(await page.evaluate(() => document.documentElement.scrollWidth), 'aucun défilement horizontal').toBeLessThanOrEqual(viewport.width);

    // Filets : exactement les liens réels (ceux d'un projet absorbé partent de sa carte), dans le
    // SVG de la géométrie affichée.
    const svg = page.locator('[data-testid="montagne-links"]:visible');
    await expect(svg).toHaveCount(1);
    const drawn = await svg.locator('path[data-from]').evaluateAll((els) => els.map((e) => `${e.getAttribute('data-from')}>${e.getAttribute('data-to')}`));
    expect(new Set(drawn)).toEqual(drawnLinks(data));
    testInfo.annotations.push({ type: 'géométrie', description: `${viewport.width}px, ${drawn.length} filets` });
  });

  test(`un ${'tap / clic'} sur la tuile ${NOTORIOUS} ouvre SA fleur, et son bouton mène à SA page projet`, async ({ page, request }, testInfo) => {
    const touch = !!testInfo.project.use.hasTouch;
    const data = await fetchMontagne(request, 'fr');
    const node = data.nodes.find((n) => n.id === NOTORIOUS)!;
    expect(node, `${NOTORIOUS} doit figurer dans la montagne`).toBeTruthy();

    await page.goto('/fr');
    await waitHydrated(page);
    await activate(page, page.locator(`[data-testid="montagne-tile"][data-project-id="${NOTORIOUS}"]`), touch);

    const flower = page.getByTestId('montagne-flower');
    await expect(flower).toBeVisible();
    await expect(flower).toHaveAttribute('data-project-id', NOTORIOUS);
    await expect(page.getByTestId('montagne-flower-name')).toHaveText(node.name);
    await expect(page.getByTestId('montagne-flower-petal')).toHaveCount(node.visuals.length);
    // La fleur couvre la fenêtre et sa carte y tient entière (téléphone compris).
    const card = (await page.getByTestId('montagne-flower-name').boundingBox())!;
    const viewport = page.viewportSize()!;
    expect(card.x).toBeGreaterThanOrEqual(0);
    expect(card.x + card.width).toBeLessThanOrEqual(viewport.width);

    const bouton = page.getByTestId('montagne-flower-portfolio');
    if (!node.curated) {
      await expect(bouton).toHaveCount(0);
      return;
    }
    await expect(bouton).toHaveAttribute('href', `/fr/portfolio/${NOTORIOUS}`);
    await activate(page, bouton, touch);
    await expect(page).toHaveURL(new RegExp(`/fr/portfolio/${NOTORIOUS}$`), { timeout: 30_000 });
    await expect(page.getByTestId('portfolio-project')).toBeVisible({ timeout: 30_000 });
  });

  test('chaque carte du sommet ouvre la fleur de SES preuves ; la fermeture rend la page', async ({ page }, testInfo) => {
    const touch = !!testInfo.project.use.hasTouch;
    await page.goto('/fr');
    await waitHydrated(page);
    for (const id of SUMMIT) {
      const flower = await openSummitFlower(page, id, touch);
      await expect(flower.getByTestId('montagne-proofs')).toBeVisible();
      await activate(page, page.getByTestId('montagne-flower-close'), touch);
      await expect(flower).toHaveCount(0);
    }
    // La fleur suspend le défilement de la page ; fermée, elle le rend.
    expect(await page.evaluate(() => document.body.style.overflow)).not.toBe('hidden');
  });

  test('la vue réseau : absente au chargement, ouverte au clic, refermable', async ({ page }, testInfo) => {
    const touch = !!testInfo.project.use.hasTouch;
    await page.goto('/fr');
    await waitHydrated(page);
    // Le canvas n'est ni rendu ni monté avant le clic : la vue principale est la montagne à plat.
    await expect(page.getByTestId('montagne-canvas')).toHaveCount(0);
    await expect(page.getByTestId('montagne-stage')).toHaveCount(0);

    await activate(page, page.getByTestId('montagne-network-open'), touch);
    await expect(page.getByTestId('montagne-network')).toBeVisible();
    const stage = page.getByTestId('montagne-stage');
    await expect(stage).toHaveAttribute('data-ready', 'true', { timeout: 60_000 });
    await expect(page.getByTestId('montagne-canvas')).toBeVisible();
    // Le réseau porte tous les projets.
    const nodes = await page.locator('[data-testid="montagne-node"]').count();
    const data = await fetchMontagne(page.request, 'fr');
    expect(nodes).toBe(data.nodes.length);
    expect(await page.getByTestId('montagne-tile').count()).toBe(data.nodes.length - notTiles(data).size);

    await activate(page, page.getByTestId('montagne-network-close'), touch);
    await expect(page.getByTestId('montagne-network')).toHaveCount(0);
  });

  test.describe('sans JavaScript', () => {
    test.use({ javaScriptEnabled: false });

    test('la voix, le sommet, la montagne et la liste sont dans le HTML ; chaque tuile reste un lien', async ({ page, request }) => {
      const data = await fetchMontagne(request, 'fr');
      await page.goto('/fr');

      await expect(page.getByTestId('montagne-voice-title')).toHaveText(FR.montagne.voice.title);
      await expect(page.getByTestId('montagne-summit-card')).toHaveCount(SUMMIT.length);
      const tiles = page.getByTestId('montagne-tile');
      await expect(tiles).toHaveCount(data.nodes.length - notTiles(data).size);
      await expect(page.locator('[data-testid="montagne-links"]:visible path[data-from]')).toHaveCount(drawnLinks(data).size);
      await expect(page.getByTestId('montagne-network-open')).toBeVisible();
      await expect(page.getByTestId('montagne-canvas')).toHaveCount(0);

      // Repli sans JS : la page projet si elle existe, sinon l'entrée de la liste (présente).
      const notorious = page.locator(`[data-testid="montagne-tile"][data-project-id="${NOTORIOUS}"]`);
      const curated = data.nodes.find((n) => n.id === NOTORIOUS)!.curated;
      await expect(notorious).toHaveAttribute('href', curated ? `/fr/portfolio/${NOTORIOUS}` : `#projet-${NOTORIOUS}`);
      const uncurated = data.nodes.find((n) => !n.curated && !notTiles(data).has(n.id));
      if (uncurated) {
        await expect(page.locator(`[data-testid="montagne-tile"][data-project-id="${uncurated.id}"]`)).toHaveAttribute('href', `#projet-${uncurated.id}`);
        await expect(page.locator(`#projet-${uncurated.id}`)).toHaveCount(1);
      }

      // La liste complète, avec ses liens portfolio : seulement et toutes les entrées curées.
      await expect(page.getByTestId('montagne-list-item')).toHaveCount(data.nodes.length);
      await expect(page.getByTestId('montagne-list-portfolio')).toHaveCount(data.nodes.filter((n) => n.curated).length);
    });
  });
});
