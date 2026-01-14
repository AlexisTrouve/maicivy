'use client';

import React, { useState } from 'react';

interface SyncStatus {
  connected: boolean;
  username?: string;
  last_sync: number;
  repo_count: number;
}

interface GitHubStatusProps {
  username: string;
  onSync?: () => void;
  onDisconnect?: () => void;
}

export function GitHubStatus({ username, onSync, onDisconnect }: GitHubStatusProps) {
  const [status, setStatus] = useState<SyncStatus | null>(null);
  const [loading, setLoading] = useState(false);
  const [syncing, setSyncing] = useState(false);

  React.useEffect(() => {
    fetchStatus();
  }, [username]);

  const fetchStatus = async () => {
    try {
      setLoading(true);
      const response = await fetch(
        `${process.env.NEXT_PUBLIC_API_URL}/api/v1/github/status?username=${username}`
      );
      const data = await response.json();
      setStatus(data);
    } catch (error) {
      console.error('Failed to fetch GitHub status:', error);
    } finally {
      setLoading(false);
    }
  };

  const handleSync = async () => {
    try {
      setSyncing(true);
      const response = await fetch(
        `${process.env.NEXT_PUBLIC_API_URL}/api/v1/github/sync`,
        {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ username }),
        }
      );

      if (response.ok) {
        // Attendre 2 secondes puis rafraîchir le statut
        setTimeout(() => {
          fetchStatus();
          setSyncing(false);
          if (onSync) onSync();
        }, 2000);
      } else {
        // Reset syncing state on error response
        setSyncing(false);
      }
    } catch (error) {
      console.error('Sync failed:', error);
      setSyncing(false);
    }
  };

  const handleDisconnect = async () => {
    if (!confirm('Voulez-vous vraiment déconnecter GitHub ?')) {
      return;
    }

    try {
      const response = await fetch(
        `${process.env.NEXT_PUBLIC_API_URL}/api/v1/github/disconnect?username=${username}`,
        { method: 'DELETE' }
      );

      if (response.ok) {
        if (onDisconnect) onDisconnect();
      }
    } catch (error) {
      console.error('Disconnect failed:', error);
    }
  };

  const formatLastSync = (timestamp: number) => {
    if (!timestamp) return 'Jamais';

    const now = Date.now();
    const diff = now - timestamp * 1000;
    const minutes = Math.floor(diff / 60000);
    const hours = Math.floor(diff / 3600000);
    const days = Math.floor(diff / 86400000);

    if (minutes < 1) return 'À l\'instant';
    if (minutes < 60) return `Il y a ${minutes} min`;
    if (hours < 24) return `Il y a ${hours}h`;
    return `Il y a ${days}j`;
  };

  if (loading) {
    return (
      <div className="bg-gray-50 rounded-lg p-4 animate-pulse">
        <div className="h-4 bg-gray-200 rounded w-1/2 mb-2"></div>
        <div className="h-3 bg-gray-200 rounded w-1/3"></div>
      </div>
    );
  }

  if (!status || !status.connected) {
    return (
      <div className="bg-gray-50 rounded-lg p-4">
        <p className="text-sm text-gray-600">GitHub non connecté</p>
      </div>
    );
  }

  return (
    <div className="bg-white border border-gray-200 rounded-lg p-4 shadow-sm">
      {/* Header */}
      <div className="flex items-center justify-between mb-4">
        <div className="flex items-center gap-2">
          <div className="w-2 h-2 bg-green-500 rounded-full animate-pulse"></div>
          <span className="font-medium text-gray-900">Connecté à GitHub</span>
        </div>
        <span className="text-sm text-gray-500">@{status.username}</span>
      </div>

      {/* Stats */}
      <div className="grid grid-cols-2 gap-4 mb-4">
        <div>
          <p className="text-xs text-gray-500 mb-1">Dernière synchro</p>
          <p className="text-sm font-medium text-gray-900">
            {formatLastSync(status.last_sync)}
          </p>
        </div>
        <div>
          <p className="text-xs text-gray-500 mb-1">Repos importés</p>
          <p className="text-sm font-medium text-gray-900">
            {status.repo_count}
          </p>
        </div>
      </div>

      {/* Actions */}
      <div className="flex gap-2">
        <button
          onClick={handleSync}
          disabled={syncing}
          className={`
            flex-1 px-4 py-2 rounded-md text-sm font-medium
            transition-colors
            ${syncing
              ? 'bg-gray-100 text-gray-400 cursor-not-allowed'
              : 'bg-blue-500 text-white hover:bg-blue-600'
            }
          `}
        >
          {syncing ? (
            <span className="flex items-center justify-center gap-2">
              <svg className="animate-spin h-4 w-4" viewBox="0 0 24 24">
                <circle
                  className="opacity-25"
                  cx="12"
                  cy="12"
                  r="10"
                  stroke="currentColor"
                  strokeWidth="4"
                  fill="none"
                />
                <path
                  className="opacity-75"
                  fill="currentColor"
                  d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"
                />
              </svg>
              Synchro...
            </span>
          ) : (
            '🔄 Synchroniser'
          )}
        </button>

        <button
          onClick={handleDisconnect}
          className="px-4 py-2 rounded-md text-sm font-medium bg-gray-100 text-gray-700 hover:bg-gray-200 transition-colors"
        >
          Déconnecter
        </button>
      </div>
    </div>
  );
}
