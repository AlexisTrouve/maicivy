import { Metadata } from 'next';
import { Suspense } from 'react';
import { getTranslations } from 'next-intl/server';

// Force dynamic rendering
export const dynamic = 'force-dynamic';
import RealtimeVisitors from '@/components/analytics/RealtimeVisitors';
import ThemeStats from '@/components/analytics/ThemeStats';
import LettersGenerated from '@/components/analytics/LettersGenerated';
import Heatmap from '@/components/analytics/Heatmap';
import DateFilter from '@/components/analytics/DateFilter';
import StatsOverview from '@/components/analytics/StatsOverview';

export const metadata: Metadata = {
  title: 'Analytics Dashboard - maicivy',
  description: 'Dashboard analytics en temps réel du CV interactif avec IA',
  openGraph: {
    title: 'Analytics Dashboard - maicivy',
    description: 'Statistiques publiques en temps réel',
  },
};

export default async function AnalyticsPage() {
  const t = await getTranslations('analytics');

  return (
    <div className="container mx-auto px-4 py-8">
      {/* Header */}
      <div className="mb-8">
        <h1 className="text-4xl font-bold mb-2">{t('title')}</h1>
        <p className="text-muted-foreground">
          {t('subtitle')} - {t('description')}
        </p>
      </div>

      {/* Filters */}
      <div className="mb-6">
        <Suspense fallback={<div className="h-10 bg-muted rounded animate-pulse" />}>
          <DateFilter />
        </Suspense>
      </div>

      {/* Stats Overview Cards */}
      <div className="mb-6">
        <Suspense fallback={<StatsOverviewSkeleton />}>
          <StatsOverview />
        </Suspense>
      </div>

      {/* Grid Layout */}
      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
        {/* Realtime Visitors - Full width */}
        <div className="lg:col-span-3">
          <Suspense fallback={<RealtimeVisitorsSkeleton />}>
            <RealtimeVisitors />
          </Suspense>
        </div>

        {/* Theme Stats - 2 cols sur desktop */}
        <div className="lg:col-span-2">
          <Suspense fallback={<StatsSkeleton />}>
            <ThemeStats />
          </Suspense>
        </div>

        {/* Letters Generated - 1 col sur desktop */}
        <div className="lg:col-span-1">
          <Suspense fallback={<StatsSkeleton />}>
            <LettersGenerated />
          </Suspense>
        </div>

        {/* Heatmap - Full width */}
        <div className="lg:col-span-3">
          <Suspense fallback={<HeatmapSkeleton />}>
            <Heatmap />
          </Suspense>
        </div>
      </div>

      {/* Footer Note */}
      <div className="mt-8 p-4 border rounded-lg bg-muted/50">
        <p className="text-sm text-muted-foreground text-center">
          {t('privacy')}
        </p>
      </div>
    </div>
  );
}

// Skeleton components pour loading states
function StatsOverviewSkeleton() {
  return (
    <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
      {[...Array(4)].map((_, i) => (
        <div key={i} className="rounded-lg border bg-card p-6 animate-pulse">
          <div className="h-6 bg-muted rounded w-1/2 mb-2" />
          <div className="h-8 bg-muted rounded w-3/4" />
        </div>
      ))}
    </div>
  );
}

function RealtimeVisitorsSkeleton() {
  return (
    <div className="rounded-lg border bg-card p-6 animate-pulse">
      <div className="h-24 bg-muted rounded" />
    </div>
  );
}

function StatsSkeleton() {
  return (
    <div className="rounded-lg border bg-card p-6 animate-pulse">
      <div className="h-64 bg-muted rounded" />
    </div>
  );
}

function HeatmapSkeleton() {
  return (
    <div className="rounded-lg border bg-card p-6 animate-pulse">
      <div className="h-96 bg-muted rounded" />
    </div>
  );
}
