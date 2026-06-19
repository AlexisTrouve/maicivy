'use client';

import { useTranslations } from 'next-intl';

// DefaultFiche — placeholder minimal affiché quand aucun onglet n'est ouvert.
// Les hints sont dans le LeftPanel — ici on garde juste un indicateur visuel.
export function DefaultFiche() {
  const t = useTranslations('chat');
  return (
    <div className="flex flex-col items-center justify-center h-full text-center p-6 gap-3">
      <div className="text-4xl opacity-20">✦</div>
      <p className="text-sm text-muted-foreground">
        {t('askQuestion')}
      </p>
    </div>
  );
}
