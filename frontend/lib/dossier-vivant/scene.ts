/**
 * scene.ts — la géométrie du dossier vivant : la montagne (dans son repère) et la scène (à l'écran).
 * Pur calcul, aucun DOM : testé en isolation.
 *
 * QUOI :
 *  - `buildScene` : la montagne RÉELLE (GET /api/v1/montagne) posée par la même fonction que la page
 *    /montagne (pyramid.ts › layoutMountain, géométrie grand écran), plus une place sous le sommet
 *    pour la tuile du visiteur — les rangs « s'ouvrent » d'une hauteur de rang pour la recevoir.
 *  - `anchorOf`, `linkPath`, `threadPath` : où aboutissent filets, curseurs et pings, rangs ouverts
 *    ou non (`open` de 0 à 1).
 *  - `stageGeometry` : où se tient la montagne avant (centrée sous le texte) et après le lancement
 *    (à gauche, équipe à droite), à partir de la taille de la scène.
 * POURQUOI réutiliser pyramid.ts : la montagne du dossier vivant EST celle de /montagne — même
 * ordre, mêmes rangs, mêmes filets. Une seconde mise en page aurait dérivé de la première.
 */

import type { MontagneLink, MontagneNode, StratumId } from '@/lib/montagne-api';
import { GEOMETRIES, layoutMountain, type PlacedBox } from '@/lib/montagne/pyramid';

/**
 * Géométrie grand écran, sur PC comme sur téléphone : la montagne est la MÊME (rangs, filets, ordre)
 * et seule son échelle change. Sur téléphone elle tient entière dans le haut de l'écran ; les noms
 * des tuiles y deviennent illisibles, d'où le bandeau des projets retenus (stageGeometry › kept).
 */
export const G = GEOMETRIES[GEOMETRIES.length - 1];

/** Ce que la scène lit de la montagne : de quoi poser les tuiles et nommer les projets. */
export interface SceneNode {
  id: string;
  name: string;
  monogram: string;
  stratum: StratumId;
  /**
   * Icône du dépôt du projet, chemin de l'API (`/api/v1/project-icons/<id>?v=…`) — absente quand le
   * dépôt n'a pas d'`icon.png` : on dessine alors le monogramme (ProjectMark).
   */
  icon?: string;
}

export interface SceneInput {
  nodes: SceneNode[];
  links: Pick<MontagneLink, 'from' | 'to'>[];
  summit: { id: string; members: string[] }[];
}

export interface SceneLink {
  key: string;
  x1: number;
  y1: number;
  x2: number;
  y2: number;
  /** Les bouts qui sont des tuiles descendent avec les rangs quand ils s'ouvrent. */
  fromTile: boolean;
  toTile: boolean;
  /** Lien carte → carte : courbe sous les cartes (pyramid.ts), immobile. */
  curve: string | null;
}

export interface Scene {
  width: number;
  /** Hauteur du repère, rangs ouverts compris. */
  height: number;
  /** Hauteur à faire tenir à l'écran (sans la brume) : règle l'échelle de la montagne. */
  fitHeight: number;
  tilesTop: number;
  cards: PlacedBox[];
  tiles: PlacedBox[];
  links: SceneLink[];
  visitor: PlacedBox;
  /** Cadre du fond peint (même cadrage que /montagne : pointe sous les cartes). */
  art: { top: number; width: number; height: number };
  isCard: ReadonlySet<string>;
}

export function buildScene(input: SceneInput): Scene {
  const absorbed = new Map(input.summit.flatMap((c) => c.members.map((m) => [m, c.id] as const)));
  const summitIds = input.summit.map((c) => c.id);
  // pyramid.ts ne lit que id/stratum des nœuds : on lui passe les nœuds de la scène tels quels.
  const layout = layoutMountain({ nodes: input.nodes as unknown as MontagneNode[], links: input.links as MontagneLink[], summit: summitIds, absorbed }, G);
  const isCard = new Set(layout.cards.map((c) => c.id));
  const isTile = new Set(layout.tiles.map((t) => t.id));
  const holder = (id: string) => absorbed.get(id) ?? id;

  const links: SceneLink[] = layout.links.map((l) => {
    const a = holder(l.from);
    const b = holder(l.to);
    return {
      key: `${l.from}>${l.to}`,
      x1: l.x1,
      y1: l.y1,
      x2: l.x2,
      y2: l.y2,
      fromTile: isTile.has(a),
      toTile: isTile.has(b),
      curve: isCard.has(a) && isCard.has(b) ? l.d : null,
    };
  });

  const tilesTop = layout.peakBottom + G.peakGap;
  const height = layout.height + G.rowHeight;
  // Fond peint : pointe 18 px sous les cartes, pied au bas du repère (montagne.module.css › .backdrop).
  const apex = 0.31;
  const ratio = 1.5;
  const artH = (height - layout.peakBottom) / (1 - apex);

  return {
    width: layout.width,
    height,
    fitHeight: height - G.mistHeight + 6,
    tilesTop,
    cards: layout.cards,
    tiles: layout.tiles,
    links,
    visitor: { id: 'visitor', x: layout.width / 2 - G.tile / 2, y: tilesTop, w: G.tile, h: G.tile },
    art: { top: layout.peakBottom + 18 - apex * artH, width: artH * ratio, height: artH },
    isCard,
  };
}

