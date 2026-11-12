// blogHref — URL d'une page de la liste du blog.
//
// QUOI : /<locale>/blog, avec ?theme= et ?page= seulement quand ils diffèrent du défaut.
// POURQUOI : la pagination et le filtre sont de VRAIS liens (rendus côté serveur) : Google suit
// chaque page et indexe tous les articles, et une URL partagée rouvre exactement la même vue.
// La page 1 sans filtre garde l'URL nue /<locale>/blog, pour ne pas créer de doublon indexé.
export function blogHref(locale: string, opts: { page?: number; theme?: string | null } = {}): string {
  const params = new URLSearchParams();
  if (opts.theme) params.set('theme', opts.theme);
  if (opts.page && opts.page > 1) params.set('page', String(opts.page));
  const qs = params.toString();
  return `/${locale}/blog${qs ? `?${qs}` : ''}`;
}

// pagesVisibles : numéros à afficher autour de la page courante — toujours la première et la
// dernière, deux voisines de chaque côté, et null pour une ellipse. 12 pages restent lisibles
// sur mobile sans aligner douze boutons.
export function pagesVisibles(page: number, total: number): (number | null)[] {
  const out: (number | null)[] = [];
  for (let p = 1; p <= total; p++) {
    if (p === 1 || p === total || Math.abs(p - page) <= 2) {
      out.push(p);
    } else if (out[out.length - 1] !== null) {
      out.push(null);
    }
  }
  return out;
}
