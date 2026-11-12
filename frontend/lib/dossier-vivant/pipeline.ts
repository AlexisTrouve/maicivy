/**
 * pipeline.ts — mise en page du schéma d'architecture du dossier (bloc `pipeline` du flux).
 * Pur calcul, aucun DOM : testé.
 *
 * QUOI : à partir de nœuds posés sur une GRILLE (colonne, rangée) et d'arêtes, rend les boîtes,
 * les tracés, les pointes de flèche, les pastilles numérotées et la bande du journal, en unités du
 * viewBox — puis l'ORDRE dans lequel les dessiner (« parts ») : le schéma se construit sous les
 * yeux, une étape après l'autre, comme dans la maquette. Rend aussi l'ÉCHELLE d'affichage.
 * POURQUOI une grille et pas des coordonnées : un agent sait dire « le trieur vient après le ticket,
 * l'escalade part en dessous » ; il ne sait pas placer des pixels. La grille est ce qu'un modèle
 * peut produire de façon fiable, et la mise en page reste au front.
 *
 * DEUX ORIENTATIONS, une seule règle de lisibilité. Une unité du viewBox vaut un pixel CSS à
 * l'échelle 1 : les libellés y font leur taille de texte courant (13,5 / 12 px). Le panneau ne
 * rétrécit donc jamais le schéma sous `minScale` :
 *  - HORIZONTALE (la maquette validée) : les colonnes de la grille vont de gauche à droite. Tenue
 *    tant qu'elle entre dans la largeur offerte à `minScale` ou plus ;
 *  - VERTICALE : sinon, la grille est TRANSPOSÉE — les colonnes deviennent des étages, de haut en
 *    bas, et les rangées des couloirs côte à côte. Un vrai agent produit des chaînes LONGUES (5, 7
 *    étapes) et peu de branches parallèles : en hauteur, la largeur ne dépend plus que du nombre
 *    de branches, et le schéma garde l'échelle 1 dans une demi-colonne. Mesuré sur le run réel du
 *    2026-10 (5 colonnes) : 742 unités de large, affichées à 0,54 dans 399 px — 7 px de texte.
 *  - si aucune n'atteint `minScale` (grille à la fois longue ET très large), on garde la plus
 *    grande des deux échelles.
 * COMMENT (horizontale) :
 *  - largeur d'une colonne = la plus large de ses boîtes, estimée sur la longueur des textes (pas de
 *    mesure DOM : le rendu serveur, le PDF et le test doivent tomber juste) ;
 *  - deux boîtes d'une même colonne sur des rangées voisines s'empileraient (pas de rangée 36 <
 *    hauteur 40, voulu pour le zigzag de la maquette) : la grille passe alors au pas d'empilement ;
 *  - arête vers une colonne plus à droite : droite sur une même rangée, courbe sinon ;
 *  - arête dans la même colonne : verticale, de bord à bord ;
 *  - arête vers une colonne à gauche (retour, escalade), selon la rangée de la cible :
 *      · plus bas que la source : descend sous la source, puis revient horizontalement vers le
 *        flanc droit de la cible ;
 *      · sur la MÊME rangée : passe AU-DESSUS de la rangée (monte du haut de la source, revient,
 *        redescend sur le haut de la cible). Le tracé « par le bas » remontait jusqu'au milieu de la
 *        source et revenait SUR la flèche aller — pointillés posés sur la pastille du garde-fou
 *        (run réel du 2026-11 : Validation → Brouillon, sur ③). Le schéma descend alors d'un
 *        couloir (`PIPE.stub`) pour faire sa place au retour ;
 *      · plus haut : part du flanc gauche de la source et monte sur le bas de la cible.
 * COMMENT (verticale) : les mêmes règles, tournées d'un quart de tour ; un retour vers un étage
 * supérieur passe par le couloir libre à droite des deux boîtes.
 */

import type { PipelineEdge, PipelineNode } from './events';

export const PIPE = {
  boxH: 40,
  rowStep: 36,
  stackStep: 52,
  colGap: 32,
  levelGap: 28,
  titleCh: 8.2,
  subCh: 6.4,
  padX: 22,
  minW: 62,
  journalGap: 12,
  journalH: 24,
  /** Écart entre une boîte et le couloir où passe un retour. */
  stub: 16,
  /** Marge du viewBox autour du schéma (traits et pastilles débordent d'un ou deux pixels). */
  margin: 2,
} as const;

