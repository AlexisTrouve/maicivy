'use client';

import { useEffect, useRef, useState } from 'react';
import { useBackground } from './BackgroundProvider';
import { BG_NONE, getBackground } from '@/lib/backgrounds/registry';
import type { BackgroundInstance } from '@/lib/backgrounds/types';

/**
 * BackgroundHost — la coquille partagée qui fait tourner UN fond à la fois.
 *
 * QUOI : possède le conteneur fixed, et centralise TOUTES les invariantes communes aux fonds —
 *   defer post-LCP, RAF loop unique, resize, pointermove, fade-in, et surtout le dispose
 *   complet au switch/unmount (zéro fuite GPU).
 * POURQUOI : c'est ce qui garantit le "plug & play sans rien casser" — un plugin ne PEUT pas
 *   oublier une invariante car elles ne sont pas dans son scope.
 * COMMENT : lit `activeId` du contexte. À chaque changement, dispose proprement l'ancien
 *   plugin puis import()e + init()e le nouveau. Le 1er montage est différé (LCP) ; un switch
 *   utilisateur est instantané.
 */

const INITIAL_DEFER_MS = 3000; // défer le 1er fond après le LCP (le thread principal se libère)
const MAX_DPR = 2; // plafond devicePixelRatio (perf)

export function BackgroundHost() {
  const { activeId } = useBackground();
  const mountRef = useRef<HTMLDivElement>(null);
  const [visible, setVisible] = useState(false); // pilote le fade-in CSS
  // Horodatage du 1er effet : sert à ne déférer que pendant la fenêtre de chargement initial.
  const mountTimeRef = useRef<number | null>(null);

  useEffect(() => {
    if (mountTimeRef.current === null) mountTimeRef.current = performance.now();

    const mount = mountRef.current;
    if (!mount) return;

    // Pas de fond à rendre : non résolu (SSR), 'none', ou reduced-motion (→ activeId='none').
    if (!activeId || activeId === BG_NONE) {
      setVisible(false);
      return;
    }
    const entry = getBackground(activeId);
    if (!entry) {
      setVisible(false);
      return;
    }

    let instance: BackgroundInstance | null = null;
    let rafId = 0;
    let cancelled = false; // garde anti-race : effet nettoyé avant la fin de l'init async
    let lastTime = 0;

    // Defer seulement le temps restant de la fenêtre de chargement initial (protège le LCP).
    // Un fond monté plus tard (switch utilisateur, ou départ depuis 'none') démarre immédiatement.
    const elapsed = performance.now() - mountTimeRef.current;
    const deferMs = Math.max(0, INITIAL_DEFER_MS - elapsed);

    const startTimer = setTimeout(() => {
      const dpr = Math.min(window.devicePixelRatio || 1, MAX_DPR);
      const ctx = {
        mount,
        width: window.innerWidth,
        height: window.innerHeight,
        dpr,
        // La coquille ne monte JAMAIS de fond sous reduced-motion (cf. provider) → false ici.
        reducedMotion: false,
      };

      Promise.resolve(entry.load())
        .then((mod) => mod.default(ctx))
        .then((inst) => {
          // L'utilisateur a pu switcher pendant le chargement async → on jette ce qu'on vient de créer.
          if (cancelled) {
            inst.dispose();
            return;
          }
          instance = inst;
          setVisible(true);

          // RAF loop UNIQUE gérée par la coquille — les plugins ne touchent pas requestAnimationFrame.
          if (inst.frame) {
            const loop = (now: number) => {
              const dt = lastTime ? now - lastTime : 16;
              lastTime = now;
              inst.frame!(dt);
              rafId = requestAnimationFrame(loop);
            };
            rafId = requestAnimationFrame(loop);
          }
        })
        .catch((err) => {
          // Échec franc et visible (pas de fond silencieusement absent qui masquerait un bug).
          // eslint-disable-next-line no-console
          console.error('[BackgroundHost] init failed for', activeId, err);
        });
    }, deferMs);

    // Resize viewport → forwardé au plugin.
    const onResize = () => instance?.resize?.(window.innerWidth, window.innerHeight);
    window.addEventListener('resize', onResize);

    // Souris : le conteneur est pointer-events-none, on écoute donc sur window et on forwarde.
    // No-op pour les fonds non interactifs (onPointerMove absent).
    const onPointer = (e: PointerEvent) => instance?.onPointerMove?.(e.clientX, e.clientY);
    window.addEventListener('pointermove', onPointer);

    // Cleanup : appelé au changement d'activeId ou à l'unmount → dispose total.
    return () => {
      cancelled = true;
      clearTimeout(startTimer);
      cancelAnimationFrame(rafId);
      window.removeEventListener('resize', onResize);
      window.removeEventListener('pointermove', onPointer);
      instance?.dispose();
      // Filet de sécurité : vider le conteneur si le plugin a laissé des nodes.
      while (mount.firstChild) mount.removeChild(mount.firstChild);
      setVisible(false);
    };
  }, [activeId]);

  return (
    <div
      ref={mountRef}
      aria-hidden="true"
      data-testid="background-host"
      // z-[1] : au-dessus des blobs aurora (z-0), sous le contenu (z-10).
      // pointer-events-none : le canvas ne capture aucun event utilisateur.
      className="fixed inset-0 z-[1] pointer-events-none"
      style={{ opacity: visible ? 1 : 0, transition: 'opacity 1.5s ease' }}
    />
  );
}
