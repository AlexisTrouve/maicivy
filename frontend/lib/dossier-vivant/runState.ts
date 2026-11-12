/**
 * runState.ts — repli du flux d'événements (events.ts) en ÉTAT d'écran. Pur, aucun DOM, testé.
 *
 * QUOI : `applyEvent(state, stamped)` rend un nouvel état ; `foldEvents` replie une liste. L'état
 * garde l'INSTANT de chaque fait (`at`) : les composants en tirent les animations (`now - at`),
 * ce qui rend l'écran fonction pure de (état, now) — rejouer, mettre en pause ou figer un instant
 * donne exactement le même rendu.
 * POURQUOI un réducteur et pas des minuteries par composant : le flux réel (SSE) arrivera dans un
 * ordre et à un rythme que le front ne choisit pas. Un repli unique est la seule façon d'avoir un
 * écran cohérent quel que soit ce rythme — et de le tester sans navigateur.
 * COMMENT : mises à jour immuables ciblées (seules les branches touchées sont copiées), pour que
 * React voie les changements et que les parties figées gardent leur identité.
 * Les sélecteurs en bas (`usageAt`, `totalsAt`, `elapsedAt`) sont les seuls calculs « continus »
 * qui ne dépendent d'aucune géométrie.
 */

import {
  SECTION_IDS,
  SUB_AGENTS,
  type AgentId,
  type AgentStatus,
  type DossierBlock,
  type LogKind,
  type Participant,
  type RunFailureReason,
  type SectionId,
  type SectionStatus,
  type StampedEvent,
  type SubAgentId,
  type VisitorBrief,
} from './events';

/** Cumul d'usage d'un agent à un instant. */
export interface Usage {
  tokens: number;
  costUsd: number;
}

/** Ligne de journal : les verdicts y entrent avec leur propre glyphe (keep / drop). */
export interface LogEntry {
  at: number;
  kind: LogKind | 'keep' | 'drop';
  text: string;
}

/** Où regarde le curseur d'un agent sur la montagne : le visiteur ou un projet. */
export type CursorTarget = { kind: 'visitor' } | { kind: 'project'; projectId: string };

export interface AgentView {
  model?: string;
  status: AgentStatus;
  activity?: string;
  /** Appel d'outil en cours d'affichage (tapé lettre à lettre depuis `at`) ; effacé à l'arrêt. */
  call: { text: string; at: number } | null;
  /** Cibles successives du curseur (montagne uniquement), dans l'ordre. */
  trail: { at: number; target: CursorTarget }[];
  /** Projet en cours de lecture : posé par get_project, levé par la note ou le verdict qui suit. */
  reading: string | null;
  log: LogEntry[];
  /** Usage précédent et courant, pour l'interpolation du compteur. */
  usage: { prev: Usage; cur: Usage; at: number };
  firstWorkAt: number | null;
  doneAt: number | null;
}

export interface MessageView {
  /**
   * Rang du message dans le run (0, 1, 2…) : son identité, la clé React du fil et des paquets.
   * POURQUOI pas (`at`, de, à) : un run réel horodate à la RÉCEPTION, et tous les événements d'un
   * même paquet réseau partagent leur instant — deux messages « orchestrateur → orchestrateur » du
   * même paquet (plan puis contrôle) avaient la même clé, et React fusionnait leurs lignes.
   */
  id: number;
  at: number;
  from: Participant;
  to: Participant;
  text: string;
  topic?: 'plan' | 'check';
}

export interface SectionView {
  title: string;
  waitingFor: SubAgentId | 'previous';
  status: SectionStatus;
  writingAt: number | null;
  doneAt: number | null;
  blocks: { at: number; block: DossierBlock }[];
}

export interface RunState {
  startedAt: number | null;
  /**
   * Sous-agents du run, dans l'ordre des colonnes : ceux qu'annonce `run.started`. Avant tout run,
   * tous (l'équipe est mesurée cachée, au montage). Le rejeu Tessera n'annonce pas le Dessinateur.
   */
  team: readonly SubAgentId[];
  visitor: VisitorBrief | null;
  completedAt: number | null;
  agents: Record<AgentId, AgentView>;
  messages: MessageView[];
  /** Filtrage de l'index : instant du balayage, candidats, taille de l'index. */
  filter: { at: number; candidates: ReadonlySet<string>; total: number } | null;
  /** Dernier verdict par projet. */
  verdicts: ReadonlyMap<string, { verdict: 'keep' | 'drop'; at: number }>;
  /** Filets tracés depuis le visiteur : projet → instant. */
  threads: ReadonlyMap<string, number>;
  /** Instants de « ping » par projet (retenu, cité en preuve, remise). */
  pings: ReadonlyMap<string, readonly number[]>;
  /** Première revue de l'orchestrateur (message `check`) : l'état du dossier passe à « relu ». */
  reviewAt: number | null;
  dossier: { openedAt: number; sections: Record<SectionId, SectionView> } | null;
  /** Arrêt du run avant la remise (run.failed) : instant et raison nommée. */
  failure: { at: number; reason: RunFailureReason } | null;
  /** Copie serveur du dossier remis (run réel) : l'identifiant que le PDF public demande. */
  dossierId: string | null;
}

