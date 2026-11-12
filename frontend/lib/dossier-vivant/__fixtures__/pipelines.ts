/**
 * Schémas d'architecture RÉELS pour les tests : celui du rejeu Tessera (lu dans le scénario) et
 * celui du run réel du 2026-10 (copié du flux brut live-run.ndjson, bloc `pipeline` de
 * l'Architecte, au caractère près) — le cas qui rendait le schéma illisible.
 */
import type { PipelineEdge, PipelineNode } from '../events';
import { tesseraScenario } from '../tessera';

export const LIVE_PIPELINE: { nodes: PipelineNode[]; edges: PipelineEdge[] } = {
  nodes: [
    { id: 'recu', title: 'Réception', subtitle: 'Pièce déposée', col: 0, row: 0 },
    { id: 'lecture', title: 'Extraction LLM', subtitle: 'LLM lit la pièce', col: 1, row: 0 },
    { id: 'controle', title: 'Contrôle', subtitle: 'Règles certaines', col: 2, row: 0 },
    { id: 'saisie', title: 'Préparation', subtitle: 'Brouillon saisie', col: 3, row: 0 },
    { id: 'valid', title: 'Validation', subtitle: 'Collaborateur OK', col: 4, row: 0, human: true },
    { id: 'anomalie', title: 'Anomalie', subtitle: 'Traitement manuel', col: 2, row: 1, human: true },
  ],
  edges: [
    { from: 'recu', to: 'lecture' },
    { from: 'lecture', to: 'controle', step: 1 },
    { from: 'controle', to: 'saisie', step: 2 },
    { from: 'controle', to: 'anomalie', dashed: true },
    { from: 'saisie', to: 'valid', step: 3 },
  ],
};

/**
 * Schéma du run réel du 2026-11 (schema-run.ndjson, Cabinet Lumen), au caractère près : un RETOUR
 * en pointillés sur la même rangée (Validation → Brouillon) — celui qui passait sur la pastille ③.
 */
export const SCHEMA_PIPELINE: { nodes: PipelineNode[]; edges: PipelineEdge[] } = {
  nodes: [
    { id: 'pieces', title: 'Pièces reçues', subtitle: 'Code, 300/jour', col: 0, row: 0 },
    { id: 'lecture', title: 'Lecture pièce', subtitle: 'Modèle, JSON', col: 1, row: 0 },
    { id: 'controle', title: 'Dossier client', subtitle: 'Code Python', col: 2, row: 0 },
    { id: 'brouillon', title: 'Brouillon saisie', subtitle: 'Code, non envoyé', col: 3, row: 0 },
    { id: 'validation', title: 'Validation', subtitle: 'Collaborateur', col: 4, row: 0, human: true },
    { id: 'revue', title: 'File de revue', subtitle: 'Collaborateur', col: 2, row: 1, human: true },
  ],
  edges: [
    { from: 'pieces', to: 'lecture' },
    { from: 'lecture', to: 'controle', step: 1 },
    { from: 'controle', to: 'brouillon', step: 2 },
    { from: 'brouillon', to: 'validation', step: 3 },
    { from: 'controle', to: 'revue', dashed: true },
    { from: 'validation', to: 'brouillon', dashed: true },
  ],
};

export function tesseraPipeline(): { nodes: PipelineNode[]; edges: PipelineEdge[] } {
  const s = tesseraScenario({ projectCount: 75, siteUrl: '/fr' });
  const e = s.events.find((x) => x.event.type === 'dossier.block' && x.event.block.type === 'pipeline')!.event;
  if (e.type !== 'dossier.block' || e.block.type !== 'pipeline') throw new Error('pas de schéma dans le scénario');
  return { nodes: e.block.nodes, edges: e.block.edges };
}

/** Largeur offerte au schéma dans la demi-colonne du dossier, à 1920×1080 (mesurée). */
export const PANEL_WIDTH = 399;
