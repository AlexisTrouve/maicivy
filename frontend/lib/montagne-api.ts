/**
 * montagne-api.ts — la montagne des projets, telle que la sert le backend Go (GET /api/v1/montagne).
 *
 * Tout est calculé côté serveur : strates, liens « sert à construire », positions 3D, monogrammes,
 * visuels, cartes du sommet et leurs preuves. Le front ne fait que lire et dessiner — aucune
 * position ni preuve n'est inventée ici.
 * Contrat : backend/internal/services/montagne_service.go et montagne_sommet.go (types Montagne*).
 */

import { api } from './api';

/** Identifiants stables des strates, du bas vers le haut. Servent aussi de clés i18n. */
export type StratumId = 'foundations' | 'tooling' | 'engines' | 'works';

export interface MontagneStratum {
  id: StratumId;
  /** 0 = en bas. */
  level: number;
  y_min: number;
  y_max: number;
  /** Rayon du cône au pied de la bande (calculé côté Go, seule définition). */
  radius: number;
  count: number;
}

/** Un pétale de la fleur : chemin servi par maiProFiles, à préfixer par portfolioImageUrl. */
export interface MontagneVisual {
  kind: 'image' | 'video';
  url: string;
  poster_url?: string;
  alt: string;
  /** Image SVG : servie telle quelle, l'optimiseur de Next refuse le SVG. */
  vector?: boolean;
}

export interface MontagneNode {
  id: string;
  /** Nom court (identité), dans la langue demandée. */
  name: string;
  /** Mini-description ; vide quand elle n'est pas traduite dans cette langue. */
  mini: string;
  status: string;
  stratum: StratumId;
  /** Deux lettres affichées tant que `icon_url` est vide. */
  monogram: string;
  /** Icône définitive (Eikon, à venir). Vide = monogramme. */
  icon_url: string;
  /** A une page /portfolio/[id]. */
  curated: boolean;
  visuals: MontagneVisual[];
  x: number;
  y: number;
  z: number;
}

/** D'où une carte du sommet tire ses preuves (montagne_sommet.go › RegleSommet). */
export type SummitRule = 'built_games' | 'video' | 'stages' | 'site';

/**
 * Une preuve d'une carte du sommet. `kind: 'page'` = une page de CE site (url vide : la capture
 * est un fichier du front, cf. lib/montagne/sitePages.ts).
 */
export interface MontagneProof {
  kind: 'image' | 'video' | 'page';
  url: string;
  poster_url?: string;
  vector?: boolean;
  alt: string;
  /** Projet qui porte la preuve (jeu bâti sur le moteur, étage de la chaîne). */
  project_id?: string;
  project_name?: string;
  curated?: boolean;
  /** Règle `stages` : étage de la chaîne. */
  stage?: 'global' | 'local';
  /** Règle `site` : identifiant de la page et chemin sans la langue. */
  page?: string;
  path?: string;
}

export interface MontagneSummitCard {
  id: string;
  rule: SummitRule;
  /** De la meilleure à la moins bonne : l'ordre de la fleur (la couverture est une illustration, lib/montagne/art.ts). */
  proofs: MontagneProof[];
  /** Projets absorbés par la carte : plus de tuile, et leurs filets partent de la carte. */
  members: string[];
}

/** `from` sert à construire `to`. */
export interface MontagneLink {
  from: string;
  to: string;
  kind: 'stack' | 'description';
}

/**
 * Le cône sur lequel reposent les projets (montagne_layout.go) : rayon à la hauteur y =
 * slope × (apex_y − y). Le pied est à y = 0. Sert à dessiner la silhouette et les courbes de niveau.
 */
export interface MontagneShape {
  apex_y: number;
  slope: number;
}

export interface Montagne {
  lang: string;
  strata: MontagneStratum[];
  shape: MontagneShape;
  nodes: MontagneNode[];
  links: MontagneLink[];
  /** Cartes du sommet, dans leur ordre d'affichage, avec leurs preuves (montagne_sommet.go). */
  summit: MontagneSummitCard[];
}

/**
 * Lecture côté serveur (Server Component). `no-store` : le backend a déjà son cache Redis
 * stale-while-revalidate devant maiProFiles ; un second cache ici retarderait une recuration
 * sans rien protéger de plus.
 */
export function getMontagne(lang: string): Promise<Montagne> {
  return api.get<Montagne>('/api/v1/montagne', { lang }, { cache: 'no-store' });
}
