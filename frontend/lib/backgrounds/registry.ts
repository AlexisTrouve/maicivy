import type { BackgroundManifestEntry } from './types';

/**
 * Manifeste des fonds animés disponibles.
 *
 * QUOI : liste légère { id, name, mobile, load } de chaque fond branchable.
 * POURQUOI léger : peuplé dans le sélecteur sans charger les déps lourdes (Three.js, shaders).
 *   Seul le fond effectivement choisi est import()é via `load()` → bundle initial intact.
 * COMMENT : ajouter un fond = (1) créer son plugin sous ./plugins/<id>.ts qui default-exporte
 *   un BackgroundInitFn, (2) ajouter une entrée ici. Rien d'autre à toucher (host + sélecteur
 *   se mettent à jour automatiquement depuis ce manifeste).
 */
export const BACKGROUNDS: BackgroundManifestEntry[] = [
  {
    id: 'constellation',
    name: 'Constellation',
    mobile: false, // Three.js trop coûteux sur petit écran — skip mobile
    load: () => import('./plugins/constellation'),
  },
  {
    id: 'conwaygol',
    name: 'Game of Life',
    mobile: true, // Canvas 2D léger — OK sur mobile (et dans le pool aléatoire mobile)
    load: () => import('./plugins/conwaygol'),
  },
  {
    id: 'fractal',
    name: 'Fractal',
    mobile: false, // shader WebGL lourd — skip mobile pour v1
    load: () => import('./plugins/fractal'),
  },
];

// IDs spéciaux du sélecteur (pas des plugins de rendu).
export const BG_RANDOM = 'random'; // tirage au sort à chaque visite
export const BG_NONE = 'none'; // aucun fond animé

// Retourne l'entrée correspondant à un id de plugin, ou undefined si inconnu/spécial.
export function getBackground(id: string): BackgroundManifestEntry | undefined {
  return BACKGROUNDS.find((b) => b.id === id);
}

// Vrai si l'id est une valeur de préférence valide (plugin connu, random, ou none).
export function isValidPreference(id: string | null | undefined): boolean {
  return !!id && (id === BG_RANDOM || id === BG_NONE || !!getBackground(id));
}

/**
 * Tire un fond concret au hasard dans le pool éligible.
 * COMMENT : filtre selon la capacité mobile, puis pick uniforme. Renvoie BG_NONE si le pool
 *   est vide (ex. mobile sans aucun fond mobile-capable) — échec franc, pas de fond cassé.
 */
export function pickRandomBackground(isMobile: boolean): string {
  const pool = BACKGROUNDS.filter((b) => (isMobile ? b.mobile : true));
  if (pool.length === 0) return BG_NONE;
  return pool[Math.floor(Math.random() * pool.length)].id;
}
