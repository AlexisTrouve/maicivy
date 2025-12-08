'use client';

import { useState, useEffect } from 'react';
import { visitorsApi } from '@/lib/api';
import type { VisitorStatus } from '@/lib/types';

export function useVisitCount() {
  const [status, setStatus] = useState<VisitorStatus | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    checkVisitStatus();
  }, []);

  const checkVisitStatus = async () => {
    try {
      setLoading(true);
      setError(null);
      const response = await visitorsApi.checkStatus();
      setStatus(response);
    } catch (err: any) {
      console.error('Error checking visit status:', err);
      setError(err.message || 'Failed to check visit status');

      // Fallback: permettre l'accès en cas d'erreur (serveur vérifiera)
      setStatus({
        visitCount: 0,
        hasAccess: true,
        remainingVisits: 0,
        sessionId: '',
      });
    } finally {
      setLoading(false);
    }
  };

  const refresh = () => {
    checkVisitStatus();
  };

  return {
    status,
    loading,
    error,
    refresh,
  };
}
