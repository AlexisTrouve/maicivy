/**
 * Géométrie : la montagne du dossier vivant (scene.ts), l'apparence des projets (looks.ts) et le
 * schéma d'architecture (pipeline.ts).
 */
import { readFileSync } from 'fs';
import path from 'path';

import { CH } from '../choreography';
import { applyEvent, initialRunState, type RunState } from '../runState';
import { projectLook } from '../looks';
import { layoutPipeline } from '../pipeline';
import { G, anchorOf, buildScene, introFoot, layoutFor, linkPath, stageGeometry } from '../scene';
import type { AgentEvent } from '../events';

const nodes = Array.from({ length: 75 }, (_, i) => ({
  id: `p${i}`,
  name: `Projet ${i}`,
  monogram: 'Pj',
  stratum: (['works', 'engines', 'tooling', 'foundations'] as const)[i % 4],
}));
const input = {
  nodes,
  links: [
    { from: 'p10', to: 'p0' },
    { from: 'p11', to: 'p12' },
  ],
  summit: ['p0', 'p1', 'p2', 'p3'].map((id) => ({ id, members: [] })),
};

describe('buildScene', () => {
  const scene = buildScene(input);

  it('pose 4 cartes et 71 tuiles, et un rang de libre sous le sommet pour le visiteur', () => {
    expect(scene.cards).toHaveLength(4);
    expect(scene.tiles).toHaveLength(71);
    expect(scene.visitor.y).toBe(scene.tilesTop);
    // Rangs ouverts : la première tuile descend d'une hauteur de rang, sous la tuile du visiteur.
    const firstRow = Math.min(...scene.tiles.map((t) => t.y));
    expect(anchorOf(scene, scene.tiles.find((t) => t.y === firstRow)!.id, 1)[1]).toBe(firstRow + G.tile / 2 + G.rowHeight);
  });

  it('même cadrage que la maquette : 1092 de large, 1150 à faire tenir', () => {
    expect(scene.width).toBe(1092);
    expect(scene.fitHeight).toBe(1150);
  });

  it('un lien vers une tuile descend avec les rangs, un lien de carte ne bouge pas à son bout', () => {
    const l = scene.links.find((x) => x.key === 'p10>p0')!;
    expect(linkPath(l, 0)).not.toEqual(linkPath(l, 1));
    expect(linkPath(l, 1).endsWith(`L${l.x2},${l.y2}`)).toBe(true);
  });
});

describe('stageGeometry', () => {
  it('1920×1024 : équipe de 845 px à droite, montagne entière à gauche', () => {
    const g = stageGeometry(1920, 1024, 420, { width: 1092, fitHeight: 1150 });
    expect(g.team.w).toBe(845);
    expect(g.team.x + g.team.w + g.pad).toBe(1920);
    expect(g.work.x + 1092 * g.work.s).toBeLessThanOrEqual(g.team.x);
    expect(g.work.y + 1150 * g.work.s).toBeLessThanOrEqual(1024);
    // Grand écran : rien ne bouge à la remise, pas de bandeau des retenus.
    expect(g.compact).toBe(false);
    expect(g.full).toEqual(g.team);
    expect(g.kept).toBeNull();
  });

  it.each([
    [390, 766],
    [393, 649],
  ])('téléphone %i×%i : montagne entière en haut, bandeau, équipe pleine largeur dessous, dossier plein écran à la remise', (W, H) => {
    const g = stageGeometry(W, H, 430, { width: 1092, fitHeight: 1150 });
    expect(g.compact).toBe(true);
    // Montagne : dans la largeur et au-dessus du bandeau.
    expect(g.work.x).toBeGreaterThanOrEqual(g.pad - 0.5);
    expect(g.work.x + 1092 * g.work.s).toBeLessThanOrEqual(W - g.pad + 0.5);
    expect(g.work.y + 1150 * g.work.s).toBeLessThanOrEqual(g.kept!.y + 0.5);
    // Bandeau puis équipe, sans chevauchement, l'équipe jusqu'en bas.
    expect(g.kept!.y + g.kept!.h).toBeLessThanOrEqual(g.team.y);
    expect(g.team.x + g.team.w + g.pad).toBe(W);
    expect(g.team.y + g.team.h + g.pad).toBe(H);
    expect(g.full).toEqual({ x: g.pad, y: g.pad, w: W - 2 * g.pad, h: H - 2 * g.pad });
    // À l'ouverture, la montagne tient dans la largeur, sous le texte.
    expect(g.intro.x).toBeGreaterThanOrEqual(0);
    expect(g.intro.y).toBeGreaterThan(430);
  });

  it('le seuil : 760 px = grand écran, 759 px = téléphone', () => {
    expect(stageGeometry(760, 900, 400, { width: 1092, fitHeight: 1150 }).compact).toBe(false);
    expect(stageGeometry(759, 900, 400, { width: 1092, fitHeight: 1150 }).compact).toBe(true);
  });
});

