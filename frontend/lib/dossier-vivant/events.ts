/**
 * events.ts — le CONTRAT du flux d'événements d'agents du « dossier vivant ».
 *
 * QUOI : la liste fermée des événements qu'une équipe d'agents émet pendant qu'elle transforme le
 * besoin d'un visiteur en dossier de conception. Règle de conception : UN ÉVÉNEMENT = UNE ACTION
 * VISIBLE à l'écran (un message passe dans le fil de l'orchestrateur, un appel d'outil s'écrit dans
 * une colonne, une carte s'allume, un filet se trace, un bloc du dossier s'écrit, un compteur de
 * tokens avance). Aucun événement « technique » sans effet visible : s'il ne se voit pas, il n'a
 * rien à faire dans le flux.
 *
 * POURQUOI un contrat explicite : deux producteurs émettent ce même flux.
 *  1. AUJOURD'HUI — le REJEU (lib/dossier-vivant/tessera.ts) : un scénario enregistré, horodaté,
 *     joué par ReplaySource. Aucun appel IA.
 *  2. le backend Go à vrais agents (backend/internal/dossiervivant, miroir Go de ce fichier), en
 *     SSE (`event: <type>` / `data: <JSON de l'événement>`), lu par SseSource (sseSource.ts).
 *     Le front ne change pas : seule la source change (cf. eventSource.ts).
 * Le front ne connaît donc ni modèle, ni tarif, ni prompt : le nom de modèle et le coût VIENNENT
 * du flux (`run.started.agents[].model`, `usage.costUsd`). Rien n'est en dur côté front.
 *
 * COMMENT le temps est porté : chaque événement voyage dans une enveloppe `StampedEvent` dont `at`
 * est l'instant de réception en secondes depuis le début du run, MESURÉ PAR LE CLIENT :
 *  - rejeu : `at` = instant programmé du scénario (rendu déterministe, seekable — `?t=33`) ;
 *  - SSE : `at` = horloge locale à la réception (une horloge serveur décalée ferait sauter les
 *    animations, qui partent toutes de `at`).
 * L'écran est une fonction PURE de (événements reçus, instant courant) : runState.ts les replie en
 * état, les composants en déduisent chaque animation (`now - at`).
 *
 * Identifiants : les agents forment une équipe FIXE (AgentId) — leurs noms sont de l'INTERFACE
 * (next-intl, 5 langues). Les TEXTES produits par les agents (activité, notes, dossier) sont du
 * CONTENU : ils sortent dans la langue de l'agent, pas dans celle de l'interface.
 */

/**
 * L'équipe : un orchestrateur et ses sous-agents spécialisés. `diagrammer` (le Dessinateur) est un
 * AJOUT au contrat : l'agent que l'Architecte lance par son outil `draw_schema` pour dessiner le
 * schéma, sur un autre modèle (annoncé par `run.started`). Il ne parle qu'à l'Architecte.
 */
export type AgentId = 'orchestrator' | 'reader' | 'researcher' | 'architect' | 'diagrammer' | 'writer';

/**
 * Les sous-agents, dans l'ordre de leurs colonnes. Un run n'en affiche que ceux qu'annonce son
 * `run.started` (le rejeu Tessera n'a pas de Dessinateur : quatre colonnes ; un run réel, cinq).
 */
export const SUB_AGENTS = ['reader', 'researcher', 'architect', 'diagrammer', 'writer'] as const satisfies readonly AgentId[];
export type SubAgentId = (typeof SUB_AGENTS)[number];

/** Interlocuteurs d'un message du fil : le visiteur ou un agent. */
export type Participant = AgentId | 'visitor';

