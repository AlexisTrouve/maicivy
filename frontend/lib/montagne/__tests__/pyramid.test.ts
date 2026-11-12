/**
 * pyramid.test.ts — la disposition à plat de la montagne (lib/montagne/pyramid.ts).
 *
 * Invariants verrouillés : les rangs s'élargissent vers le bas et portent exactement tous les
 * projets hors sommet ; les tuiles d'un rang ne se chevauchent pas et tiennent dans la largeur de
 * leur géométrie ; sur téléphone, une tuile fait au moins 44 px (cible tactile) ; les strates se
 * lisent de haut en bas ; un filet par lien réel, et seulement eux.
 */

import type { MontagneLink, MontagneNode, StratumId } from '@/lib/montagne-api';
import { GEOMETRIES, centerOutSlot, layoutMountain, orderTiles, rowSizes } from '../pyramid';

function node(id: string, stratum: StratumId): MontagneNode {
  return {
    id,
    name: id,
    mini: '',
    status: 'active',
    stratum,
    monogram: id.slice(0, 2).toUpperCase(),
    icon_url: '',
    curated: false,
    visuals: [],
    x: 0,
    y: 0,
    z: 0,
  };
}

/** 75 projets répartis comme les vraies données (39 œuvres, 2 moteurs, 25 outils, 9 fondations). */
function realisticNodes(): MontagneNode[] {
  const out: MontagneNode[] = [];
  const counts: [StratumId, number][] = [
    ['works', 39],
    ['engines', 2],
    ['tooling', 25],
    ['foundations', 9],
  ];
  for (const [s, n] of counts) for (let i = 0; i < n; i++) out.push(node(`${s}_${String(i).padStart(2, '0')}`, s));
  return out;
}

const SUMMIT = ['works_00', 'works_01', 'works_02', 'works_03'];

describe('rowSizes', () => {
  it.each(GEOMETRIES.map((g, i) => [i, g] as const))('géométrie %i : somme exacte, rangs non décroissants, ≤ maxPerRow', (_i, g) => {
    for (const n of [0, 1, 7, 30, 71, 120]) {
      const sizes = rowSizes(n, g);
      expect(sizes.reduce((a, b) => a + b, 0)).toBe(n);
      sizes.forEach((s, i) => {
        expect(s).toBeGreaterThan(0);
        expect(s).toBeLessThanOrEqual(g.maxPerRow);
        if (i > 0) expect(s).toBeGreaterThanOrEqual(sizes[i - 1]);
      });
    }
  });

  it('grand écran : les 71 tuiles réelles forment la pyramide 5-7-9-11-13-13-13', () => {
    expect(rowSizes(71, GEOMETRIES[2])).toEqual([5, 7, 9, 11, 13, 13, 13]);
  });
});

describe('centerOutSlot', () => {
  it('remplit un rang du centre vers les bords, chaque place une seule fois', () => {
    for (const size of [1, 2, 5, 6, 13]) {
      const slots = Array.from({ length: size }, (_, k) => centerOutSlot(k, size));
      expect(new Set(slots).size).toBe(size);
      expect(Math.min(...slots)).toBe(0);
      expect(Math.max(...slots)).toBe(size - 1);
      expect(slots[0]).toBe(Math.floor((size - 1) / 2));
    }
  });
});

describe('orderTiles', () => {
  it('écarte le sommet, range les strates du haut vers le bas, le plus relié d’abord', () => {
    const nodes = [node('f1', 'foundations'), node('t1', 'tooling'), node('w2', 'works'), node('w1', 'works'), node('e1', 'engines'), node('top', 'works')];
    const links: MontagneLink[] = [{ from: 'e1', to: 'w2', kind: 'stack' }];
    const ids = orderTiles(nodes, links, ['top']).map((n) => n.id);
    expect(ids).toEqual(['w2', 'w1', 'e1', 't1', 'f1']);
  });
});

