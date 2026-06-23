'use client';

import { useEffect, useState } from 'react';
import { Users, Eye, FileText, TrendingUp, BookOpen } from 'lucide-react';
import { useTranslations } from 'next-intl';
import { useSearchParams } from 'next/navigation';

interface AnalyticsStats {
  totalVisitors: number;
  totalPageViews: number;
  totalLetters: number;
  conversionRate: number;
  totalBlogReads: number;
}

const ZERO_STATS: AnalyticsStats = {
  totalVisitors: 0,
  totalPageViews: 0,
  totalLetters: 0,
  conversionRate: 0,
  totalBlogReads: 0,
};

// Le filtre de période (DateFilter) écrit ?period=<preset> dans l'URL ; on le mappe vers la période
// attendue par le backend (day|week|month). 'all' → month (le backend ne gère pas 'all').
const PRESET_TO_PERIOD: Record<string, string> = {
  today: 'day',
  '7d': 'week',
  '30d': 'month',
  all: 'month',
};

export default function StatsOverview() {
  const t = useTranslations('analytics.widgets.stats');
  const searchParams = useSearchParams();
  // Période pilotée par le filtre (DateFilter écrit ?period=). Défaut 7 jours → 'week'.
  const period = PRESET_TO_PERIOD[searchParams.get('period') || '7d'] || 'week';
  const [stats, setStats] = useState<AnalyticsStats>(ZERO_STATS);
  const [isLoading, setIsLoading] = useState(true);

  // Refetch au montage, toutes les 30s, ET à chaque changement de période (le filtre).
  useEffect(() => {
    fetchStats();
    const interval = setInterval(fetchStats, 30000);
    return () => clearInterval(interval);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [period]);

  const fetchStats = async () => {
    try {
      const backendUrl = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080';
      const response = await fetch(`${backendUrl}/api/v1/analytics/stats?period=${period}`, {
        credentials: 'include',
      });

      if (!response.ok) throw new Error('Failed to fetch stats');

      const statsJson = await response.json();

      // Map API response (snake_case, {success, data}) → format affiché. Toutes les valeurs viennent du
      // backend (réel + synthétique DemoMetrics si DEMO_METRICS, gaté par les vrais users). Plus aucun
      // delta/sous-titre fabriqué côté front.
      if (statsJson.success && statsJson.data) {
        setStats({
          totalVisitors: statsJson.data.unique_visitors || 0,
          totalPageViews: statsJson.data.total_events || 0,
          totalLetters: statsJson.data.letters_generated || 0,
          conversionRate: Math.round((statsJson.data.conversion_rate || 0) * 100 * 10) / 10,
          totalBlogReads: statsJson.data.blog_reads || 0,
        });
      } else {
        throw new Error('Invalid response format');
      }
    } catch (error) {
      console.error('Error fetching stats:', error);
      setStats(ZERO_STATS); // échec → zéros honnêtes (jamais de mock en prod)
    } finally {
      setIsLoading(false);
    }
  };

  const statCards = [
    { title: t('visitors'), value: stats.totalVisitors, icon: Users, color: 'text-blue-500' },
    { title: t('pageViews'), value: stats.totalPageViews, icon: Eye, color: 'text-purple-500' },
    { title: t('letters'), value: stats.totalLetters, icon: FileText, color: 'text-green-500' },
    { title: t('blogReads'), value: stats.totalBlogReads, icon: BookOpen, color: 'text-cyan-500' },
    { title: t('conversion'), value: `${stats.conversionRate}%`, icon: TrendingUp, color: 'text-orange-500' },
  ];

  if (isLoading) {
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

  return (
    <div className="grid grid-cols-2 md:grid-cols-3 lg:grid-cols-5 gap-4">
      {statCards.map((card, index) => {
        const Icon = card.icon;
        return (
          <div key={index} className="rounded-lg border bg-card p-6">
            <div className="flex items-center justify-between mb-2">
              <h3 className="text-sm font-medium text-muted-foreground">{card.title}</h3>
              <Icon className={`h-4 w-4 ${card.color}`} />
            </div>
            <div className="text-3xl font-bold">{card.value}</div>
          </div>
        );
      })}
    </div>
  );
}
