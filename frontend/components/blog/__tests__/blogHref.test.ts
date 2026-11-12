import { blogHref, pagesVisibles } from '../blogHref';

describe('blogHref', () => {
  it('page 1 sans filtre = URL nue (pas de doublon indexé)', () => {
    expect(blogHref('fr')).toBe('/fr/blog');
    expect(blogHref('fr', { page: 1 })).toBe('/fr/blog');
  });
  it('porte le thème et la page quand ils diffèrent du défaut', () => {
    expect(blogHref('en', { page: 3 })).toBe('/en/blog?page=3');
    expect(blogHref('fr', { theme: 'games', page: 2 })).toBe('/fr/blog?theme=games&page=2');
    expect(blogHref('fr', { theme: null, page: 1 })).toBe('/fr/blog');
  });
});

describe('pagesVisibles', () => {
  it('peu de pages : toutes', () => {
    expect(pagesVisibles(1, 4)).toEqual([1, 2, 3, 4]);
  });
  it('beaucoup de pages : extrémités, voisines, ellipses', () => {
    expect(pagesVisibles(1, 12)).toEqual([1, 2, 3, null, 12]);
    expect(pagesVisibles(6, 12)).toEqual([1, null, 4, 5, 6, 7, 8, null, 12]);
    expect(pagesVisibles(12, 12)).toEqual([1, null, 10, 11, 12]);
  });
});