describe('layoutMountain', () => {
  const nodes = realisticNodes();
  const links: MontagneLink[] = [
    { from: 'engines_00', to: 'works_00', kind: 'stack' }, // vers une carte du sommet
    { from: 'tooling_00', to: 'works_10', kind: 'stack' }, // entre deux tuiles
    { from: 'foundations_00', to: 'ghost', kind: 'stack' }, // bout absent : pas de filet
  ];

  it.each(GEOMETRIES.map((g, i) => [i, g] as const))('géométrie %i : toutes les tuiles, sans chevauchement, dans la largeur', (_i, g) => {
    const m = layoutMountain({ nodes, links, summit: SUMMIT }, g);
    expect(m.cards.map((c) => c.id)).toEqual(SUMMIT);
    expect(m.tiles).toHaveLength(nodes.length - SUMMIT.length);
    for (const t of m.tiles) {
      expect(t.x).toBeGreaterThanOrEqual(0);
      expect(t.x + t.w).toBeLessThanOrEqual(m.width);
      expect(t.y + t.h).toBeLessThanOrEqual(m.height);
    }
    for (let i = 0; i < m.tiles.length; i++) {
      for (let j = i + 1; j < m.tiles.length; j++) {
        const a = m.tiles[i];
        const b = m.tiles[j];
        const overlap = a.x < b.x + b.w && b.x < a.x + a.w && a.y < b.y + b.h && b.y < a.y + a.h;
        expect(overlap).toBe(false);
      }
    }
    // Les tuiles commencent sous les cartes.
    const cardsBottom = Math.max(...m.cards.map((c) => c.y + c.h));
    expect(Math.min(...m.tiles.map((t) => t.y))).toBeGreaterThan(cardsBottom);
  });

  it('téléphone : tuiles d’au moins 44 px, montagne de 360 px de large au plus', () => {
    const g = GEOMETRIES[0];
    const m = layoutMountain({ nodes, links, summit: SUMMIT }, g);
    expect(g.tile).toBeGreaterThanOrEqual(44);
    expect(m.width).toBeLessThanOrEqual(360);
  });

  it('les strates se lisent de haut en bas : aucune tuile d’une strate inférieure au-dessus d’une supérieure', () => {
    const m = layoutMountain({ nodes, links, summit: SUMMIT }, GEOMETRIES[0]);
    const byId = new Map(nodes.map((n) => [n.id, n]));
    const order: StratumId[] = ['works', 'engines', 'tooling', 'foundations'];
    const lowest = new Map<StratumId, number>();
    const highest = new Map<StratumId, number>();
    for (const t of m.tiles) {
      const s = byId.get(t.id)!.stratum;
      lowest.set(s, Math.max(lowest.get(s) ?? -Infinity, t.y));
      highest.set(s, Math.min(highest.get(s) ?? Infinity, t.y));
    }
    for (let i = 1; i < order.length; i++) {
      expect(highest.get(order[i])!).toBeGreaterThanOrEqual(lowest.get(order[i - 1])!);
    }
  });

  it.each(GEOMETRIES.map((g, i) => [i, g] as const))('géométrie %i : la pointe du fond (peakBottom) est sous les cartes et au-dessus des tuiles', (_i, g) => {
    const m = layoutMountain({ nodes, links, summit: SUMMIT }, g);
    // La montagne peinte doit naître sous les cartes, jamais derrière une tuile ni au milieu d'une carte.
    expect(m.peakBottom).toBe(Math.max(...m.cards.map((c) => c.y + c.h)));
    for (const t of m.tiles) expect(t.y).toBeGreaterThanOrEqual(m.peakBottom + g.peakGap);
  });

  it('un filet par lien dont les deux bouts sont dessinés ; vers une carte, il s’arrête sous elle', () => {
    const m = layoutMountain({ nodes, links, summit: SUMMIT }, GEOMETRIES[2]);
    expect(m.links.map((l) => `${l.from}>${l.to}`)).toEqual(['engines_00>works_00', 'tooling_00>works_10']);
    const card = m.cards.find((c) => c.id === 'works_00')!;
    expect(m.links[0].x2).toBe(card.x + card.w / 2);
    expect(m.links[0].y2).toBe(card.y + card.h);
    expect(m.links[0].d).toMatch(/^M[\d.]+,[\d.]+L/);
  });

  it('entre deux cartes, le filet plonge en courbe sous elles', () => {
    const m = layoutMountain({ nodes, links: [{ from: 'works_01', to: 'works_00', kind: 'description' }], summit: SUMMIT }, GEOMETRIES[2]);
    expect(m.links).toHaveLength(1);
    expect(m.links[0].d).toContain('Q');
  });

  it('est déterministe et ne dépend pas de l’ordre des données', () => {
    const a = layoutMountain({ nodes, links, summit: SUMMIT }, GEOMETRIES[1]);
    const b = layoutMountain({ nodes: [...nodes].reverse(), links: [...links].reverse(), summit: SUMMIT }, GEOMETRIES[1]);
    expect(b.tiles).toEqual(a.tiles);
    expect(b.cards).toEqual(a.cards);
  });
});

describe('projets absorbés par une carte du sommet', () => {
  // works_10 est absorbé par la carte works_01 (comme Emergika par Theomen dès qu'elle a un visuel).
  const absorbed = new Map([['works_10', 'works_01']]);
  const links: MontagneLink[] = [
    { from: 'works_01', to: 'works_10', kind: 'description' }, // carte → son propre membre : boucle
    { from: 'engines_00', to: 'works_10', kind: 'stack' }, // vers le membre : arrive sur la carte
    { from: 'works_10', to: 'works_20', kind: 'stack' }, // depuis le membre : part de la carte
  ];

  it.each(GEOMETRIES.map((g, i) => [i, g] as const))('géométrie %i : pas de tuile, filets ancrés sur la carte, boucle écartée', (_i, g) => {
    const m = layoutMountain({ nodes: realisticNodes(), links, summit: SUMMIT, absorbed }, g);
    expect(m.tiles.map((t) => t.id)).not.toContain('works_10');
    expect(m.tiles).toHaveLength(75 - SUMMIT.length - 1);
    const card = m.cards.find((c) => c.id === 'works_01')!;
    const cardAnchor = [card.x + card.w / 2, card.y + card.h];
    expect(m.links.map((l) => `${l.from}>${l.to}`)).toEqual(['engines_00>works_10', 'works_10>works_20']);
    expect([m.links[0].x2, m.links[0].y2]).toEqual(cardAnchor);
    expect([m.links[1].x1, m.links[1].y1]).toEqual(cardAnchor);
  });

  it('un membre dont la carte est absente redevient une tuile', () => {
    const m = layoutMountain({ nodes: realisticNodes(), links: [], summit: ['works_00'], absorbed }, GEOMETRIES[0]);
    expect(m.tiles.map((t) => t.id)).toContain('works_10');
  });
});
