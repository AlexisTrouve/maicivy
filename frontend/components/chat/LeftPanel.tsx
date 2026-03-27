'use client';

import { useState, useCallback } from 'react';
import { Tip } from './TipBar';
import { pickHints, Hint } from './hints';

interface LeftPanelProps {
  tips: Tip[];
  onTipClose: (id: string) => void;
  onHintClick: (message: string) => void;
}

// LeftPanel — panel gauche (20%) avec tips Claude et hints cliquables.
// Priorité : si des tips sont présents, on les affiche (max 3 FIFO).
// Sinon : hints aléatoires en boutons pill + bouton refresh.
export function LeftPanel({ tips, onTipClose, onHintClick }: LeftPanelProps) {
  // Hints tirés aléatoirement au mount, re-tirés au clic refresh
  const [hints, setHints] = useState<Hint[]>(() => pickHints(5));

  // Re-tire 5 hints aléatoires
  const refreshHints = useCallback(() => {
    setHints(pickHints(5));
  }, []);

  const hasTips = tips.length > 0;

  return (
    <div className="flex flex-col h-full overflow-hidden border-r bg-muted/10">
      {hasTips ? (
        /* Zone tips — empilés en haut, max 3 FIFO */
        <div className="flex flex-col gap-2 p-3 overflow-y-auto">
          {tips.slice(-3).map((tip) => (
            <div
              key={tip.id}
              className="flex items-start gap-2 p-3 rounded-lg bg-amber-50 dark:bg-amber-950/20
                         border border-amber-200 dark:border-amber-800 text-sm"
            >
              <span className="shrink-0 mt-0.5 text-base">{tip.icon ?? '💡'}</span>
              <p className="flex-1 text-amber-900 dark:text-amber-200 leading-snug text-xs">
                {tip.text}
              </p>
              <button
                onClick={() => onTipClose(tip.id)}
                aria-label="Fermer ce tip"
                className="shrink-0 text-amber-400 hover:text-amber-600 dark:hover:text-amber-300
                           text-base leading-none mt-0.5"
              >
                ×
              </button>
            </div>
          ))}
        </div>
      ) : (
        /* Zone hints — boutons cliquables quand pas de tips */
        <div className="flex flex-col h-full p-3 gap-3">
          {/* Header hints */}
          <div className="flex items-center justify-between">
            <span className="text-xs font-medium text-muted-foreground uppercase tracking-wider">
              Questions
            </span>
            <button
              onClick={refreshHints}
              aria-label="Nouvelles suggestions"
              title="Nouvelles suggestions"
              className="text-muted-foreground/50 hover:text-muted-foreground transition-colors text-sm leading-none p-1"
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
                className="text-left px-3 py-2 rounded-lg text-xs text-muted-foreground
                           bg-background border border-border/50
                           hover:border-primary/40 hover:text-foreground hover:bg-primary/5
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
