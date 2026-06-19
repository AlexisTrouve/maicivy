import { test, expect } from '@playwright/test';

/**
 * E2E du sélecteur de fond pluggable (étape 1 : coquille + registry + constellation en plugin).
 *
 * QUOI : vérifie en cliquant RÉELLEMENT que le sélecteur existe, ouvre ses options, et que
 *   choisir un fond monte/démonte effectivement un <canvas> dans la coquille.
 * POURQUOI : doctrine — une UI sans E2E qui clique dessus n'est pas "fonctionnelle". Le rendu
 *   WebGL pixel par pixel n'est pas assertable, mais la présence du <canvas> et le câblage du
 *   sélecteur le sont.
 * COMMENT : on pilote via ?bg= pour des états déterministes sans attendre le defer initial de 3s,
 *   puis on teste le switch en live (immédiat, sans defer).
 */

const HOST = '[data-testid="background-host"]';

test.describe('Background switcher', () => {
  test('le sélecteur est visible et ouvre ses options', async ({ page }) => {
    await page.goto('/fr?bg=none'); // ?bg=none → aucun canvas, pas d'attente de defer

    const switcher = page.getByTestId('background-switcher');
    await expect(switcher).toBeVisible();

    await switcher.click();
    // Les 3 options de l'étape 1 : Aléatoire, Constellation, Aucun
    await expect(page.getByTestId('bg-option-random')).toBeVisible();
    await expect(page.getByTestId('bg-option-constellation')).toBeVisible();
    await expect(page.getByTestId('bg-option-none')).toBeVisible();
  });

  test('?bg=none ne rend aucun canvas', async ({ page }) => {
    await page.goto('/fr?bg=none');
    await page.waitForLoadState('networkidle');
    // Laisser passer le delai initial pour être sûr qu'aucun canvas ne monte.
    await page.waitForTimeout(3500);
    await expect(page.locator(`${HOST} canvas`)).toHaveCount(0);
  });

  test('choisir Constellation monte un canvas (switch immédiat, sans defer)', async ({ page }) => {
    await page.goto('/fr?bg=none');
    await page.getByTestId('background-switcher').click();
    await page.getByTestId('bg-option-constellation').click();

    // Switch user = init immédiat → un canvas WebGL doit apparaître rapidement dans la coquille.
    await expect(page.locator(`${HOST} canvas`)).toHaveCount(1, { timeout: 10000 });
  });

  test('switcher vers Aucun démonte le canvas et persiste en localStorage', async ({ page }) => {
    // Partir d'un état avec canvas (constellation forcée), puis basculer sur Aucun.
    await page.goto('/fr?bg=constellation');
    await expect(page.locator(`${HOST} canvas`)).toHaveCount(1, { timeout: 10000 });

    await page.getByTestId('background-switcher').click();
    await page.getByTestId('bg-option-none').click();

    // Le canvas doit être démonté (dispose + retrait du DOM).
    await expect(page.locator(`${HOST} canvas`)).toHaveCount(0, { timeout: 5000 });

    // La préférence explicite est persistée.
    const stored = await page.evaluate(() => localStorage.getItem('maicivy_bg'));
    expect(stored).toBe('none');
  });
});

/**
 * Régression reduced-motion : l'étape 1 avait introduit un bug où reduced-motion forçait 'none'
 * ET masquait le sélecteur → l'utilisateur ne voyait rien et n'avait aucun contrôle. Le bon
 * comportement : défaut calme (none) MAIS sélecteur visible + choix explicite honoré.
 */
test.describe('Background switcher — reduced motion', () => {
  test('défaut = aucun canvas, MAIS le sélecteur reste visible (agency préservée)', async ({
    page,
  }) => {
    await page.emulateMedia({ reducedMotion: 'reduce' });
    await page.goto('/fr'); // aucun ?bg → on prend le défaut
    await expect(page.getByTestId('background-switcher')).toBeVisible();
    await page.waitForTimeout(3500); // laisser passer le defer initial
    await expect(page.locator(`${HOST} canvas`)).toHaveCount(0);
  });

  test('choix explicite honoré sous reduced-motion (?bg=constellation rend le canvas)', async ({
    page,
  }) => {
    await page.emulateMedia({ reducedMotion: 'reduce' });
    await page.goto('/fr?bg=constellation');
    await expect(page.locator(`${HOST} canvas`)).toHaveCount(1, { timeout: 10000 });
  });
});

