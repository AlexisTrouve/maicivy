/**
 * Lisibilité du schéma d'architecture : le choix d'orientation et l'échelle (pipeline.ts), sur
 * les deux schémas réels — le rejeu Tessera (maquette validée, doit rester horizontal) et le run
 * réel qui s'affichait à l'échelle 0,54 (texte de 7 px) dans la demi-colonne du dossier.
 */
import { LIVE_PIPELINE, PANEL_WIDTH, tesseraPipeline } from '../__fixtures__/pipelines';
import { layoutPipeline, PIPE, SCREEN_FIT, type PipelineLayout } from '../pipeline';
import { PRINT_FIT } from '../pdfExport';

const panel = { maxWidth: PANEL_WIDTH, minScale: SCREEN_FIT.minScale };

function overlaps(L: PipelineLayout): string[] {
  const boxes = [...L.boxes.values()];
  const hits: string[] = [];
  for (const a of boxes)
    for (const b of boxes)
      if (a.id < b.id && a.x < b.x + b.w && b.x < a.x + a.w && a.y < b.y + PIPE.boxH && b.y < a.y + PIPE.boxH) hits.push(`${a.id}/${b.id}`);
  return hits;
}

/** Largeur réellement affichée (px CSS) : celle que PipelineDiagram donne au SVG. */
const shown = (L: PipelineLayout) => (L.width + 2 * PIPE.margin) * L.scale;

describe('layoutPipeline — lisibilité dans le panneau (1920×1080)', () => {
  it('run réel : passe en vertical, à l’échelle 1, dans la largeur offerte', () => {
    const before = layoutPipeline(LIVE_PIPELINE.nodes, LIVE_PIPELINE.edges);
    // Le défaut mesuré : 742 unités écrasées dans 399 px.
    expect(before.width).toBeGreaterThan(740);
    expect(PANEL_WIDTH / before.width).toBeLessThan(0.55);

    const L = layoutPipeline(LIVE_PIPELINE.nodes, LIVE_PIPELINE.edges, panel);
    expect(L.orientation).toBe('vertical');
    expect(L.scale).toBe(1);
    expect(shown(L)).toBeLessThanOrEqual(PANEL_WIDTH);
    expect(overlaps(L)).toEqual([]);
  });

  it('rejeu Tessera : garde l’horizontale de la maquette, au-dessus du plancher', () => {
    const { nodes, edges } = tesseraPipeline();
    const L = layoutPipeline(nodes, edges, panel);
    expect(L.orientation).toBe('horizontal');
    expect(L.scale).toBeGreaterThanOrEqual(SCREEN_FIT.minScale);
    expect(shown(L)).toBeLessThanOrEqual(PANEL_WIDTH + 0.5);
  });

  it('jamais agrandi au-delà de la taille nominale', () => {
    const L = layoutPipeline(LIVE_PIPELINE.nodes, LIVE_PIPELINE.edges, { maxWidth: 4000, minScale: SCREEN_FIT.minScale });
    expect(L.orientation).toBe('horizontal');
    expect(L.scale).toBe(1);
  });

  it('chaîne longue d’un vrai agent (9 étapes, 2 branches) : verticale à l’échelle 1', () => {
    const nodes = Array.from({ length: 9 }, (_, i) => ({ id: `n${i}`, title: `Étape numéro ${i}`, subtitle: 'sous-titre assez long', col: i, row: 0 }));
    nodes.push({ id: 'esc', title: 'Escalade', subtitle: 'file humaine', col: 4, row: 1 });
    const edges = nodes.slice(1, 9).map((n, i) => ({ from: `n${i}`, to: n.id }));
    edges.push({ from: 'n4', to: 'esc' }, { from: 'n8', to: 'n2' });
    const L = layoutPipeline(nodes, edges, panel);
    expect(L.orientation).toBe('vertical');
    expect(L.scale).toBe(1);
    expect(shown(L)).toBeLessThanOrEqual(PANEL_WIDTH);
    expect(overlaps(L)).toEqual([]);
  });

  it('grille trop large dans les deux sens : la plus grande des deux échelles, jamais pire', () => {
    const nodes = Array.from({ length: 4 }, (_, r) => Array.from({ length: 6 }, (_, c) => ({ id: `n${c}${r}`, title: 'Un titre très très long', subtitle: 'x', col: c, row: r }))).flat();
    const L = layoutPipeline(nodes, [], panel);
    const h = layoutPipeline(nodes, []);
    expect(L.scale).toBeGreaterThanOrEqual(Math.min(1, PANEL_WIDTH / (h.width + 2 * PIPE.margin)));
  });
});

describe('layoutPipeline — tracés', () => {
  it('horizontale : deux boîtes d’une même colonne sur des rangées voisines ne se chevauchent plus', () => {
    const L = layoutPipeline(LIVE_PIPELINE.nodes, LIVE_PIPELINE.edges);
    expect(overlaps(L)).toEqual([]);
    // Et l'arête Contrôle → Anomalie descend tout droit, de bord à bord.
    const e = L.parts.flatMap((p) => p.edges).find((x) => x.key === 'controle>anomalie')!;
    const c = L.boxes.get('controle')!;
    const a = L.boxes.get('anomalie')!;
    expect(e.d).toBe(`M${c.x + c.w / 2},${c.y + PIPE.boxH}V${a.y - 6}`);
  });

  it('verticale : une arête descend du bord bas de la source au bord haut de la cible', () => {
    const L = layoutPipeline(LIVE_PIPELINE.nodes, LIVE_PIPELINE.edges, panel);
    const e = L.parts.flatMap((p) => p.edges).find((x) => x.key === 'recu>lecture')!;
    const r = L.boxes.get('recu')!;
    const l = L.boxes.get('lecture')!;
    expect(e.d).toBe(`M${r.x + r.w / 2},${r.y + PIPE.boxH}V${l.y - 6}`);
    // Même étage (Contrôle → Anomalie) : horizontale de flanc à flanc.
    const side = L.parts.flatMap((p) => p.edges).find((x) => x.key === 'controle>anomalie')!;
    const c = L.boxes.get('controle')!;
    expect(side.d.startsWith(`M${c.x + c.w},`)).toBe(true);
  });

  it('verticale : un retour passe par le couloir à droite, dans la largeur du schéma', () => {
    const nodes = [
      { id: 'a', title: 'A', subtitle: 'a', col: 0, row: 0 },
      { id: 'b', title: 'B', subtitle: 'b', col: 1, row: 0 },
      { id: 'c', title: 'C', subtitle: 'c', col: 2, row: 0 },
    ];
    const L = layoutPipeline(nodes, [{ from: 'a', to: 'b' }, { from: 'b', to: 'c' }, { from: 'c', to: 'a', dashed: true }], { maxWidth: 70, minScale: SCREEN_FIT.minScale });
    expect(L.orientation).toBe('vertical');
    const back = L.parts.flatMap((p) => p.edges).find((x) => x.key === 'c>a')!;
    const cx = Number(/H([\d.]+)V/.exec(back.d)![1]);
    expect(cx).toBeLessThanOrEqual(L.width);
  });
});

describe('PRINT_FIT — la page A4', () => {
  it('le run réel tient en horizontale sur la largeur utile, au-dessus du plancher d’impression', () => {
    const L = layoutPipeline(LIVE_PIPELINE.nodes, LIVE_PIPELINE.edges, PRINT_FIT);
    expect(L.orientation).toBe('horizontal');
    expect(L.scale).toBeGreaterThanOrEqual(PRINT_FIT.minScale);
    expect(shown(L)).toBeLessThanOrEqual(PRINT_FIT.maxWidth);
  });
});
