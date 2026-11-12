import { readFileSync } from 'fs';
import path from 'path';

import { test, expect, devices, type Page } from '@playwright/test';

import { mockLiveRun, recordedRun } from './helpers/dossierVivant';
import { trackDuplicateKeys } from './helpers/reactKeys';

/**
 * Dossier vivant — la version TÉLÉPHONE, au doigt (projet « Mobile Chrome » : Pixel 5, tactile).
 * Le même parcours que sur PC, raconté de haut en bas : le besoin, l'équipe au travail sous la
 * montagne, les projets qui s'allument (tuiles d'or + bandeau des retenus), le dossier remis en
 * plein écran qui se lit en DÉFILANT LA PAGE (plus d'ascenseur interne), et son PDF déclenché par un TAP.
 *
 * Invariants vérifiés à chaque moment clé (ouverture, travail, remise) :
 *  - rien ne déborde à droite : ni la page (`scrollWidth <= clientWidth`), ni un élément visible
 *    hors de l'écran, ni un conteneur qui défilerait de côté — seul le bandeau des retenus défile
 *    horizontalement, c'est voulu ;
 *  - toute cible tactile visible (lien, bouton, champ) fait au moins 44 × 44 px ;
 *  - aucune clé React en double en console (helpers/reactKeys.ts).
 * Gestes : `tap()` (vrai événement tactile) et, pour défiler, un glissement au DOIGT (touchStart /
 * touchMove / touchEnd par CDP, cf. swipeUp) — pas un `scrollTop` posé à la main.
 *
 * Téléphone COUCHÉ (dernier bloc, Pixel 5 paysage, 802 × 293) : mêmes invariants, sa propre
 * disposition (`data-layout="landscape"` : côte à côte, densité du téléphone), le parcours rejoué
 * jusqu'au dossier et son PDF ; une tuile allumée y porte la marque de son projet.
 *
 * Banc : le même que la version PC (MONTAGNE_API_URL, backend réel pour la montagne, les fiches,
 * les compétences et le PDF du rejeu). Lancer :
 * `npx playwright test tests/e2e/dossier-vivant-mobile.spec.ts --project="Mobile Chrome"`
 */

const FR = JSON.parse(readFileSync(path.join(__dirname, '../../messages/fr.json'), 'utf-8'));
/** Cible tactile minimale (WCAG 2.5.5 / recommandations iOS et Android). */
const MIN_TARGET = 44;

/** Les cibles tactiles visibles de la page qui font moins de 44 px de côté. */
async function smallTargets(page: Page): Promise<string[]> {
  return page.evaluate((min) => {
    const root = document.querySelector('[data-testid="dv-root"]')!;
    return [...root.querySelectorAll<HTMLElement>('a[href], button, input, textarea')]
      .filter((el) => el.checkVisibility({ visibilityProperty: true }) && !el.closest('[aria-hidden="true"]'))
      .map((el) => ({ el, r: el.getBoundingClientRect() }))
      .filter(({ r }) => r.width < min || r.height < min)
      .map(({ el, r }) => `${el.dataset.testid ?? el.textContent?.trim().slice(0, 30)} ${Math.round(r.width)}×${Math.round(r.height)}`);
  }, MIN_TARGET);
}

/**
 * Débordements horizontaux : la page, chaque élément visible (hors montagne, image mise à l'échelle
 * dont le fond peint est volontairement plus large et rogné), et tout conteneur qui défilerait de côté.
 */
async function horizontalOverflow(page: Page): Promise<string[]> {
  return page.evaluate(() => {
    const doc = document.documentElement;
    const W = doc.clientWidth;
    const out: string[] = [];
    if (doc.scrollWidth > W) out.push(`page : scrollWidth ${doc.scrollWidth} > ${W}`);
    const root = document.querySelector('[data-testid="dv-root"]')!;
    const name = (el: Element) => `${el.tagName.toLowerCase()}.${(el as HTMLElement).className || ''} « ${el.textContent?.trim().slice(0, 30)} »`;
    const scrollsSideways = (el: Element) => ['auto', 'scroll'].includes(getComputedStyle(el).overflowX);
    for (const el of root.querySelectorAll<HTMLElement>('*')) {
      if (el.closest('[data-testid="dv-mountain"], [aria-hidden="true"]')) continue;
      if (!el.checkVisibility({ visibilityProperty: true, opacityProperty: true })) continue;
      // Le bandeau des retenus défile au doigt, par conception : son contenu peut dépasser.
      if (el.closest('[data-testid="dv-kept"]')) continue;
      if (scrollsSideways(el) && el.scrollWidth > el.clientWidth + 1) out.push(`défile de côté : ${name(el)} ${el.scrollWidth} > ${el.clientWidth}`);
      const r = el.getBoundingClientRect();
      if (r.width > 0 && (r.right > W + 1 || r.left < -1)) out.push(`hors écran : ${name(el)} [${Math.round(r.left)}, ${Math.round(r.right)}]`);
    }
    return out.slice(0, 12);
  });
}

