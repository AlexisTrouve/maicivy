import {
  AnalyticsStats,
  ThemeStatsResponse,
  LettersStatsResponse,
  HeatmapData,
} from './types';

const BACKEND_URL = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080';

/**
 * Fetch general analytics stats
 */
export async function fetchAnalyticsStats(
  period: 'day' | 'week' | 'month' = 'week'
): Promise<AnalyticsStats> {
  const response = await fetch(
    `${BACKEND_URL}/api/v1/analytics/stats?period=${period}`,
    {
      credentials: 'include',
    }
  );

  if (!response.ok) {
    throw new Error(`Failed to fetch analytics stats: ${response.statusText}`);
  }

  return response.json();
}

/**
 * Fetch theme statistics
 */
export async function fetchThemeStats(): Promise<ThemeStatsResponse> {
  const response = await fetch(`${BACKEND_URL}/api/v1/analytics/themes`, {
    credentials: 'include',
  });

  if (!response.ok) {
    throw new Error(`Failed to fetch theme stats: ${response.statusText}`);
  }

  return response.json();
}

/**
 * Fetch letters generation statistics
 */
export async function fetchLettersStats(
  period: 'day' | 'week' | 'month' = 'week'
): Promise<LettersStatsResponse> {
  const response = await fetch(
    `${BACKEND_URL}/api/v1/analytics/letters?period=${period}`,
    {
      credentials: 'include',
    }
  );

  if (!response.ok) {
    throw new Error(`Failed to fetch letters stats: ${response.statusText}`);
  }

  return response.json();
}

/**
 * Fetch heatmap interaction data
 */
export async function fetchHeatmapData(page?: string): Promise<HeatmapData> {
  const url = page
    ? `${BACKEND_URL}/api/v1/analytics/heatmap?page=${page}`
    : `${BACKEND_URL}/api/v1/analytics/heatmap`;

  const response = await fetch(url, {
    credentials: 'include',
  });

  if (!response.ok) {
    throw new Error(`Failed to fetch heatmap data: ${response.statusText}`);
  }

  return response.json();
}

/**
 * Create WebSocket connection for realtime analytics
 */
export function createAnalyticsWebSocket(
  onMessage: (data: any) => void,
  onError?: (error: Event) => void,
  onOpen?: () => void,
  onClose?: () => void
): WebSocket {
  const wsUrl = BACKEND_URL.replace('http', 'ws') + '/ws/analytics';
  const socket = new WebSocket(wsUrl);

  socket.onopen = () => {
    console.log('[Analytics WebSocket] Connected');
    if (onOpen) onOpen();
  };

  socket.onmessage = (event) => {
    try {
      const data = JSON.parse(event.data);
      onMessage(data);
    } catch (error) {
      console.error('[Analytics WebSocket] Failed to parse message:', error);
    }
  };

  socket.onerror = (error) => {
    console.error('[Analytics WebSocket] Error:', error);
    if (onError) onError(error);
  };

  socket.onclose = () => {
    console.log('[Analytics WebSocket] Disconnected');
    if (onClose) onClose();
  };

  return socket;
}

/**
 * Export analytics data as CSV
 */
export async function exportAnalyticsCSV(
  type: 'visitors' | 'themes' | 'letters',
  period: 'day' | 'week' | 'month' = 'week'
): Promise<Blob> {
  const response = await fetch(
    `${BACKEND_URL}/api/v1/analytics/export/${type}?period=${period}&format=csv`,
    {
      credentials: 'include',
    }
  );

  if (!response.ok) {
    throw new Error(`Failed to export analytics: ${response.statusText}`);
  }

  return response.blob();
}
