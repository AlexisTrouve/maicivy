import { test, expect } from '@playwright/test';

/**
 * QUOI : un article ou un projet inexistant répond un VRAI statut HTTP 404 (pas 200 + noindex).
 *
 * POURQUOI : `[locale]/loading.tsx` enveloppait les pages dans un Suspense ; Next diffusait le
 *   shell (en-tête 200 déjà envoyé) avant que notFound() ne soit appelé → « Soft 404 » côté Google.
 *   Ce loading racine a été retiré le 2026-10-12 ; ce test empêche qu'un loading.tsx parent revienne.
 *
 * COMMENT : lecture directe du statut via `request` (pas de rendu navigateur), sans suivre les
 *   redirections i18n. Identifiants inventés, volontairement improbables. Tourne contre la prod :
 *   PLAYWRIGHT_TEST_BASE_URL=https://maicivy.etheryale.com
 */
test.describe('Ressource absente → vrai 404', () => {
  const chemins = [
    '/fr/blog/article-inexistant-e2e-404-zz9',
    '/fr/portfolio/projet-inexistant-e2e-404-zz9',
  ];

  for (const chemin of chemins) {
    test(`${chemin} renvoie 404`, async ({ request }) => {
      const res = await request.get(chemin, { maxRedirects: 0 });
      expect(res.status()).toBe(404);
    });
  }
});
