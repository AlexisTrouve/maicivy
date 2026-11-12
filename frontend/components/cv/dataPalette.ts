// ============================================================================
// Teintes de DONNÉES des pages /cv et /gitstats — exception 1 du brief de refonte.
// ----------------------------------------------------------------------------
// QUOI : les seules couleurs littérales de ces pages. Elles portent une information (une série de
// graphe, un langage dans la barre de répartition) : un jeton de thème ne suffit pas, il en faut
// plusieurs qui se distinguent entre eux.
//
// POURQUOI des hex et pas des variables CSS : recharts écrit ces valeurs dans des attributs SVG
// (`stroke`, `stopColor`), où `var(--…)` n'est pas résolu de façon fiable d'un navigateur à
// l'autre. Un hex y est lu partout.
//
// COMMENT elles sont choisies : ce sont les teintes de strates de la montagne
// (dossier-vivant.module.css › --dv-tint-*), mais ÉCLAIRCIES à ~72-75 % de luminance oklch et
// gardées peu saturées (chroma 0,06-0,10). Sur le fond nuit (#04060b) elles se lisent sans
// crier, et aucune n'est l'or (réservé au signal) ni le bleu vif de l'ancien site.
// Valeur oklch d'origine en commentaire : c'est elle qu'il faut retoucher, puis reconvertir.
// ============================================================================

export const DATA_TINTS = {
  /** oklch(93% 0.025 85) — l'ivoire du site : la série principale (total des commits). */
  ivory: '#f0e7d6',
  /** oklch(74% 0.06 245) — acier, strate « fondations ». */
  steel: '#8bafcf',
  /** oklch(75% 0.07 135) — sauge, strate « outillage » ; aussi les lignes AJOUTÉES. */
  sage: '#9bb88c',
  /** oklch(72% 0.07 320) — mauve, strate « moteurs ». */
  mauve: '#b997c0',
  /** oklch(74% 0.08 45) — cuivre, strate « œuvres » ; aussi la série GitLab. */
  copper: '#d69b81',
  /** oklch(80% 0.05 85) — sable, ivoire assourdi. */
  sand: '#cdbc9a',
  /** oklch(66% 0.025 250) — ardoise, la teinte la plus effacée. */
  slate: '#8794a1',
  /** oklch(72% 0.10 30) — terre cuite, proche de --destructive : les lignes SUPPRIMÉES. */
  terra: '#dc8c7f',
  /** oklch(52% 0.016 252) — gris effacé : les langages au-delà des six teintes (/gitstats). */
  faint: '#626a72',
  /** oklch(42% 0.018 255) — filet appuyé : l'agrégat « autres » de la bande des langages. */
  rule: '#474e57',
} as const;

/** Barre de répartition du code par langage : une teinte stable par RANG (pas par langage). */
export const LANGUAGE_TINTS = [
  DATA_TINTS.steel,
  DATA_TINTS.sage,
  DATA_TINTS.mauve,
  DATA_TINTS.copper,
  DATA_TINTS.sand,
  DATA_TINTS.slate,
] as const;
