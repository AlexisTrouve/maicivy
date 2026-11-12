import { getTranslations } from 'next-intl/server';
import { Download, MessageCircle } from 'lucide-react';

import { cvExportUrl } from '@/lib/montagne/links';
import styles from './montagne.module.css';

/**
 * MontagneVoice — la voix d'Alexis, en tête de la montagne : une phrase, un titre, deux actions.
 *
 * QUOI : le grand titre serif (`montagne.voice.title`, la phrase qu'Alexis réécrira lui-même —
 * une seule clé, dans les 5 fichiers de messages, FR = source), la ligne dorée espacée
 * (`montagne.voice.tagline`), puis le CV en PDF et le chat avec son agent.
 * POURQUOI côté serveur : c'est le premier écran, il doit s'afficher sans attendre le JavaScript.
 * Doctrine de l'or : le CV est L'action principale de l'écran, la seule en or plein ; le chat est
 * secondaire (contour neutre). La ligne dorée est un signal d'identité, courte, pas du texte courant.
 * E2E : `montagne-voice-*`.
 */
export async function MontagneVoice({ locale }: { locale: string }) {
  const t = await getTranslations({ locale, namespace: 'montagne.voice' });
  return (
    <section data-testid="montagne-voice" className="px-5 pb-8 pt-9 text-center sm:pt-12 xl:pb-10">
      <h1
        data-testid="montagne-voice-title"
        className={`${styles.serif} mx-auto max-w-4xl text-balance text-[2.35rem] xl:max-w-6xl font-medium leading-[1.05] text-[#efe6d2] sm:text-5xl xl:text-[4.4rem]`}
      >
        {t('title')}
      </h1>
      <p
        data-testid="montagne-voice-tagline"
        className="mx-auto mt-4 whitespace-nowrap text-[10px] font-medium uppercase tracking-[0.24em] text-gold sm:text-[13px] sm:tracking-[0.42em]"
      >
        {t('tagline')}
      </p>
      <div className="mt-7 flex flex-wrap items-center justify-center gap-3">
        <a
          href={cvExportUrl(locale)}
          data-testid="montagne-voice-cv"
          className="inline-flex min-h-[44px] items-center gap-2 rounded-md bg-gradient-to-b from-gold-bright to-gold-deep px-5 text-sm font-semibold tracking-wide text-slate-950 shadow-gold transition-[filter] hover:brightness-110 focus:outline-none focus-visible:ring-2 focus-visible:ring-gold-soft"
        >
          <Download className="h-4 w-4" aria-hidden="true" />
          {t('cv')}
        </a>
        <a
          href={`/${locale}/chat`}
          data-testid="montagne-voice-chat"
          className="inline-flex min-h-[44px] items-center gap-2 rounded-md border border-white/20 bg-white/[0.03] px-5 text-sm text-slate-200 transition-colors hover:border-white/40 hover:text-white focus:outline-none focus-visible:ring-2 focus-visible:ring-gold/60"
        >
          <MessageCircle className="h-4 w-4" aria-hidden="true" />
          {t('chat')}
        </a>
      </div>
    </section>
  );
}
