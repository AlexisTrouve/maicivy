/**
 * eventSource.ts — d'où vient le flux d'événements, et l'horloge du run.
 *
 * QUOI : `AgentEventSource`, le seul point de contact entre l'écran et ceux qui produisent le flux ;
 * `ReplaySource`, la source de CETTE étape (scénario enregistré) ; `RunClock`, l'horloge du run.
 * POURQUOI une interface : à l'étape suivante, une source SSE (backend Go, vrais agents) prendra
 * la place de ReplaySource sans qu'aucun composant ne change. Elle n'aura qu'à : ouvrir un
 * EventSource, parser `data` en AgentEvent, l'horodater à la RÉCEPTION avec `clock.now()`
 * (cf. events.ts › StampedEvent) et le passer au `sink`.
 * COMMENT le rejeu reste déterministe : l'horloge peut être mise en pause, accélérée (`rate`,
 * utilisé par l'E2E) ou placée à un instant précis (`?t=33`, captures) ; ReplaySource livre, à
 * chaque image, tous les événements dont l'instant programmé est passé — avec CET instant, pas
 * l'heure de livraison, si bien qu'un saut à t=33 donne exactement l'écran d'un run vécu jusqu'à 33 s.
 */

import type { StampedEvent } from './events';

/** Producteur du flux. `open` commence la livraison et rend la fonction qui l'arrête. */
export interface AgentEventSource {
  open(sink: (batch: readonly StampedEvent[]) => void): () => void;
}

/** Horloge d'un run : secondes depuis le lancement, pausable, à vitesse réglable. */
export class RunClock {
  private origin = 0;
  private frozenAt: number | null = 0;

  constructor(private readonly rate = 1) {}

  /** Instant courant du run (s). */
  now(): number {
    return this.frozenAt ?? ((performance.now() - this.origin) / 1000) * this.rate;
  }

  get playing(): boolean {
    return this.frozenAt === null;
  }

  /** Repart de l'instant `t` (et joue). */
  start(t = 0): void {
    this.origin = performance.now() - (t / this.rate) * 1000;
    this.frozenAt = null;
  }

  pause(): void {
    this.frozenAt = this.now();
  }

  resume(): void {
    this.start(this.now());
  }

  /** Fige l'horloge à l'instant `t` (captures, mouvement réduit). */
  freeze(t: number): void {
    this.frozenAt = t;
  }
}

/** Rejeu d'un scénario enregistré : livre chaque événement quand l'horloge atteint son instant. */
export class ReplaySource implements AgentEventSource {
  constructor(
    private readonly events: readonly StampedEvent[],
    private readonly clock: RunClock,
  ) {}

  open(sink: (batch: readonly StampedEvent[]) => void): () => void {
    let next = 0;
    let frame = 0;
    const pump = () => {
      const t = this.clock.now();
      const start = next;
      while (next < this.events.length && this.events[next].at <= t) next++;
      if (next > start) sink(this.events.slice(start, next));
      if (next < this.events.length) frame = requestAnimationFrame(pump);
    };
    pump();
    return () => cancelAnimationFrame(frame);
  }
}