const ZERO: Usage = { tokens: 0, costUsd: 0 };
const AGENT_IDS: AgentId[] = ['orchestrator', ...SUB_AGENTS];

function emptyAgent(): AgentView {
  return {
    status: 'waiting',
    call: null,
    trail: [],
    reading: null,
    log: [],
    usage: { prev: ZERO, cur: ZERO, at: 0 },
    firstWorkAt: null,
    doneAt: null,
  };
}

export function initialRunState(): RunState {
  return {
    startedAt: null,
    team: SUB_AGENTS,
    visitor: null,
    completedAt: null,
    agents: Object.fromEntries(AGENT_IDS.map((id) => [id, emptyAgent()])) as Record<AgentId, AgentView>,
    messages: [],
    filter: null,
    verdicts: new Map(),
    threads: new Map(),
    pings: new Map(),
    reviewAt: null,
    dossier: null,
    failure: null,
    dossierId: null,
  };
}

/** Copie l'état d'un agent avec un correctif. */
function patchAgent(s: RunState, id: AgentId, patch: (a: AgentView) => Partial<AgentView>): RunState {
  const a = s.agents[id];
  return { ...s, agents: { ...s.agents, [id]: { ...a, ...patch(a) } } };
}

function addPing(pings: ReadonlyMap<string, readonly number[]>, id: string, at: number): Map<string, readonly number[]> {
  const next = new Map(pings);
  next.set(id, [...(pings.get(id) ?? []), at]);
  return next;
}

function patchSection(s: RunState, id: SectionId, patch: (sec: SectionView) => Partial<SectionView>): RunState {
  if (!s.dossier) return s;
  const sec = s.dossier.sections[id];
  return { ...s, dossier: { ...s.dossier, sections: { ...s.dossier.sections, [id]: { ...sec, ...patch(sec) } } } };
}

export function applyEvent(s: RunState, { at, event: e }: StampedEvent): RunState {
  switch (e.type) {
    case 'run.started': {
      const base = initialRunState();
      const agents = { ...base.agents };
      for (const { id, model } of e.agents) agents[id] = { ...agents[id], model };
      const announced = new Set(e.agents.map((a) => a.id));
      return { ...base, startedAt: at, visitor: e.visitor, agents, team: SUB_AGENTS.filter((id) => announced.has(id)) };
    }
    case 'agent.status':
      return patchAgent(s, e.agent, (a) => ({
        status: e.status,
        activity: e.activity,
        // Un agent qui s'arrête ou attend n'a plus d'appel en cours : la ligne d'outil se vide.
        call: e.status === 'working' ? a.call : null,
        reading: e.status === 'working' ? a.reading : null,
        firstWorkAt: a.firstWorkAt ?? (e.status === 'working' ? at : null),
        doneAt: e.status === 'done' ? at : null,
      }));
    case 'message':
      return {
        ...s,
        // La liste ne fait que s'allonger (run.started repart d'un état vide) : sa longueur est un rang unique.
        messages: [...s.messages, { id: s.messages.length, at, from: e.from, to: e.to, text: e.text, topic: e.topic }],
        reviewAt: e.topic === 'check' && s.reviewAt === null ? at : s.reviewAt,
      };
    case 'tool.call':
      return patchAgent(s, e.agent, (a) => {
        const f = e.focus;
        const target: CursorTarget | null =
          f?.kind === 'project' ? { kind: 'project', projectId: f.projectId } : f?.kind === 'visitor' ? { kind: 'visitor' } : null;
        return {
          call: { text: e.call, at },
          trail: target ? [...a.trail, { at, target }] : a.trail,
          reading: f?.kind === 'project' ? f.projectId : null,
        };
      });
    case 'agent.log':
      // Écrire une note = avoir fini de lire la fiche en cours.
      return patchAgent(s, e.agent, (a) => ({ log: [...a.log, { at, kind: e.kind, text: e.text }], reading: null }));
    case 'index.filtered':
      return { ...s, filter: { at, candidates: new Set(e.candidates), total: e.total } };
    case 'project.verdict': {
      const verdicts = new Map(s.verdicts);
      verdicts.set(e.projectId, { verdict: e.verdict, at });
      const next: RunState = { ...s, verdicts, pings: e.verdict === 'keep' ? addPing(s.pings, e.projectId, at) : s.pings };
      return patchAgent(next, e.agent, (a) => ({
        log: e.note ? [...a.log, { at, kind: e.verdict, text: e.note }] : a.log,
        reading: a.reading === e.projectId ? null : a.reading,
      }));
    }
    case 'link.traced': {
      const threads = new Map(s.threads);
      if (!threads.has(e.to)) threads.set(e.to, at);
      return { ...s, threads };
    }
    case 'usage':
      return patchAgent(s, e.agent, (a) => ({
        usage: { prev: usageOf(a, at), cur: { tokens: e.inputTokens + e.outputTokens, costUsd: e.costUsd }, at },
      }));
    case 'dossier.opened': {
      const sections = Object.fromEntries(
        SECTION_IDS.map((id) => {
          const plan = e.sections.find((p) => p.id === id);
          const view: SectionView = {
            title: plan?.title ?? '',
            waitingFor: plan?.waitingFor ?? 'previous',
            status: 'waiting',
            writingAt: null,
            doneAt: null,
            blocks: [],
          };
          return [id, view];
        }),
      ) as Record<SectionId, SectionView>;
      return { ...s, dossier: { openedAt: at, sections } };
    }
    case 'dossier.section':
      return patchSection(s, e.section, (sec) => ({
        status: e.status,
        writingAt: e.status === 'writing' ? (sec.writingAt ?? at) : sec.writingAt,
        doneAt: e.status === 'done' ? at : sec.doneAt,
      }));
    case 'dossier.block': {
      const next = patchSection(s, e.section, (sec) => ({ blocks: [...sec.blocks, { at, block: e.block }] }));
      // Un projet cité en preuve « pinge » sur la montagne : on voit d'où vient la ligne écrite.
      return e.block.type === 'proof' ? { ...next, pings: addPing(next.pings, e.block.projectId, at) } : next;
    }
    case 'run.completed': {
      // Dernière respiration : tous les projets retenus et le visiteur pingent ensemble.
      let pings = new Map(s.pings);
      for (const [id, v] of s.verdicts) if (v.verdict === 'keep') pings = addPing(pings, id, at + BREATH_DELAY);
      pings = addPing(pings, 'visitor', at + BREATH_DELAY);
      return { ...s, completedAt: at, pings, dossierId: e.dossierId ?? null };
    }
    case 'run.failed':
      // Le premier arrêt fait foi (la source peut ajouter `network` quand le flux se ferme ensuite).
      return s.failure ? s : { ...s, failure: { at, reason: e.reason } };
  }
}

