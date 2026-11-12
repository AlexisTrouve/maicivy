/*
 * palette.ts — les SEULES couleurs littérales des pages de contenu, toutes porteuses d'information.
 *
 * QUOI : la rampe de chaleur de la heatmap et deux couleurs d'état (en ligne / hors ligne).
 * POURQUOI ici et pas en variables de thème : ce ne sont pas des surfaces mais des DONNÉES — la
 *   chaleur d'un clic, l'état d'une connexion. Règle 1 du brief (exception « couleurs porteuses
 *   d'information ») : on les garde, mais accordées à la nuit de la montagne — désaturées, lisibles
 *   sur #04060b, et SANS bleu (le bleu est retiré du site public).
 * COMMENT : chaque valeur est relevée dans une palette déjà validée (dossier-vivant.module.css) ou
 *   construite sur la même teinte. Un état n'est jamais porté par la couleur seule : l'UI ajoute
 *   toujours le glyphe ✓ / ✕ (doctrine du panneau admin, reprise par le dossier vivant).
 */

/**
 * Rampe de chaleur, du froid au chaud — quatre paliers, utilisés par la carte ET par sa légende
 * (la légende doit montrer exactement les couleurs de la carte).
 *   0 : ardoise froide  — --dv-rule-strong éclairci (peu de clics : présent mais discret)
 *   1 : ocre            — teinte « works » du dossier vivant (oklch 62% 0.05 40), plus saturée
 *   2 : or              — #d9b45f, l'or de la montagne : ici une valeur de donnée, pas un signal
 *   3 : braise          — --dv-alert (oklch 74% 0.12 30) : le plus cliqué
 * Format [r, g, b] pour pouvoir moduler l'opacité selon l'intensité.
 */
export const RAMPE_CHALEUR: ReadonlyArray<readonly [number, number, number]> = [
  [118, 128, 146], // ardoise
  [176, 128, 84], // ocre
  [217, 180, 95], // or
  [237, 141, 125], // braise
];

/** rgba() d'un palier de la rampe à l'opacité donnée (bornée à [0, 1]). */
export function couleurChaleur(palier: number, opacite: number): string {
  const [r, g, b] = RAMPE_CHALEUR[Math.max(0, Math.min(RAMPE_CHALEUR.length - 1, palier))];
  return `rgba(${r}, ${g}, ${b}, ${Math.max(0, Math.min(1, opacite))})`;
}

/**
 * États — toujours doublés d'un glyphe.
 *   ok   : vert lichen, teinte « tooling » du dossier vivant (oklch 58% 0.03 135) relevée en
 *          luminance pour rester lisible en petit sur fond nuit (≈ oklch 76% 0.08 150).
 *   arret: --dv-alert, identique à --destructive du thème.
 */
export const ETAT = {
  ok: '#8fc39a',
  arret: '#ed8d7d',
} as const;
