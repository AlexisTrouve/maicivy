'use client';

import { useTranslations } from 'next-intl';

// Un tip contextuel affiché dans la barre supérieure du panel droit.
// Persistant jusqu'à fermeture manuelle (×).
export interface Tip {
  id: string;
  text: string;
  icon?: string;
}

interface TipBarProps {
  tip: Tip;
  onClose: (id: string) => void;
}

export function TipBar({ tip, onClose }: TipBarProps) {
  const t = useTranslations('chat');
  return (
    <div
      className="flex items-start gap-2 px-3 py-2 bg-amber-50 dark:bg-amber-950/20
                 border-b border-amber-200 dark:border-amber-800 text-sm shrink-0"
    >
      <span className="shrink-0 mt-0.5">{tip.icon ?? '💡'}</span>
      <p className="flex-1 text-amber-900 dark:text-amber-200 leading-snug">{tip.text}</p>
      <button
        onClick={() => onClose(tip.id)}
        aria-label={t('closeTip')}
        className="shrink-0 text-amber-400 hover:text-amber-600 dark:hover:text-amber-300 text-base leading-none mt-0.5"
      >
        ×
      </button>
    </div>
  );
}
