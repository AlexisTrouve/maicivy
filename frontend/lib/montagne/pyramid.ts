/**
 * pyramid.ts — la montagne « à plat » : où poser les cartes du sommet, les tuiles et les filets.
 * Pur calcul, aucun DOM : testable en isolation, exécuté côté SERVEUR (la vue est rendue en
 * HTML/SVG dans la réponse, sans JavaScript).
 *
 * QUOI :
 *  - en haut, les CARTES des œuvres du sommet (liste décidée côté backend, `summit`) ;
 *  - dessous, les TUILES de tous les autres projets, en rangs CENTRÉS qui s'élargissent vers le
 *    bas, des œuvres jusqu'aux fondations ;
 *  - les FILETS : un tracé par lien réel « sert à construire » dont les deux bouts sont à l'écran ;
 *    un projet ABSORBÉ par une carte (`absorbed`, cf. montagne_sommet.go › members) n'a pas de
 *    tuile : ses filets partent de la carte qui le porte, et un filet devenu une boucle (carte →
 *    elle-même) n'est pas tracé ;
 *  - PEAKBOTTOM, le bas des cartes : c'est là que l'illustration de fond (MontagneBackdrop) pose
 *    la pointe de sa montagne peinte. La roche n'est plus dessinée ici (v4) : un triangle calculé
 *    ne s'accorderait pas avec la montagne de l'illustration, il la doublerait.
 *
 * Pas de repère de strate écrit à côté des rangs : les rangs sont remplis à la suite (sinon la
 * strate des moteurs, deux projets, ferait un rang étroit au milieu de la pente) et deux strates
 * partagent souvent un rang — un repère y serait faux pour la moitié des tuiles. La strate se lit
 * à la teinte de la bordure des tuiles et dans la liste.
 *
 * POURQUOI des coordonnées en px calculées ici, et non une grille CSS : les filets SVG doivent
 * relier les centres exacts des tuiles. Une grille laissée au navigateur ne donnerait ces centres
 * qu'après mesure, côté client — plus de rendu serveur, et des filets qui sautent au chargement.
 * Ici chaque géométrie (téléphone, tablette, grand écran) est fixée en px : le serveur connaît
 * TOUTES les positions, et le CSS choisit la géométrie par media query (MontagnePeak).
 *
 * COMMENT :
 *  1. ORDRE (orderTiles, commun à toutes les géométries) : strate du haut vers le bas (œuvres,
 *     moteurs, outillage, fondations) ; dans une strate, le plus relié d'abord, puis l'id. Le DOM
 *     suit cet ordre : lecteur d'écran et clavier lisent la montagne de haut en bas.
 *  2. RANGS (rowSizes) : une pyramide (firstRow, firstRow + step, …) jusqu'à maxPerRow, puis des
 *     rangs pleins ; le reste éventuel forme un rang de plus. Les tailles sont ensuite TRIÉES : un
 *     rang n'est jamais plus étroit que celui du dessus, la forme reste une montagne. Sur téléphone
 *     maxPerRow est bas (tuiles de 44 px au moins) : la montagne s'allonge, elle ne rapetisse pas.
 *  3. DANS UN RANG : le plus prioritaire au centre, puis alternativement à gauche et à droite —
 *     les carrefours du réseau sont au milieu, leurs filets traversent moins la montagne.
 */

import type { MontagneLink, MontagneNode, StratumId } from '@/lib/montagne-api';

/** Une géométrie de rendu, en px CSS. */
export interface Geometry {
  /** Largeur d'une cellule de tuile : la tuile plus son écart horizontal. */
  cell: number;
  /** Côté de la tuile (carrée). Sur téléphone, ≥ 44 : la cible tactile minimale. */
  tile: number;
  /** Hauteur d'un rang : tuile, nom sur deux lignes, écart. */
  rowHeight: number;
  /** Tuiles au plus par rang. */
  maxPerRow: number;
  /** Tuiles du rang du haut, et élargissement d'un rang au suivant. */
  firstRow: number;
  step: number;
  /** Cartes du sommet : par rangée, taille, écart. */
  cardsPerRow: number;
  cardWidth: number;
  cardHeight: number;
  cardGap: number;
  /** Vide entre le bas des cartes et le premier rang de tuiles (les filets y montent). */
  peakGap: number;
  /** Brume sous le dernier rang. */
  mistHeight: number;
}