/** Point d'ancrage d'un projet ou du visiteur : centre de tuile, ou milieu du bas d'une carte. */
export function anchorOf(scene: Scene, id: string, open = 1): [number, number] {
  if (id === 'visitor') return [scene.visitor.x + G.tile / 2, scene.visitor.y + G.tile / 2];
  const c = scene.cards.find((b) => b.id === id);
  if (c) return [c.x + c.w / 2, c.y + c.h];
  const t = scene.tiles.find((b) => b.id === id);
  if (!t) return [scene.width / 2, scene.tilesTop];
  return [t.x + t.w / 2, t.y + t.h / 2 + open * G.rowHeight];
}

/** Boîte d'un projet rangs ouverts (anneau de ping). */
export function boxOf(scene: Scene, id: string): PlacedBox | null {
  if (id === 'visitor') return scene.visitor;
  const c = scene.cards.find((b) => b.id === id);
  if (c) return c;
  const t = scene.tiles.find((b) => b.id === id);
  return t ? { ...t, y: t.y + G.rowHeight } : null;
}

/** Tracé d'un lien réel, rangs ouverts à `open`. */
export function linkPath(l: SceneLink, open: number): string {
  if (l.curve) return l.curve;
  const dy = open * G.rowHeight;
  return `M${l.x1},${l.y1 + (l.fromTile ? dy : 0)}L${l.x2},${l.y2 + (l.toTile ? dy : 0)}`;
}

/** Bout d'arrivée d'un lien réel (son point). */
export function linkEnd(l: SceneLink, open: number): [number, number] {
  return [l.x2, l.y2 + (l.toTile ? open * G.rowHeight : 0)];
}

/**
 * Filet du visiteur vers un projet retenu : une courbe qui s'écarte de l'axe vertical (du côté de
 * la cible, à gauche pour l'axe lui-même). Plusieurs projets retenus sont dans la colonne
 * centrale, sous le visiteur : un trait droit y passerait entièrement caché derrière les tuiles.
 */
export function threadPath(scene: Scene, id: string): string {
  const [x1, y1] = anchorOf(scene, 'visitor');
  const [x2, y2] = anchorOf(scene, id);
  const dx = x2 - x1;
  const dy = y2 - y1;
  const len = Math.hypot(dx, dy) || 1;
  const side = Math.abs(dx) < 4 ? -1 : Math.sign(dx);
  const bow = scene.isCard.has(id) ? 0 : Math.min(140, len * 0.22);
  const cx = (x1 + x2) / 2 + (side * bow * Math.abs(dy)) / len;
  const cy = (y1 + y2) / 2 - ((side * bow * dx) / len) * 0.3;
  return `M${x1},${y1}Q${cx.toFixed(1)},${cy.toFixed(1)} ${x2},${y2}`;
}

/** Rectangle en px de la scène. */
export interface Rect {
  x: number;
  y: number;
  w: number;
  h: number;
}

/**
 * Les trois dispositions de la scène :
 *  - `wide` : grand écran, montagne à gauche, équipe à droite ;
 *  - `compact` : téléphone en PORTRAIT, l'histoire lue de haut en bas ;
 *  - `landscape` : téléphone à l'HORIZONTALE — côte à côte comme le grand écran, mais avec la
 *    densité du téléphone (le grand écran y serait à l'étroit : ~240 px de haut sous la barre).
 */
export type StageLayout = 'wide' | 'compact' | 'landscape';

