/**
 * sitePages.ts — les captures des pages de CE site, preuves de la carte maicivy du sommet.
 *
 * QUOI : pour une page servie par le backend (règle `site` de montagne_sommet.go : `page` +
 * `path`), le chemin de ses deux captures, téléphone et grand écran, rangées dans
 * `public/montagne/maicivy/<page>-{mobile,desktop}.webp`.
 * POURQUOI des fichiers du front et pas des images maiProFiles : les visuels maicivy de
 * maiProFiles sont d'anciennes captures de l'accueil et du CV, aucune du blog, des statistiques
 * git ni du chat. Ces captures-ci sont prises sur la PROD (Playwright, 393×851 @2x et 1440×900),
 * puis optimisées en webp — refaire la capture = remplacer le fichier, même nom.
 * COMMENT : le nom de fichier se déduit de l'identifiant de page, rien n'est listé deux fois ; un
 * test (sitePages.test.ts) vérifie que chaque page de la règle a bien ses deux fichiers.
 */

/** Pages couvertes par une capture — mêmes identifiants que la règle `site` du backend. */
export const SITE_PAGES = ['blog', 'gitstats', 'chat', 'cv'] as const;

export interface SiteCapture {
  /** Capture téléphone (portrait). */
  mobile: string;
  /** Capture grand écran (paysage). */
  desktop: string;
}

export function siteCapture(page: string): SiteCapture {
  return {
    mobile: `/montagne/maicivy/${page}-mobile.webp`,
    desktop: `/montagne/maicivy/${page}-desktop.webp`,
  };
}
