/**
 * looks.ts — l'apparence d'un projet de la montagne à un instant : son état visible et son recul.
 * Pur, testé.
 *
 * QUOI : `projectLook` rend
 *  - `look` — 'idle' (au repos), 'candidate' (retenu par le filtrage, teinte machine), 'reading'
 *    (un agent lit sa fiche), 'lit' (retenu : or), 'dropped' (écarté, rendu au rang). Porté tel quel
 *    par l'attribut `data-state` des tuiles et cartes (E2E) ;
 *  - `quiet` — de 0 à 1 : combien le projet recule (opacité ½ au plus). Après le filtrage, tout ce
 *    qui n'est pas en jeu recule pour que les projets en jeu se lisent ; un candidat écarté y
 *    retourne en douceur.
 * POURQUOI hors du composant : c'est la règle du « qui s'allume quand », celle qu'Alexis a validée
 * dans la maquette ; elle se teste sans navigateur.
 */

import { prog } from './anim';
import { CH } from './choreography';
import type { RunState } from './runState';

export type ProjectLookName = 'idle' | 'candidate' | 'reading' | 'lit' | 'dropped';

export interface ProjectLook {
  look: ProjectLookName;
  quiet: number;
}

/**
 * @param sweepPos position du projet dans le balayage (0 = haut de la montagne, 1 = bas) : la ligne
 *   de balayage le « touche » à cet avancement, et c'est là qu'il se teinte s'il est candidat.
 */
export function projectLook(s: RunState, id: string, now: number, sweepPos: number): ProjectLook {
  const verdict = s.verdicts.get(id);
  if (verdict?.verdict === 'keep') return { look: 'lit', quiet: 0 };

  const reading = Object.values(s.agents).some((a) => a.reading === id);
  const f = s.filter;
  if (!f) return { look: reading ? 'reading' : 'idle', quiet: 0 };

  const quiet = prog(now, f.at + CH.sweepD, CH.quietD);
  const isCandidate = f.candidates.has(id);
  if (!isCandidate) return { look: reading ? 'reading' : 'idle', quiet };

  if (verdict?.verdict === 'drop') return { look: 'dropped', quiet: prog(now, verdict.at, CH.dropD) * quiet };
  if (reading) return { look: 'reading', quiet: 0 };
  const swept = now >= f.at + CH.sweepD * sweepPos;
  return { look: swept ? 'candidate' : 'idle', quiet: 0 };
}