/** Placement de la scène à l'écran (px de la scène, sous la barre). */
export interface StageGeometry {
  pad: number;
  layout: StageLayout;
  /** Densité TÉLÉPHONE (portrait ou paysage) : colonnes resserrées, pas d'éventail, bandeau des retenus. */
  compact: boolean;
  team: Rect;
  /**
   * Boîte de l'équipe à la remise. Grand écran : la même (le dossier prend la colonne). Téléphone :
   * toute la scène — le dossier remis se lit en plein écran, en défilant, la montagne dessous.
   */
  full: Rect;
  /** Bandeau des projets retenus, sous la montagne (téléphone seulement : les noms des tuiles y sont illisibles). */
  kept: Rect | null;
  /** Montagne avant le lancement : centrée sous le texte d'ouverture. */
  intro: { s: number; x: number; y: number };
  /** Montagne pendant le travail : à gauche de l'équipe, entière (en haut, entière, sur téléphone). */
  work: { s: number; x: number; y: number };
}

/**
 * Largeur de fenêtre sous laquelle on passe à la disposition téléphone en portrait (layoutFor). Le
 * CSS bascule au même seuil (`@media (max-width: 759px), …` dans dossier-vivant.module.css).
 */
export const COMPACT_MAX_WIDTH = 759;

/**
 * Hauteur de FENÊTRE (viewport) sous laquelle un écran à l'horizontale est un téléphone couché.
 * Relevé : Pixel 5 couché 802×293, iPhone 12 couché 750×340, iPhone 13 Pro Max couché 832×380 ; un
 * portable de bureau a au moins ~650 px de haut. Le CSS bascule à la même mesure
 * (`(orientation: landscape) and (max-height: 500px)` dans dossier-vivant.module.css).
 */
export const LANDSCAPE_MAX_HEIGHT = 500;

/**
 * Disposition d'une fenêtre de `vw × vh` px — la même règle que les media queries du CSS, dans le
 * même ordre : un téléphone couché l'emporte sur le seuil de largeur (couché, un iPhone 12 fait
 * 750 px de large et passerait sinon en portrait, la montagne sur 120 px de haut).
 */
export function layoutFor(vw: number, vh: number): StageLayout {
  if (vw > vh && vh <= LANDSCAPE_MAX_HEIGHT) return 'landscape';
  return vw <= COMPACT_MAX_WIDTH ? 'compact' : 'wide';
}

/** Part de la hauteur de scène donnée à la montagne sur téléphone (le reste : l'équipe). */
const COMPACT_MOUNTAIN_SHARE = 0.36;
/** Hauteur du bandeau des projets retenus, écart compris (px). */
const COMPACT_KEPT_H = 36;

/**
 * Mêmes règles que la maquette (measure()) : équipe à droite sur ~44 % de la largeur (600 à 860 px),
 * montagne mise à l'échelle dans ce qui reste ; à l'ouverture, 62 % de la largeur sous le texte.
 * Sous `COMPACT_MAX_WIDTH` : disposition téléphone (compactGeometry).
 */
export function stageGeometry(
  W: number,
  H: number,
  heroBottom: number,
  scene: Pick<Scene, 'width' | 'fitHeight'>,
  layout: StageLayout = W <= COMPACT_MAX_WIDTH ? 'compact' : 'wide',
): StageGeometry {
  if (layout === 'compact') return compactGeometry(W, H, heroBottom, scene);
  if (layout === 'landscape') return landscapeGeometry(W, H, scene);
  const pad = Math.round(Math.min(24, Math.max(16, W * 0.0125)));
  const teamW = Math.round(Math.min(860, Math.max(600, W * 0.44)));
  const teamX = W - pad - teamW;
  const zoneW = teamX - 2 * pad;
  const zoneH = H - 2 * pad;
  const sW = Math.min(zoneW / scene.width, zoneH / scene.fitHeight);
  const sI = Math.min(1, (W * 0.62) / scene.width);
  const team = { x: teamX, y: pad, w: teamW, h: H - 2 * pad };
  return {
    pad,
    layout: 'wide',
    compact: false,
    team,
    full: team,
    kept: null,
    intro: { s: sI, x: (W - scene.width * sI) / 2, y: heroBottom + 34 },
    work: { s: sW, x: pad + (zoneW - scene.width * sW) / 2, y: pad + (zoneH - scene.fitHeight * sW) / 2 },
  };
}

/**
 * Téléphone (portrait) : l'histoire se lit de HAUT EN BAS au lieu de gauche à droite.
 * QUOI : la montagne entière en haut (~36 % de la hauteur), le bandeau des projets retenus dessous,
 * puis l'équipe sur toute la largeur ; à la remise, l'équipe — donc le dossier — prend toute la scène.
 * POURQUOI pas la mise en page grand écran rétrécie : côte à côte, chaque moitié n'aurait que
 * ~180 px, ni la montagne ni les colonnes ne seraient lisibles.
 */
