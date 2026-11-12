'use client';

import { easeIO, lerp, prog } from '@/lib/dossier-vivant/anim';
import { CH } from '@/lib/dossier-vivant/choreography';
import type { SubAgentId } from '@/lib/dossier-vivant/events';
import type { CursorTarget, RunState } from '@/lib/dossier-vivant/runState';
import { G, anchorOf, type Scene } from '@/lib/dossier-vivant/scene';

import type { MountainMotion } from './MountainScene';
import styles from './dossier-vivant.module.css';

/** Place de chaque agent quand il travaille « à côté » de la tuile du visiteur (px écran). */
const VISITOR_SLOT: Partial<Record<SubAgentId, [number, number]>> = {
  reader: [34, 6],
  researcher: [-150, 6],
  architect: [150, 6],
};
const CURSOR_AGENTS = Object.keys(VISITOR_SLOT) as SubAgentId[];
/**
 * Téléphone : les places autour du visiteur se resserrent. La montagne y est ~4 fois plus petite ;
 * à ±150 px, le Chercheur et l'Architecte sortiraient de l'écran.
 */
const COMPACT_SLOT_SCALE = 0.4;

/**
 * AgentCursors — les curseurs des agents qui parcourent la montagne (bleu acier : la machine).
 *
 * QUOI : un curseur étiqueté par agent qui a une cible sur la montagne (`tool.call` avec un `focus`
 * projet ou visiteur). Il glisse vers chaque nouvelle cible, s'efface peu après que l'agent a fini.
 * Le Rédacteur n'en a pas : il travaille sur le dossier, pas sur la montagne.
 * COMMENT : en px de la SCÈNE (taille réelle, pas mise à l'échelle avec la montagne) ; la position
 * suit la montagne pendant qu'elle glisse, via `motion`.
 */
export function AgentCursors({
  scene,
  state,
  now,
  motion,
  names,
  compact,
}: {
  scene: Scene;
  state: RunState;
  now: number;
  motion: MountainMotion;
  names: Record<SubAgentId, string>;
  compact: boolean;
}) {
  const slot = compact ? COMPACT_SLOT_SCALE : 1;
  const toScreen = (id: string): [number, number] => {
    const [ax, ay] = anchorOf(scene, id);
    return [motion.x + ax * motion.s, motion.y + ay * motion.s];
  };
  const place = (agent: SubAgentId, target: CursorTarget): [number, number] => {
    if (target.kind === 'visitor') {
      const [vx, vy] = toScreen('visitor');
      const [ox, oy] = VISITOR_SLOT[agent] ?? [0, 0];
      return [vx + ox * slot, vy + oy * slot];
    }
    const [ax, ay] = toScreen(target.projectId);
    // Sur une carte, le curseur se pose au-dessus de son bas ; sur une tuile, sur son coin bas-droit.
    const dy = scene.isCard.has(target.projectId) ? -40 * motion.s : 0.3 * G.tile * motion.s;
    return [ax + 0.3 * G.tile * motion.s, ay + dy];
  };

  return (
    <>
      {CURSOR_AGENTS.map((id) => {
        const a = state.agents[id];
        if (a.trail.length === 0) return null;
        const first = a.trail[0].at;
        const hideAt = a.doneAt === null ? Infinity : a.doneAt + CH.cursorHide;
        const vis = Math.min(prog(now, first - CH.cursorFade, CH.cursorFade), 1 - prog(now, hideAt, CH.cursorFade));
        if (vis <= 0) return null;
        let [x, y] = place(id, a.trail[0].target);
        for (let i = 1; i < a.trail.length; i++) {
          const step = a.trail[i];
          if (now < step.at) break;
          const k = easeIO(prog(now, step.at, CH.cursorMove));
          const [tx, ty] = place(id, step.target);
          x = lerp(x, tx, k);
          y = lerp(y, ty, k);
        }
        return (
          <div
            key={id}
            className={styles.cursor}
            style={{ transform: `translate(${x.toFixed(1)}px, ${y.toFixed(1)}px)`, opacity: vis }}
            aria-hidden="true"
            data-testid="dv-cursor"
            data-agent={id}
          >
            <svg viewBox="0 0 16 18">
              <path d="M1.5 1.5l12 9-5.2.7 2.9 5.6-2.1 1-2.9-5.6-4.7 3.4z" />
            </svg>
            <span className={styles.cursorTag}>{names[id]}</span>
          </div>
        );
      })}
    </>
  );
}
