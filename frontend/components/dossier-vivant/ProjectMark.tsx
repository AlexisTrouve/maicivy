'use client';

import { useState } from 'react';

import type { SceneNode } from '@/lib/dossier-vivant/scene';

/** Base de l'API côté navigateur (même règle que lib/chat-api.ts, DossierVivant). */
const API_BASE = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080';

/**
 * ProjectMark — la marque d'un projet partout où le dossier vivant le représente : tuile et carte
 * de la montagne, bandeau des retenus, tableau des preuves.
 *
 * QUOI : l'icône du dépôt du projet (`icon.png`, servie et réduite par le backend :
 * GET /api/v1/project-icons/:id) quand elle existe, sinon le monogramme de deux lettres.
 * POURQUOI un seul composant : le même projet doit porter la même marque aux quatre endroits — et
 * dans le PDF, que le backend compose avec la même icône.
 * COMMENT : décorative (`alt=""`, `aria-hidden`) — le nom du projet est toujours écrit à côté (ou
 * en `title` sur la tuile). Si l'image ne charge pas (cache évincé, backend redémarré), retour au
 * monogramme plutôt qu'une icône cassée. `<img>` et pas next/image : 128 px au plus, déjà réduite
 * par le backend, souvent posée dans un repère mis à l'échelle — l'optimiseur n'apporterait rien.
 * E2E : l'image porte `data-testid="dv-project-icon"`.
 */
export function ProjectMark({ node, monoClass, iconClass }: { node: SceneNode | undefined; monoClass: string; iconClass: string }) {
  const [failed, setFailed] = useState(false);
  if (node?.icon && !failed) {
    return (
      // eslint-disable-next-line @next/next/no-img-element -- icône déjà réduite par le backend (cf. en-tête)
      <img
        className={iconClass}
        src={`${API_BASE}${node.icon}`}
        alt=""
        aria-hidden="true"
        decoding="async"
        draggable={false}
        onError={() => setFailed(true)}
        data-testid="dv-project-icon"
      />
    );
  }
  return (
    <span className={monoClass} aria-hidden="true">
      {node?.monogram}
    </span>
  );
}
