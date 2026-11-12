'use client';

import { useCallback, useEffect, useMemo, useRef, useState, type ReactNode } from 'react';
import dynamic from 'next/dynamic';
import { useTranslations } from 'next-intl';
import { Network, X } from 'lucide-react';

import type { Montagne, MontagneNode } from '@/lib/montagne-api';
import { MontagneFlower } from './MontagneFlower';
import { MontagneProofFlower } from './MontagneProofFlower';
import { useLinkHighlight } from './useLinkHighlight';

/**
 * MontagneExplorer — ce qui, dans la montagne, demande du JavaScript : la fleur et la vue réseau.
 *
 * QUOI :
 *  - enveloppe la montagne rendue côté serveur (`children`, MontagnePeak) et ouvre la FLEUR du
 *    projet quand on clique ou touche une tuile (pétales), ou celle de ses PREUVES quand c'est une
 *    carte du sommet ;
 *  - allume les filets du projet survolé ou focalisé (useLinkHighlight) ;
 *  - porte le bouton secondaire « vue réseau » : le canvas 3D, ouvert en plein écran.
 *
 * POURQUOI une délégation de clic et pas un composant client par tuile : les 75 cartes et tuiles
 * restent du HTML serveur, sans hydratation chacune. Un seul gestionnaire lit `data-project-id`
 * sur l'élément `[data-open-flower]` touché. Sans JavaScript, ces éléments restent des liens
 * (page projet ou entrée de la liste) : rien n'est perdu.
 *
 * POURQUOI `dynamic` sans SSR pour le canvas : son code (projection, gestes, étiquettes) n'est
 * TÉLÉCHARGÉ qu'au premier clic sur « vue réseau » — la page d'arrivée n'en paie rien.
 */
const MontagneCanvas = dynamic(() => import('./MontagneCanvas'), {
  ssr: false,
  loading: () => <NetworkLoading />,
});

export function MontagneExplorer({ montagne, children }: { montagne: Montagne; children: ReactNode }) {
  const t = useTranslations('montagne');
  const byId = useMemo(() => new Map(montagne.nodes.map((n) => [n.id, n])), [montagne.nodes]);
  const cards = useMemo(() => new Map(montagne.summit.map((c) => [c.id, c])), [montagne.summit]);
  const [selected, setSelected] = useState<MontagneNode | null>(null);
  const [network, setNetwork] = useState(false);
  /** Élément qui a ouvert la fleur : le focus y revient à la fermeture (clavier). */
  const triggerRef = useRef<HTMLElement | null>(null);
  const networkButtonRef = useRef<HTMLButtonElement>(null);
  /** Racine de la montagne : délégation du clic (fleur) et de l'allumage des filets. */
  const mountainRef = useRef<HTMLDivElement>(null);
  useLinkHighlight(mountainRef);
  // Hydratation faite = la délégation de clic est en place. Signal OBSERVABLE pour l'E2E :
  // « réseau inactif » n'arrive plus jamais depuis que la démo vidéo du sommet tourne en boucle.
  const [hydrated, setHydrated] = useState(false);
  useEffect(() => setHydrated(true), []);

  const onClick = (e: React.MouseEvent) => {
    // Ouvrir dans un nouvel onglet (Ctrl, Cmd, Maj, clic du milieu) garde le lien de repli.
    if (e.defaultPrevented || e.button !== 0 || e.metaKey || e.ctrlKey || e.shiftKey || e.altKey) return;
    const target = (e.target as Element).closest<HTMLElement>('[data-open-flower]');
    const node = target ? byId.get(target.dataset.projectId ?? '') : undefined;
    if (!target || !node) return;
    e.preventDefault();
    triggerRef.current = target;
    setSelected(node);
  };

  const closeFlower = useCallback(() => {
    setSelected(null);
    triggerRef.current?.focus();
  }, []);

  const closeNetwork = () => {
    setNetwork(false);
    networkButtonRef.current?.focus();
  };

  return (
    <>
      <div ref={mountainRef} onClick={onClick} data-testid="montagne-explorer" data-hydrated={hydrated ? 'true' : 'false'}>
        {children}
      </div>

      {/* `relative` : la brume du pied (positionnée) déborde jusqu'ici ; sans contexte propre, elle
          se peindrait par-dessus le bouton. */}
      <div className="relative flex justify-center px-4 pb-14 pt-2">
        {/* Action SECONDAIRE : neutre (l'or reste au CV, l'action principale de l'écran). */}
        <button
          ref={networkButtonRef}
          type="button"
          data-testid="montagne-network-open"
          onClick={() => setNetwork(true)}
          className="inline-flex min-h-[44px] items-center gap-2 rounded-md border border-white/15 bg-[#0a0f1a]/80 px-4 text-sm text-slate-300 backdrop-blur transition-colors hover:border-white/35 hover:text-white focus:outline-none focus-visible:ring-2 focus-visible:ring-gold/60"
        >
          <Network className="h-4 w-4" aria-hidden="true" />
          {t('network.open')}
        </button>
      </div>

      {selected &&
        (cards.has(selected.id) ? (
          <MontagneProofFlower node={selected} card={cards.get(selected.id)!} onClose={closeFlower} />
        ) : (
          <MontagneFlower node={selected} onClose={closeFlower} />
        ))}

      {network && (
        <div
          role="dialog"
          aria-modal="true"
          aria-label={t('network.title')}
          data-testid="montagne-network"
          className="fixed inset-0 z-[250] flex flex-col bg-[#04060b]"
        >
          <div className="flex items-center justify-between border-b border-white/5 px-4 py-2">
            <p className="text-sm text-slate-300">{t('network.title')}</p>
            <button
              type="button"
              data-testid="montagne-network-close"
              onClick={closeNetwork}
              aria-label={t('network.close')}
              className="inline-flex h-11 w-11 items-center justify-center rounded-md text-slate-400 transition-colors hover:bg-white/5 hover:text-white focus:outline-none focus-visible:ring-1 focus-visible:ring-gold/70"
            >
              <X className="h-5 w-5" />
            </button>
          </div>
          <div className="relative min-h-0 flex-1">
            <MontagneCanvas montagne={montagne} />
          </div>
        </div>
      )}
    </>
  );
}

/** Pendant le téléchargement du canvas : un message centré, même fond, aucune secousse. */
function NetworkLoading() {
  const t = useTranslations('montagne.stage');
  return (
    <div data-testid="montagne-stage-loading" className="flex h-full w-full items-center justify-center px-6 text-center text-sm text-slate-500">
      {t('loading')}
    </div>
  );
}