/**
 * ConwayGOL (plugin #2) — Game of Life en fond, grille cachée + glows désalignés.
 * On ne peut pas asserter le rendu WebGL/2D au pixel près sur le visuel, mais on peut prouver
 * que le canvas monte, que l'option existe, et que le plugin DESSINE réellement (pixels non vides).
 */
test.describe('ConwayGOL', () => {
  test('le sélecteur liste "Game of Life"', async ({ page }) => {
    await page.goto('/fr?bg=none');
    await page.getByTestId('background-switcher').click();
    await expect(page.getByTestId('bg-option-conwaygol')).toBeVisible();
  });

  test('?bg=conwaygol monte un canvas', async ({ page }) => {
    await page.goto('/fr?bg=conwaygol');
    await expect(page.locator(`${HOST} canvas`)).toHaveCount(1, { timeout: 10000 });
  });

  test('peindre à la souris → le canvas rend des glows (pixels non vides)', async ({ page }) => {
    await page.goto('/fr?bg=conwaygol');
    await expect(page.locator(`${HOST} canvas`)).toHaveCount(1, { timeout: 10000 });

    // Peindre : déplacer la souris en diagonale à travers le viewport (génère des pointermove).
    const vw = page.viewportSize();
    const w = vw?.width ?? 1280;
    const h = vw?.height ?? 720;
    for (let i = 1; i <= 8; i++) {
      await page.mouse.move((w * i) / 9, (h * i) / 9);
    }
    await page.waitForTimeout(800); // laisser le fade-in + quelques frames de rendu

    // Échantillonner le canvas : au moins un pixel avec alpha > 0 → le plugin dessine bien.
    const nonEmpty = await page.evaluate(() => {
      const cv = document.querySelector(
        '[data-testid="background-host"] canvas'
      ) as HTMLCanvasElement | null;
      if (!cv) return false;
      const cx = cv.getContext('2d');
      if (!cx) return false;
      const data = cx.getImageData(0, 0, cv.width, cv.height).data;
      for (let i = 3; i < data.length; i += 40) {
        if (data[i] > 0) return true; // canal alpha
      }
      return false;
    });
    expect(nonEmpty).toBe(true);
  });
});

/**
 * Fractal (plugin #3) — Mandelbrot↔Julia hybride en shader WebGL.
 * On vérifie l'option, le montage du canvas, et que le shader DESSINE (lecture pixels via
 * drawImage du canvas WebGL sur un canvas 2D — preserveDrawingBuffer:true le permet).
 */
test.describe('Fractal', () => {
  test('le sélecteur liste "Fractal"', async ({ page }) => {
    await page.goto('/fr?bg=none');
    await page.getByTestId('background-switcher').click();
    await expect(page.getByTestId('bg-option-fractal')).toBeVisible();
  });

  test('?bg=fractal monte un canvas', async ({ page }) => {
    await page.goto('/fr?bg=fractal');
    await expect(page.locator(`${HOST} canvas`)).toHaveCount(1, { timeout: 10000 });
  });

  test('le shader WebGL rend des pixels non vides', async ({ page }) => {
    await page.goto('/fr?bg=fractal');
    await expect(page.locator(`${HOST} canvas`)).toHaveCount(1, { timeout: 10000 });
    await page.waitForTimeout(1500); // sortir du fondu d'entrée + quelques frames

    const nonEmpty = await page.evaluate(() => {
      const cv = document.querySelector(
        '[data-testid="background-host"] canvas'
      ) as HTMLCanvasElement | null;
      if (!cv) return false;
      // Le canvas est WebGL → recopier sur un canvas 2D pour lire les pixels.
      const tmp = document.createElement('canvas');
      tmp.width = cv.width;
      tmp.height = cv.height;
      const t = tmp.getContext('2d');
      if (!t) return false;
      t.drawImage(cv, 0, 0);
      const data = t.getImageData(0, 0, tmp.width, tmp.height).data;
      for (let i = 3; i < data.length; i += 40) {
        if (data[i] > 0) return true;
      }
      return false;
    });
    expect(nonEmpty).toBe(true);
  });
});
