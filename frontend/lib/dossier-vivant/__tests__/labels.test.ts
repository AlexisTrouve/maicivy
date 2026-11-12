/**
 * Noms de la montagne (labels.ts) : place offerte par les voisins nommés, lignes permises par le
 * rang du dessous. Les noms eux-mêmes (chevauchements, lignes, mots coupés) sont mesurés dans le
 * navigateur par l'E2E dossier-vivant.spec.ts.
 */
import { LABEL, labelFit, labelLines, labelWidths } from '../labels';
import { G } from '../scene';

/** Un rang de `n` tuiles au pas de la géométrie grand écran, à la hauteur `y`. */
const row = (n: number, y: number, prefix: string) =>
  Array.from({ length: n }, (_, i) => ({ id: `${prefix}${i}`, x: i * G.cell, y, w: G.tile, h: G.tile }));

describe('labelWidths', () => {
  const tiles = [...row(5, 0, 'a'), ...row(5, G.rowHeight, 'b')];

  it('un nom seul sur son rang prend la largeur maximale', () => {
    expect(labelWidths(tiles, new Set(['a2'])).get('a2')).toBe(LABEL.maxWidth);
  });

  it('deux voisins nommés se partagent l’espace : leurs plaques centrées ne se touchent pas', () => {
    const w = labelWidths(tiles, new Set(['a1', 'a2']));
    expect(w.get('a1')).toBe(G.cell - LABEL.gap);
    // Demi-plaque + demi-plaque + écart ≤ distance entre centres.
    expect(w.get('a1')! / 2 + w.get('a2')! / 2 + LABEL.gap).toBeLessThanOrEqual(G.cell);
  });

  it('un voisin nommé d’un AUTRE rang ne réduit rien (le nom loge sous sa tuile, au-dessus du rang suivant)', () => {
    expect(labelWidths(tiles, new Set(['a2', 'b2'])).get('a2')).toBe(LABEL.maxWidth);
  });

  it('une tuile muette reçoit la largeur maximale (son nom n’apparaît qu’au survol)', () => {
    expect(labelWidths(tiles, new Set(['a1', 'a2'])).get('a3')).toBe(LABEL.maxWidth);
  });
});

describe('labelLines / labelFit', () => {
  it('deux lignes tiennent entre la tuile et le rang du dessous au grand écran (police ~15 unités)', () => {
    const font = 15;
    expect(labelLines(font)).toBe(2);
    expect(G.tile + LABEL.top + LABEL.padY + 2 * font * LABEL.lineHeight).toBeLessThan(G.rowHeight);
  });

  it('montagne petite à l’écran (police grossie dans le repère) : une seule ligne', () => {
    expect(labelLines(24)).toBe(1);
  });

  it('une plaque étroite (voisins nommés) reste sur une ligne : aucun mot long ne la déborde', () => {
    expect(labelFit(G.cell - LABEL.gap, 15)).toBe(1);
    expect(labelFit(LABEL.maxWidth, 15)).toBe(2);
  });
});