function compactGeometry(W: number, H: number, heroBottom: number, scene: Pick<Scene, 'width' | 'fitHeight'>): StageGeometry {
  const pad = 12;
  const innerW = W - 2 * pad;
  const mtnH = Math.round(H * COMPACT_MOUNTAIN_SHARE);
  const sW = Math.min(innerW / scene.width, mtnH / scene.fitHeight);
  const sI = Math.min(1, innerW / scene.width);
  const keptY = pad + mtnH;
  const teamY = keptY + COMPACT_KEPT_H;
  return {
    pad,
    layout: 'compact',
    compact: true,
    team: { x: pad, y: teamY, w: innerW, h: H - pad - teamY },
    full: { x: pad, y: pad, w: innerW, h: H - 2 * pad },
    kept: { x: pad, y: keptY, w: innerW, h: COMPACT_KEPT_H - 8 },
    intro: { s: sI, x: (W - scene.width * sI) / 2, y: heroBottom + 24 },
    work: { s: sW, x: (W - scene.width * sW) / 2, y: pad + (mtnH - scene.fitHeight * sW) / 2 },
  };
}

/**
 * Pied de la montagne d'ouverture, en px de la scène : le plus bas du repère (rangs ouverts compris)
 * et du fond peint, à l'échelle et à la place d'ouverture.
 * POURQUOI : avant le lancement, la page s'allonge jusque-là (DossierVivant › pageH) — la montagne
 * se montre ENTIÈRE, on défile pour voir son pied, puis vient le pied de page. On ne la réduit pas
 * pour la faire tenir dans un écran.
 */
export function introFoot(scene: Pick<Scene, 'height' | 'art'>, intro: { s: number; y: number }): number {
  return intro.y + Math.max(scene.height, scene.art.top + scene.art.height) * intro.s;
}

/** Paysage : part de la largeur que la colonne de la montagne prend au moins, et au plus. */
const LANDSCAPE_MOUNTAIN_SHARE = { min: 0.26, max: 0.4 } as const;

/**
 * Téléphone couché : côte à côte, comme le grand écran — c'est la largeur qui abonde, pas la hauteur.
 * QUOI : à gauche, la montagne entière (mise à l'échelle sur la HAUTEUR) avec le bandeau des retenus
 * dessous ; à droite, l'équipe sur toute la hauteur ; à la remise, le dossier prend toute la scène
 * et se lit en défilant, comme en portrait.
 * POURQUOI pas la disposition portrait : sur ~240 px de haut, la montagne n'aurait que 85 px et
 * l'équipe moins de 130. Pourquoi pas le grand écran : son équipe veut 600 px de large et sa feuille
 * 340 px de haut — il ne restait que 200 px pour la montagne et des colonnes de 40 px.
 * Avant le lancement, la montagne attend SOUS la scène (le texte d'ouverture occupe toute la hauteur),
 * centrée : la page s'allonge pour la montrer (introFoot), on la voit en défilant. Elle monte à sa
 * place, à gauche, quand l'équipe se met au travail.
 */
function landscapeGeometry(W: number, H: number, scene: Pick<Scene, 'width' | 'fitHeight'>): StageGeometry {
  const pad = 10;
  const keptH = COMPACT_KEPT_H - 8;
  const zoneH = H - 2 * pad - COMPACT_KEPT_H;
  // La colonne épouse la montagne mise à l'échelle sur la hauteur, dans des bornes de largeur.
  const fitW = (scene.width * zoneH) / scene.fitHeight;
  const colW = Math.round(Math.min(W * LANDSCAPE_MOUNTAIN_SHARE.max, Math.max(W * LANDSCAPE_MOUNTAIN_SHARE.min, fitW)));
  const s = Math.min(colW / scene.width, zoneH / scene.fitHeight);
  const teamX = pad + colW + pad;
  const work = { s, x: pad + (colW - scene.width * s) / 2, y: pad + (zoneH - scene.fitHeight * s) / 2 };
  return {
    pad,
    layout: 'landscape',
    compact: true,
    team: { x: teamX, y: pad, w: W - pad - teamX, h: H - 2 * pad },
    full: { x: pad, y: pad, w: W - 2 * pad, h: H - 2 * pad },
    kept: { x: pad, y: H - pad - keptH, w: colW, h: keptH },
    intro: { s, x: (W - scene.width * s) / 2, y: H + 24 },
    work,
  };
}
