'use client';

import { useEffect, useState } from 'react';
import { Flame } from 'lucide-react';
import { useTranslations } from 'next-intl';

interface HeatmapPoint {
  x: number;
  y: number;
  intensity: number;
  element?: string;
}

interface HeatmapData {
  points: HeatmapPoint[];
  maxIntensity: number;
}

export default function Heatmap() {
  const t = useTranslations('analytics.widgets.heatmap');
  const [heatmapData, setHeatmapData] = useState<HeatmapData>({
    points: [],
    maxIntensity: 0,
  });
  const [isLoading, setIsLoading] = useState(true);
  const [hoveredPoint, setHoveredPoint] = useState<HeatmapPoint | null>(null);

  useEffect(() => {
    fetchHeatmapData();

    // Auto-refresh every 60 seconds
    const interval = setInterval(fetchHeatmapData, 60000);
    return () => clearInterval(interval);
  }, []);

  const fetchHeatmapData = async () => {
    try {
      const backendUrl = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080';
      const response = await fetch(`${backendUrl}/api/v1/analytics/heatmap`, {
        credentials: 'include',
      });

      if (!response.ok) throw new Error('Failed to fetch heatmap data');

      const json = await response.json();

      // Handle API response format: { success: true, data: [...] }
      if (json.success && Array.isArray(json.data)) {
        const points = json.data.map((item: { x: number; y: number; intensity?: number; count?: number; element?: string }) => ({
          x: item.x || 0,
          y: item.y || 0,
          intensity: item.intensity || item.count || 0,
          element: item.element,
        }));
        const maxIntensity = points.length > 0
          ? Math.max(...points.map((p: HeatmapPoint) => p.intensity))
          : 0;
        setHeatmapData({ points, maxIntensity });
      } else {
        setHeatmapData({ points: [], maxIntensity: 0 });
      }
    } catch (error) {
      console.error('Error fetching heatmap data:', error);
      setHeatmapData({ points: [], maxIntensity: 0 });
    } finally {
      setIsLoading(false);
    }
  };

  const getHeatColor = (intensity: number, max: number): string => {
    if (max === 0) return 'rgba(59, 130, 246, 0.1)'; // blue-500 very light

    const ratio = intensity / max;

    // Gradient bleu → vert → jaune → rouge
    if (ratio < 0.25) {
      return `rgba(59, 130, 246, ${0.2 + ratio * 2})`; // blue
    } else if (ratio < 0.5) {
      return `rgba(34, 197, 94, ${0.3 + ratio * 2})`; // green
    } else if (ratio < 0.75) {
      return `rgba(234, 179, 8, ${0.4 + ratio * 2})`; // yellow
    } else {
      return `rgba(239, 68, 68, ${0.5 + ratio * 2})`; // red
    }
  };

  const formatElementName = (element?: string): string => {
    if (!element) return t('unknownElement');
    return element
      .split('_')
      .map((word) => word.charAt(0).toUpperCase() + word.slice(1))
      .join(' ');
  };

  if (isLoading) {
    return (
      <div className="rounded-lg border bg-card p-6 animate-pulse h-96">
        <div className="h-6 bg-muted rounded w-1/3 mb-4" />
        <div className="h-80 bg-muted rounded" />
      </div>
    );
  }

  // Agrège les points par LIBELLÉ (somme des intensités) → liste "interactions principales" TOUJOURS
  // visible : le tooltip spatial seul était clippé par overflow-hidden et n'apparaissait qu'au survol.
  const topInteractions = Array.from(
    heatmapData.points.reduce((acc, p) => {
      const label = formatElementName(p.element);
      acc.set(label, (acc.get(label) || 0) + p.intensity);
      return acc;
    }, new Map<string, number>())
  )
    .sort((a, b) => b[1] - a[1])
    .slice(0, 7);
  const topMax = topInteractions.length > 0 ? topInteractions[0][1] : 1;

  return (
    <div className="rounded-lg border bg-card p-6">
      <div className="flex items-start justify-between mb-4 gap-4">
        <div>
          <h2 className="text-xl font-semibold flex items-center gap-2">
            <Flame className="h-5 w-5" />
            {t('title')}
          </h2>
          {/* Sous-titre explicatif : avant, on ne comprenait pas ce que la heatmap racontait. */}
          <p className="text-sm text-muted-foreground mt-1">{t('explain')}</p>
        </div>
        <div className="flex items-center gap-2 text-xs text-muted-foreground shrink-0">
          <span>{t('low')}</span>
          <div className="flex gap-1">
            <div
              className="w-6 h-4 rounded"
              style={{ backgroundColor: 'rgba(59, 130, 246, 0.5)' }}
            />
            <div
              className="w-6 h-4 rounded"
              style={{ backgroundColor: 'rgba(34, 197, 94, 0.6)' }}
            />
            <div
              className="w-6 h-4 rounded"
              style={{ backgroundColor: 'rgba(234, 179, 8, 0.7)' }}
            />
            <div
              className="w-6 h-4 rounded"
              style={{ backgroundColor: 'rgba(239, 68, 68, 0.8)' }}
            />
          </div>
          <span>{t('high')}</span>
        </div>
      </div>

      {/* Heatmap spatiale. Conteneur externe SANS overflow-hidden (sinon le tooltip était coupé près
          des bords) ; un inner overflow-hidden contient les pastilles floues. Plus haut (h-96) pour
          que tout tienne dedans. */}
      <div className="relative w-full h-96">
        <div className="absolute inset-0 bg-muted/20 rounded-lg overflow-hidden">
          {heatmapData.points.map((point, index) => (
            <div
              key={index}
              className="absolute rounded-full blur-xl transition-all duration-500 cursor-pointer"
              style={{
                left: `${point.x}%`,
                top: `${point.y}%`,
                width: '90px',
                height: '90px',
                transform: 'translate(-50%, -50%)',
                backgroundColor: getHeatColor(point.intensity, heatmapData.maxIntensity),
                pointerEvents: 'auto',
              }}
              onMouseEnter={() => setHoveredPoint(point)}
              onMouseLeave={() => setHoveredPoint(null)}
            />
          ))}

          {/* Overlay grid for reference */}
          <div className="absolute inset-0 grid grid-cols-10 grid-rows-10 pointer-events-none opacity-10">
            {Array.from({ length: 100 }).map((_, i) => (
              <div key={i} className="border border-foreground" />
            ))}
          </div>
        </div>

        {heatmapData.points.length === 0 && (
          <div className="absolute inset-0 flex items-center justify-center text-muted-foreground">
            {t('noData')}
          </div>
        )}

        {/* Tooltip — rendu HORS de l'inner overflow-hidden → plus jamais coupé. x clampé pour ne pas
            sortir sur les côtés ; positionné au-dessus de la pastille. */}
        {hoveredPoint && (
          <div
            className="absolute bg-popover text-popover-foreground px-3 py-2 rounded-md shadow-lg text-sm pointer-events-none z-20"
            style={{
              left: `${Math.min(Math.max(hoveredPoint.x, 12), 88)}%`,
              top: `${hoveredPoint.y}%`,
              transform: 'translate(-50%, calc(-100% - 8px))',
            }}
          >
            <div className="font-medium">{formatElementName(hoveredPoint.element)}</div>
            <div className="text-xs text-muted-foreground">
              {t('interactions', { count: hoveredPoint.intensity })}
            </div>
          </div>
        )}
      </div>

      {/* Interactions principales — libellés TOUJOURS visibles (le tooltip seul était souvent invisible
          / clippé). Agrégé par libellé, trié, avec une barre colorée à l'intensité. */}
      {topInteractions.length > 0 && (
        <div className="mt-5">
          <h3 className="text-sm font-medium text-muted-foreground mb-2">{t('topTitle')}</h3>
          <div className="space-y-1.5">
            {topInteractions.map(([label, count]) => (
              <div key={label} className="flex items-center gap-3 text-sm">
                <span className="flex-1 truncate" title={label}>
                  {label}
                </span>
                <div className="w-24 sm:w-32 h-2 rounded-full bg-muted overflow-hidden shrink-0">
                  <div
                    className="h-full rounded-full"
                    style={{ width: `${(count / topMax) * 100}%`, backgroundColor: getHeatColor(count, topMax) }}
                  />
                </div>
                <span className="w-8 text-right text-xs text-muted-foreground tabular-nums shrink-0">
                  {count}
                </span>
              </div>
            ))}
          </div>
        </div>
      )}

      <div className="text-xs text-muted-foreground text-center mt-4">
        {t('based', { count: heatmapData.points.length })}
      </div>
    </div>
  );
}
