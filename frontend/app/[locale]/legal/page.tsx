// /[locale]/legal — mentions légales. Mise en page : TexteJuridique (partagée avec /privacy) ;
// texte : namespace next-intl `legal`, une clé par section.
import { Metadata } from 'next';
import { getTranslations } from 'next-intl/server';

import { TexteJuridique } from '@/components/site-page/TexteJuridique';

// Ordre de lecture des sections (clés du namespace).
const SECTIONS = ['editor', 'hosting', 'ip', 'ai', 'liability'] as const;

type PageProps = { params: Promise<{ locale: string }> };

export async function generateMetadata({ params }: PageProps): Promise<Metadata> {
  const { locale } = await params;
  const t = await getTranslations({ locale, namespace: 'legal' });
  return { title: t('title') };
}

export default async function LegalPage({ params }: PageProps) {
  const { locale } = await params;
  const t = await getTranslations({ locale, namespace: 'legal' });

  return (
    <TexteJuridique
      surtitre={t('eyebrow')}
      titre={t('title')}
      sections={SECTIONS.map((id) => ({ id, titre: t(`${id}.title`), texte: t(`${id}.text`) }))}
    />
  );
}
