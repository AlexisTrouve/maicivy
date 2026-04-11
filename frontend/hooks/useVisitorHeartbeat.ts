'use client';

import { useEffect, useRef, useCallback } from 'react';
import { visitorsApi } from '@/lib/api';
import { usePathname } from 'next/navigation';

interface UseVisitorHeartbeatOptions {
  /**
   * Intervalle en millisecondes pour envoyer des heartbeats
   * @default 30000 (30 secondes)
   */
  interval?: number;

  /**
   * Activer/désactiver le heartbeat
   * @default true
   */
  enabled?: boolean;

  /**
   * Inclure l'URL de la page actuelle dans le heartbeat
   * @default true
   */
  trackPageUrl?: boolean;

  /**
   * Données additionnelles à envoyer avec chaque heartbeat
   */
  eventData?: Record<string, any>;

  /**
   * Callback appelé après chaque heartbeat réussi
   */
  onHeartbeat?: (activeVisitors?: number) => void;

  /**
   * Callback appelé en cas d'erreur
   */
  onError?: (error: Error) => void;
}

/**
 * Hook pour envoyer automatiquement des heartbeats au serveur
 * pour indiquer que le visiteur est actif.
 *
 * Basé sur les best practices 2026:
 * - Heartbeat toutes les 30 secondes (configurable)
 * - Détection automatique de l'URL de la page
 * - Nettoyage automatique lors du démontage
 * - Support de la visibilité de la page (arrête les heartbeats si page cachée)
 *
 * @example
 * ```tsx
 * // Usage simple
 * useVisitorHeartbeat();
 *
 * // Usage avec options
 * useVisitorHeartbeat({
 *   interval: 20000, // 20 secondes
 *   onHeartbeat: (activeVisitors) => {
 *     console.log('Active visitors:', activeVisitors);
 *   },
 * });
 * ```
 */
export function useVisitorHeartbeat(options: UseVisitorHeartbeatOptions = {}) {
  const {
    interval = 30000, // 30 secondes par défaut
    enabled = true,
    trackPageUrl = true,
    eventData,
    onHeartbeat,
    onError,
  } = options;

  const pathname = usePathname();
  const intervalRef = useRef<NodeJS.Timeout | null>(null);
  const isPageVisibleRef = useRef(true);
  const isMountedRef = useRef(true);

  /**
   * Envoie un heartbeat au serveur
   */
  const sendHeartbeat = useCallback(async () => {
    // Ne pas envoyer si désactivé ou page cachée
    if (!enabled || !isPageVisibleRef.current || !isMountedRef.current) {
      return;
    }

    try {
      const pageUrl = trackPageUrl ? pathname : undefined;
      const response = await visitorsApi.sendHeartbeat(pageUrl, eventData);

      if (onHeartbeat && response.active_visitors !== undefined) {
        onHeartbeat(response.active_visitors);
      }
    } catch (error) {
      const err = error instanceof Error ? error : new Error('Heartbeat failed');

      // Ne log que les erreurs importantes (pas les 404 de session)
      if (onError) {
        onError(err);
      } else {
        console.debug('Heartbeat error:', err.message);
      }
    }
  }, [enabled, pathname, trackPageUrl, eventData, onHeartbeat, onError]);

  /**
   * Gestion de la visibilité de la page
   * Arrête les heartbeats quand la page est cachée pour économiser les ressources
   */
  useEffect(() => {
    const handleVisibilityChange = () => {
      isPageVisibleRef.current = !document.hidden;

      // Si la page redevient visible, envoyer immédiatement un heartbeat
      if (!document.hidden) {
        sendHeartbeat();
      }
    };

    document.addEventListener('visibilitychange', handleVisibilityChange);

    return () => {
      document.removeEventListener('visibilitychange', handleVisibilityChange);
    };
  }, [sendHeartbeat]);

  /**
   * Configuration de l'intervalle de heartbeat
   */
  useEffect(() => {
    if (!enabled) {
      return;
    }

    // Envoyer un heartbeat immédiatement au montage
    sendHeartbeat();

    // Configurer l'intervalle
    intervalRef.current = setInterval(() => {
      sendHeartbeat();
    }, interval);

    return () => {
      if (intervalRef.current) {
        clearInterval(intervalRef.current);
        intervalRef.current = null;
      }
    };
  }, [enabled, interval, sendHeartbeat]);

  /**
   * Cleanup au démontage
   */
  useEffect(() => {
    isMountedRef.current = true;

    return () => {
      isMountedRef.current = false;
      if (intervalRef.current) {
        clearInterval(intervalRef.current);
        intervalRef.current = null;
      }
    };
  }, []);

  return {
    /**
     * Envoie manuellement un heartbeat
     */
    sendHeartbeat,
  };
}