describe('layoutFor — la disposition se décide sur la fenêtre, comme les media queries du CSS', () => {
  it.each([
    [1920, 1080, 'wide'],
    [1366, 768, 'wide'],
    [393, 727, 'compact'], // Pixel 5
    [390, 844, 'compact'],
    [802, 293, 'landscape'], // Pixel 5 couché
    [750, 340, 'landscape'], // iPhone 12 couché : < 760 px de large, mais couché l'emporte
    [832, 380, 'landscape'], // iPhone 13 Pro Max couché
    [568, 320, 'landscape'], // iPhone SE couché
    [900, 501, 'wide'], // une fenêtre de bureau basse reste un grand écran au-delà de 500 px
  ] as const)('%i×%i → %s', (vw, vh, layout) => {
    expect(layoutFor(vw, vh)).toBe(layout);
  });
});

describe('stageGeometry — téléphone couché', () => {
  const scene = { width: 1092, fitHeight: 1150 };
  it.each([
    [802, 241], // Pixel 5 couché, sous une barre de 52 px
    [750, 288],
    [832, 328],
    [568, 268],
  ])('%i×%i : montagne entière à gauche au-dessus du bandeau, équipe à droite, dossier plein écran', (W, H) => {
    const g = stageGeometry(W, H, 200, scene, 'landscape');
    expect(g.layout).toBe('landscape');
    expect(g.compact).toBe(true);
    // Montagne : dans sa colonne, au-dessus du bandeau des retenus.
    expect(g.work.x).toBeGreaterThanOrEqual(g.pad - 0.5);
    expect(g.work.x + scene.width * g.work.s).toBeLessThanOrEqual(g.kept!.x + g.kept!.w + 0.5);
    expect(g.work.y + scene.fitHeight * g.work.s).toBeLessThanOrEqual(g.kept!.y + 0.5);
    expect(g.kept!.y + g.kept!.h + g.pad).toBe(H);
    // Équipe : à droite de la colonne, jusqu'au bord, toute la hauteur — et d'au moins 55 % de large.
    expect(g.team.x).toBeGreaterThanOrEqual(g.kept!.x + g.kept!.w);
    expect(g.team.x + g.team.w + g.pad).toBe(W);
    expect(g.team.h).toBe(H - 2 * g.pad);
    expect(g.team.w).toBeGreaterThanOrEqual(W * 0.55);
    expect(g.full).toEqual({ x: g.pad, y: g.pad, w: W - 2 * g.pad, h: H - 2 * g.pad });
    // Avant le lancement, la montagne attend sous la scène (l'ouverture occupe toute la hauteur).
    expect(g.intro.y).toBeGreaterThan(H);
    expect(g.intro.s).toBe(g.work.s);
    // … centrée : la page s'allonge pour la montrer, on la voit en défilant.
    expect(g.intro.x + (scene.width * g.intro.s) / 2).toBeCloseTo(W / 2, 6);
  });
});

describe('introFoot — la page d’ouverture s’allonge jusqu’au pied de la montagne', () => {
  const scene = buildScene(input);

  it('sous le dernier rang de tuiles ET sous le fond peint, à l’échelle d’ouverture', () => {
    const g = stageGeometry(1920, 827, 420, scene);
    const foot = introFoot(scene, g.intro);
    const lastRow = Math.max(...scene.tiles.map((t) => t.y + t.h));
    expect(foot).toBeGreaterThan(g.intro.y + lastRow * g.intro.s);
    expect(foot).toBeGreaterThanOrEqual(g.intro.y + (scene.art.top + scene.art.height) * g.intro.s);
    expect(foot).toBeGreaterThanOrEqual(g.intro.y + scene.height * g.intro.s);
    // Sur un écran de PC (947 px), la montagne d'ouverture dépasse le premier écran : d'où l'allongement.
    expect(foot).toBeGreaterThan(827);
  });

  it('jamais réduite pour tenir : l’échelle d’ouverture ne dépend pas de la hauteur de l’écran', () => {
    expect(stageGeometry(1920, 600, 420, scene).intro).toEqual(stageGeometry(1920, 1400, 420, scene).intro);
  });
});

