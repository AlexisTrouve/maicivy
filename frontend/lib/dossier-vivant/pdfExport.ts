/**
 * pdfExport.ts — « Télécharger le dossier » côté navigateur, et les sections que lit la route des
 * compétences. Pur, aucun DOM : testé.
 *
 * QUOI : le PDF du dossier est mis en page A4 et imprimé par le backend (Chromium headless, même
 * moteur que le CV). Le navigateur ne lui envoie QUE des identifiants et des choix fermés
 * (`DossierPdfQuery`) : la langue, et pour un run réel l'id de la copie que le serveur a gardée
 * (`run.completed.dossierId`). Il n'envoie plus jamais le contenu.
 * POURQUOI : le document porte le nom et les coordonnées d'Alexis Trouvé et ses routes sont
 * publiques. Quand le navigateur envoyait le dossier replié, n'importe qui pouvait faire imprimer
 * n'importe quel texte sous ce nom. Le serveur imprime désormais sa copie : le dossier du run qu'il
 * a exécuté, ou le scénario Tessera tel qu'il le connaît (copies verrouillées par
 * __tests__/serverCopies.test.ts, mise en page du schéma portée en Go au nombre près).
 * Ce qui reste ici : `pdfSections` (les sections repliées que lit la route des compétences de
 * l'écran) et `printDiagram` (la mise en page imprimée dont le port Go est la copie).
 */

import { portfolioImageUrl } from '@/lib/portfolio-api';

import { SECTION_IDS, type SectionId } from './events';
import { layoutPipeline, PIPE } from './pipeline';
import type { RunState } from './runState';

/**
 * Largeur utile d'une page A4 en pixels CSS : 210 mm − 2 × 16 mm de marges, à 96 px par pouce.
 * Plancher plus bas qu'à l'écran : sur papier, 0,8 donne encore des titres de 8 pt.
 */
export const PRINT_FIT = { maxWidth: 672, minScale: 0.8 } as const;

/**
 * Chemins des deux routes, publiques toutes deux : le rejeu (le PDF porte sa mention) et le run
 * réel qu'un visiteur vient de lancer.
 */
export const REPLAY_PDF_PATH = '/api/v1/dossier-vivant/pdf';
export const LIVE_PDF_PATH = '/api/v1/dossier-vivant/run/pdf';
/** La section « Compétences mobilisées » à l'écran : mêmes sections en entrée, même sélection (Go). */
export const SKILLS_PATH = '/api/v1/dossier-vivant/skills';

export interface PdfDiagram {
  label: string;
  journal: string;
  /** Dimensions du schéma (unités du viewBox), marge autour, échelle d'impression. */
  width: number;
  height: number;
  margin: number;
  scale: number;
  journalY: number;
  journalH: number;
  boxH: number;
  boxes: { x: number; y: number; w: number; title: string; subtitle: string; human: boolean }[];
  edges: { d: string; head: string; dashed: boolean }[];
  steps: { n: number; x: number; y: number }[];
}

export type PdfBlock =
  | { type: 'lead' | 'paragraph' | 'item'; text: string }
  | { type: 'diagram'; diagram: PdfDiagram }
  | { type: 'proof'; projectId: string; why: string; demo?: { kind: 'video' | 'site'; url: string } }
  | { type: 'also'; projectIds: string[] }
  | { type: 'contact' };

/**
 * La section des compétences, telle que la rend le backend (services/dossier_skills.go) : chaque
 * compétence du profil mobilisée par au moins une preuve, avec ses preuves (`projects`, ids) ; `need`
 * = son nom figure dans le besoin ou l'architecture. Rien n'est choisi côté front.
 */
export interface DossierSkills {
  skills: { name: string; level: string; need: boolean; projects: string[] }[];
  technologies: { name: string; projects: string[] }[];
}

/** Les sections du dossier, blocs repliés : ce que lit la route des compétences (et la golden du schéma). */
export type PdfSection = { id: SectionId; title: string; blocks: PdfBlock[] };

/**
 * Le corps des routes publiques du PDF : rien d'imprimable. `dossierId` (run réel) désigne la copie
 * serveur ; le rejeu n'en a pas (le serveur connaît son scénario).
 */
export interface DossierPdfQuery {
  lang: string;
  dossierId?: string;
}

/**
 * Échec nommé d'une demande de PDF : `code` est celui du backend (`DOSSIER_NOT_FOUND` : dossier
 * inconnu ou expiré ; `PDF_BUSY`, `PDF_FAILED`…), ou `HTTP_<statut>` sans corps lisible.
 */