/**
 * Les trois géométries, de la plus étroite à la plus large. Leurs seuils (0, 768, 1280 px de
 * large) sont repris par les media queries de montagne.module.css — un changement ici s'y reporte.
 *  - téléphone : 6 tuiles de 46 px au plus par rang (348 px, tient dans un écran de 360 px) ;
 *  - tablette / petit portable : 9 tuiles de 52 px ;
 *  - grand écran : 13 tuiles de 58 px, la pyramide 5-7-9-11-13 porte les 71 tuiles en 7 rangs.
 */
export const GEOMETRIES: readonly Geometry[] = [
  {
    cell: 58,
    tile: 46,
    rowHeight: 84,
    maxPerRow: 6,
    firstRow: 2,
    step: 1,
    cardsPerRow: 2,
    cardWidth: 168,
    cardHeight: 150,
    cardGap: 12,
    peakGap: 64,
    mistHeight: 90,
  },
  {
    cell: 72,
    tile: 52,
    rowHeight: 96,
    maxPerRow: 9,
    firstRow: 3,
    step: 2,
    cardsPerRow: 4,
    cardWidth: 164,
    cardHeight: 150,
    cardGap: 14,
    peakGap: 72,
    mistHeight: 110,
  },
  {
    cell: 84,
    tile: 58,
    rowHeight: 108,
    maxPerRow: 13,
    firstRow: 5,
    step: 2,
    cardsPerRow: 4,
    cardWidth: 236,
    cardHeight: 196,
    cardGap: 22,
    peakGap: 84,
    mistHeight: 130,
  },
];

/** Ordre vertical des strates dans la vue : du sommet vers la base. */
const STRATA_TOP_DOWN: StratumId[] = ['works', 'engines', 'tooling', 'foundations'];

export interface PlacedBox {
  id: string;
  /** Coin haut-gauche et taille, px dans le repère de la montagne. */
  x: number;
  y: number;
  w: number;
  h: number;
}

export interface PlacedLink {
  from: string;
  to: string;
  /** Tracé SVG : segment droit, ou courbe sous les cartes quand le lien relie deux cartes. */
  d: string;
  x1: number;
  y1: number;
  x2: number;
  y2: number;
}

export interface MountainLayout {
  width: number;
  height: number;
  cards: PlacedBox[];
  tiles: PlacedBox[];
  links: PlacedLink[];
  /** Bas du bloc des cartes (px) : la pointe de la montagne peinte s'y pose (MontagneBackdrop). */
  peakBottom: number;
}

/** Degré (liens entrants + sortants) de chaque projet. */
function degrees(links: MontagneLink[]): Map<string, number> {
  const d = new Map<string, number>();
  for (const l of links) {
    d.set(l.from, (d.get(l.from) ?? 0) + 1);
    d.set(l.to, (d.get(l.to) ?? 0) + 1);
  }
  return d;
}

/** Étape 1 de l'en-tête : les projets hors sommet (cartes et projets absorbés), dans l'ordre de la montagne. */
export function orderTiles(nodes: MontagneNode[], links: MontagneLink[], summit: string[], absorbed: ReadonlyMap<string, string> = new Map()): MontagneNode[] {
  const top = new Set([...summit, ...absorbed.keys()]);
  const deg = degrees(links);
  const rank = (s: StratumId) => STRATA_TOP_DOWN.indexOf(s);
  return nodes
    .filter((n) => !top.has(n.id))
    .sort(
      (a, b) =>
        rank(a.stratum) - rank(b.stratum) ||
        (deg.get(b.id) ?? 0) - (deg.get(a.id) ?? 0) ||
        (a.id < b.id ? -1 : a.id > b.id ? 1 : 0),
    );
}

/** Étape 2 de l'en-tête : tailles des rangs, du haut vers le bas. Somme = n. */
export function rowSizes(n: number, g: Pick<Geometry, 'firstRow' | 'step' | 'maxPerRow'>): number[] {
  const sizes: number[] = [];
  let remaining = n;
  for (let s = g.firstRow; remaining > 0 && s < g.maxPerRow; s += g.step) {
    const take = Math.min(s, remaining);
    sizes.push(take);
    remaining -= take;
  }
  while (remaining > 0) {
    const take = Math.min(g.maxPerRow, remaining);
    sizes.push(take);
    remaining -= take;
  }
  return sizes.sort((a, b) => a - b);
}

/**
 * Étape 3 de l'en-tête : place d'arrivée (0 = gauche) du k-ième élément d'un rang de `size`
 * tuiles. k = 0 au centre, puis droite, gauche, droite… (centre-gauche pour un rang pair).
 */
export function centerOutSlot(k: number, size: number): number {
  const center = Math.floor((size - 1) / 2);
  const offset = Math.ceil(k / 2) * (k % 2 === 1 ? 1 : -1);
  return center + offset;
}