describe('projectLook', () => {
  const ev = (s: RunState, at: number, event: AgentEvent) => applyEvent(s, { at, event });
  let s = initialRunState();
  s = ev(s, 1, { type: 'index.filtered', agent: 'researcher', total: 75, candidates: ['a', 'b'] });

  it('un candidat se teinte quand la ligne de balayage le touche', () => {
    expect(projectLook(s, 'a', 1 + CH.sweepD * 0.4, 0.5).look).toBe('idle');
    expect(projectLook(s, 'a', 1 + CH.sweepD * 0.6, 0.5).look).toBe('candidate');
  });

  it('hors candidats, le projet recule après le balayage', () => {
    expect(projectLook(s, 'z', 1 + CH.sweepD + CH.quietD, 0).quiet).toBe(1);
  });

  it('retenu = allumé, écarté = rendu au rang en douceur', () => {
    const s2 = ev(ev(s, 5, { type: 'project.verdict', agent: 'researcher', projectId: 'a', verdict: 'keep' }), 6, {
      type: 'project.verdict',
      agent: 'researcher',
      projectId: 'b',
      verdict: 'drop',
    });
    expect(projectLook(s2, 'a', 7, 0)).toEqual({ look: 'lit', quiet: 0 });
    expect(projectLook(s2, 'b', 6, 0)).toEqual({ look: 'dropped', quiet: 0 });
    expect(projectLook(s2, 'b', 6 + CH.dropD, 0).quiet).toBeCloseTo(1);
  });
});

describe('layoutPipeline', () => {
  const L = layoutPipeline(
    [
      { id: 'a', title: 'Ticket', subtitle: 'entrant', col: 0, row: 1 },
      { id: 'b', title: 'Trieur', subtitle: 'léger', col: 1, row: 1 },
      { id: 'c', title: 'Répondeur', subtitle: 'cite', col: 2, row: 0 },
      { id: 'h', title: 'File humaine', subtitle: 'brief', col: 2, row: 2, human: true },
      { id: 'd', title: 'Contrôle', subtitle: 'sans LLM', col: 3, row: 0 },
    ],
    [
      { from: 'a', to: 'b', step: 1 },
      { from: 'b', to: 'c' },
      { from: 'c', to: 'd' },
      { from: 'b', to: 'h', dashed: true },
      { from: 'd', to: 'h', dashed: true, step: 2 },
    ],
  );

  it('dessine chaque boîte une fois, chaque arête une fois, puis le journal et les pastilles', () => {
    expect(L.parts.flatMap((p) => p.boxes).map((b) => b.id)).toEqual(['a', 'b', 'c', 'h', 'd']);
    expect(L.parts.flatMap((p) => p.edges)).toHaveLength(5);
    expect(L.parts.at(-2)?.journal).toBe(true);
    expect(L.parts.at(-1)?.steps).toBe(true);
  });

  it("un retour vers la gauche part SOUS la source et arrive au flanc droit de la cible", () => {
    const back = L.parts.flatMap((p) => p.edges).find((e) => e.key === 'd>h')!;
    const d = L.boxes.get('d')!;
    const h = L.boxes.get('h')!;
    expect(back.d.startsWith(`M${d.x + d.w / 2},${d.y + 40}V`)).toBe(true);
    expect(back.d.endsWith(`H${h.x + h.w + 6}`)).toBe(true);
  });

  it('aucune boîte ne se chevauche', () => {
    const boxes = [...L.boxes.values()];
    for (const a of boxes)
      for (const b of boxes) {
        if (a === b) continue;
        const overlap = a.x < b.x + b.w && b.x < a.x + a.w && a.y < b.y + 40 && b.y < a.y + 40;
        expect(overlap).toBe(false);
      }
  });
});

/** Contraste WCAG (sRGB) entre deux couleurs hex. */
function luminance(hex: string): number {
  const c = hex.replace('#', '').match(/../g)!.map((h) => parseInt(h, 16) / 255);
  const [r, g, b] = c.map((v) => (v <= 0.03928 ? v / 12.92 : Math.pow((v + 0.055) / 1.055, 2.4)));
  return 0.2126 * r + 0.7152 * g + 0.0722 * b;
}
const contrast = (a: string, b: string) => {
  const [hi, lo] = [luminance(a), luminance(b)].sort((x, y) => y - x);
  return (hi + 0.05) / (lo + 0.05);
};

describe('bouton or (défaut de la maquette : 4,13:1)', () => {
  const css = readFileSync(path.join(__dirname, '../../../components/dossier-vivant/dossier-vivant.module.css'), 'utf-8');
  const token = (name: string) => css.match(new RegExp(`--${name}:\\s*(#[0-9a-fA-F]{6})`))![1];

  it('texte du bouton ≥ 4,5:1 sur l’or, au repos comme au survol', () => {
    expect(contrast(token('dv-gold-ink'), token('dv-gold'))).toBeGreaterThanOrEqual(4.5);
    expect(contrast(token('dv-gold-ink'), token('dv-gold-bright'))).toBeGreaterThanOrEqual(4.5);
  });
});