/** Invariants d'un moment du parcours : rien ne déborde, toutes les cibles sont tapables. */
async function expectPhoneFriendly(page: Page, moment: string) {
  expect(await horizontalOverflow(page), `débordement — ${moment}`).toEqual([]);
  expect(await smallTargets(page), `cibles < 44 px — ${moment}`).toEqual([]);
}

/**
 * Glissement tactile vertical (doigt de bas en haut = contenu qui monte) depuis le point (x, y) :
 * un VRAI doigt — touchStart, dix touchMove, touchEnd — envoyé dans la file d'entrée du navigateur.
 * POURQUOI pas `Input.synthesizeScrollGesture` en mode « touch » : mesuré, il ne défile rien en
 * headless (0 px), alors que la même distance au doigt défile ; et pas `mouse.wheel`, qui n'est
 * pas un geste de téléphone.
 */
async function swipeUp(page: Page, x: number, y: number, distance: number) {
  const cdp = await page.context().newCDPSession(page);
  const at = (yy: number) => [{ x, y: Math.round(yy) }];
  await cdp.send('Input.dispatchTouchEvent', { type: 'touchStart', touchPoints: at(y) });
  for (let i = 1; i <= 10; i++) await cdp.send('Input.dispatchTouchEvent', { type: 'touchMove', touchPoints: at(y - (distance * i) / 10) });
  await cdp.send('Input.dispatchTouchEvent', { type: 'touchEnd', touchPoints: [] });
  await cdp.detach();
}

