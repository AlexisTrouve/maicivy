'use client';

import { useEffect, useState } from 'react';
import { Users, Eye, FileText, TrendingUp } from 'lucide-react';
import { useTranslations } from 'next-intl';

interface AnalyticsStats {
  totalVisitors: number;
  totalPageViews: number;
  totalLetters: number;
  conversionRate: number;
  activeVisitors?: number;
}

export default function StatsOverview() {
  const t = useTranslations('analytics.widgets.stats');
  const [stats, setStats] = useState<AnalyticsStats>({
    totalVisitors: 0,
    totalPageViews: 0,
    totalLetters: 0,
    conversionRate: 0,
    activeVisitors: 0,
  });
  const [isLoading, setIsLoading] = useState(true);

  useEffect(() => {
    fetchStats();

    // Auto-refresh every 30 seconds
    const interval = setInterval(fetchStats, 30000);
    return () => clearInterval(interval);
  }, []);

  const fetchStats = async () => {
    try {
      const backendUrl = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080';
      const response = await fetch(`${backendUrl}/api/v1/analytics/stats`, {
        credentials: 'include',
      });

      if (!response.ok) throw new Error('Failed to fetch stats');

      const data = await response.json();
      setStats(data);
    } catch (error) {
      console.error('Error fetching stats:', error);
      // Mock data for development
      setStats({
        totalVisitors: 1543,
        totalPageViews: 8234,
        totalLetters: 456,
        conversionRate: 29.6,
        activeVisitors: 12,
      });
    } finally {
      setIsLoading(false);
    }
  };

  const statCards = [
    {
      title: t('visitors'),
      value: stats.totalVisitors,
      icon: Users,
      subtitle: `+${stats.activeVisitors || 0} actifs`,
      color: 'text-blue-500',
    },
    {
      title: t('pageViews'),
      value: stats.totalPageViews,
      icon: Eye,
      subtitle: '+234 aujourd\'hui',
      color: 'text-purple-500',
    },
    {
      title: t('letters'),
      value: stats.totalLetters,
      icon: FileText,
      subtitle: '+12 aujourd\'hui',
      color: 'text-green-500',
    },
    {
      title: t('conversion'),
      value: `${stats.conversionRate}%`,
      icon: TrendingUp,
      subtitle: '+2.3% vs hier',
      color: 'text-orange-500',
    },
  ];

  if (isLoading) {
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

  return (
    <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
      {statCards.map((card, index) => {
        const Icon = card.icon;
        return (
          <div key={index} className="rounded-lg border bg-card p-6">
            <div className="flex items-center justify-between mb-2">
              <h3 className="text-sm font-medium text-muted-foreground">{card.title}</h3>
              <Icon className={`h-4 w-4 ${card.color}`} />
            </div>
            <div className="flex flex-col">
              <div className="text-3xl font-bold">{card.value}</div>
              <p className="text-xs text-muted-foreground mt-1">{card.subtitle}</p>
            </div>
          </div>
        );
      })}
    </div>
  );
}
