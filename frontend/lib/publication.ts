import type { Metadata } from 'next';

/**
 * publication.ts — LE point de bascule des pages en ligne mais pas encore ouvertes au public.
 *
 * QUOI : une page déployée n'est pas forcément publiée. Tant que son drapeau est `false`, elle
 * porte `<meta name="robots" content="noindex, nofollow">` (unpublishedRobots, lu par son
 * generateMetadata) ET reste hors du sitemap (app/sitemap.ts lit publishedRoutes()). Les deux
 * décisions partent du même drapeau : impossible d'en ouvrir une sans l'autre.
 * POURQUOI : une page démontrée en privé (lien envoyé à la main) ne doit pas apparaître dans Google
 * avant qu'Alexi ne la décide publique — et une page au sitemap mais en noindex est un signal
 * contradictoire que la Search Console remonte en erreur.
 * COMMENT l'ouvrir : passer son drapeau à `true`. Rien d'autre à toucher (le test
 * app/__tests__/publication.test.ts suit le drapeau).
 */
export const PUBLICATION = {
  // /[locale]/dossier-vivant — démo des agents ; fermée tant qu'Alexi ne l'a pas ouverte.
  'dossier-vivant': false,
} as const;

export type GatedRoute = keyof typeof PUBLICATION;

/** robots d'une page à bascule : noindex/nofollow tant qu'elle n'est pas publiée, sinon rien (défaut du site). */
export function unpublishedRobots(route: GatedRoute): Metadata['robots'] | undefined {
  return PUBLICATION[route] ? undefined : { index: false, follow: false };
}

/** Les routes à bascule déjà publiées — à ajouter au sitemap. */
export function publishedRoutes(): GatedRoute[] {
  return (Object.keys(PUBLICATION) as GatedRoute[]).filter((r) => PUBLICATION[r]);
}
