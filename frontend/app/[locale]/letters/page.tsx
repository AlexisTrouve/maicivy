import { Metadata } from 'next';
import { getTranslations } from 'next-intl/server';
import { LetterGenerator } from '@/components/letters/LetterGenerator';

// Force dynamic rendering
export const dynamic = 'force-dynamic';
import { AccessGate } from '@/components/letters/AccessGate';

// Metadata is now generated dynamically via generateMetadata
export async function generateMetadata({ params }: { params: Promise<{ locale: string }> | { locale: string } }): Promise<Metadata> {
  const resolvedParams = params instanceof Promise ? await params : params;
  const locale = resolvedParams.locale || 'en';
  const messages = (await import(`@/messages/${locale}.json`)).default;

  return {
    title: `${messages.letters.title} | maicivy`,
    description: messages.letters.subtitle,
    openGraph: {
      title: messages.letters.title,
      description: messages.letters.subtitle,
    },
  };
}

export default async function LettersPage({ params }: { params: Promise<{ locale: string }> | { locale: string } }) {
  // Handle both Promise and non-Promise params (Next.js 14.2+ compatibility)
  const resolvedParams = params instanceof Promise ? await params : params;
  const t = await getTranslations({ locale: resolvedParams.locale, namespace: 'letters' });

  return (
    <div className="min-h-screen bg-gradient-to-br from-slate-50 via-white to-blue-50 dark:from-slate-900 dark:via-slate-800 dark:to-slate-900">
      <div className="container mx-auto px-4 py-12">
        {/* Header */}
        <div className="text-center mb-12">
          <h1 className="text-4xl md:text-5xl font-bold mb-4 text-slate-900 dark:text-white">
            {t('title')}
          </h1>
          <p className="text-lg text-slate-600 dark:text-slate-300 max-w-2xl mx-auto">
            {t('subtitle')}
          </p>
        </div>

        {/* Access Gate + Generator */}
        <AccessGate>
          <LetterGenerator />
        </AccessGate>
      </div>
    </div>
  );
}
