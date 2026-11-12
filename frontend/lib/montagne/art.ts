/**
 * art.ts — les illustrations d'AMBIANCE de la page /montagne : le fond (montagne nocturne dorée)
 * et la couverture de chaque carte du sommet.
 *
 * QUOI : des chemins de fichiers de `public/montagne/art/`, produits par
 * `scripts/montagne-art.mjs` depuis les PNG d'Eikon (webp, budgets de poids vérifiés au passage).
 * POURQUOI ces images ne sont PAS des preuves : ce sont des illustrations. Les preuves (captures
 * des jeux, démo vidéo, pages du site) viennent du backend (montagne_sommet.go) et s'ouvrent dans
 * la fleur de la carte — la couverture donne envie, la fleur montre ce qui existe.
 * COMMENT : le nom de fichier se déduit de l'id de la carte (`cover-<id>.webp`) ; seules les
 * cartes listées ici ont une illustration. Une carte que le backend ajouterait au sommet sans
 * illustration garde son icône en grand (MontagnePeak) — jamais un cadre vide. Un test
 * (art.test.ts) vérifie que chaque fichier existe, en webp, dans son budget.
 */

/** Cartes du sommet illustrées — mêmes ids que montagne_sommet.go. */
export const SUMMIT_ART_IDS = ['grove_engine', 'melodicode', 'theomen', 'maicivy'] as const;

/** Couverture illustrée d'une carte du sommet, ou `null` si la carte n'en a pas. */
export function summitCover(id: string): string | null {
  return (SUMMIT_ART_IDS as readonly string[]).includes(id) ? `/montagne/art/cover-${id}.webp` : null;
}

/**
 * Le fond, en deux cadrages : portrait sous 768 px (le seuil de la géométrie téléphone de
 * pyramid.ts), paysage au-delà.
 */
export const BACKDROP = {
  mobile: '/montagne/art/fond-mobile.webp',
  desktop: '/montagne/art/fond-desktop.webp',
  /** Media query du cadrage paysage — le même seuil que montagne.module.css. */
  desktopMedia: '(min-width: 768px)',
} as const;