/**
 * Le panneau du dossier : jamais sous 0,82 (titres ≥ 11 px, sous-titres ≥ 9,8 px). Plancher choisi
 * pour garder la maquette Tessera VALIDÉE telle quelle : elle s'affichait à 0,83 dans la
 * demi-colonne de 399 px (1920×1080). En dessous, la verticale prend le relais.
 */
export const SCREEN_FIT = { minScale: 0.82 } as const;

export type PipeOrientation = 'horizontal' | 'vertical';

export interface PipeBox {
  id: string;
  x: number;
  y: number;
  w: number;
  title: string;
  subtitle: string;
  human: boolean;
}

export interface PipeEdgeDraw {
  key: string;
  d: string;
  dashed: boolean;
  /** Pointe de flèche (triangle) déjà orientée. */
  head: string;
  /** Pastille numérotée au milieu de l'arête. */
  step?: { n: number; x: number; y: number };
}

export interface PipePart {
  boxes: PipeBox[];
  edges: PipeEdgeDraw[];
  journal: boolean;
  steps: boolean;
}

export interface PipelineLayout {
  orientation: PipeOrientation;
  width: number;
  height: number;
  journalY: number;
  /** Pixels CSS par unité du viewBox : 1 = taille nominale des libellés, jamais plus. */
  scale: number;
  boxes: Map<string, PipeBox>;
  /** Dans l'ordre de dessin. */
  parts: PipePart[];
}

/** Largeur offerte et échelle plancher ; sans elle, le schéma garde sa taille nominale (horizontale). */
export interface PipelineFit {
  maxWidth: number;
  minScale: number;
}

/** Largeur d'une boîte, estimée sur ses textes. */
function boxWidth(n: PipelineNode): number {
  return Math.max(PIPE.minW, Math.ceil(Math.max(n.title.length * PIPE.titleCh, n.subtitle.length * PIPE.subCh) + PIPE.padX));
}

/** Largeurs par piste (colonne ou couloir) et abscisses cumulées. */
function tracks(nodes: PipelineNode[], key: (n: PipelineNode) => number): { w: number[]; x: number[] } {
  const count = Math.max(...nodes.map(key)) + 1;
  const w = Array.from({ length: count }, (_, i) => Math.max(PIPE.minW, ...nodes.filter((n) => key(n) === i).map(boxWidth)));
  const x = w.map((_, i) => w.slice(0, i).reduce((a, b) => a + b, 0) + i * PIPE.colGap);
  return { w, x };
}

interface Geometry {
  width: number;
  boxesBottom: number;
  boxes: Map<string, PipeBox>;
  draw: (e: PipelineEdge) => PipeEdgeDraw;
}

