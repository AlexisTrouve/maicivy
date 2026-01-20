'use client';

import { useVisitorHeartbeat } from '@/hooks/useVisitorHeartbeat';
import { useState, useEffect } from 'react';

/**
 * Provider global pour le heartbeat des visiteurs
 * Envoie automatiquement des heartbeats toutes les 30 secondes
 * et affiche optionnellement le nombre de visiteurs actifs
 */
export function VisitorHeartbeatProvider({
  children,
  showActiveVisitors = false,
}: {
  children: React.ReactNode;
  showActiveVisitors?: boolean;
}) {
  const [activeVisitors, setActiveVisitors] = useState<number>(0);

  // Hook de heartbeat automatique
  useVisitorHeartbeat({
    interval: 30000, // 30 secondes
    enabled: true,
    trackPageUrl: true,
    onHeartbeat: (count) => {
      if (count !== undefined) {
        setActiveVisitors(count);
      }
    },
    onError: (error) => {
      // Log silencieux des erreurs (évite de polluer la console)
      console.debug('Visitor heartbeat error:', error.message);
    },
  });

  return (
    <>
      {children}
      {/* Badge optionnel pour afficher le nombre de visiteurs actifs */}
      {showActiveVisitors && activeVisitors > 0 && (
        <div
          className="fixed bottom-4 right-4 z-50 rounded-full bg-green-500/90 px-3 py-1 text-xs text-white shadow-lg backdrop-blur-sm"
          aria-live="polite"
        >
          🟢 {activeVisitors} visiteur{activeVisitors > 1 ? 's' : ''} actif
          {activeVisitors > 1 ? 's' : ''}
        </div>
      )}
    </>
  );
}
