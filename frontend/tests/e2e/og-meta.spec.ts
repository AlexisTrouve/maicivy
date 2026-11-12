import { test, expect, type APIRequestContext } from '@playwright/test';

/**
 * Aperçu de lien et icônes — chaque page publique annonce une image d'aperçu qui existe vraiment.
 *
 * QUOI : pour chaque page du lot, en fr et en en, la balise `og:image` est présente, son URL répond
 *   200 en `image/png` de 1200 × 630 (lus dans l'en-tête PNG, pas déduits du type), et la carte
 *   Twitter/X est `summary_large_image` avec une `twitter:image`. Les trois icônes du site
 *   (`/icon.png`, `/apple-icon.png`, `/favicon.ico`) répondent 200.
 * POURQUOI : Next fusionne les métadonnées CLÉ PAR CLÉ, pas en profondeur. Une page qui déclare son
 *   propre `openGraph` sans `images` efface celle du layout — l'accueil partait ainsi SANS image
 *   dans Discord, Slack ou LinkedIn, sans la moindre erreur. Et une balise présente ne prouve rien
 *   si l'URL qu'elle désigne casse ou sort au mauvais format : on suit donc le lien jusqu'aux octets.
 * COMMENT : contre le front servi à PLAYWRIGHT_TEST_BASE_URL. L'URL de l'image est absolue (elle
 *   dérive de NEXT_PUBLIC_BASE_URL côté serveur) : en local, lancer le front avec
 *   NEXT_PUBLIC_BASE_URL=http://localhost:3000 pour tester les images locales et non celles de prod.
 */

const PAGES = ['', '/gitstats', '/portfolio', '/cv', '/analytics', '/architecture', '/dossier-vivant'];
const LOCALES = ['fr', 'en'];

/** Largeur et hauteur lues dans le bloc IHDR, qui suit toujours la signature PNG (octets 16–23). */
function pngSize(buf: Buffer): { width: number; height: number } {
  const signature = Buffer.from([0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a]);
  expect(buf.subarray(0, 8).equals(signature), 'signature PNG').toBe(true);
  return { width: buf.readUInt32BE(16), height: buf.readUInt32BE(20) };
}

async function expectPreviewImage(request: APIRequestContext, url: string) {
  const res = await request.get(url);
  expect(res.status(), `GET ${url}`).toBe(200);
  expect(res.headers()['content-type']).toContain('image/png');
  expect(pngSize(await res.body())).toEqual({ width: 1200, height: 630 });
}

for (const locale of LOCALES) {
  for (const path of PAGES) {
    const url = `/${locale}${path}`;
    test(`${url} — og:image 1200×630 et carte summary_large_image`, async ({ page, request }) => {
      // Les balises sont dans le <head> du rendu serveur : inutile d'attendre la 3D ou les données.
      await page.goto(url, { waitUntil: 'domcontentloaded' });

      // Assertion explicite sur la PRÉSENCE : un getAttribute sur une balise absente attendrait
      // jusqu'au délai du test et l'échec ne dirait pas ce qui manque.
      const og = page.locator('meta[property="og:image"]');
      await expect(og, `${url} n'a pas de og:image`).toHaveCount(1);
      await expectPreviewImage(request, (await og.getAttribute('content'))!);

      await expect(page.locator('meta[name="twitter:card"]')).toHaveAttribute('content', 'summary_large_image');
      const tw = page.locator('meta[name="twitter:image"]');
      await expect(tw, `${url} n'a pas de twitter:image`).toHaveCount(1);
      await expectPreviewImage(request, (await tw.getAttribute('content'))!);
    });
  }
}

test('icônes du site servies', async ({ request }) => {
  for (const icon of ['/icon.png', '/apple-icon.png', '/favicon.ico']) {
    const res = await request.get(icon);
    expect(res.status(), `GET ${icon}`).toBe(200);
  }
});