/** Le visiteur dont le besoin lance le run. */
export interface VisitorBrief {
  /** Nom affiché sous sa tuile et dans le fil (« Inès Morel »). */
  name: string;
  /** Fonction et organisation, telles qu'il les a écrites (« CTO chez Tessera »). */
  affiliation: string;
  /** Organisation seule, seconde ligne de sa tuile (« Tessera »). */
  organisation: string;
  /** Deux lettres de sa tuile (« IM »). */
  initials: string;
  /** Le besoin, tel qu'il l'a tapé. */
  need: string;
}

/** État d'un agent : la ligne d'état de sa colonne (glyphe + libellé). */
export type AgentStatus = 'waiting' | 'working' | 'done';

/** Sur quoi porte l'attention d'un agent pendant un appel d'outil : pilote son curseur. */
export type ToolFocus =
  | { kind: 'project'; projectId: string } // le curseur va sur la carte/tuile du projet (lecture)
  | { kind: 'visitor' } // le curseur travaille à côté de la tuile du visiteur
  | { kind: 'dossier' }; // pas de curseur sur la montagne : l'agent écrit le dossier

/** Nature d'une ligne du journal d'un agent : choisit le glyphe et la teinte. */
export type LogKind = 'note' | 'question' | 'done';

/** Les cinq sections du dossier, dans l'ordre de lecture. */
export const SECTION_IDS = ['need', 'architecture', 'proofs', 'contribution', 'next'] as const;
export type SectionId = (typeof SECTION_IDS)[number];

export type SectionStatus = 'waiting' | 'writing' | 'done';

/** Un nœud du schéma d'architecture, posé sur une grille (colonne, rangée). */
export interface PipelineNode {
  id: string;
  title: string;
  subtitle: string;
  col: number;
  row: number;
  /** Étape tenue par un humain : bordure claire au lieu de la bordure neutre. */
  human?: boolean;
}

export interface PipelineEdge {
  from: string;
  to: string;
  /** Chemin d'exception (escalade) : tracé en pointillé. */
  dashed?: boolean;
  /** Numéro de garde-fou porté par l'arête (pastille dorée). */
  step?: number;
}

/** Un bloc du dossier. Chaque bloc écrit = un événement `dossier.block`. */
export type DossierBlock =
  /** Phrase d'ouverture d'une section, tapée lettre à lettre. */
  | { type: 'lead'; text: string }
  /** Paragraphe courant, tapé lettre à lettre. */
  | { type: 'paragraph'; text: string }
  /** Puce : apparaît d'un coup (fondu). */
  | { type: 'item'; text: string }
  /** Schéma d'architecture, dessiné élément par élément. `journal` = bande basse (trace). */
  | { type: 'pipeline'; nodes: PipelineNode[]; edges: PipelineEdge[]; journal: string; label: string }
  /**
   * Une ligne du tableau des preuves. Statut, lignes et tests NE SONT PAS dans l'événement : le
   * front les lit sur la fiche réelle du projet (backend → maiProFiles) — un agent ne peut pas
   * inventer un chiffre. `demo` : lien public éventuel (vidéo, site en ligne).
   */
  | { type: 'proof'; projectId: string; why: string; demo?: { kind: 'video' | 'site'; url: string } }
  /** « Aussi retenus : … » — noms lus sur la montagne réelle. */
  | { type: 'also'; projectIds: string[] }
  /** Le bouton de contact de la prochaine étape. */
  | { type: 'contact' };

