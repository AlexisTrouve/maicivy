/**
 * Les retours du schéma (arêtes vers une colonne à gauche) ne passent jamais sur une pastille de
 * garde-fou ni sur une flèche aller. Cas réel : run du 2026-11 (Cabinet Lumen), retour en pointillés
 * Validation → Brouillon tracé SUR la flèche ③ — à l'écran comme dans le PDF, même mise en page.
 */
import { layoutPipeline, PIPE, SCREEN_FIT } from '../pipeline';
import { PRINT_FIT } from '../pdfExport';
import { LIVE_PIPELINE, SCHEMA_PIPELINE, tesseraPipeline } from '../__fixtures__/pipelines';

type Pt = { x: number; y: number };

/** Sommets d'un tracé fait de M, H, V (et de la fin d'un C) : assez pour les retours. */
function vertices(d: string): Pt[] {
  const out: Pt[] = [];
  let cur: Pt = { x: 0, y: 0 };
  for (const [, cmd, args] of d.matchAll(/([MHVC])([^MHVC]*)/g)) {
    const n = args.split(/[ ,]+/).filter(Boolean).map(Number);
    if (cmd === 'M') cur = { x: n[0], y: n[1] };
    else if (cmd === 'H') cur = { x: n[0], y: cur.y };
    else if (cmd === 'V') cur = { x: cur.x, y: n[0] };
    else if (cmd === 'C') cur = { x: n[4], y: n[5] };
    out.push(cur);
  }
  return out;
}

/** Distance d'un point à un segment axé (les retours n'ont que des segments droits). */
function distToSegment(p: Pt, a: Pt, b: Pt): number {
  const cx = Math.max(Math.min(a.x, b.x), Math.min(p.x, Math.max(a.x, b.x)));
  const cy = Math.max(Math.min(a.y, b.y), Math.min(p.y, Math.max(a.y, b.y)));
  return Math.hypot(p.x - cx, p.y - cy);
}

const CASES = {
  'run réel 2026-11 (retour même rangée)': SCHEMA_PIPELINE,
  'run réel 2026-10': LIVE_PIPELINE,
  'rejeu Tessera': tesseraPipeline(),
};

describe.each(Object.entries(CASES))('%s', (_name, { nodes, edges }) => {
  const backs = edges.filter((e) => nodes.find((n) => n.id === e.to)!.col < nodes.find((n) => n.id === e.from)!.col);

  it.each([
    ['PDF (A4)', PRINT_FIT],
    ['écran (demi-colonne)', { maxWidth: 399, minScale: SCREEN_FIT.minScale }],
    ['taille nominale', undefined],
  ])('%s : aucun retour ne touche une pastille ni ne longe une flèche aller', (_label, fit) => {
    const L = layoutPipeline(nodes, edges, fit);
    const draws = L.parts.flatMap((p) => p.edges);
    for (const back of backs) {
      const draw = draws.find((e) => e.key === `${back.from}>${back.to}`)!;
      // Les pastilles des AUTRES arêtes (un retour peut porter la sienne, posée sur lui).
      const steps = draws.flatMap((e) => (e !== draw && e.step ? [e.step] : []));
      const pts = vertices(draw.d);
      for (let i = 1; i < pts.length; i++) {
        // Une pastille a un rayon de 8 : le retour passe à plus de 10 de son centre.
        for (const s of steps) expect(distToSegment(s, pts[i - 1], pts[i])).toBeGreaterThan(10);
        // Aucun segment horizontal du retour ne court sur l'ordonnée d'une flèche aller droite.
        if (pts[i - 1].y === pts[i].y) {
          for (const f of draws.filter((e) => e !== draw && /^M[\d.]+,[\d.]+H[\d.]+$/.test(e.d))) {
            const [fy] = f.d.match(/,([\d.]+)H/)!.slice(1).map(Number);
            expect(Math.abs(fy - pts[i].y)).toBeGreaterThan(4);
          }
        }
      }
    }
    // Le couloir du haut ne déborde pas du cadre : tout tracé reste dans la hauteur du schéma.
    for (const e of draws) for (const p of vertices(e.d)) expect(p.y).toBeGreaterThanOrEqual(-PIPE.margin);
  });
});
