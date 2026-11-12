'use client';

import { useLayoutEffect, useRef, useState } from 'react';
import Image from 'next/image';
import { useTranslations } from 'next-intl';

import type { MontagneNode } from '@/lib/montagne-api';
import { computePetalLayout, type Size } from '@/lib/montagne/petals';
import { portfolioImageUrl } from '@/lib/portfolio-api';

import { FlowerCard, flowerTitleId, useFlowerDialog } from './FlowerCard';

/**
 * MontagneFlower — la « fleur » d'une TUILE, ouverte au clic. (Une carte du sommet ouvre la fleur
 * de ses preuves, MontagneProofFlower.)
 *
 * QUOI : la carte du projet au centre (icône, nom, statut, mini-description, bouton vers la page
 * portfolio SI le projet est curé), ses captures et sa vidéo en pétales autour. Un projet sans
 * visuel ouvre une fleur réduite : la carte seule.
 * POURQUOI un calque DOM et pas du canvas : du texte sélectionnable, une vidéo lisible, des liens
 * réels, et un dialogue accessible (focus, Échap).
 * COMMENT : la carte est mesurée après rendu, puis computePetalLayout place les pétales autour
 * (ellipse en paysage, rangées dessus/dessous en portrait). Fermeture : bouton, Échap, clic sur le
 * fond. Calque FIXE sur toute la fenêtre (la montagne à plat défile, la fleur ne doit pas défiler
 * avec elle) ; le défilement de la page est suspendu tant qu'elle est ouverte — sur téléphone, un
 * glisser sur le fond ferait sinon défiler la montagne derrière.
 * E2E : `data-project-id` (le BON projet), `data-variant` (full | reduced), testids stables.
 */
export function MontagneFlower({ node, onClose }: { node: MontagneNode; onClose: () => void }) {
  const t = useTranslations('montagne');
  const rootRef = useRef<HTMLDivElement>(null);
  const cardRef = useRef<HTMLDivElement>(null);
  const closeRef = useFlowerDialog(onClose);
  const [viewport, setViewport] = useState<Size | null>(null);
  const [card, setCard] = useState<Size | null>(null);

  // Mesure du calque et de la carte : les pétales se placent autour de la carte RÉELLE (sa hauteur
  // dépend de la longueur du mini et de la langue).
  useLayoutEffect(() => {
    const measure = () => {
      if (rootRef.current) setViewport({ width: rootRef.current.clientWidth, height: rootRef.current.clientHeight });
      if (cardRef.current) setCard({ width: cardRef.current.offsetWidth, height: cardRef.current.offsetHeight });
    };
    measure();
    const ro = new ResizeObserver(measure);
    if (rootRef.current) ro.observe(rootRef.current);
    if (cardRef.current) ro.observe(cardRef.current);
    return () => ro.disconnect();
  }, [node.id]);

  const petals = viewport && card ? computePetalLayout(node.visuals.length, viewport, card) : null;

  return (
    <div
      ref={rootRef}
      role="dialog"
      aria-modal="true"
      aria-labelledby={flowerTitleId(node)}
      data-testid="montagne-flower"
      data-project-id={node.id}
      data-variant={node.visuals.length > 0 ? 'full' : 'reduced'}
      className="fixed inset-0 z-[300] overflow-hidden bg-[#04060b]/80 backdrop-blur-sm animate-in fade-in-0 duration-200"
      // Un geste dans la fleur ne fait pas tourner la montagne derrière (la scène écoute pointerdown).
      onPointerDown={(e) => e.stopPropagation()}
      // Clic sur le fond (et seulement sur lui) : ferme.
      onClick={(e) => {
        if (e.target === e.currentTarget) onClose();
      }}
    >
      {/* Pétales — positionnés une fois carte et calque mesurés. */}
      {petals &&
        node.visuals.map((v, i) => {
          const p = petals.positions[i];
          return (
            <div
              key={`${v.url}-${i}`}
              data-testid="montagne-flower-petal"
              data-kind={v.kind}
              className="absolute overflow-hidden rounded-xl border border-white/10 bg-[#0b111c] shadow-panel animate-in fade-in-0 zoom-in-90 duration-300"
              style={{
                left: p.x - petals.size / 2,
                top: p.y - petals.size / 2,
                width: petals.size,
                height: petals.size,
                animationDelay: `${60 * i}ms`,
                animationFillMode: 'backwards',
              }}
            >
              {v.kind === 'video' ? (
                <video
                  controls
                  playsInline
                  // « metadata » et jamais « none » : avec « none », Chrome ignore le clic sur le
                  // lecteur (mesuré, cf. portfolio.spec.ts).
                  preload="metadata"
                  poster={v.poster_url ? portfolioImageUrl(v.poster_url) : undefined}
                  src={portfolioImageUrl(v.url)}
                  aria-label={v.alt || t('flower.video')}
                  className="h-full w-full bg-black object-cover"
                />
              ) : (
                <a
                  href={portfolioImageUrl(v.url)}
                  target="_blank"
                  rel="noopener noreferrer"
                  aria-label={`${t('flower.openImage')} — ${v.alt || node.name}`}
                  className="relative block h-full w-full focus:outline-none focus-visible:ring-2 focus-visible:ring-gold/70"
                >
                  {/* Optimiseur Next pour les captures (pleine définition côté maiProFiles),
                      jamais pour le SVG qu'il refuse. */}
                  <Image
                    src={portfolioImageUrl(v.url)}
                    alt={v.alt || node.name}
                    fill
                    sizes={`${Math.round(petals.size)}px`}
                    unoptimized={v.vector}
                    className="object-cover object-top transition-transform duration-300 hover:scale-105"
                  />
                </a>
              )}
            </div>
          );
        })}

      {/* La carte, au centre. */}
      <FlowerCard
        ref={cardRef}
        node={node}
        onClose={onClose}
        closeRef={closeRef}
        className="absolute left-1/2 top-1/2 w-[min(20rem,calc(100%-2rem))] -translate-x-1/2 -translate-y-1/2"
      />
    </div>
  );
}
