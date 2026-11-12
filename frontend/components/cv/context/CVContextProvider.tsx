'use client';

import { createContext, useContext, useEffect, useState, type ReactNode } from 'react';
import { useLocale } from 'next-intl';
import { fetchCVContext, type CVContextView } from '@/lib/cvContext';

// ============================================================================
// État partagé du CV adapté sur la page /cv.
// ----------------------------------------------------------------------------
// QUOI : lit GET /cv/context au montage, re-lit tant que le CV est « en préparation », et garde la
// version affichée (adaptée ou standard). Le bandeau (dans l'en-tête) et le contenu (plus bas) lisent
// ce même état — d'où un contexte React plutôt qu'un composant unique : le portrait git s'intercale
// entre les deux et ne doit pas bouger.
//
// POURQUOI côté client : le contexte dépend du cookie de session du visiteur. Le rendu serveur de la
// page est mis en cache (revalidate 300) et partagé entre visiteurs : il ne doit JAMAIS porter le CV
// adapté de quelqu'un. Le CV standard est donc toujours le premier rendu, sans question (décision
// d'Alexi) ; l'adapté le remplace quand il est prêt.
//
// COMMENT : polling à intervalle fixe borné (POLL_MAX lectures, ~4 min — au-delà de la durée de vie
// d'une génération côté serveur). Aucune relecture hors de l'état « pending ».
// ============================================================================

export type CVView = 'adapted' | 'standard';

interface CVContextState {
  view: CVContextView;
  shown: CVView;
  setShown: (v: CVView) => void;
}

const Ctx = createContext<CVContextState>({
  view: { status: 'none' },
  shown: 'standard',
  setShown: () => {},
});

export const useCVContext = () => useContext(Ctx);

const POLL_MS = 4000;
const POLL_MAX = 60;

export function CVContextProvider({ children }: { children: ReactNode }) {
  const locale = useLocale();
  const [view, setView] = useState<CVContextView>({ status: 'none' });
  // Le visiteur qui a choisi « standard » le garde : l'arrivée du CV adapté ne lui reprend pas la main.
  const [choice, setChoice] = useState<CVView | null>(null);

  useEffect(() => {
    const ac = new AbortController();
    let timer: ReturnType<typeof setTimeout> | undefined;
    let reads = 0;
    const read = async () => {
      const next = await fetchCVContext(locale, ac.signal);
      if (ac.signal.aborted) return;
      setView(next);
      reads += 1;
      if (next.status === 'pending' && reads < POLL_MAX) timer = setTimeout(read, POLL_MS);
    };
    read();
    return () => {
      ac.abort();
      if (timer) clearTimeout(timer);
    };
  }, [locale]);

  // Adapté par défaut dès qu'il est prêt (c'est le but de la page), sauf choix explicite.
  const shown: CVView =
    view.status === 'ready' && view.cv && choice !== 'standard' ? 'adapted' : 'standard';

  return <Ctx.Provider value={{ view, shown, setShown: setChoice }}>{children}</Ctx.Provider>;
}