export class DossierPdfError extends Error {
  constructor(readonly code: string) {
    super(`dossier pdf: ${code}`);
  }
}

/**
 * Géométrie d'impression du schéma : la mise en page de l'écran, à la largeur d'une page A4.
 * Portée en Go (backend services/dossier_print_layout.go) : le PDF public est mis en page par le
 * serveur ; les deux sont tenues égales par la golden de serverCopies.test.ts.
 */
export function printDiagram(block: { nodes: Parameters<typeof layoutPipeline>[0]; edges: Parameters<typeof layoutPipeline>[1]; journal: string; label: string }): PdfDiagram {
  const L = layoutPipeline(block.nodes, block.edges, PRINT_FIT);
  const edges = L.parts.flatMap((p) => p.edges);
  return {
    label: block.label,
    journal: block.journal,
    width: L.width,
    height: L.height,
    margin: PIPE.margin,
    scale: L.scale,
    journalY: L.journalY,
    journalH: PIPE.journalH,
    boxH: PIPE.boxH,
    boxes: [...L.boxes.values()].map((b) => ({ x: b.x, y: b.y, w: b.w, title: b.title, subtitle: b.subtitle, human: b.human })),
    edges: edges.map((e) => ({ d: e.d, head: e.head, dashed: e.dashed })),
    steps: edges.flatMap((e) => (e.step ? [e.step] : [])),
  };
}

/**
 * Les sections du dossier, blocs repliés pour le backend. Partagées par le PDF et la route des
 * compétences : l'écran et le document lisent les compétences dans les mêmes sections.
 */
export function pdfSections(state: RunState, demoUrl: (demo: { kind: 'video' | 'site'; url: string }) => string): PdfSection[] {
  const dossier = state.dossier!;
  return SECTION_IDS.map((id) => {
    const sec = dossier.sections[id];
    const blocks = sec.blocks.map(({ block: b }): PdfBlock => {
      switch (b.type) {
        case 'pipeline':
          return { type: 'diagram', diagram: printDiagram(b) };
        case 'proof':
          // Lien ABSOLU : un PDF n'a pas d'origine contre laquelle résoudre un chemin relatif.
          return { type: 'proof', projectId: b.projectId, why: b.why, demo: b.demo ? { kind: b.demo.kind, url: demoUrl(b.demo) } : undefined };
        default:
          return b;
      }
    });
    return { id, title: sec.title, blocks };
  });
}

/**
 * Lien de démo ABSOLU : la vidéo est servie par maiProFiles, le site par l'origine de la page. Un
 * PDF n'a pas d'origine contre laquelle résoudre un chemin relatif, et le backend n'accepte que
 * http(s).
 */
export function absoluteDemoUrl(demo: { kind: 'video' | 'site'; url: string }, origin: string): string {
  return new URL(demo.kind === 'video' ? portfolioImageUrl(demo.url) : demo.url, origin).href;
}

/** Demande la section des compétences pour ces sections ; toute réponse non 200 est une erreur. */
export async function fetchDossierSkills(url: string, body: { lang: string; sections: PdfSection[] }, signal?: AbortSignal): Promise<DossierSkills> {
  const res = await fetch(url, { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(body), signal });
  if (!res.ok) throw new Error(`dossier skills: ${res.status}`);
  return (await res.json()) as DossierSkills;
}

/** Nom de fichier : le gabarit traduit, réduit à des lettres, chiffres et tirets. */
export function pdfFileName(base: string): string {
  const slug = base
    .normalize('NFD')
    .replace(/\p{M}+/gu, '')
    .toLowerCase()
    .replace(/[^\p{L}\p{N}]+/gu, '-')
    .replace(/^-+|-+$/g, '');
  return `${slug || 'dossier'}.pdf`;
}

/**
 * Demande le PDF et le rend ; toute autre réponse est une DossierPdfError nommée (jamais un faux
 * fichier) — l'écran dit laquelle (dossier expiré ≠ panne).
 */
export async function fetchDossierPdf(url: string, query: DossierPdfQuery): Promise<Blob> {
  const res = await fetch(url, { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(query) });
  if (res.ok && (res.headers.get('Content-Type') ?? '').includes('application/pdf')) return res.blob();
  const body = (await res.json().catch(() => null)) as { code?: unknown } | null;
  throw new DossierPdfError(typeof body?.code === 'string' ? body.code : `HTTP_${res.status}`);
}
