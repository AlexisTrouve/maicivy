'use client';

import { useEffect, type RefObject } from 'react';

/**
 * useLinkHighlight — allume les filets d'UN projet quand on le survole (souris) ou le focalise
 * (clavier), dans la montagne à plat (MontagnePeak).
 *
 * QUOI : sur le projet actif (carte du sommet ou tuile, `[data-open-flower][data-project-id]`),
 * pose `data-lit` sur chaque filet dont il est une extrémité (`path[data-from|data-to]`) et sur
 * leurs points d'arrivée (`circle[data-end-from|data-end-to]`), et `data-link-focus` sur la
 * racine. Le style (montagne.module.css › .link / .linkEnd) fait le reste : au repos tout est
 * discret, l'actif s'allume, les autres reculent encore.
 * POURQUOI : en v4, chaque lien réel était un trait plein — une vingtaine partaient de GroveEngine
 * et brouillaient la composition. On garde TOUS les liens réels (aucun inventé, aucun retiré),
 * mais on ne montre en clair que ceux du projet qu'on regarde.
 * COMMENT :
 *  - une seule délégation d'événements sur la racine (les ~75 cartes et tuiles restent du HTML
 *    serveur, sans composant client chacune — même principe que la délégation de clic) ;
 *  - survol : `pointerover` hors tactile (au doigt, le tap ouvre la fleur ; un allumage fugace
 *    juste avant n'apporterait rien), `pointerleave` de la racine éteint ;
 *  - focus : `focusin` allume, `focusout` vers un élément sans projet éteint ;
 *  - sans JavaScript, rien ne s'allume : les filets restent visibles au repos, discrets.
 */
export function useLinkHighlight(rootRef: RefObject<HTMLElement | null>) {
  useEffect(() => {
    const root = rootRef.current;
    if (!root) return;
    let active: string | null = null;

    /** Projet porté par l'élément (ou un ancêtre), s'il en porte un. */
    const projectOf = (target: EventTarget | null): string | null =>
      target instanceof Element ? (target.closest('[data-open-flower]')?.getAttribute('data-project-id') ?? null) : null;

    const light = (id: string | null) => {
      if (id === active) return;
      active = id;
      root.querySelectorAll('[data-lit]').forEach((el) => el.removeAttribute('data-lit'));
      root.toggleAttribute('data-link-focus', id !== null);
      if (id === null) return;
      const q = CSS.escape(id);
      root
        .querySelectorAll(`path[data-from="${q}"], path[data-to="${q}"], circle[data-end-from="${q}"], circle[data-end-to="${q}"]`)
        .forEach((el) => el.setAttribute('data-lit', ''));
    };

    const onPointerOver = (e: PointerEvent) => {
      if (e.pointerType !== 'touch') light(projectOf(e.target));
    };
    const onPointerLeave = (e: PointerEvent) => {
      if (e.pointerType !== 'touch') light(null);
    };
    const onFocusIn = (e: FocusEvent) => light(projectOf(e.target));
    const onFocusOut = (e: FocusEvent) => light(projectOf(e.relatedTarget));

    root.addEventListener('pointerover', onPointerOver);
    root.addEventListener('pointerleave', onPointerLeave);
    root.addEventListener('focusin', onFocusIn);
    root.addEventListener('focusout', onFocusOut);
    return () => {
      root.removeEventListener('pointerover', onPointerOver);
      root.removeEventListener('pointerleave', onPointerLeave);
      root.removeEventListener('focusin', onFocusIn);
      root.removeEventListener('focusout', onFocusOut);
    };
  }, [rootRef]);
}