/** La montagne complète dans une géométrie. */
export function layoutMountain(
  input: {
    nodes: MontagneNode[];
    links: MontagneLink[];
    summit: string[];
    /** Projet absorbé → id de la carte qui le porte. */
    absorbed?: ReadonlyMap<string, string>;
  },
  g: Geometry,
): MountainLayout {
  const byId = new Map(input.nodes.map((n) => [n.id, n]));
  const summit = input.summit.filter((id) => byId.has(id));
  const absorbed = new Map([...(input.absorbed ?? new Map<string, string>())].filter(([, card]) => summit.includes(card)));
  const ordered = orderTiles(input.nodes, input.links, summit, absorbed);
  const sizes = rowSizes(ordered.length, g);

  const cardsBlockWidth = g.cardsPerRow * g.cardWidth + (g.cardsPerRow - 1) * g.cardGap;
  const width = Math.max(g.maxPerRow * g.cell, cardsBlockWidth);
  const mid = width / 2;

  // Cartes du sommet : rangées centrées (la dernière, incomplète, centrée elle aussi).
  const cards: PlacedBox[] = [];
  const cardRows = Math.ceil(summit.length / g.cardsPerRow);
  summit.forEach((id, i) => {
    const row = Math.floor(i / g.cardsPerRow);
    const inRow = Math.min(g.cardsPerRow, summit.length - row * g.cardsPerRow);
    const col = i - row * g.cardsPerRow;
    const rowWidth = inRow * g.cardWidth + (inRow - 1) * g.cardGap;
    cards.push({
      id,
      x: mid - rowWidth / 2 + col * (g.cardWidth + g.cardGap),
      y: row * (g.cardHeight + g.cardGap),
      w: g.cardWidth,
      h: g.cardHeight,
    });
  });
  const peakBottom = cardRows > 0 ? cardRows * g.cardHeight + (cardRows - 1) * g.cardGap : 0;
  const tilesTop = peakBottom + g.peakGap;

  // Tuiles : rang par rang, centre d'abord. La tuile est centrée dans sa cellule, en haut du rang
  // (le nom occupe le bas du rang).
  const tiles: PlacedBox[] = [];
  let cursor = 0;
  sizes.forEach((size, r) => {
    const y = tilesTop + r * g.rowHeight;
    const left = mid - (size * g.cell) / 2;
    for (let k = 0; k < size; k++) {
      const node = ordered[cursor + k];
      const slot = centerOutSlot(k, size);
      tiles.push({ id: node.id, x: left + slot * g.cell + (g.cell - g.tile) / 2, y, w: g.tile, h: g.tile });
    }
    cursor += size;
  });
  // Ordre du DOM = ordre de la montagne (orderTiles), pas ordre des places.
  const tileOrder = new Map(ordered.map((n, i) => [n.id, i]));
  tiles.sort((a, b) => tileOrder.get(a.id)! - tileOrder.get(b.id)!);

  // Filets : d'un centre de tuile au centre d'une tuile, ou au milieu du BAS d'une carte (le
  // filet monte sous la carte et s'y arrête, il ne traverse pas la capture). Entre deux cartes, un
  // segment longerait leurs bords et se confondrait avec eux : il plonge en courbe sous elles.
  const anchor = new Map<string, [number, number]>();
  for (const t of tiles) anchor.set(t.id, [t.x + t.w / 2, t.y + t.h / 2]);
  for (const c of cards) anchor.set(c.id, [c.x + c.w / 2, c.y + c.h]);
  const isCard = new Set(cards.map((c) => c.id));
  // Un projet absorbé se lit sur sa carte : ses filets en partent (ou y arrivent).
  const holder = (id: string) => absorbed.get(id) ?? id;
  const links: PlacedLink[] = [];
  for (const l of input.links) {
    if (holder(l.from) === holder(l.to)) continue; // boucle carte → elle-même : rien à tracer
    const a = anchor.get(holder(l.from));
    const b = anchor.get(holder(l.to));
    if (!a || !b) continue;
    const d =
      isCard.has(holder(l.from)) && isCard.has(holder(l.to))
        ? `M${a[0]},${a[1]}Q${(a[0] + b[0]) / 2},${Math.max(a[1], b[1]) + g.peakGap * 0.7} ${b[0]},${b[1]}`
        : `M${a[0]},${a[1]}L${b[0]},${b[1]}`;
    links.push({ from: l.from, to: l.to, d, x1: a[0], y1: a[1], x2: b[0], y2: b[1] });
  }

  const tilesBottom = tilesTop + sizes.length * g.rowHeight;
  const height = tilesBottom + g.mistHeight;

  return { width, height, cards, tiles, links, peakBottom };
}
