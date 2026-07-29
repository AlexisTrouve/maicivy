import { useEffect, useState, useCallback } from 'react';

interface UseAnalyticsDataOptions {
  endpoint: string;
  refreshInterval?: number; // in milliseconds
  enabled?: boolean;
}

interface UseAnalyticsDataReturn<T> {
  data: T | null;
  isLoading: boolean;
  error: Error | null;
  refetch: () => Promise<void>;
}

export function useAnalyticsData<T>(
  options: UseAnalyticsDataOptions
): UseAnalyticsDataReturn<T> {
  const { endpoint, refreshInterval = 30000, enabled = true } = options;

  const [data, setData] = useState<T | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<Error | null>(null);

  const fetchData = useCallback(async () => {
    if (!enabled) return;

    try {
      const backendUrl = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080';
      const response = await fetch(`${backendUrl}${endpoint}`, {
        credentials: 'include',
      });

      if (!response.ok) {
        throw new Error(`Failed to fetch data from ${endpoint}: ${response.statusText}`);
      }

      const result = await response.json();
      setData(result);
      setError(null);
    } catch (err) {
      console.error(`[useAnalyticsData] Error fetching ${endpoint}:`, err);
      setError(err as Error);
    } finally {
      setIsLoading(false);
    }
  }, [endpoint, enabled]);

  useEffect(() => {
    if (!enabled) {
      setIsLoading(false);
      return;
    }

    // Initial fetch
    fetchData();

    // Set up polling if refreshInterval is specified
    if (refreshInterval > 0) {
      const interval = setInterval(fetchData, refreshInterval);
      return () => clearInterval(interval);
    }
  }, [fetchData, refreshInterval, enabled]);

  return {
    data,
    isLoading,
    error,
    refetch: fetchData,
  };
}
