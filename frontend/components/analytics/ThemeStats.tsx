'use client';

import { useEffect, useState } from 'react';
import { BarChart3 } from 'lucide-react';
import { useTranslations } from 'next-intl';

interface ThemeStat {
  theme: string;
  count: number;
  percentage: number;
}

export default function ThemeStats() {
  const t = useTranslations('analytics.widgets.themes');
  const [stats, setStats] = useState<ThemeStat[]>([]);
  const [isLoading, setIsLoading] = useState(true);

  useEffect(() => {
    fetchThemeStats();

    // Auto-refresh every 30 seconds
    const interval = setInterval(fetchThemeStats, 30000);
    return () => clearInterval(interval);
  }, []);

  const fetchThemeStats = async () => {
    try {
      const backendUrl = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080';
      const response = await fetch(`${backendUrl}/api/v1/analytics/themes`, {
        credentials: 'include',
      });

      if (!response.ok) throw new Error('Failed to fetch theme stats');

      const json = await response.json();

      // Handle API response format: { success: true, data: [...] }
      if (json.success && Array.isArray(json.data)) {
        // Calculate total for percentages
        // Backend returns 'views' field, not 'count'
        const total = json.data.reduce((sum: number, item: { views?: number; count?: number }) => sum + (item.views || item.count || 0), 0);

        const mappedStats = json.data.map((item: { theme: string; views?: number; count?: number }) => ({
          theme: item.theme || 'unknown',
          count: item.views || item.count || 0,
          percentage: total > 0 ? ((item.views || item.count || 0) / total) * 100 : 0,
        }));
        setStats(mappedStats);
      } else {
        setStats([]);
      }
    } catch (error) {
      console.error('Error fetching theme stats:', error);
      setStats([]);
    } finally {
      setIsLoading(false);
    }
  };

  if (isLoading) {
    return (
      <div className="rounded-lg border bg-card p-5 md:p-6 animate-pulse h-80">
        <div className="h-6 bg-muted rounded w-1/3 mb-4" />
        <div className="space-y-3">
          {[...Array(5)].map((_, i) => (
            <div key={i} className="h-8 bg-muted rounded" />
          ))}
        </div>
      </div>
    );
  }

  return (
    <div className="h-full rounded-lg border bg-card p-5 md:p-6">
      <h2 className="text-xl font-medium flex items-center gap-2 mb-6">
        <BarChart3 aria-hidden="true" className="h-5 w-5 text-muted-foreground" />
        {t('title')}
      </h2>

      <div className="space-y-4">
        {stats.map((stat, index) => (
          <div key={stat.theme} className="space-y-2">
            <div className="flex items-center justify-between text-sm">
              <div className="flex items-center gap-2">
                <span className="font-medium text-muted-foreground w-6">
                  #{index + 1}
                </span>
                <span className="font-medium capitalize">{stat.theme}</span>
              </div>
              <div className="flex items-center gap-2">
                <span className="text-muted-foreground">{stat.count} {t('views')}</span>
                <span className="text-xs bg-muted px-2 py-1 rounded min-w-[3rem] text-center">
                  {stat.percentage.toFixed(1)}%
                </span>
              </div>
            </div>
            {/* Barre monochrome (ivoire) : le thème est déjà nommé et classé à côté, une couleur par
                thème n'ajoutait aucune information — seulement du bruit sur la nuit. */}
            <div className="w-full bg-muted rounded-full h-2 overflow-hidden">
              <div
                data-theme={stat.theme}
                className="h-full rounded-full bg-foreground/70 transition-all duration-500"
                style={{ width: `${stat.percentage}%` }}
              />
            </div>
          </div>
        ))}
      </div>

      <div className="text-xs text-muted-foreground text-center mt-6">
        {t('update')}
      </div>
    </div>
  );
}
