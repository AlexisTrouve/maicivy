'use client';

/*
 * Frontière d'erreur de la section localisée. QUOI : ce qui s'est passé, quoi faire, et UNE action
 *   principale (or) — réessayer, qui relance le rendu du segment via `reset()` —, plus un retour sobre
 *   à l'accueil si l'erreur persiste.
 * Le message technique n'est affiché qu'en développement : en production il ne dirait rien au
 *   visiteur et pourrait exposer un détail interne.
 */
import { useEffect } from 'react';
import { useTranslations } from 'next-intl';

import { Link } from '@/i18n/navigation';
import { boutonOr, boutonSobre, etiquette } from '@/components/site-page/kit';

export default function Error({
  error,
  reset,
}: {
  error: Error & { digest?: string };
  reset: () => void;
}) {
  const t = useTranslations('errors');
  const tCommon = useTranslations('common');

  useEffect(() => {
    // Trace côté navigateur (pourrait être envoyée à un service de supervision).
    console.error('Error boundary:', error);
  }, [error]);

  return (
    <div className="container flex min-h-[60vh] max-w-3xl flex-col justify-center px-5 py-20 md:px-8">
      <p className={etiquette}>
        {t('errorEyebrow')}
        <span aria-hidden="true" className="mt-3 block h-px w-12 bg-gold/70" />
      </p>
      <h1 className="mt-5 text-4xl font-medium leading-tight text-foreground md:text-5xl">{t('generic')}</h1>
      <p className="mt-5 max-w-xl text-base leading-relaxed text-muted-foreground md:text-lg">{t('tryAgain')}</p>

      {process.env.NODE_ENV === 'development' && (
        <pre className="mt-6 overflow-x-auto rounded-md border border-border bg-background p-4 font-mono text-sm text-muted-foreground">
          {error.message}
        </pre>
      )}

      <div className="mt-9 flex flex-wrap gap-3">
        <button type="button" onClick={reset} className={boutonOr}>
          {t('retry')}
        </button>
        <Link href="/" className={boutonSobre}>
          {tCommon('backToHome')}
        </Link>
      </div>
    </div>
  );
}
