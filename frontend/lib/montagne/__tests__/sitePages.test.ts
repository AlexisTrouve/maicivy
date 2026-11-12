/**
 * sitePages.test.ts — chaque page preuve de la carte maicivy a ses DEUX captures dans public/.
 * Une capture manquante ferait un cadre cassé dans la fleur : on le voit ici, pas en prod.
 */

import { existsSync, statSync } from 'fs';
import path from 'path';

import { SITE_PAGES, siteCapture } from '../sitePages';

const PUBLIC = path.join(__dirname, '../../../public');

describe('captures des pages du site', () => {
  it('les quatre pages de la règle `site` (montagne_sommet.go), dans le même ordre', () => {
    expect([...SITE_PAGES]).toEqual(['blog', 'gitstats', 'chat', 'cv']);
  });

  it.each(SITE_PAGES.map((p) => [p]))('%s : capture téléphone et grand écran présentes, en webp, non vides', (page) => {
    const { mobile, desktop } = siteCapture(page);
    for (const src of [mobile, desktop]) {
      expect(src).toMatch(/\.webp$/);
      const file = path.join(PUBLIC, src);
      expect(existsSync(file)).toBe(true);
      expect(statSync(file).size).toBeGreaterThan(5_000);
    }
  });
});
