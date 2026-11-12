'use client';

import type { MontagneNode, MontagneSummitCard } from '@/lib/montagne-api';

import { FlowerCard, flowerTitleId, useFlowerDialog } from './FlowerCard';
import { MontagneProofs } from './MontagneProofs';

/**
 * MontagneProofFlower — la fleur d'une CARTE DU SOMMET : la carte du projet, puis ses preuves.
 *
 * POURQUOI pas des pétales : une preuve a besoin d'une légende (quel jeu, quel étage, quelle
 * page) et d'une taille où on la lit ; des pétales carrés tournant autour de la carte ne portent
 * ni l'un ni l'autre sur un téléphone. Ici, une colonne qui défile DANS le calque (la page derrière
 * reste immobile), centrée et bornée en largeur sur grand écran.
 * Fermeture : bouton, Échap, ou toucher le fond (hors de la colonne).
 * E2E : mêmes testids que la fleur d'une tuile (`montagne-flower`, `data-project-id`),
 * `data-variant="proofs"`.
 */
export function MontagneProofFlower({ node, card, onClose }: { node: MontagneNode; card: MontagneSummitCard; onClose: () => void }) {
  const closeRef = useFlowerDialog(onClose);
  const closeOnBackdrop = (e: React.MouseEvent) => {
    if (e.target === e.currentTarget) onClose();
  };

  return (
    <div
      role="dialog"
      aria-modal="true"
      aria-labelledby={flowerTitleId(node)}
      data-testid="montagne-flower"
      data-project-id={node.id}
      data-variant="proofs"
      className="fixed inset-0 z-[300] overflow-y-auto overscroll-contain bg-[#04060b]/85 backdrop-blur-sm animate-in fade-in-0 duration-200"
      onClick={closeOnBackdrop}
    >
      <div className="mx-auto flex min-h-full w-full max-w-4xl flex-col items-center gap-5 px-4 py-6 md:py-12" onClick={closeOnBackdrop}>
        <FlowerCard node={node} onClose={onClose} closeRef={closeRef} className="relative w-full max-w-md" />
        <MontagneProofs card={card} node={node} />
      </div>
    </div>
  );
}
