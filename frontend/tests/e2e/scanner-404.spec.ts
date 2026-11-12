import { test, expect } from '@playwright/test';

/**
 * QUOI : vérifie que les chemins inexistants renvoient un VRAI HTTP 404 (et pas un soft-200).
 *
 * POURQUOI : un scanner (cf. attaque 34.165.149.147 du 2026-06-11) tape des chemins type
 *   /aws-credentials.json, /phpinfo.php, /.env. Ces chemins à SEGMENT UNIQUE contenant un point
 *   bypassent le middleware next-intl (matcher exclut `.*\..*`) et matchent le segment dynamique
 *   [locale] avec une locale invalide (ex: locale="aws-credentials.json"). Avant le fix, le layout
 *   rendait la homepage en HTTP 200 (fallback silencieux sur 'fr') → faux signal de succès pour le
 *   scanner + aucun incrément du score sus (qui ne bump que sur 4xx). On veut un 404 franc.
 *
 * COMMENT : on interroge directement le status HTTP via le contexte `request` de Playwright
 *   (pas de clic UI — c'est un comportement de status code, la preuve = le code réel renvoyé par
 *   l'app qui tourne). maxRedirects:0 pour observer le status brut sans suivre les redirects i18n.
 */
test.describe('Scanner paths → vrai 404 (anti-soft-200)', () => {
  // Chemins single-segment avec point : avant le fix ils sortaient 200 (homepage). Doivent être 404.
  const scannerPaths = [
    '/aws-credentials.json',
    '/phpinfo.php',
    '/db.sql',
    '/credentials.json',
    '/config.php',
  ];

  for (const path of scannerPaths) {
    test(`${path} doit renvoyer 404`, async ({ request }) => {
      const res = await request.get(path, { maxRedirects: 0 });
      expect(res.status()).toBe(404);
    });
  }

  // Régression inverse : les locales VALIDES doivent continuer à servir un 200.
  test('locale valide /fr → 200', async ({ request }) => {
    const res = await request.get('/fr', { maxRedirects: 0 });
    expect(res.status()).toBe(200);
  });

  // Régression : une vraie page existante sous une locale valide reste 200.
  test('page valide /fr/cv → 200', async ({ request }) => {
    const res = await request.get('/fr/cv', { maxRedirects: 0 });
    expect(res.status()).toBe(200);
  });

  // Régression : un chemin inexistant SOUS une locale valide doit aussi 404 (comportement déjà OK).
  test('chemin inconnu sous locale valide /fr/zzz-inexistant → 404', async ({ request }) => {
    const res = await request.get('/fr/zzz-inexistant', { maxRedirects: 0 });
    expect(res.status()).toBe(404);
  });
});
