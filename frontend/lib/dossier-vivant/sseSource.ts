/**
 * sseSource.ts — la source RÉELLE du flux : un run de vrais agents, lu en SSE sur le backend Go
 * (POST /api/v1/dossier-vivant/run, route publique : tout visiteur, sans connexion).
 *
 * QUOI : `SseSource` implémente `AgentEventSource` (eventSource.ts) — l'écran ne sait pas s'il
 * regarde un rejeu ou un vrai run. `SseParser` découpe le texte reçu en événements du contrat.
 * POURQUOI fetch + ReadableStream et pas `EventSource` : le run se lance par un POST (le besoin
 * voyage dans le corps), ce qu'EventSource ne sait pas faire.
 * COMMENT le temps est porté : chaque événement est horodaté À LA RÉCEPTION, sur l'horloge du run
 * (`clock.now()`, cf. events.ts › StampedEvent) — jamais par le serveur, dont l'horloge décalée
 * ferait sauter les animations. Les événements d'un même paquet réseau partagent leur instant.
 * Les échecs de transport (refus, coupure) deviennent un `run.failed` nommé, émis par la source :
 * un seul chemin d'affichage pour tout arrêt, et un flux qui se ferme sans `run.completed` ne
 * laisse jamais l'écran « au travail » pour toujours.
 */

import { RUN_FAILURE_REASONS, type AgentEvent, type AgentEventType, type RunFailureReason, type StampedEvent } from './events';
import type { AgentEventSource } from './eventSource';

/** Ce que le visiteur envoie : son besoin, « Nom, rôle chez organisation », la langue de l'interface. */
export interface LiveRequest {
  need: string;
  who: string;
  lang: string;
}

export const LIVE_RUN_PATH = '/api/v1/dossier-vivant/run';

/** Les types du contrat : un `type` inconnu (serveur plus récent que la page) est ignoré. */
const KNOWN = new Set<AgentEventType>([
  'run.started', 'agent.status', 'message', 'tool.call', 'agent.log', 'index.filtered', 'project.verdict',
  'link.traced', 'usage', 'dossier.opened', 'dossier.section', 'dossier.block', 'run.completed', 'run.failed',
]);
const REASONS = new Set<string>(RUN_FAILURE_REASONS);

/**
 * Découpe incrémentale d'un flux SSE (champ `data`, trames séparées par une ligne vide). Les
 * paquets réseau coupent n'importe où, y compris au milieu d'un caractère ou d'une ligne : seul
 * ce qui forme une trame COMPLÈTE sort.
 */
export class SseParser {
  private buffer = '';
  private data: string[] = [];

  feed(chunk: string): AgentEvent[] {
    this.buffer += chunk;
    const out: AgentEvent[] = [];
    // Un `\r` final peut être la moitié d'un `\r\n` : il attend le paquet suivant.
    const cut = this.buffer.endsWith('\r') ? this.buffer.length - 1 : this.buffer.length;
    const lines = this.buffer.slice(0, cut).replace(/\r\n?/g, '\n').split('\n');
    this.buffer = lines.pop()! + this.buffer.slice(cut);
    for (const line of lines) {
      if (line === '') {
        const ev = this.dispatch();
        if (ev) out.push(ev);
      } else if (line.startsWith('data:')) {
        this.data.push(line.slice(line.startsWith('data: ') ? 6 : 5));
      }
      // `event:`, `id:`, commentaires `:` — le type est déjà dans `data`, rien d'autre ne sert.
    }
    return out;
  }

  private dispatch(): AgentEvent | null {
    if (this.data.length === 0) return null;
    const raw = this.data.join('\n');
    this.data = [];
    try {
      const ev = JSON.parse(raw) as AgentEvent;
      if (!ev || typeof ev !== 'object' || !KNOWN.has(ev.type)) return null;
      // Une raison d'arrêt hors contrat (serveur d'une autre version pendant un déploiement, ex.
      // l'ancien `budget`) reste un arrêt : affiché comme panne de service, jamais sans libellé.
      if (ev.type === 'run.failed' && !REASONS.has(ev.reason)) return { type: 'run.failed', reason: 'upstream' };
      return ev;
    } catch {
      return null; // trame illisible : ignorée, la suite du flux reste valable
    }
  }
}

/** Statut HTTP d'un refus → raison affichée (la route est publique : seul 400 vient de la saisie). */
function refusal(status: number): RunFailureReason {
  return status === 400 ? 'invalid' : 'upstream';
}

const isTerminal = (e: AgentEvent) => e.type === 'run.completed' || e.type === 'run.failed';

export class SseSource implements AgentEventSource {
  constructor(
    private readonly url: string,
    private readonly request: LiveRequest,
    private readonly clock: { now(): number },
    private readonly fetchImpl: typeof fetch = (...args) => fetch(...args),
  ) {}

  open(sink: (batch: readonly StampedEvent[]) => void): () => void {
    const abort = new AbortController();
    let ended = false;
    const deliver = (events: AgentEvent[]) => {
      if (events.length === 0 || ended) return;
      const at = this.clock.now();
      sink(events.map((event) => ({ at, event })));
      ended = events.some(isTerminal);
    };
    const fail = (reason: RunFailureReason) => {
      if (!abort.signal.aborted) deliver([{ type: 'run.failed', reason }]);
    };

    void (async () => {
      let res: Response;
      try {
        res = await this.fetchImpl(this.url, {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify(this.request),
          signal: abort.signal,
        });
      } catch {
        fail('network');
        return;
      }
      if (!res.ok || !res.body) {
        fail(refusal(res.status));
        return;
      }
      const reader = res.body.getReader();
      const decoder = new TextDecoder();
      const parser = new SseParser();
      try {
        for (;;) {
          const { done, value } = await reader.read();
          if (done) break;
          deliver(parser.feed(decoder.decode(value, { stream: true })));
        }
      } catch {
        // Lecture interrompue : arrêt volontaire (abort) ou coupure — `fail` tranche.
      }
      // Flux fermé sans remise ni arrêt nommé : la connexion a été perdue.
      fail('network');
    })();

    return () => abort.abort();
  }
}