test.describe('dossier vivant — téléphone', () => {
  test.setTimeout(120_000);
  // Gestes tactiles synthétisés par CDP : Chromium seulement, et c'est la version téléphone.
  test.skip(({ isMobile, browserName }) => !isMobile || browserName !== 'chromium', 'version téléphone, Chromium tactile');

  test('au doigt : lancer, regarder l’équipe, défiler le dossier, taper « Télécharger » → un vrai PDF', async ({ page }) => {
    const duplicates = trackDuplicateKeys(page);
    await page.goto('/fr/dossier-vivant?speed=4');
    const root = page.getByTestId('dv-root');
    await expect(root).toHaveAttribute('data-layout', 'compact');
    await expect(page.getByTestId('dv-need')).toHaveValue(/1 400 tickets/, { timeout: 15_000 });

    // 1. L'ouverture tient dans la largeur : titre, champ, bouton pleine largeur.
    const launch = page.getByTestId('dv-launch');
    await expect(launch).toBeInViewport();
    await expectPhoneFriendly(page, 'ouverture');

    // 2. Un TAP sur « Voir la démo » lance le rejeu : la montagne en haut, l'orchestrateur et ses
    //    quatre agents dessous (le run réel au doigt : test « run réel » plus bas).
    await page.getByTestId('dv-pause').tap();
    await expect(root).toHaveAttribute('data-run', 'running');
    await expect(page.getByTestId('dv-visitor-tile')).toBeVisible({ timeout: 10_000 });
    await expect(page.getByTestId('dv-orchestrator')).toBeInViewport();
    const columns = page.getByTestId('dv-agent-column');
    await expect(columns).toHaveCount(4);
    for (const agent of ['reader', 'researcher', 'architect', 'writer']) {
      const col = page.locator(`[data-testid="dv-agent-column"][data-agent="${agent}"]`);
      await expect(col).toHaveAttribute('data-status', /working|done/, { timeout: 15_000 });
      await expect(col).toBeInViewport();
    }

    // 3. Les projets s'allument : la tuile passe à l'or ET son nom entre dans le bandeau des retenus
    //    (sur téléphone, les noms sous les tuiles seraient illisibles).
    await expect(page.locator('[data-testid="dv-tile"][data-project-id="melodicode"], [data-testid="dv-card"][data-project-id="melodicode"]')).toHaveAttribute('data-state', 'lit', {
      timeout: 15_000,
    });
    const chip = page.locator('[data-testid="dv-kept-chip"][data-kept-id="melodicode"]');
    await expect(chip).toBeVisible();
    await expect(page.getByTestId('dv-kept')).toHaveAttribute('aria-label', FR.dossierVivant.mountain.kept);
    await expectPhoneFriendly(page, 'équipe au travail');

    // 4. La remise : le dossier prend tout l'écran, le bouton de téléchargement en vue.
    await expect(root).toHaveAttribute('data-run', 'completed', { timeout: 60_000 });
    const dossier = page.getByTestId('dv-dossier');
    await expect(dossier).toHaveAttribute('data-ready', '');
    await expect(page.getByTestId('dv-skills')).toHaveAttribute('data-status', 'ready', { timeout: 20_000 });
    // Plein écran : le haut de la feuille sous la barre, sa largeur celle de l'écran (aux marges près).
    await expect.poll(async () => (await dossier.boundingBox())!.y, { timeout: 5_000 }).toBeLessThan(120);
    const viewport = page.viewportSize()!;
    expect((await dossier.boundingBox())!.width).toBeGreaterThan(viewport.width - 40);
    const download = page.getByTestId('dv-download');
    await expect(download).toBeInViewport();
    await expectPhoneFriendly(page, 'dossier remis');

    // 5. Le dossier se lit en DÉFILANT LA PAGE, au doigt (glissé sur la feuille), jusqu'à la
    //    prochaine étape et son contact. La feuille n'a plus d'ascenseur interne.
    const body = dossier.locator('[class*="dosBody"]');
    const contact = page.getByTestId('dv-contact');
    await expect(contact).not.toBeInViewport();
    for (let i = 0; i < 12 && !(await contact.evaluate((el) => { const r = el.getBoundingClientRect(); return r.bottom <= innerHeight && r.top >= 0; })); i++) {
      await swipeUp(page, viewport.width / 2, viewport.height * 0.75, Math.round(viewport.height * 0.5));
    }
    expect(await page.evaluate(() => scrollY)).toBeGreaterThan(0);
    expect(await body.evaluate((el) => el.scrollTop)).toBe(0);
    await expect(contact).toBeInViewport();
    // Les preuves, repliées en fiches lisibles : chiffres précédés de leur en-tête, rien de coupé.
    await expect(page.getByTestId('dv-proof-row')).toHaveCount(4);
    await expectPhoneFriendly(page, 'dossier défilé');

    // 6. Retour en tête de la feuille au doigt, puis un TAP sur « Télécharger le dossier » remet un vrai PDF.
    for (let i = 0; i < 12 && !(await download.evaluate((el) => el.getBoundingClientRect().top >= 0)); i++) {
      await swipeUp(page, viewport.width / 2, viewport.height * 0.3, -Math.round(viewport.height * 0.5));
    }
    await expect(download).toBeInViewport();
    await expect(download).toHaveText(FR.dossierVivant.dossier.download);
    const [file] = await Promise.all([page.waitForEvent('download', { timeout: 60_000 }), download.tap()]);
    expect(file.suggestedFilename()).toBe('dossier-de-conception-tessera.pdf');
    const bytes = readFileSync((await file.path())!);
    expect(bytes.length).toBeGreaterThan(5_000);
    expect(bytes.subarray(0, 4).toString('latin1')).toBe('%PDF');
    await expect(download).toHaveAttribute('data-pdf', 'idle');

    expect(duplicates, duplicates.join('\n')).toEqual([]);
  });

  test('pause et rejouer : des boutons-icônes de 44 px qui gardent leur nom', async ({ page }) => {
    await page.goto('/fr/dossier-vivant?speed=2');
    const pause = page.getByTestId('dv-pause');
    // Avant le départ, le bouton de lecture lance la démo — et le dit, même masqué à l'œil.
    await expect(pause).toHaveAccessibleName(FR.dossierVivant.bar.demo);
    await pause.tap();
    await expect(pause).toBeEnabled({ timeout: 10_000 });
    // Le libellé reste le nom accessible du bouton, même masqué à l'œil.
    await expect(pause).toHaveAccessibleName(FR.dossierVivant.bar.pause);
    await pause.tap();
    await expect(pause).toHaveAccessibleName(FR.dossierVivant.bar.play);
    const before = await page.getByTestId('dv-totals').textContent();
    await page.waitForTimeout(1200);
    expect(await page.getByTestId('dv-totals').textContent()).toBe(before);
    await expect(page.getByTestId('dv-replay')).toHaveAccessibleName(FR.dossierVivant.bar.replay);
    await expectPhoneFriendly(page, 'en pause');
  });

  test('run réel (flux brut enregistré, six agents) : cinq colonnes tiennent, dossier remis, aucune clé en double', async ({ page }) => {
    const duplicates = trackDuplicateKeys(page);
    await mockLiveRun(page, recordedRun());
    await page.goto('/fr/dossier-vivant');
    await expect(page.getByTestId('dv-need')).toHaveAttribute('data-editable', '', { timeout: 15_000 });
    await expectPhoneFriendly(page, 'ouverture');
    await page.getByTestId('dv-launch').tap();
    const root = page.getByTestId('dv-root');
    await expect(root).toHaveAttribute('data-mode', 'live');
    await expect(page.getByTestId('dv-agent-column')).toHaveCount(5);
    await expect(root).toHaveAttribute('data-run', 'completed', { timeout: 30_000 });
    await expect(page.getByTestId('dv-download')).toBeInViewport();
    await expectPhoneFriendly(page, 'run réel remis');
    expect(duplicates, duplicates.join('\n')).toEqual([]);
  });
});

