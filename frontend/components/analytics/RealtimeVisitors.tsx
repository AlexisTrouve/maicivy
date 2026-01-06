'use client';

import { useEffect, useState } from 'react';
import { Activity } from 'lucide-react';
import { useTranslations } from 'next-intl';

// Backend WebSocket message format
interface WSMessage {
  type: 'heartbeat' | 'initial_stats' | 'pong';
  data?: {
    current_visitors?: number;
    unique_today?: number;
    total_events?: number;
    letters_today?: number;
    timestamp?: number;
  };
  time?: number;
}

export default function RealtimeVisitors() {
  const t = useTranslations('analytics.widgets.visitors');
  const [visitors, setVisitors] = useState<number>(0);
  // eslint-disable-next-line @typescript-eslint/no-unused-vars
  const [prevVisitors, setPrevVisitors] = useState<number>(0);
  const [isConnected, setIsConnected] = useState(false);
  // eslint-disable-next-line @typescript-eslint/no-unused-vars
  const [ws, setWs] = useState<WebSocket | null>(null);

  useEffect(() => {
    // WebSocket connection
    const backendUrl = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080';
    const wsUrl = backendUrl.replace('http', 'ws') + '/ws/analytics';

    let socket: WebSocket;
    let reconnectAttempts = 0;
    const maxReconnectAttempts = 5;
    let reconnectTimeout: NodeJS.Timeout;

    const connect = () => {
      try {
        socket = new WebSocket(wsUrl);

        socket.onopen = () => {
          console.log('[WS] Connected to analytics');
          setIsConnected(true);
          reconnectAttempts = 0;
        };

        socket.onmessage = (event) => {
          try {
            const message: WSMessage = JSON.parse(event.data);

            // Handle different message types from backend
            if ((message.type === 'heartbeat' || message.type === 'initial_stats') && message.data) {
              // Backend uses snake_case: current_visitors
              const currentVisitors = message.data.current_visitors || 0;
              setPrevVisitors(visitors);
              setVisitors(currentVisitors);
            }
          } catch (error) {
            console.error('[WS] Failed to parse message:', error);
          }
        };

        socket.onerror = (error) => {
          console.error('[WS] Error:', error);
        };

        socket.onclose = () => {
          console.log('[WS] Disconnected from analytics');
          setIsConnected(false);

          // Auto-reconnect with exponential backoff
          if (reconnectAttempts < maxReconnectAttempts) {
            const delay = Math.min(1000 * Math.pow(2, reconnectAttempts), 30000);
            reconnectAttempts++;
            console.log(`[WS] Reconnecting in ${delay}ms (attempt ${reconnectAttempts}/${maxReconnectAttempts})`);
            reconnectTimeout = setTimeout(connect, delay);
          }
        };

        setWs(socket);
      } catch (error) {
        console.error('[WS] Connection failed:', error);
      }
    };

    connect();

    // Cleanup on unmount
    return () => {
      if (reconnectTimeout) clearTimeout(reconnectTimeout);
      if (socket && socket.readyState === WebSocket.OPEN) {
        socket.close();
      }
    };
  }, []); // eslint-disable-line react-hooks/exhaustive-deps

  // Count up animation effect
  const [displayCount, setDisplayCount] = useState(0);

  useEffect(() => {
    if (visitors === displayCount) return;

    const duration = 1000; // 1 second
    const steps = 20;
    const stepDuration = duration / steps;
    const increment = (visitors - displayCount) / steps;

    let currentStep = 0;
    const interval = setInterval(() => {
      currentStep++;
      if (currentStep >= steps) {
        setDisplayCount(visitors);
        clearInterval(interval);
      } else {
        setDisplayCount(Math.round(displayCount + increment * currentStep));
      }
    }, stepDuration);

    return () => clearInterval(interval);
  }, [visitors, displayCount]);

  return (
    <div className="rounded-lg border bg-card p-6">
      <div className="flex items-center justify-between mb-4">
        <h2 className="text-xl font-semibold flex items-center gap-2">
          <Activity className="h-5 w-5" />
          {t('title')}
        </h2>
        <div className="flex items-center gap-2">
          <div
            className={`h-2 w-2 rounded-full ${
              isConnected ? 'bg-green-500' : 'bg-red-500'
            } animate-pulse`}
          />
          <span className="text-xs text-muted-foreground">
            {isConnected ? t('online') : t('offline')}
          </span>
        </div>
      </div>

      <div className="flex items-center justify-center py-8">
        <div className="text-center">
          <div className="text-6xl font-bold text-primary">
            {displayCount}
          </div>
          <p className="text-sm text-muted-foreground mt-2">
            {visitors === 0 || visitors === 1 ? t('person') : t('people')} {t('rightNow')}
          </p>
        </div>
      </div>

      <div className="text-xs text-muted-foreground text-center">
        {t('realtime')}
      </div>
    </div>
  );
}
