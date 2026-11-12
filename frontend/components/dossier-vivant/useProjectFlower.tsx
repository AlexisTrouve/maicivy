'use client';

import { useCallback, useMemo, useRef, useState, type ReactNode } from 'react';

import { MontagneFlower } from '@/components/montagne/MontagneFlower';
import { MontagneProofFlower } from '@/components/montagne/MontagneProofFlower';
import type { MontagneNode, MontagneSummitCard } from '@/lib/montagne-api';

/** Ce que la fleur d'un projet lit de la montagne : les nœuds complets et les cartes du sommet. */
export interface FlowerProjects {
  nodes: MontagneNode[];
  summit: MontagneSummitCard[];
}

/** Ouvre la fleur du projet `id` ; `trigger` reprend le focus à la fermeture. */
export type OpenProject = (id: string, trigger: HTMLElement) => void;

/**
 * useProjectFlower — la fleur d'un projet, ouverte depuis le dossier vivant (tuile ou carte de la
 * montagne, projet retenu, ligne de preuve du dossier).
 *
 * QUOI : un état (le projet ouvert) et le calque à rendre. Une CARTE du sommet ouvre la fleur de
 * ses preuves (MontagneProofFlower), une tuile celle de ses pétales (MontagneFlower) — même règle
 * que MontagneExplorer sur /montagne. La fleur porte le bouton vers /portfolio/<id> pour un projet
 * curé : c'est le « second clic ».
 * POURQUOI réutiliser les fleurs de la montagne : le dossier vivant montre les MÊMES projets ; une
 * seconde fiche dériverait de la première (contenu, testids, comportement de dialogue).
 * COMMENT : la fleur est un calque fixe hors de la scène ; l'ouvrir ne touche ni à l'horloge ni à
 * la source du run — un rejeu ou un run réel (SSE) continue derrière. Échap, fond et bouton
 * ferment (useFlowerDialog) ; le focus revient à l'élément qui l'a ouverte.
 * `canOpen` : un projet cité par un run réel mais absent de la montagne n'a pas de fleur — son
 * élément reste du texte, pas un bouton qui ne ferait rien.
 */
export function useProjectFlower(projects: FlowerProjects): { open: OpenProject; canOpen: (id: string) => boolean; flower: ReactNode } {
  const byId = useMemo(() => new Map(projects.nodes.map((n) => [n.id, n])), [projects.nodes]);
  const cards = useMemo(() => new Map(projects.summit.map((c) => [c.id, c])), [projects.summit]);
  const [selected, setSelected] = useState<MontagneNode | null>(null);
  const triggerRef = useRef<HTMLElement | null>(null);

  const open = useCallback<OpenProject>(
    (id, trigger) => {
      const node = byId.get(id);
      if (!node) return;
      triggerRef.current = trigger;
      setSelected(node);
    },
    [byId],
  );
  const canOpen = useCallback((id: string) => byId.has(id), [byId]);

  const close = useCallback(() => {
    setSelected(null);
    triggerRef.current?.focus();
  }, []);

  const card = selected ? cards.get(selected.id) : undefined;
  const flower = selected ? card ? <MontagneProofFlower node={selected} card={card} onClose={close} /> : <MontagneFlower node={selected} onClose={close} /> : null;
  return { open, canOpen, flower };
}