// Pixel 5 couché, sans `defaultBrowserType` (interdit dans un `test.use` de groupe).
const { viewport, userAgent, deviceScaleFactor, isMobile, hasTouch } = devices['Pixel 5 landscape'];
const PIXEL5_LANDSCAPE = { viewport, userAgent, deviceScaleFactor, isMobile, hasTouch };

test.describe('dossier vivant — téléphone couché', () => {
  test.setTimeout(120_000);
  test.use(PIXEL5_LANDSCAPE);
  // Un seul projet le joue : le `test.use` ci-dessus rendrait sinon « mobile » le projet chromium aussi.
  test.beforeEach(({ browserName }, testInfo) => {
    test.skip(testInfo.project.name !== 'Mobile Chrome' || browserName !== 'chromium', 'téléphone couché : projet « Mobile Chrome »');
  });

  test('couché : ouverture sur deux colonnes, équipe à côté de la montagne, tuiles allumées identifiables, dossier lu au doigt → un vrai PDF', async ({ page }) => {
    const duplicates = trackDuplicateKeys(page);
    await page.goto('/fr/dossier-vivant?speed=4');
    const root = page.getByTestId('dv-root');
    await expect(root).toHaveAttribute('data-layout', 'landscape');
    await expect(page.getByTestId('dv-need')).toHaveValue(/1 400 tickets/, { timeout: 15_000 });

    // 1. L'ouverture tient dans ~240 px de haut : champ, présentation et bouton en vue.
    for (const id of ['dv-need', 'dv-who', 'dv-launch']) await expect(page.getByTestId(id)).toBeInViewport({ ratio: 1 });
    await expectPhoneFriendly(page, 'couché — ouverture');

    // 2. Un TAP sur « Voir la démo » : la montagne à gauche, l'orchestrateur et ses colonnes à droite, en vue.
    await page.getByTestId('dv-pause').tap();
    await expect(root).toHaveAttribute('data-run', 'running');
    await expect(page.getByTestId('dv-visitor-tile')).toBeVisible({ timeout: 10_000 });
    const mountain = (await page.getByTestId('dv-mountain').boundingBox())!;
    const team = page.getByTestId('dv-orchestrator');
    await expect(team).toBeInViewport({ ratio: 1 });
    expect((await team.boundingBox())!.x).toBeGreaterThanOrEqual(mountain.x + mountain.width - 1);
    const columns = page.getByTestId('dv-agent-column');
    await expect(columns).toHaveCount(4);
    for (const agent of ['reader', 'researcher', 'architect', 'writer']) {
      const col = page.locator(`[data-testid="dv-agent-column"][data-agent="${agent}"]`);
      await expect(col).toHaveAttribute('data-status', /working|done/, { timeout: 15_000 });
      await expect(col).toBeInViewport();
    }

    // 3. Projets allumés IDENTIFIABLES : la tuile grandit à ~22 px à l'écran et porte sa marque
    //    (icône du dépôt ou monogramme) ; sa pastille entre dans le bandeau des retenus.
    const tile = page.locator('[data-testid="dv-tile"][data-project-id="openclaw_platform"]');
    await expect(tile).toHaveAttribute('data-state', 'lit', { timeout: 15_000 });
    const box = (await tile.boundingBox())!;
    expect(box.width).toBeGreaterThanOrEqual(18);
    await expect(tile.locator('[class*="mono"]')).toBeVisible();
    await expect(tile.locator('[class*="mono"]')).toHaveText('OP');
    await expect(page.locator('[data-testid="dv-kept-chip"][data-kept-id="openclaw_platform"]')).toBeVisible();
    await expectPhoneFriendly(page, 'couché — équipe au travail');

    // 4. La remise : le dossier prend toute la scène, le bouton de téléchargement en vue.
    await expect(root).toHaveAttribute('data-run', 'completed', { timeout: 60_000 });
    const dossier = page.getByTestId('dv-dossier');
    await expect(dossier).toHaveAttribute('data-ready', '');
    await expect(page.getByTestId('dv-skills')).toHaveAttribute('data-status', 'ready', { timeout: 20_000 });
    const viewport = page.viewportSize()!;
    await expect.poll(async () => (await dossier.boundingBox())!.width, { timeout: 5_000 }).toBeGreaterThan(viewport.width - 40);
    const download = page.getByTestId('dv-download');
    await expect(download).toBeInViewport({ ratio: 1 });
    await expectPhoneFriendly(page, 'couché — dossier remis');

    // 5. Lu en DÉFILANT LA PAGE au doigt jusqu'au contact ; les preuves sont là, rien ne déborde.
    const body = dossier.locator('[class*="dosBody"]');
    const bodyBox = (await body.boundingBox())!;
    expect(bodyBox.height, 'il reste de quoi lire sous l’en-tête du dossier').toBeGreaterThan(100);
    const contact = page.getByTestId('dv-contact');
    for (let i = 0; i < 30 && !(await contact.evaluate((el) => { const r = el.getBoundingClientRect(); return r.bottom <= innerHeight && r.top >= 0; })); i++) {
      await swipeUp(page, viewport.width / 2, viewport.height * 0.8, Math.round(viewport.height * 0.6));
    }
    expect(await page.evaluate(() => scrollY)).toBeGreaterThan(0);
    await expect(contact).toBeInViewport();
    await expect(page.getByTestId('dv-proof-row')).toHaveCount(4);
    await expectPhoneFriendly(page, 'couché — dossier défilé');

    // 6. Retour en tête au doigt, puis TAP sur « Télécharger le dossier » → un vrai PDF.
    for (let i = 0; i < 30 && !(await download.evaluate((el) => el.getBoundingClientRect().top >= 0)); i++) {
      await swipeUp(page, viewport.width / 2, viewport.height * 0.2, -Math.round(viewport.height * 0.6));
    }
    await expect(download).toBeInViewport();
    const [file] = await Promise.all([page.waitForEvent('download', { timeout: 60_000 }), download.tap()]);
    expect(file.suggestedFilename()).toBe('dossier-de-conception-tessera.pdf');
    const bytes = readFileSync((await file.path())!);
    expect(bytes.subarray(0, 4).toString('latin1')).toBe('%PDF');

    expect(duplicates, duplicates.join('//n')).toEqual([]);
  });

  test('couché : pause et rejouer restent des boutons de 44 px, barre sur une ligne', async ({ page }) => {
    await page.goto('/fr/dossier-vivant?speed=2');
    const pause = page.getByTestId('dv-pause');
    // Avant le départ, le bouton de lecture lance la démo — et le dit, même masqué à l'œil.
    await expect(pause).toHaveAccessibleName(FR.dossierVivant.bar.demo);
    await pause.tap();
    await expect(pause).toBeEnabled({ timeout: 10_000 });
    await expect(pause).toHaveAccessibleName(FR.dossierVivant.bar.pause);
    // Une ligne : marque, mention et boutons partagent la même bande horizontale.
    const caption = (await page.getByTestId('dv-caption').boundingBox())!;
    const p = (await pause.boundingBox())!;
    expect(Math.abs(caption.y + caption.height / 2 - (p.y + p.height / 2))).toBeLessThan(6);
    await expectPhoneFriendly(page, 'couché — barre');
  });
});
