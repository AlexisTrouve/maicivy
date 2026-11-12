import { Metadata } from 'next';
import { getTranslations } from 'next-intl/server';
import GitStatsPanel from '@/components/cv/GitStatsPanel';
import { socialMetadata } from '@/lib/seo/social';

export const dynamic = 'force-dynamic';

interface GitStatsPageProps {
  params: Promise<{ locale: string }>;
}

// Métadonnées traduites : le titre et la description étaient en français en dur, donc servis tels
// quels aux moteurs et aux aperçus de lien sur /en, /de, /it et /zh.
export async function generateMetadata({ params }: GitStatsPageProps): Promise<Metadata> {
  const { locale } = await params;
  const t = await getTranslations({ locale, namespace: 'gitstats' });
  return {
    title: t('metaTitle'),
    description: t('metaDescription'),
    ...socialMetadata({ locale, title: t('metaTitle'), description: t('metaDescription') }),
  };
}

// Page /gitstats, maquette A « Le relevé ». L'en-tête (signature, titre, chapeau) est rendu ici,
// côté serveur ; le panneau le place à gauche de la date du relevé et du bouton « Actualiser »,
// qui dépendent des données. Pas de filet sous l'en-tête : c'est la ligne des quatre chiffres
// qui ouvre le registre.
export default async function GitStatsPage({ params }: GitStatsPageProps) {
  const { locale } = await params;
  const t = await getTranslations({ locale, namespace: 'gitstats' });
  // Signature du site (nom + positionnement), lue là où elle est déjà traduite.
  const tNav = await getTranslations({ locale, namespace: 'nav' });
  const tVoice = await getTranslations({ locale, namespace: 'montagne.voice' });

  const heading = (
    <header>
      <p className="text-xs uppercase tracking-[0.22em] text-muted-foreground">{`${tNav('brand')} · ${tVoice('tagline')}`}</p>
      <h1 className="mt-3 text-[clamp(2.6rem,5vw,4rem)] font-medium leading-[1.04] text-foreground">{t('title')}</h1>
      <p className="mt-3 max-w-[52ch] text-base text-foreground/80 md:text-[17px]">{t('subtitle')}</p>
    </header>
  );

  return (
    <div className="mx-auto w-full max-w-6xl px-5 pb-16 md:px-10">
      <GitStatsPanel heading={heading} />
    </div>
  );
}
