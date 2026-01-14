'use client';

import { useEffect, useState } from 'react';
import { FileText } from 'lucide-react';
import { useTranslations } from 'next-intl';

interface LettersStat {
  date: string;
  count: number;
}

type Period = 'day' | 'week' | 'month';

export default function LettersGenerated() {
  const t = useTranslations('analytics.widgets.letters');
  const tPeriods = useTranslations('analytics.periods');
  const [stats, setStats] = useState<LettersStat[]>([]);
  const [period, setPeriod] = useState<Period>('week');
  const [totalLetters, setTotalLetters] = useState(0);
  const [isLoading, setIsLoading] = useState(true);

  useEffect(() => {
    fetchLettersStats();

    // Auto-refresh every 30 seconds
    const interval = setInterval(fetchLettersStats, 30000);
    return () => clearInterval(interval);
  }, [period]);

  const fetchLettersStats = async () => {
    try {
      const backendUrl = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080';
      const response = await fetch(
        `${backendUrl}/api/v1/analytics/letters?period=${period}`,
        { credentials: 'include' }
      );

      if (!response.ok) throw new Error('Failed to fetch letters stats');

      const json = await response.json();

      // Handle API response format: { success: true, data: {...} }
      if (json.success && json.data) {
        const data = json.data;
        // Total is motivation + anti_motivation
        const total = (data.motivation || 0) + (data.anti_motivation || 0);
        setTotalLetters(total);

        // If there's a history array, use it; otherwise create a simple display
        if (Array.isArray(data.history) && data.history.length > 0) {
          setStats(data.history);
        } else {
          // Create a single data point for display
          setStats([
            { date: new Date().toISOString().split('T')[0], count: total }
          ]);
        }
      } else {
        setStats([]);
        setTotalLetters(0);
      }
    } catch (error) {
      console.error('Error fetching letters stats:', error);
      setStats([]);
      setTotalLetters(0);
    } finally {
      setIsLoading(false);
    }
  };

  const maxCount = Math.max(...stats.map((s) => s.count), 1);

  const formatDate = (dateString: string): string => {
    const date = new Date(dateString);
    return date.toLocaleDateString('fr-FR', {
      day: '2-digit',
      month: 'short',
    });
  };

  if (isLoading) {
    return (
      <div className="rounded-lg border bg-card p-6 animate-pulse h-80">
        <div className="h-6 bg-muted rounded w-1/2 mb-4" />
        <div className="h-48 bg-muted rounded" />
      </div>
    );
  }

  return (
    <div className="rounded-lg border bg-card p-6">
      <div className="flex items-center justify-between mb-4">
        <h2 className="text-xl font-semibold flex items-center gap-2">
          <FileText className="h-5 w-5" />
          {t('title')}
        </h2>

        {/* Period selector */}
        <div className="flex gap-1 bg-muted p-1 rounded-md">
          {(['day', 'week', 'month'] as Period[]).map((p) => (
            <button
              key={p}
              onClick={() => setPeriod(p)}
              className={`px-3 py-1 text-xs rounded transition-colors ${
                period === p
                  ? 'bg-primary text-primary-foreground'
                  : 'hover:bg-background'
              }`}
            >
              {p === 'day' ? tPeriods('day') : p === 'week' ? tPeriods('week') : tPeriods('month')}
            </button>
          ))}
        </div>
      </div>

      {/* Total counter */}
      <div className="mb-6 text-center">
        <div className="text-3xl font-bold text-primary">{totalLetters}</div>
        <div className="text-xs text-muted-foreground">{t('total')}</div>
      </div>

      {/* Simple line chart using SVG */}
      <div className="h-48 relative">
        {stats.length > 0 ? (
          <svg className="w-full h-full" viewBox="0 0 400 150" preserveAspectRatio="none">
            {/* Grid lines */}
            <line x1="0" y1="0" x2="400" y2="0" stroke="currentColor" strokeWidth="0.5" opacity="0.1" />
            <line x1="0" y1="37.5" x2="400" y2="37.5" stroke="currentColor" strokeWidth="0.5" opacity="0.1" />
            <line x1="0" y1="75" x2="400" y2="75" stroke="currentColor" strokeWidth="0.5" opacity="0.1" />
            <line x1="0" y1="112.5" x2="400" y2="112.5" stroke="currentColor" strokeWidth="0.5" opacity="0.1" />
            <line x1="0" y1="150" x2="400" y2="150" stroke="currentColor" strokeWidth="0.5" opacity="0.1" />

            {/* Area fill */}
            <path
              d={`M 0 150 ${stats
                .map((stat, i) => {
                  const x = (i / (stats.length - 1)) * 400;
                  const y = 150 - (stat.count / maxCount) * 140;
                  return `L ${x} ${y}`;
                })
                .join(' ')} L 400 150 Z`}
              fill="hsl(var(--primary))"
              fillOpacity="0.1"
            />

            {/* Line */}
            <polyline
              points={stats
                .map((stat, i) => {
                  const x = (i / (stats.length - 1)) * 400;
                  const y = 150 - (stat.count / maxCount) * 140;
                  return `${x},${y}`;
                })
                .join(' ')}
              fill="none"
              stroke="hsl(var(--primary))"
              strokeWidth="2"
            />

            {/* Points */}
            {stats.map((stat, i) => {
              const x = (i / (stats.length - 1)) * 400;
              const y = 150 - (stat.count / maxCount) * 140;
              return (
                <circle
                  key={i}
                  cx={x}
                  cy={y}
                  r="3"
                  fill="hsl(var(--primary))"
                  className="hover:r-5 transition-all"
                />
              );
            })}
          </svg>
        ) : (
          <div className="flex items-center justify-center h-full text-muted-foreground">
            {t('noData')}
          </div>
        )}
      </div>

      {/* X-axis labels */}
      {stats.length > 0 && (
        <div className="flex justify-between mt-2 text-xs text-muted-foreground">
          <span>{formatDate(stats[0].date)}</span>
          {stats.length > 2 && (
            <span>{formatDate(stats[Math.floor(stats.length / 2)].date)}</span>
          )}
          <span>{formatDate(stats[stats.length - 1].date)}</span>
        </div>
      )}

      <div className="text-xs text-muted-foreground text-center mt-4">
        {t('evolution', { period: period === 'day' ? '24h' : period === 'week' ? '7 jours' : '30 jours' })}
      </div>
    </div>
  );
}
