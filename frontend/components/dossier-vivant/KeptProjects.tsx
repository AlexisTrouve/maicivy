'use client';

import { reveal } from '@/lib/dossier-vivant/anim';
import { CH } from '@/lib/dossier-vivant/choreography';
import type { RunState } from '@/lib/dossier-vivant/runState';
import type { Rect, SceneNode } from '@/lib/dossier-vivant/scene';

import { ProjectMark } from './ProjectMark';
import styles from './dossier-vivant.module.css';
import type { OpenProject } from './useProjectFlower';

/**
 * Marge de frappe au-dessus et au-dessous du bandeau (px) : la pastille garde ses 28 px de la
 * maquette, mais son bouton en fait 44 (28 + 2 × 8) — la cible tactile minimale. Le bandeau
 * déborde de 8 px sur le bas de la montagne (de la brume, non cliquable sur téléphone) et vient
 * juste toucher l'équipe (COMPACT_KEPT_H = 36 = 28 + 8, scene.ts) : rien ne bouge à l'écran.
 */
const HIT_PAD = 8;

/**
 * KeptProjects — téléphone seulement : le bandeau des projets RETENUS, sous la montagne.
 *
 * QUOI : une pastille or par projet retenu (icône du dépôt ou monogramme, + nom), la plus récente en tête, qui
 * apparaît à l'instant de son verdict — au moment même où sa tuile s'allume sur la montagne.
 * POURQUOI : sur un téléphone la montagne entière tient dans ~370 px de large ; ses tuiles font
 * une douzaine de pixels et leurs noms seraient illisibles. On voit QUE des cartes s'allument ; le
 * bandeau dit LESQUELLES. Le grand écran n'en a pas besoin (les noms y sont sous les tuiles).
 * COMMENT : dérivé de `state.verdicts` (dernier verdict = « keep »), aucun état propre ; un
 * projet repris puis écarté sort du bandeau comme sa tuile s'éteint.
 * CLIC : chaque pastille est un bouton qui ouvre la fleur du projet (sur téléphone, c'est elle qui
 * remplace la tuile, trop petite pour être touchée).
 * E2E : `dv-kept`, `dv-kept-chip` (`data-kept-id`, pas `data-project-id` : ce dernier désigne
 * la tuile ou la carte de la montagne, que l'E2E retrouve de façon unique) — c'est le bouton.
 */
export function KeptProjects({
  state,
  now,
  nodes,
  box,
  label,
  onOpen,
  canOpen,
}: {
  state: RunState;
  now: number;
  nodes: ReadonlyMap<string, SceneNode>;
  box: Rect;
  label: string;
  onOpen: OpenProject;
  canOpen: (id: string) => boolean;
}) {
  const kept = [...state.verdicts]
    .filter(([, v]) => v.verdict === 'keep')
    .sort((a, b) => b[1].at - a[1].at)
    .map(([id, v]) => ({ id, at: v.at, node: nodes.get(id) }));
  return (
    <ul className={styles.kept} style={{ left: box.x, top: box.y - HIT_PAD, width: box.w, height: box.h + 2 * HIT_PAD }} aria-label={label} data-testid="dv-kept">
      {kept.map(({ id, at, node }) => (
        <li key={id} style={reveal(now, at, CH.blockReveal, 0)}>
          <button
            type="button"
            className={styles.keptChip}
            disabled={!canOpen(id)}
            aria-haspopup="dialog"
            onClick={(e) => onOpen(id, e.currentTarget)}
            data-testid="dv-kept-chip"
            data-kept-id={id}
          >
            <span className={styles.keptPill}>
              <ProjectMark node={node} monoClass={styles.keptMono} iconClass={styles.keptIcon} />
              <span>{node?.name ?? id}</span>
            </span>
          </button>
        </li>
      ))}
    </ul>
  );
}
