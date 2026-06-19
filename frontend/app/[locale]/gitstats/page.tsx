import { Metadata } from 'next';
import { useTranslations } from 'next-intl';
import GitStatsPanel from '@/components/cv/GitStatsPanel';

export const dynamic = 'force-dynamic';

export const metadata: Metadata = {
  title: 'Git Stats - Alexis',
  description: 'Statistiques Git : commits, lignes de code, repos actifs sur 6 mois',
};

export default function GitStatsPage() {
  const t = useTranslations('gitstats');
  return (
    <div className="container mx-auto px-4 py-8 max-w-6xl">
      <header className="mb-10 text-center">
        <h1 className="text-4xl md:text-5xl font-bold mb-3 bg-gradient-to-r from-orange-500 to-red-500 bg-clip-text text-transparent">
          Git Stats
        </h1>
        <p className="text-lg text-gray-600 dark:text-gray-300">
          {t('subtitle')}
        </p>
      </header>

      <GitStatsPanel />
    </div>
  );
}
