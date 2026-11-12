import { chooseLabels, comparePriority, intersects, wrapLabel, type LabelCandidate, type Rect } from '../labels';

/**
 * Arbitrage des étiquettes (labels.ts) : qui gagne, et pourquoi. Pur calcul, aucun DOM.
 */

const BOUNDS: Rect = { left: 0, top: 0, right: 1000, bottom: 1000 };

/** Nœud à (x, y) : icône 20 px, étiquette 80×14 sous l'icône. */
function node(id: string, x: number, y: number, extra: Partial<LabelCandidate> = {}): LabelCandidate {
  return {
    id,
    icon: { left: x - 10, top: y - 10, right: x + 10, bottom: y + 10 },
    label: { left: x - 40, top: y + 13, right: x + 40, bottom: y + 27 },
    depth: 10,
    degree: 0,
    focus: false,
    ...extra,
  };
}

describe('chooseLabels', () => {
  it('deux étiquettes qui se chevauchent : la plus proche de la caméra gagne', () => {
    const near = node('near', 100, 100, { depth: 5 });
    const far = node('far', 150, 100, { depth: 9 });
    expect([...chooseLabels([far, near], BOUNDS)]).toEqual(['near']);
  });

  it('à profondeur égale, le plus relié gagne ; à degré égal, l’id', () => {
    const hub = node('zz_hub', 100, 100, { degree: 12 });
    const leaf = node('aa_leaf', 150, 100, { degree: 1 });
    expect([...chooseLabels([leaf, hub], BOUNDS)]).toEqual(['zz_hub']);
    const a = node('a', 100, 100);
    const b = node('b', 150, 100);
    expect([...chooseLabels([b, a], BOUNDS)]).toEqual(['a']);
  });

  it('le nœud désigné passe avant tout, même plus loin', () => {
    const near = node('near', 100, 100, { depth: 1 });
    const focused = node('focused', 150, 100, { depth: 9, focus: true });
    expect([...chooseLabels([near, focused], BOUNDS)]).toEqual(['focused']);
  });

  it('une étiquette ne recouvre jamais l’icône d’un autre nœud', () => {
    const a = node('a', 100, 100, { depth: 1 });
    // L'icône de b est juste sous le nom de a.
    const b = node('b', 100, 120, { depth: 9 });
    const shown = chooseLabels([a, b], BOUNDS);
    expect(shown.has('a')).toBe(false);
  });

  it('une étiquette coupée par le bord n’est pas affichée ; l’espace réservé est respecté', () => {
    const edge = node('edge', 20, 100);
    expect(chooseLabels([edge], BOUNDS).size).toBe(0);
    const free = node('free', 500, 500);
    expect(chooseLabels([free], BOUNDS, [{ left: 480, top: 510, right: 520, bottom: 520 }]).size).toBe(0);
  });

  it('l’étiquette réapparaît dès que la place se libère', () => {
    const a = node('a', 100, 100, { depth: 1 });
    expect(chooseLabels([a, node('b', 150, 100, { depth: 2 })], BOUNDS).has('b')).toBe(false);
    expect(chooseLabels([a, node('b', 400, 104, { depth: 2 })], BOUNDS).has('b')).toBe(true);
  });

  it('aucune paire d’étiquettes affichées ne se chevauche, sur une grille dense', () => {
    const many: LabelCandidate[] = [];
    for (let i = 0; i < 80; i++) many.push(node(`n${i}`, 60 + (i % 10) * 45, 60 + Math.floor(i / 10) * 30, { depth: (i * 7) % 13 }));
    const shown = many.filter((c) => chooseLabels(many, BOUNDS).has(c.id));
    expect(shown.length).toBeGreaterThan(0);
    for (let i = 0; i < shown.length; i++) {
      for (let j = i + 1; j < shown.length; j++) expect(intersects(shown[i].label, shown[j].label)).toBe(false);
    }
  });

  it('l’ordre est total et ne dépend pas de l’ordre d’entrée', () => {
    const xs = [node('c', 0, 0, { depth: 2 }), node('a', 0, 0, { depth: 2 }), node('b', 0, 0, { depth: 1 })];
    expect([...xs].sort(comparePriority).map((x) => x.id)).toEqual(['b', 'a', 'c']);
    expect([...xs].reverse().sort(comparePriority).map((x) => x.id)).toEqual(['b', 'a', 'c']);
  });
});

describe('wrapLabel', () => {
  // Police fictive : 7 px par caractère.
  const measure = (s: string) => s.length * 7;

  it('un nom court tient sur une ligne', () => {
    expect(wrapLabel('Drifterra', 150, measure)).toEqual(['Drifterra']);
  });

  it('un nom long passe sur deux lignes, entier', () => {
    const lines = wrapLabel('GitHub Claude Co-Author Scraper', 150, measure);
    expect(lines).toEqual(['GitHub Claude', 'Co-Author Scraper']);
    expect(lines.every((l) => measure(l) <= 150)).toBe(true);
  });

  it('ce qui dépasse encore la seconde ligne est abrégé, jamais débordant', () => {
    const lines = wrapLabel('Suite de développement IA pour Unreal Engine', 150, measure);
    expect(lines).toHaveLength(2);
    expect(lines[1].endsWith('…')).toBe(true);
    expect(lines.every((l) => measure(l) <= 150)).toBe(true);
  });
});
