/**
 * data.ts — les données RÉELLES du dossier vivant, lues côté serveur sur les endpoints du site.
 *
 * QUOI :
 *  - `toSceneInput` : de la montagne (GET /api/v1/montagne) ne garde que ce que la scène dessine
 *    (ids, noms, monogrammes, strates, liens, sommet) — le reste (positions 3D, visuels, preuves)
 *    ne voyage pas jusqu'au navigateur — et y joint l'icône de chaque projet qui en a une ;
 *  - `getProjectIcons` : les projets dont le dépôt a un `icon.png` (GET /api/v1/project-icons) —
 *    facultatif : en cas d'échec, aucune icône, les monogrammes restent ;
 *  - `getProjectFacts` : statut, lignes et tests de chaque projet, lus sur les fiches maiProFiles
 *    via le backend (GET /api/v1/projects, mêmes chiffres que le CV) — c'est là que le tableau des
 *    preuves du dossier prend ses chiffres, jamais dans le flux des agents ;
 *  - `replayContext` : ce que le scénario rejoué emprunte au réel (nombre de projets, démo vidéo de
 *    MelodiCode servie par sa carte du sommet, adresse du site).
 */

import { api } from '@/lib/api';
import type { Montagne } from '@/lib/montagne-api';
import { portfolioImageUrl } from '@/lib/portfolio-api';

import type { SceneInput } from './scene';
import type { ReplayContext } from './tessera';

/** Fiche projet telle que la sert GET /api/v1/projects (sous-ensemble lu ici). */
interface ApiProject {
  sourceId: string;
  status: string;
  loc: number;
  testCount: number;
}

export interface ProjectFactsMap {
  [id: string]: { status: string; loc: number | null; tests: number | null };
}

/** id de projet → chemin de son icône sur l'API (`/api/v1/project-icons/<id>?v=…`). */
export type ProjectIcons = Readonly<Record<string, string>>;

export function toSceneInput(m: Montagne, icons: ProjectIcons = {}): SceneInput {
  return {
    nodes: m.nodes.map((n) => ({ id: n.id, name: n.name, monogram: n.monogram, stratum: n.stratum, ...(icons[n.id] ? { icon: icons[n.id] } : {}) })),
    links: m.links.map((l) => ({ from: l.from, to: l.to })),
    summit: m.summit.map((c) => ({ id: c.id, members: c.members })),
  };
}

export async function getProjectIcons(): Promise<ProjectIcons> {
  try {
    const res = await api.get<{ icons: Record<string, string> }>('/api/v1/project-icons', undefined, { cache: 'no-store' });
    return res.icons ?? {};
  } catch {
    // Une icône est un plus, jamais une condition : la page s'affiche avec les monogrammes.
    return {};
  }
}

export async function getProjectFacts(lang: string): Promise<ProjectFactsMap> {
  const res = await api.get<{ projects: ApiProject[] }>('/api/v1/projects', { lang }, { cache: 'no-store' });
  const facts: ProjectFactsMap = {};
  // Un zéro dans la fiche veut dire « non mesuré », pas « zéro ligne » : on l'affiche comme absent.
  for (const p of res.projects) facts[p.sourceId] = { status: p.status, loc: p.loc || null, tests: p.testCount || null };
  return facts;
}

export function replayContext(m: Montagne, locale: string): ReplayContext {
  const video = m.summit.find((c) => c.id === 'melodicode')?.proofs.find((p) => p.kind === 'video');
  return {
    projectCount: m.nodes.length,
    melodicodeVideoUrl: video ? portfolioImageUrl(video.url) : undefined,
    siteUrl: `/${locale}`,
  };
}