function horizontal(nodes: PipelineNode[], edges: PipelineEdge[]): Geometry {
  const { w: colW, x: colX } = tracks(nodes, (n) => n.col);
  // Deux boîtes d'une même colonne sur des rangées voisines : le pas de 36 les ferait se chevaucher.
  const stacked = nodes.some((a) => nodes.some((b) => a.col === b.col && b.row === a.row + 1));
  const rowStep = stacked ? PIPE.stackStep : PIPE.rowStep;
  const rows = Math.max(...nodes.map((n) => n.row)) + 1;
  const node = new Map(nodes.map((n) => [n.id, n]));
  // Un retour sur la même rangée passe au-dessus d'elle : le schéma lui réserve un couloir en haut.
  const sameRowBack = (e: PipelineEdge) => node.get(e.to)!.col < node.get(e.from)!.col && node.get(e.to)!.row === node.get(e.from)!.row;
  const top = edges.some(sameRowBack) ? PIPE.stub : 0;
  const boxes = new Map<string, PipeBox>(
    nodes.map((n) => [n.id, { id: n.id, x: colX[n.col], y: top + n.row * rowStep, w: colW[n.col], title: n.title, subtitle: n.subtitle, human: !!n.human }]),
  );
  const colOf = new Map(nodes.map((n) => [n.id, n.col]));

  const draw = (e: PipelineEdge): PipeEdgeDraw => {
    const s = boxes.get(e.from)!;
    const t = boxes.get(e.to)!;
    const sCy = s.y + PIPE.boxH / 2;
    const tCy = t.y + PIPE.boxH / 2;
    const key = `${e.from}>${e.to}`;
    const dashed = !!e.dashed;
    const sc = colOf.get(e.from)!;
    const tc = colOf.get(e.to)!;
    if (tc > sc) {
      const sx = s.x + s.w;
      const tx = t.x;
      const sy = tCy === sCy ? sCy : sCy + (tCy < sCy ? -6 : 6);
      const mid = (sx + tx) / 2;
      const d = tCy === sCy ? `M${sx},${sy}H${tx - 6}` : `M${sx},${sy}C${mid},${sy} ${mid},${tCy} ${tx - 6},${tCy}`;
      return { key, d, dashed, head: `M${tx},${tCy}l-6,-3.5v7z`, step: e.step ? { n: e.step, x: mid, y: (sy + tCy) / 2 } : undefined };
    }
    if (tc === sc) {
      // Même colonne : verticale de bord à bord, au milieu de la boîte.
      const cx = s.x + s.w / 2;
      const down = t.y > s.y;
      const sy = down ? s.y + PIPE.boxH : s.y;
      const ty = down ? t.y : t.y + PIPE.boxH;
      return {
        key,
        d: `M${cx},${sy}V${down ? ty - 6 : ty + 6}`,
        dashed,
        head: down ? `M${cx},${ty}l-3.5,-6h7z` : `M${cx},${ty}l-3.5,6h7z`,
        step: e.step ? { n: e.step, x: cx, y: (sy + ty) / 2 } : undefined,
      };
    }
    const sx = s.x + s.w / 2;
    const tcx = t.x + t.w / 2;
    if (sameRowBack(e)) {
      // Retour sur la même rangée : par le couloir du haut, jamais sur la flèche aller.
      const lane = s.y - PIPE.stub;
      return {
        key,
        d: `M${sx},${s.y}V${lane}H${tcx}V${t.y - 6}`,
        dashed,
        head: `M${tcx},${t.y}l-3.5,-6h7z`,
        step: e.step ? { n: e.step, x: (sx + tcx) / 2, y: lane } : undefined,
      };
    }
    if (t.y < s.y) {
      // Retour vers une rangée plus haute : du flanc gauche de la source, puis sur le bas de la cible.
      const tb = t.y + PIPE.boxH;
      return { key, d: `M${s.x},${sCy}H${tcx}V${tb + 6}`, dashed, head: `M${tcx},${tb}l-3.5,6h7z`, step: e.step ? { n: e.step, x: (s.x + tcx) / 2, y: sCy } : undefined };
    }
    // Retour vers une rangée plus basse : sous la source, puis horizontal vers le flanc droit de la cible.
    const sy = s.y + PIPE.boxH;
    const tx = t.x + t.w;
    return { key, d: `M${sx},${sy}V${tCy}H${tx + 6}`, dashed, head: `M${tx},${tCy}l6,-3.5v7z`, step: e.step ? { n: e.step, x: sx, y: (sy + tCy) / 2 } : undefined };
  };

  return { width: colX[colX.length - 1] + colW[colW.length - 1], boxesBottom: top + (rows - 1) * rowStep + PIPE.boxH, boxes, draw };
}