/** Le flux. Union discriminée par `type`. */
export type AgentEvent =
  /**
   * Le besoin est envoyé : le champ se replie en tuile du visiteur, la montagne glisse à gauche,
   * l'équipe entre. `agents[].model` : identifiant lisible du modèle, AFFICHÉ tel quel s'il est
   * fourni (le rejeu n'en fournit pas : il ne prétend à aucun modèle).
   */
  | { type: 'run.started'; visitor: VisitorBrief; agents: { id: AgentId; model?: string }[] }
  /** Ligne d'état d'un agent. `activity` = ce qu'il fait, écrit par lui (contenu). */
  | { type: 'agent.status'; agent: AgentId; status: AgentStatus; activity?: string }
  /** Un message passe dans le fil de l'orchestrateur (et un paquet dans l'éventail s'il va d'un agent à l'autre). */
  | { type: 'message'; from: Participant; to: Participant; text: string; topic?: 'plan' | 'check' }
  /** Appel d'outil : s'écrit sous l'en-tête de la colonne, déplace le curseur de l'agent. */
  | { type: 'tool.call'; agent: AgentId; call: string; focus?: ToolFocus }
  /** Une ligne du journal de l'agent. */
  | { type: 'agent.log'; agent: SubAgentId; kind: LogKind; text: string }
  /** L'index est filtré : une ligne de balayage parcourt la montagne, les candidats se teintent. */
  | { type: 'index.filtered'; agent: SubAgentId; total: number; candidates: string[] }
  /**
   * Verdict sur un projet : `keep` l'allume en or (et le fait « pinger »), `drop` le rend au rang.
   * `note` (facultatif) s'ajoute au journal de l'agent avec le glyphe du verdict.
   */
  | { type: 'project.verdict'; agent: SubAgentId; projectId: string; verdict: 'keep' | 'drop'; note?: string }
  /** Un filet d'or se trace de la tuile du visiteur vers un projet retenu. */
  | { type: 'link.traced'; from: 'visitor'; to: string }
  /** Compteurs CUMULÉS d'un agent. Le coût est calculé par le producteur (le front ne connaît aucun tarif). */
  | { type: 'usage'; agent: AgentId; inputTokens: number; outputTokens: number; costUsd: number }
  /** La feuille du dossier apparaît, avec le plan de ses sections et ce que chacune attend. */
  | { type: 'dossier.opened'; sections: { id: SectionId; title: string; waitingFor: SubAgentId | 'previous' }[] }
  /** Une section change d'état (« en cours d'écriture », écrite). */
  | { type: 'dossier.section'; section: SectionId; status: SectionStatus }
  /** Un bloc s'écrit dans une section. */
  | { type: 'dossier.block'; section: SectionId; block: DossierBlock }
  /** Le dossier est remis : bouton de téléchargement, dernière respiration de la montagne. */
  | {
      type: 'run.completed';
      /**
       * AJOUT au contrat — run réel seulement : l'identifiant de la copie que le serveur garde du
       * dossier remis (24 h). C'est TOUT ce que le front renvoie pour obtenir le PDF : le document
       * imprime la copie du serveur, jamais l'état replié du navigateur. Absent : le rejeu (son PDF
       * est le scénario que le serveur connaît), ou une copie que le serveur n'a pas pu garder.
       */
      dossierId?: string;
    }
  /**
   * Le run s'arrête avant la remise, sur une VRAIE erreur — l'arrêt se VOIT (barre du haut, ✕).
   * Raisons du serveur : `model` (sortie inutilisable : tronquée, JSON invalide), `upstream`
   * (passerelle ou données). Raisons posées par la source SSE elle-même (le serveur n'a pas pu le
   * dire) : `invalid` (400, saisie hors bornes), `network` (connexion perdue, flux coupé sans fin).
   * Ni la consommation ni la durée d'un run ne l'arrêtent plus (le serveur rappelle le temps aux
   * agents, il ne coupe rien) : il n'y a donc ni raison `budget` ni raison `timeout`.
   * La route du run est publique et sans verrou : ni refus de session, ni « run déjà en cours ».
   */
  | { type: 'run.failed'; reason: RunFailureReason };

export const RUN_FAILURE_REASONS = ['model', 'upstream', 'invalid', 'network'] as const;
export type RunFailureReason = (typeof RUN_FAILURE_REASONS)[number];

export type AgentEventType = AgentEvent['type'];

/** Enveloppe : l'événement et son instant de réception (s depuis le début du run, horloge client). */
export interface StampedEvent {
  at: number;
  event: AgentEvent;
}
