import { test, expect } from '@playwright/test';
import adminFr from '../../messages/fr.json';

// Verrou de non-retour : le HTML servi sur les pages PUBLIQUES ne doit contenir aucun libellé
// du namespace `admin`.
//
// L'incident (26/08/2026) : `app/[locale]/layout.tsx` passait l'objet `messages` ENTIER à
// NextIntlClientProvider, qui le sérialise dans le HTML de chaque page. Les libellés du panneau
// owner (titres des sept outils, aides de saisie, « Mot de passe invalide ») étaient donc lisibles
// en view-source sur maicivy.etheryale.com, et indexables. Pas une fuite de données : une fuite de
// produit, l'outil interne se décrivant lui-même à qui passait. Le namespace pèse ~20 % du fichier
// de messages et aucun composant public ne le lit : il partait pour rien.
//
// Le correctif est une destructuration dans le layout, donc trivial à annuler par inadvertance au
// prochain refactor. D'où ce test : il échoue si `admin` revient dans le HTML public, quelle que
// soit la façon dont il y revient (provider, prefetch RSC, dump de config).
//
// Choix d'implémentation : on ne code pas en dur des phrases (elles changent), on LIT les valeurs
// depuis messages/fr.json et on cherche les plus distinctives. Le test suit donc les traductions.
//
// Tourne contre la prod : PLAYWRIGHT_TEST_BASE_URL=https://maicivy.etheryale.com

/** Aplatit un objet de messages imbriqué en la liste de ses chaînes feuilles. */
function chainesFeuilles(valeur: unknown): string[] {
  if (typeof valeur === 'string') return [valeur];
  if (valeur && typeof valeur === 'object') {
    return Object.values(valeur as Record<string, unknown>).flatMap(chainesFeuilles);
  }
  return [];
}

// Les chaînes du namespace admin, les plus longues d'abord : une phrase complète ne peut pas
// apparaître par hasard dans le HTML public, alors qu'un mot isolé (« Chat », « Stats ») est
// partagé avec le site public et produirait des faux positifs.
const LIBELLES_ADMIN = chainesFeuilles((adminFr as Record<string, unknown>).admin)
  .filter((s) => s.length >= 25)
  .sort((a, b) => b.length - a.length)
  .slice(0, 30);

const PAGES_PUBLIQUES = ['/fr', '/fr/cv', '/en'];

test.describe('Pages publiques — aucun libellé du panneau admin dans le HTML', () => {
  test('le namespace admin ne doit jamais être sérialisé côté public', async ({ page }) => {
    // Garde-fou sur le test lui-même : si le filtrage ne retient rien, le test passerait à vide.
    expect(LIBELLES_ADMIN.length).toBeGreaterThan(5);

    for (const chemin of PAGES_PUBLIQUES) {
      const reponse = await page.goto(chemin, { waitUntil: 'load' });
      expect(reponse?.status(), `${chemin} doit répondre 200`).toBe(200);

      const html = await page.content();
      const fuites = LIBELLES_ADMIN.filter((libelle) => html.includes(libelle));

      expect(
        fuites,
        `${chemin} expose ${fuites.length} libellé(s) du namespace admin, ex. : ${fuites[0] ?? ''}`
      ).toEqual([]);
    }
  });
});
