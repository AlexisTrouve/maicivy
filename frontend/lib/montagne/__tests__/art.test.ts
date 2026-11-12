/**
 * art.test.ts — chaque illustration de la montagne existe dans public/, en webp, dans son budget.
 * Un fichier manquant ferait une carte sans couverture ou une page sans fond : on le voit ici, pas
 * en prod. Les budgets sont ceux de scripts/montagne-art.mjs (le fond est la première image de la
 * page sur téléphone).
 */

import { existsSync, statSync } from 'fs';
import path from 'path';

import { BACKDROP, SUMMIT_ART_IDS, summitCover } from '../art';

const PUBLIC = path.join(__dirname, '../../../public');
const KB = 1024;

/** Taille du fichier public derrière un chemin servi. */
function sizeOf(src: string): number {
  const file = path.join(PUBLIC, src);
  expect(existsSync(file)).toBe(true);
  return statSync(file).size;
}

describe('illustrations de la montagne', () => {
  it('les quatre cartes du sommet (montagne_sommet.go), dans le même ordre', () => {
    expect([...SUMMIT_ART_IDS]).toEqual(['grove_engine', 'melodicode', 'theomen', 'maicivy']);
  });

  it.each(SUMMIT_ART_IDS.map((id) => [id]))('%s : couverture webp, non vide, sous 200 Ko', (id) => {
    const src = summitCover(id)!;
    expect(src).toMatch(/\.webp$/);
    const size = sizeOf(src);
    expect(size).toBeGreaterThan(5_000);
    expect(size).toBeLessThan(200 * KB);
  });

  it('une carte hors de la liste n’a pas d’illustration (elle garde son icône)', () => {
    expect(summitCover('drifterra')).toBeNull();
  });

  it.each([['mobile', BACKDROP.mobile], ['desktop', BACKDROP.desktop]])('fond %s : webp, non vide, sous 350 Ko', (_name, src) => {
    expect(src).toMatch(/\.webp$/);
    const size = sizeOf(src);
    expect(size).toBeGreaterThan(5_000);
    expect(size).toBeLessThan(350 * KB);
  });

  it('les quatre couvertures sont des fichiers distincts', () => {
    const sizes = SUMMIT_ART_IDS.map((id) => sizeOf(summitCover(id)!));
    expect(new Set(sizes).size).toBe(SUMMIT_ART_IDS.length);
  });
});
