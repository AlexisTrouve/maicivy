import { Metadata } from 'next';
import { getTranslations } from 'next-intl/server';
import { LetterGenerator } from '@/components/letters/LetterGenerator';

// Force dynamic rendering
export const dynamic = 'force-dynamic';
import { AccessGate } from '@/components/letters/AccessGate';

export const metadata: Metadata = {
  title: 'Générateur de Lettres IA | maicivy',
  description: 'Générez des lettres de motivation et anti-motivation personnalisées par IA pour vos candidatures.',
  openGraph: {
    title: 'Générateur de Lettres IA',
    description: 'Lettres de motivation/anti-motivation générées par IA',
  },
};

export default async function LettersPage() {
  const t = await getTranslations('letters');

  return (
    <div className="min-h-screen bg-gradient-to-br from-slate-50 via-white to-blue-50 dark:from-slate-900 dark:via-slate-800 dark:to-slate-900">
      <div className="container mx-auto px-4 py-12">
        {/* Header */}
        <div className="text-center mb-12">
          <h1 className="text-4xl md:text-5xl font-bold mb-4 bg-clip-text text-transparent bg-gradient-to-r from-blue-600 to-purple-600 dark:from-blue-400 dark:to-purple-400">
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
