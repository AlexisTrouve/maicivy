/**
 * labels.ts — les NOMS de la montagne : lesquels se montrent, sur quelle largeur, en combien de lignes.
 * Pur calcul, aucun DOM : testé en isolation.
 *
 * POURQUOI : 71 noms sous des tuiles espacées de 84 unités ne tiennent pas. Mesuré sur le PC
 * 1920×1080 : noms sur trois lignes qui mordaient le rang du dessous, voisins qui se chevauchaient,
 * mots coupés en plein milieu (« Transcrip / tor »), filets d'or lisibles À TRAVERS le texte.
 * QUOI (le choix de design) : seuls les projets RETENUS (allumés) et le visiteur portent leur nom en
 * permanence, sur une plaque opaque — l'or désigne ce qui compte, le nom dit quoi. Les autres se
 * nomment au survol (même plaque, au-dessus de tout) et par `title`. Un nom affiché ne passe jamais
 * sous deux lignes et ne coupe jamais un mot : trop long, il finit par « … ».
 * COMMENT : un nom est centré sous sa tuile ; il ne peut déborder sur un VOISIN de rang que si ce
 * voisin est muet. La largeur offerte vient donc du plus proche voisin nommé du même rang
 * (`labelWidths`) ; le nombre de lignes, de la place verticale libre entre la tuile et le rang du
 * dessous à la taille de police courante (`labelLines`).
 */

import type { PlacedBox } from '@/lib/montagne/pyramid';

import { G } from './scene';

/** Mesures d'un nom, en unités du repère de la montagne (celles de pyramid.ts). */
export const LABEL = {
  /** Écart minimal entre deux plaques voisines. */
  gap: 8,
  /** Largeur maximale d'une plaque : deux cellules, moins l'écart. */
  maxWidth: 2 * G.cell - 8,
  /**
   * Largeur sous laquelle un nom ne passe PAS à la ligne (il finit par « … ») : sur 80 unités, un
   * seul mot long (« Orchestrator ») dépasserait la plaque et serait rogné en plein mot.
   */
  wrapMin: 120,
  /** Vide entre le bas de la tuile et la plaque. */
  top: 4,
  /** Rembourrage vertical de la plaque (haut + bas) et vide gardé au-dessus du rang suivant. */
  padY: 4,
  margin: 6,
  /** Interligne (multiple de la police) — repris par le CSS (.tileName). */
  lineHeight: 1.15,
} as const;

/**
 * Lignes que la plaque peut prendre sans toucher le rang du dessous, à la police `font` (unités du
 * repère) : deux sur le grand écran (~15 unités), une seule quand la montagne est petite à l'écran
 * et que la police, gardée lisible, grossit dans le repère.
 */
export function labelLines(font: number): 1 | 2 {
  const room = G.rowHeight - G.tile - LABEL.top - LABEL.padY - LABEL.margin;
  return room >= 2 * font * LABEL.lineHeight ? 2 : 1;
}

/**
 * Largeur offerte au nom de chaque tuile. Une tuile nommée (`shown`) s'arrête à mi-chemin du plus
 * proche voisin nommé de son rang — deux plaques centrées sur leurs tuiles ne peuvent alors pas se
 * toucher. Une tuile muette reçoit la largeur maximale : son nom n'apparaît qu'au survol, au-dessus
 * de tout, un seul à la fois.
 */
export function labelWidths(tiles: readonly PlacedBox[], shown: ReadonlySet<string>): Map<string, number> {
  const center = (b: PlacedBox) => b.x + b.w / 2;
  const named = tiles.filter((b) => shown.has(b.id));
  const widths = new Map<string, number>();
  for (const b of tiles) {
    if (!shown.has(b.id)) {
      widths.set(b.id, LABEL.maxWidth);
      continue;
    }
    const nearest = named.reduce((d, o) => (o.id !== b.id && o.y === b.y ? Math.min(d, Math.abs(center(o) - center(b))) : d), Infinity);
    widths.set(b.id, Math.min(LABEL.maxWidth, nearest - LABEL.gap));
  }
  return widths;
}

/** Lignes permises pour une plaque de `width` unités à la police `font` (voir LABEL.wrapMin). */
export function labelFit(width: number, font: number): 1 | 2 {
  return width >= LABEL.wrapMin ? labelLines(font) : 1;
}
