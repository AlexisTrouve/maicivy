'use client';

import { useEffect, useState } from 'react';
import { Activity } from 'lucide-react';
import { useTranslations } from 'next-intl';
import { ETAT } from '@/components/site-page/palette';

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

// NB : le « social proof » (plancher vivant de visiteurs) n'est PLUS géré ici. Il est généré côté
// BACKEND (DemoMetrics, seedé par les commits + gaté par les vrais users) et arrive déjà blendé dans
// `current_visitors` via le WebSocket. Le composant ne fait qu'afficher la valeur reçue. cf. CLAUDE.md.

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
    <div className="rounded-lg border bg-card p-5 md:p-6">
      <div className="flex items-center justify-between gap-3 mb-4">
        <h2 className="text-xl font-medium flex items-center gap-2">
          <Activity aria-hidden="true" className="h-5 w-5 text-muted-foreground" />
          {t('title')}
        </h2>
        {/* État de la connexion : glyphe ✓/✕ + libellé + pastille. La couleur (palette ETAT, seule
            exception de couleur de la page) double l'information, elle ne la porte jamais seule. */}
        <div className="flex items-center gap-2 text-xs text-muted-foreground">
          <span
            data-testid="ws-status-dot"
            data-state={isConnected ? 'online' : 'offline'}
            className={`h-2 w-2 rounded-full ${isConnected ? 'animate-pulse' : ''}`}
            style={{ backgroundColor: isConnected ? ETAT.ok : ETAT.arret }}
          />
          <span aria-hidden="true">{isConnected ? '✓' : '✕'}</span>
          <span>{isConnected ? t('online') : t('offline')}</span>
        </div>
      </div>

      <div className="flex items-center justify-center py-8">
        <div className="text-center">
          {/* Valeur reçue du WebSocket (déjà blendée côté backend), animée en count-up. */}
          <div className="font-heading text-6xl font-medium tabular-nums text-foreground">
            {displayCount}
          </div>
          {/* Une seule clé par cas (et non « personne » + « en ce moment » concaténés) : l'ordre des
              mots change selon la langue. */}
          <p className="text-sm text-muted-foreground mt-2">
            {displayCount <= 1 ? t('oneNow') : t('manyNow')}
          </p>
        </div>
      </div>

      <div className="text-xs text-muted-foreground text-center">
        {t('realtime')}
      </div>
    </div>
  );
}
