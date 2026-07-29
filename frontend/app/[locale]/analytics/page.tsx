import { Metadata } from 'next';
import { Suspense } from 'react';
import { getTranslations } from 'next-intl/server';
import { loadMessages } from '@/i18n/messages';

// Force dynamic rendering
export const dynamic = 'force-dynamic';
import TestStatsCard from '@/components/analytics/TestStatsCard';
import RealtimeVisitors from '@/components/analytics/RealtimeVisitors';
import ThemeStats from '@/components/analytics/ThemeStats';
import LettersGenerated from '@/components/analytics/LettersGenerated';
import Heatmap from '@/components/analytics/Heatmap';
import DateFilter from '@/components/analytics/DateFilter';
import StatsOverview from '@/components/analytics/StatsOverview';

// Generate dynamic metadata
export async function generateMetadata({ params }: { params: Promise<{ locale: string }> | { locale: string } }): Promise<Metadata> {
  const resolvedParams = params instanceof Promise ? await params : params;
  const locale = resolvedParams.locale || 'en';
  const messages = loadMessages(locale);

  return {
    title: `${messages.analytics.title} - maicivy`,
    description: messages.analytics.subtitle,
    openGraph: {
      title: `${messages.analytics.title} - maicivy`,
      description: messages.analytics.subtitle,
    },
  };
}

export default async function AnalyticsPage({ params }: { params: Promise<{ locale: string }> | { locale: string } }) {
  const resolvedParams = params instanceof Promise ? await params : params;
  const t = await getTranslations({ locale: resolvedParams.locale, namespace: 'analytics' });

  return (
    <div className="container mx-auto px-4 py-8">
      {/* Header */}
      <div className="mb-8">
        <h1 className="text-4xl font-bold mb-2">{t('title')}</h1>
        <p className="text-muted-foreground">
          {t('subtitle')} - {t('description')}
        </p>
      </div>

      {/* Tests & Qualité — santé projet : nombre réel de tests automatisés (cf. TestStatsCard).
          Le portrait dev (LOC/commits/tendances), lui, est sur le CV. */}
      <div className="mb-8">
        <TestStatsCard />
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
    <div className="grid grid-cols-2 md:grid-cols-3 lg:grid-cols-5 gap-4">
      {[...Array(5)].map((_, i) => (
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
