'use client';

import { forwardRef, useEffect, useRef, type RefObject } from 'react';
import { useTranslations } from 'next-intl';
import { X } from 'lucide-react';

import { Link } from '@/i18n/navigation';
import type { MontagneNode } from '@/lib/montagne-api';
import { nodeIconSrc } from '@/lib/montagne/monogram';

/**
 * FlowerCard — ce que les deux fleurs (pétales d'une tuile, preuves d'une carte du sommet) ont en
 * commun : la carte du projet (icône, nom, strate · statut, mini, bouton vers la page portfolio
 * si le projet est curé) et le comportement de dialogue.
 *
 * POURQUOI un composant partagé : la fleur d'une tuile et celle d'une carte du sommet ne diffèrent
 * que par ce qui entoure la carte. Le titre, le bouton d'action principale (la seule action en or
 * à l'écran) et la fermeture doivent rester identiques — mêmes testids pour l'E2E.
 */
export const FlowerCard = forwardRef<HTMLDivElement, { node: MontagneNode; onClose: () => void; closeRef: RefObject<HTMLButtonElement | null>; className?: string }>(
  function FlowerCard({ node, onClose, closeRef, className = '' }, ref) {
    const t = useTranslations('montagne');
    const statusLabel = t.has(`status.${node.status}`) ? t(`status.${node.status}`) : node.status;
    return (
      <div ref={ref} className={`rounded-xl border border-white/10 bg-[#121b2c] p-5 shadow-panel-gold ${className}`}>
        <button
          ref={closeRef}
          type="button"
          onClick={onClose}
          aria-label={t('flower.close')}
          data-testid="montagne-flower-close"
          className="absolute right-1 top-1 inline-flex h-11 w-11 items-center justify-center rounded-md text-slate-400 transition-colors hover:bg-white/5 hover:text-white focus:outline-none focus-visible:ring-1 focus-visible:ring-gold/70"
        >
          <X className="h-4 w-4" />
        </button>
        <div className="flex items-center gap-3 pr-9">
          {/* eslint-disable-next-line @next/next/no-img-element -- monogramme en data-URL SVG (ou icône Eikon) : rien à optimiser */}
          <img src={nodeIconSrc(node)} alt="" width={56} height={56} className="h-14 w-14 shrink-0 rounded-full" />
          <div className="min-w-0">
            <h2 id={flowerTitleId(node)} className="font-heading text-lg font-semibold leading-tight text-slate-100" data-testid="montagne-flower-name">
              {node.name}
            </h2>
            <p className="mt-0.5 text-xs text-slate-400">
              {t(`strata.${node.stratum}`)} · {statusLabel}
            </p>
          </div>
        </div>
        {node.mini && <p className="mt-3 text-sm leading-relaxed text-slate-300">{node.mini}</p>}
        {node.curated && (
          // L'action principale de la fleur : la seule en or à l'écran pendant qu'elle est ouverte.
          <Link
            href={`/portfolio/${node.id}`}
            data-testid="montagne-flower-portfolio"
            className="mt-4 inline-flex min-h-[44px] items-center gap-1.5 rounded-md bg-gradient-to-b from-gold-bright to-gold-deep px-3 py-1.5 text-sm font-semibold text-slate-950 shadow-gold transition-[filter] hover:brightness-110 focus:outline-none focus-visible:ring-2 focus-visible:ring-gold-soft"
          >
            {t('flower.openPortfolio')} →
          </Link>
        )}
      </div>
    );
  },
);

/** Identifiant du titre de la fleur : `aria-labelledby` du dialogue. */
export function flowerTitleId(node: MontagneNode) {
  return `montagne-flower-title-${node.id}`;
}

/**
 * Comportement commun des fleurs : défilement de la page suspendu tant qu'elle est ouverte (sur
 * téléphone, un glisser sur le fond ferait sinon défiler la montagne derrière), focus sur le
 * bouton de fermeture à l'ouverture, Échap ferme. Renvoie la référence du bouton de fermeture.
 */
export function useFlowerDialog(onClose: () => void) {
  const closeRef = useRef<HTMLButtonElement>(null);

  useEffect(() => {
    const previous = document.body.style.overflow;
    document.body.style.overflow = 'hidden';
    return () => {
      document.body.style.overflow = previous;
    };
  }, []);

  useEffect(() => {
    closeRef.current?.focus();
    const onKey = (e: KeyboardEvent) => {
      if (e.key === 'Escape') onClose();
    };
    window.addEventListener('keydown', onKey);
    return () => window.removeEventListener('keydown', onKey);
  }, [onClose]);

  return closeRef;
}
