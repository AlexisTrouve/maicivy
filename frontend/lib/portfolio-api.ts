/**
 * portfolio-api.ts — accès au portfolio CURÉ.
 *
 * La donnée vient de maiProFiles (`data/book.yaml`), la couche de curation : 25 projets choisis,
 * avec leur récit, leur image, leurs badges et leurs chiffres. C'est la MÊME source qui produit le
 * PDF du portfolio book — le site en rend les pages, bookgen en rend le PDF, aucun des deux ne
 * recopie le contenu.
 *
 * Langues : les éditions curées sont en/fr/zh. Toute autre locale retombe sur l'anglais (jamais le
 * français), règle appliquée côté backend Go — le front passe simplement sa locale.
 */

import { api } from './api';

/** Une image de la curation. `url` est la route servie par maiProFiles, la seule utilisable ici. */
export interface PortfolioImage {
  src: string;
  alt: string;
  portrait: boolean;
  url: string;
}

/** Pastille de stack, avec la classe de couleur décidée à la curation. */
export interface PortfolioBadge {
  label: string;
  class: string;
}

/** Chiffre mis en avant : « 33 000+ » + « lignes Rust ». */
export interface PortfolioStat {
  value: string;
  label: string;
}

/** Un visuel de la galerie. Déjà résolu dans la langue demandée côté maiProFiles. */
export interface PortfolioGalleryImage {
  src: string;
  alt: string;
  caption: string;
  url: string;
  portrait: boolean;
}

/**
 * La démo vidéo d'un projet. `url` et `poster_url` sont des chemins maiProFiles (`/images/<id>`),
 * à préfixer par `portfolioImageUrl` comme les images. La légende arrive déjà dans la langue du livre.
 */
export interface PortfolioVideo {
  caption: string;
  url: string;
  poster_url: string;
}

export interface PortfolioProject {
  num: number;
  anchor: string;
  id: string;
  name: string;
  category: string;
  index_name: string;
  index_domain: string;
  index_stack: string;
  image: PortfolioImage;
  /** La galerie : la page A4 du livre n'en tient qu'une, le web les rend toutes. */
  images: PortfolioGalleryImage[];
  /** Démo vidéo, null quand le projet n'en déclare pas. Fichiers servis par maiProFiles. */
  video?: PortfolioVideo | null;
  paragraphs: string[];
  badges: PortfolioBadge[];
  stats: PortfolioStat[];
}

export interface PortfolioBook {
  lang: string;
  title: string;
  projects: PortfolioProject[];
}

/** La page complète : le récit curé ET la fiche technique, plus liens, statut et galerie. */
export interface PortfolioPage extends PortfolioProject {
  description_portfolio: string;
  /** Fiche technique rendue en HTML côté Go (goldmark) — le front n'a pas de moteur markdown. */
  description_portfolio_html: string;
  links: Record<string, string>;
  status: string;
  tags: string[];
  gallery: Array<{ id: string; filename?: string; alt?: string }>;
}

/**
 * Base des images : elles sont servies par maiProFiles, pas par maicivy.
 * Le backend rend un chemin relatif (`/portfolio-images/x.jpg`) — on le préfixe ici, côté rendu,
 * plutôt que de figer un domaine dans la donnée.
 */
const MAIPROFILES_URL =
  process.env.NEXT_PUBLIC_MAIPROFILES_URL || 'https://maiprofiles.etheryale.com';

export function portfolioImageUrl(url: string): string {
  if (!url) return '';
  if (url.startsWith('http')) return url;
  return `${MAIPROFILES_URL}${url}`;
}

// Pas de cache HTTP : la curation change rarement, mais quand elle change on veut la voir
// tout de suite — et le backend Go a déjà son cache de 5 minutes devant maiProFiles.
const NO_CACHE: RequestInit = { cache: 'no-store' };

export const portfolioApi = {
  getIndex: (lang?: string) =>
    api.get<PortfolioBook>('/api/v1/portfolio', lang ? { lang } : undefined, NO_CACHE),

  getPage: (id: string, lang?: string) =>
    api.get<PortfolioPage>(`/api/v1/portfolio/${id}`, lang ? { lang } : undefined, NO_CACHE),
};
