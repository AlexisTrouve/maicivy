// /[locale]/privacy — politique de confidentialité. Mise en page : TexteJuridique (partagée avec
// /legal) ; texte : namespace next-intl `privacy`, une clé par section.
import { Metadata } from 'next';
import { getTranslations } from 'next-intl/server';

import { TexteJuridique } from '@/components/site-page/TexteJuridique';

// Ordre de lecture des sections (clés du namespace).
const SECTIONS = ['collection', 'cookies', 'ai', 'retention', 'rights', 'contact'] as const;

type PageProps = { params: Promise<{ locale: string }> };

export async function generateMetadata({ params }: PageProps): Promise<Metadata> {
  const { locale } = await params;
  const t = await getTranslations({ locale, namespace: 'privacy' });
  return { title: t('title') };
}

export default async function PrivacyPage({ params }: PageProps) {
  const { locale } = await params;
  const t = await getTranslations({ locale, namespace: 'privacy' });

  return (
    <TexteJuridique
      surtitre={t('eyebrow')}
      titre={t('title')}
      sections={SECTIONS.map((id) => ({ id, titre: t(`${id}.title`), texte: t(`${id}.text`) }))}
    />
  );
}