/** Délai entre la remise et la dernière respiration de la montagne (s). */
export const BREATH_DELAY = 0.2;

export function foldEvents(events: readonly StampedEvent[], state: RunState = initialRunState()): RunState {
  return events.reduce(applyEvent, state);
}

/* --------------------------------------------------------------------------------------------
   Sélecteurs continus
   -------------------------------------------------------------------------------------------- */

/**
 * Durée de l'interpolation d'un compteur : un événement `usage` ne fait pas SAUTER le compteur, il
 * le fait courir jusqu'à sa nouvelle valeur. Le rejeu émet un relevé toutes les 0,5 s : le compteur
 * court donc en continu, comme dans la maquette.
 */
export const USAGE_TWEEN = 0.5;

function usageOf(a: AgentView, now: number): Usage {
  const { prev, cur, at } = a.usage;
  const k = Math.min(1, Math.max(0, (now - at) / USAGE_TWEEN));
  return { tokens: prev.tokens + (cur.tokens - prev.tokens) * k, costUsd: prev.costUsd + (cur.costUsd - prev.costUsd) * k };
}

/** Usage affiché d'un agent à l'instant `now`. */
export function usageAt(s: RunState, agent: AgentId, now: number): Usage {
  return usageOf(s.agents[agent], now);
}

/** Totaux de l'équipe : somme des compteurs de tous les agents. */
export function totalsAt(s: RunState, now: number): Usage {
  return AGENT_IDS.reduce<Usage>(
    (acc, id) => {
      const u = usageAt(s, id, now);
      return { tokens: acc.tokens + u.tokens, costUsd: acc.costUsd + u.costUsd };
    },
    { tokens: 0, costUsd: 0 },
  );
}

/** Temps de travail de l'équipe (s) : du premier agent au travail jusqu'à la remise. */
export function elapsedAt(s: RunState, now: number): number {
  const starts = AGENT_IDS.map((id) => s.agents[id].firstWorkAt).filter((x): x is number => x !== null);
  if (starts.length === 0) return 0;
  const end = s.completedAt ?? now;
  return Math.max(0, Math.min(now, end) - Math.min(...starts));
}

/** Instant où tous les sous-agents de recherche ont fini (le dossier monte alors sur leurs colonnes). */
export function researchDoneAt(s: RunState): number | null {
  const ids: SubAgentId[] = ['reader', 'researcher', 'architect'];
  const done = ids.map((id) => s.agents[id].doneAt);
  return done.every((d): d is number => d !== null) ? Math.max(...done) : null;
}
