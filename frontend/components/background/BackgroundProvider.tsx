'use client';

import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useState,
  type ReactNode,
} from 'react';
import {
  BG_NONE,
  BG_RANDOM,
  getBackground,
  isValidPreference,
  pickRandomBackground,
} from '@/lib/backgrounds/registry';

/**
 * BackgroundProvider — état de sélection du fond, partagé entre la coquille (BackgroundHost,
 * niveau layout) et le sélecteur (BackgroundSwitcher, dans le Header).
 *
 * QUOI : résout et expose deux notions distinctes —
 *   - `preference` : ce que l'utilisateur a choisi / ce que le sélecteur coche
 *     ('random' | 'none' | <pluginId>). Persisté en localStorage.
 *   - `activeId`   : le fond CONCRET rendu par la coquille ('none' | <pluginId> | null=non résolu).
 *     Quand preference='random', activeId est un plugin tiré au sort.
 * POURQUOI séparer les deux : "🎲 Aléatoire" doit rester coché dans le sélecteur tout en
 *   affichant un fond concret ; et re-cliquer "Aléatoire" doit pouvoir re-tirer un autre fond.
 * COMMENT résolution initiale (client only) : ?bg= (one-shot) > localStorage > random.
 *   reduced-motion force activeId='none' (accessibilité non négociable).
 */

const STORAGE_KEY = 'maicivy_bg';

interface BackgroundState {
  preference: string; // 'random' | 'none' | <pluginId>
  activeId: string | null; // fond concret rendu, ou null tant que non résolu (SSR)
  reducedMotion: boolean;
  setPreference: (p: string) => void;
}

const BackgroundCtx = createContext<BackgroundState | null>(null);

export function useBackground(): BackgroundState {
  const ctx = useContext(BackgroundCtx);
  if (!ctx) throw new Error('useBackground must be used within <BackgroundProvider>');
  return ctx;
}

// Calcule le fond concret à rendre depuis une préférence + le contexte d'écran.
function resolveActive(preference: string, isMobile: boolean): string {
  if (preference === BG_NONE) return BG_NONE;
  if (preference === BG_RANDOM) return pickRandomBackground(isMobile);
  // Préférence = un plugin précis : valable seulement s'il existe et est éligible mobile.
  const entry = getBackground(preference);
  if (!entry) return pickRandomBackground(isMobile);
  if (isMobile && !entry.mobile) return BG_NONE;
  return entry.id;
}

export function BackgroundProvider({ children }: { children: ReactNode }) {
  // Valeurs SSR-safe déterministes AVANT résolution client → aucun mismatch d'hydratation.
  const [preference, setPreferenceState] = useState<string>(BG_RANDOM);
  const [activeId, setActiveId] = useState<string | null>(null);
  const [reducedMotion, setReducedMotion] = useState(false);
  const [isMobile, setIsMobile] = useState(false);

  // Résolution initiale — client uniquement (window/localStorage indisponibles en SSR).
  useEffect(() => {
    const mobile = window.innerWidth < 768;
    const reduced = window.matchMedia('(prefers-reduced-motion: reduce)').matches;
    setIsMobile(mobile);
    setReducedMotion(reduced);

    // Ordre de résolution : ?bg= (override URL one-shot, ne persiste pas) > localStorage > défaut.
    // Le défaut est 'random', mais 'none' sous reduced-motion (défaut calme, accessibilité).
    // IMPORTANT : reduced-motion ne change QUE le défaut — un choix explicite (URL ou localStorage),
    // y compris un fond animé, est TOUJOURS honoré. Ne pas retirer l'agency de l'utilisateur :
    // c'était la régression de l'étape 1 (reduced-motion forçait 'none' + masquait le sélecteur).
    const urlBg = new URLSearchParams(window.location.search).get('bg');
    let pref: string;
    if (isValidPreference(urlBg)) {
      pref = urlBg as string;
    } else {
      const stored = localStorage.getItem(STORAGE_KEY);
      pref = isValidPreference(stored)
        ? (stored as string)
        : reduced
          ? BG_NONE
          : BG_RANDOM;
    }

    setPreferenceState(pref);
    setActiveId(resolveActive(pref, mobile));
  }, []);

  // Changement explicite via le sélecteur — persiste + recalcule le fond concret.
  const setPreference = useCallback(
    (p: string) => {
      setPreferenceState(p);
      // try/catch : localStorage peut throw (mode privé Safari). Échec silencieux toléré ici —
      // le choix reste actif pour la session, seule la persistance entre visites est perdue.
      try {
        localStorage.setItem(STORAGE_KEY, p);
      } catch {
        /* persistance best-effort */
      }
      setActiveId(resolveActive(p, isMobile));
    },
    [isMobile]
  );

  return (
    <BackgroundCtx.Provider value={{ preference, activeId, reducedMotion, setPreference }}>
      {children}
    </BackgroundCtx.Provider>
  );
}