function vertical(nodes: PipelineNode[], edges: PipelineEdge[]): Geometry {
  // Transposée : la rangée de la grille devient un couloir (abscisse), la colonne un étage.
  const { w: laneW, x: laneX } = tracks(nodes, (n) => n.row);
  const levelStep = PIPE.boxH + PIPE.levelGap;
  const levels = Math.max(...nodes.map((n) => n.col)) + 1;
  const boxes = new Map<string, PipeBox>(
    nodes.map((n) => [n.id, { id: n.id, x: laneX[n.row], y: n.col * levelStep, w: laneW[n.row], title: n.title, subtitle: n.subtitle, human: !!n.human }]),
  );
  const colOf = new Map(nodes.map((n) => [n.id, n.col]));
  const lanesRight = laneX[laneX.length - 1] + laneW[laneW.length - 1];

  const draw = (e: PipelineEdge): PipeEdgeDraw => {
    const s = boxes.get(e.from)!;
    const t = boxes.get(e.to)!;
    const key = `${e.from}>${e.to}`;
    const dashed = !!e.dashed;
    const sc = colOf.get(e.from)!;
    const tc = colOf.get(e.to)!;
    if (tc > sc) {
      // Vers un étage plus bas : du bord bas de la source au bord haut de la cible.
      const sx = s.x + s.w / 2;
      const sy = s.y + PIPE.boxH;
      const tx = t.x + t.w / 2;
      const ty = t.y;
      const mid = (sy + ty) / 2;
      const d = sx === tx ? `M${sx},${sy}V${ty - 6}` : `M${sx},${sy}C${sx},${mid} ${tx},${mid} ${tx},${ty - 6}`;
      return { key, d, dashed, head: `M${tx},${ty}l-3.5,-6h7z`, step: e.step ? { n: e.step, x: (sx + tx) / 2, y: mid } : undefined };
    }
    const sCy = s.y + PIPE.boxH / 2;
    const tCy = t.y + PIPE.boxH / 2;
    if (tc === sc) {
      // Même étage : horizontale de flanc à flanc.
      const right = t.x > s.x;
      const sx = right ? s.x + s.w : s.x;
      const tx = right ? t.x : t.x + t.w;
      return {
        key,
        d: `M${sx},${sCy}H${right ? tx - 6 : tx + 6}`,
        dashed,
        head: right ? `M${tx},${tCy}l-6,-3.5v7z` : `M${tx},${tCy}l6,-3.5v7z`,
        step: e.step ? { n: e.step, x: (sx + tx) / 2, y: sCy } : undefined,
      };
    }
    // Retour vers un étage supérieur : par le couloir libre à droite des deux boîtes.
    const cx = Math.max(s.x + s.w, t.x + t.w) + PIPE.stub;
    const tx = t.x + t.w;
    return {
      key,
      d: `M${s.x + s.w},${sCy}H${cx}V${tCy}H${tx + 6}`,
      dashed,
      head: `M${tx},${tCy}l6,-3.5v7z`,
      step: e.step ? { n: e.step, x: cx, y: (sCy + tCy) / 2 } : undefined,
    };
  };

  // Un retour déborde à droite du dernier couloir : la largeur lui fait sa place.
  const goesBack = edges.some((e) => colOf.get(e.to)! < colOf.get(e.from)!);
  return { width: lanesRight + (goesBack ? PIPE.stub + 2 : 0), boxesBottom: (levels - 1) * levelStep + PIPE.boxH, boxes, draw };
}

/** Assemble la mise en page d'une géométrie : ordre de dessin, journal, dimensions. */
function assemble(orientation: PipeOrientation, g: Geometry, nodes: PipelineNode[], edges: PipelineEdge[], scale: number): PipelineLayout {
  // Ordre de dessin : colonne par colonne, chaque boîte avec les arêtes qui y arrivent depuis une
  // boîte déjà dessinée ; puis les arêtes restantes (retours), le journal, et enfin les pastilles.
  // Même ordre dans les deux orientations : la construction raconte la chaîne, pas la page.
  const order = [...nodes].sort((a, b) => a.col - b.col || a.row - b.row);
  const drawn = new Set<string>();
  const used = new Set<PipelineEdge>();
  const parts: PipePart[] = order.map((n) => {
    const incoming = edges.filter((e) => e.to === n.id && drawn.has(e.from));
    incoming.forEach((e) => used.add(e));
    drawn.add(n.id);
    return { boxes: [g.boxes.get(n.id)!], edges: incoming.map(g.draw), journal: false, steps: false };
  });
  const rest = edges.filter((e) => !used.has(e));
  if (rest.length) parts.push({ boxes: [], edges: rest.map(g.draw), journal: false, steps: false });
  parts.push({ boxes: [], edges: [], journal: true, steps: false });
  if (edges.some((e) => e.step)) parts.push({ boxes: [], edges: [], journal: false, steps: true });

  const journalY = g.boxesBottom + PIPE.journalGap;
  return { orientation, width: g.width, height: journalY + PIPE.journalH, journalY, scale, boxes: g.boxes, parts };
}

/** Échelle à laquelle une largeur d'unités entre dans `maxWidth` (jamais au-dessus de 1). */
function fitScale(width: number, maxWidth: number): number {
  return Math.min(1, maxWidth / (width + 2 * PIPE.margin));
}

export function layoutPipeline(nodes: PipelineNode[], edges: PipelineEdge[], fit?: PipelineFit): PipelineLayout {
  const h = horizontal(nodes, edges);
  if (!fit) return assemble('horizontal', h, nodes, edges, 1);
  const hs = fitScale(h.width, fit.maxWidth);
  if (hs >= fit.minScale) return assemble('horizontal', h, nodes, edges, hs);
  const v = vertical(nodes, edges);
  const vs = fitScale(v.width, fit.maxWidth);
  return vs > hs ? assemble('vertical', v, nodes, edges, vs) : assemble('horizontal', h, nodes, edges, hs);
}
