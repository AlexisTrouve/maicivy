import { Metadata } from 'next';
import { getTranslations } from 'next-intl/server';

export async function generateMetadata({ params }: { params: Promise<{ locale: string }> | { locale: string } }): Promise<Metadata> {
  const resolvedParams = await Promise.resolve(params);
  const t = await getTranslations({ locale: resolvedParams.locale, namespace: 'privacy' });
  return { title: t('title') };
}

export default async function PrivacyPage({ params }: { params: Promise<{ locale: string }> | { locale: string } }) {
  const resolvedParams = await Promise.resolve(params);
  const t = await getTranslations({ locale: resolvedParams.locale, namespace: 'privacy' });

  return (
    <div className="container max-w-3xl py-12 md:py-16">
      <h1 className="text-3xl font-bold tracking-tight">{t('title')}</h1>

      <div className="mt-8 space-y-8 text-sm leading-relaxed text-muted-foreground">
        <section>
          <h2 className="mb-3 text-lg font-semibold text-foreground">{t('collection.title')}</h2>
          <p>{t('collection.text')}</p>
        </section>

        <section>
          <h2 className="mb-3 text-lg font-semibold text-foreground">{t('cookies.title')}</h2>
          <p>{t('cookies.text')}</p>
        </section>

        <section>
          <h2 className="mb-3 text-lg font-semibold text-foreground">{t('ai.title')}</h2>
          <p>{t('ai.text')}</p>
        </section>

        <section>
          <h2 className="mb-3 text-lg font-semibold text-foreground">{t('retention.title')}</h2>
          <p>{t('retention.text')}</p>
        </section>

        <section>
          <h2 className="mb-3 text-lg font-semibold text-foreground">{t('rights.title')}</h2>
          <p>{t('rights.text')}</p>
        </section>

        <section>
          <h2 className="mb-3 text-lg font-semibold text-foreground">{t('contact.title')}</h2>
          <p>{t('contact.text')}</p>
        </section>
      </div>
    </div>
  );
}
