'use client';

import { useState, useCallback, useEffect } from 'react';
import { useTranslations } from 'next-intl';
import { pickHints, Hint } from './hints';

// Un tip contextuel affiché dans la zone haute du panel gauche (persistant jusqu'à fermeture ×).
export interface Tip {
  id: string;
  text: string;
  icon?: string;
}

interface LeftPanelProps {
  tips: Tip[];
  onTipClose: (id: string) => void;
  onHintClick: (message: string) => void;
  // Questions de relance contextuelles (tool suggest_followups, LLM) — priorité intermédiaire entre
  // les tips et le pool statique de hints (cf. priorité ci-dessous).
  followups: string[];
}

// LeftPanel — panel gauche (20%) avec tips Claude et hints cliquables.
// Priorité d'affichage : tips (max 3 FIFO) > relances contextuelles (LLM, suggest_followups) >
// hints statiques aléatoires (pool fixe, fallback quand la conversation n'a encore rien suggéré).
export function LeftPanel({ tips, onTipClose, onHintClick, followups }: LeftPanelProps) {
  const t = useTranslations('chat');
  // Pool de hints localisé (messages/*.json → chat.hints), lu via t.raw (renvoie le tableau brut).
  const allHints = t.raw('hints') as Hint[];

  // QUOI : état initial DÉTERMINISTE (les 5 premiers hints) — identique SSR et client.
  // POURQUOI : pickHints utilise Math.random ; l'appeler dans l'initialiseur useState produisait des
  //   hints différents au SSR et à l'hydratation → mismatch React (#425/#422) sur /chat. On rend donc
  //   un set stable d'abord, puis on mélange APRÈS le mount (useEffect, côté client uniquement).
  const [hints, setHints] = useState<Hint[]>(() => allHints.slice(0, 5));

  // Mélange aléatoire une fois monté (post-hydratation → pas de mismatch).
  useEffect(() => {
    setHints(pickHints(allHints, 5));
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  // Re-tire 5 hints aléatoires (clic refresh)
  const refreshHints = useCallback(() => {
    setHints(pickHints(allHints, 5));
  }, [allHints]);

  const hasTips = tips.length > 0;
  const hasFollowups = !hasTips && followups.length > 0;

  return (
    <div className="flex flex-col h-full overflow-hidden bg-card/40">
      {hasTips ? (
        /* Zone tips — empilés en haut, max 3 FIFO */
        <div className="flex flex-col gap-2 p-4 overflow-y-auto">
          {tips.slice(-3).map((tip) => (
            // Un tip = une note posée par l'agent : panneau de la nuit, filet d'or à gauche (le
            // signal « à lire »), texte ivoire. L'ancien aplat ambre est retiré (pas de couleur en dur).
            <div
              key={tip.id}
              className="flex items-start gap-2 p-3 rounded-lg bg-card border border-border
                         border-l-2 border-l-gold/70 text-sm"
            >
              <p className="flex-1 text-foreground leading-snug text-xs">
                {tip.text}
              </p>
              <button
                onClick={() => onTipClose(tip.id)}
                aria-label={t('closeTip')}
                className="shrink-0 text-muted-foreground hover:text-foreground
                           text-base leading-none mt-0.5 p-1 -m-1"
              >
                ×
              </button>
            </div>
          ))}
        </div>
      ) : hasFollowups ? (
        /* Zone relances — questions contextuelles générées par le LLM (suggest_followups) */
        <div className="flex flex-col h-full p-4 gap-3 overflow-y-auto">
          <span className="text-xs font-medium text-muted-foreground uppercase tracking-[0.18em]">
            {t('followupsHeader')}
          </span>
          <div className="flex flex-col gap-1.5">
            {followups.map((question, i) => (
              <button
                key={i}
                onClick={() => onHintClick(question)}
                className="text-left px-3 py-2.5 rounded-lg text-sm text-muted-foreground
                           bg-background border border-border
                           hover:border-muted-foreground hover:text-foreground
                           transition-colors cursor-pointer leading-snug"
              >
                {question}
              </button>
            ))}
          </div>
        </div>
      ) : (
        /* Zone hints — boutons cliquables quand pas de tips ni de relances contextuelles */
        <div className="flex flex-col h-full p-4 gap-3 overflow-y-auto">
          {/* Header hints */}
          <div className="flex items-center justify-between">
            <span className="text-xs font-medium text-muted-foreground uppercase tracking-[0.18em]">
              {t('questionsHeader')}
            </span>
            <button
              onClick={refreshHints}
              aria-label={t('newSuggestions')}
              title={t('newSuggestions')}
              className="text-muted-foreground hover:text-foreground transition-colors text-base leading-none p-2 -m-1"
            >
              ↺
            </button>
          </div>

          {/* Boutons hints */}
          <div className="flex flex-col gap-1.5">
            {hints.map((hint, i) => (
              <button
                key={i}
                onClick={() => onHintClick(hint.message)}
                className="text-left px-3 py-2.5 rounded-lg text-sm text-muted-foreground
                           bg-background border border-border
                           hover:border-muted-foreground hover:text-foreground
                           transition-colors cursor-pointer leading-snug"
              >
                {hint.text}
              </button>
            ))}
          </div>
        </div>
      )}
    </div>
  );
}
