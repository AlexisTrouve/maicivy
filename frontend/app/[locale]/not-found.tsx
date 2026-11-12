'use client';

/*
 * 404 de la section localisée. QUOI : le code en grand (serif, ivoire), une phrase qui dit ce qui
 *   s'est passé, une phrase qui dit quoi faire, et UNE action principale (or) — retour à l'accueil —
 *   plus deux chemins sobres vers les pages les plus visitées.
 * POURQUOI `min-h-[60vh]` et pas `min-h-screen` : la page est rendue ENTRE l'en-tête et le pied ;
 *   un écran entier poussait le pied hors de la vue et forçait un défilement pour rien.
 */
import { Link } from '@/i18n/navigation';
import { useTranslations } from 'next-intl';

import { boutonOr, boutonSobre, etiquette } from '@/components/site-page/kit';

export default function NotFound() {
  const t = useTranslations('errors');
  const tCommon = useTranslations('common');

  return (
    <div className="container flex min-h-[60vh] max-w-3xl flex-col justify-center px-5 py-20 md:px-8">
      <p className={etiquette}>
        {t('notFoundEyebrow')}
        <span aria-hidden="true" className="mt-3 block h-px w-12 bg-gold/70" />
      </p>
      <h1 className="mt-5 text-4xl font-medium leading-tight text-foreground md:text-5xl">
        <span className="font-heading tabular-nums text-muted-foreground">404</span>
        <span aria-hidden="true" className="mx-3 text-border">·</span>
        {t('notFound')}
      </h1>
      <p className="mt-5 max-w-xl text-base leading-relaxed text-muted-foreground md:text-lg">{t('notFoundHint')}</p>
      <div className="mt-9 flex flex-wrap gap-3">
        <Link href="/" className={boutonOr}>
          {tCommon('backToHome')}
        </Link>
        <Link href="/portfolio" className={boutonSobre}>
          {t('goPortfolio')}
        </Link>
        <Link href="/cv" className={boutonSobre}>
          {t('goCv')}
        </Link>
      </div>
    </div>
  );
}
