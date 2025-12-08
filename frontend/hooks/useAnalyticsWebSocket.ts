import { useEffect, useState, useCallback } from 'react';

interface RealtimeData {
  currentVisitors: number;
  timestamp: number;
  recentEvents?: any[];
}

interface UseAnalyticsWebSocketReturn {
  data: RealtimeData | null;
  isConnected: boolean;
  error: Error | null;
  reconnect: () => void;
}

export function useAnalyticsWebSocket(): UseAnalyticsWebSocketReturn {
  const [data, setData] = useState<RealtimeData | null>(null);
  const [isConnected, setIsConnected] = useState(false);
  const [error, setError] = useState<Error | null>(null);
  const [ws, setWs] = useState<WebSocket | null>(null);
  const [shouldConnect, setShouldConnect] = useState(true);

  const connect = useCallback(() => {
    const backendUrl = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080';
    const wsUrl = backendUrl.replace('http', 'ws') + '/ws/analytics';

    try {
      const socket = new WebSocket(wsUrl);

      socket.onopen = () => {
        console.log('[useAnalyticsWebSocket] Connected');
        setIsConnected(true);
        setError(null);
      };

      socket.onmessage = (event) => {
        try {
          const parsedData: RealtimeData = JSON.parse(event.data);
          setData(parsedData);
        } catch (err) {
          console.error('[useAnalyticsWebSocket] Failed to parse message:', err);
          setError(err as Error);
        }
      };

      socket.onerror = (event) => {
        console.error('[useAnalyticsWebSocket] Error:', event);
        setError(new Error('WebSocket error'));
      };

      socket.onclose = () => {
        console.log('[useAnalyticsWebSocket] Disconnected');
        setIsConnected(false);
      };

      setWs(socket);
    } catch (err) {
      console.error('[useAnalyticsWebSocket] Connection failed:', err);
      setError(err as Error);
    }
  }, []);

  const reconnect = useCallback(() => {
    if (ws) {
      ws.close();
    }
    setShouldConnect(true);
  }, [ws]);

  useEffect(() => {
    if (!shouldConnect) return;

    connect();
    setShouldConnect(false);

    return () => {
      if (ws && ws.readyState === WebSocket.OPEN) {
        ws.close();
      }
    };
  }, [shouldConnect, connect, ws]);

  return {
    data,
    isConnected,
    error,
    reconnect,
  };
}
