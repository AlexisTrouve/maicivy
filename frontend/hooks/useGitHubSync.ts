'use client';

import { useState, useEffect, useCallback } from 'react';

interface SyncStatus {
  connected: boolean;
  username?: string;
  last_sync: number;
  repo_count: number;
}

interface GitHubRepo {
  id: number;
  repo_name: string;
  full_name: string;
  description: string;
  url: string;
  stars: number;
  language: string;
  topics: string[];
  is_private: boolean;
  pushed_at: string;
}

type SyncState = 'idle' | 'connecting' | 'syncing' | 'connected' | 'error';

interface UseGitHubSyncReturn {
  // State
  state: SyncState;
  status: SyncStatus | null;
  repos: GitHubRepo[];
  error: string | null;
  loading: boolean;

  // Actions
  connect: () => Promise<void>;
  sync: () => Promise<void>;
  disconnect: () => Promise<void>;
  fetchStatus: () => Promise<void>;
  fetchRepos: () => Promise<void>;
}

export function useGitHubSync(username?: string): UseGitHubSyncReturn {
  const [state, setState] = useState<SyncState>('idle');
  const [status, setStatus] = useState<SyncStatus | null>(null);
  const [repos, setRepos] = useState<GitHubRepo[]>([]);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);

  // Fetch status au montage si username fourni
  useEffect(() => {
    if (username) {
      fetchStatus();
    }
  }, [username]);

  // Déterminer l'état basé sur le status (sauf si déjà en erreur)
  useEffect(() => {
    if (status && state !== 'error') {
      setState(status.connected ? 'connected' : 'idle');
    }
  }, [status, state]);

  const fetchStatus = useCallback(async () => {
    if (!username) return;

    try {
      setLoading(true);
      const response = await fetch(
        `${process.env.NEXT_PUBLIC_API_URL}/api/v1/github/status?username=${username}`
      );

      if (!response.ok) {
        throw new Error('Failed to fetch status');
      }

      const data = await response.json();
      setStatus(data);
      setError(null);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to fetch status');
      setState('error');
    } finally {
      setLoading(false);
    }
  }, [username]);

  const fetchRepos = useCallback(async () => {
    if (!username) return;

    try {
      setLoading(true);
      const url = new URL(`${process.env.NEXT_PUBLIC_API_URL}/api/v1/github/repos`);
      url.searchParams.set('username', username);

      const response = await fetch(url.toString());
      const data = await response.json();

      if (!response.ok) {
        throw new Error(data.message || 'Failed to fetch repos');
      }

      setRepos(data.repositories || []);
      setError(null);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to fetch repos');
    } finally {
      setLoading(false);
    }
  }, [username]);

  const connect = useCallback(async () => {
    try {
      setState('connecting');
      setError(null);

      // Récupérer l'URL d'authentification GitHub
      const response = await fetch(
        `${process.env.NEXT_PUBLIC_API_URL}/api/v1/github/auth-url`
      );
      const data = await response.json();

      if (!response.ok || !data.auth_url) {
        throw new Error('Failed to get GitHub auth URL');
      }

      // Ouvrir popup OAuth GitHub
      const width = 600;
      const height = 700;
      const left = window.screen.width / 2 - width / 2;
      const top = window.screen.height / 2 - height / 2;

      const popup = window.open(
        data.auth_url,
        'GitHub OAuth',
        `width=${width},height=${height},left=${left},top=${top}`
      );

      // Attendre fermeture popup
      const checkPopup = setInterval(() => {
        if (!popup || popup.closed) {
          clearInterval(checkPopup);

          // Vérifier si connexion réussie
          const connected = localStorage.getItem('github_connected');
          if (connected) {
            const connectedUsername = localStorage.getItem('github_username');
            localStorage.removeItem('github_connected');

            if (connectedUsername) {
              setState('connected');
              fetchStatus();
              fetchRepos();
            }
          } else {
            setState('idle');
          }
        }
      }, 500);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Connection failed');
      setState('error');
    }
  }, [fetchStatus, fetchRepos]);

  const sync = useCallback(async () => {
    if (!username) {
      setError('Username required');
      return;
    }

    try {
      setState('syncing');
      setError(null);

      const response = await fetch(
        `${process.env.NEXT_PUBLIC_API_URL}/api/v1/github/sync`,
        {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ username }),
        }
      );

      const data = await response.json();

      if (!response.ok) {
        throw new Error(data.message || 'Sync failed');
      }

      // Attendre 2 secondes pour que la sync se termine
      await new Promise((resolve) => setTimeout(resolve, 2000));

      // Rafraîchir status et repos
      await Promise.all([fetchStatus(), fetchRepos()]);

      setState('connected');
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Sync failed');
      setState('error');
    }
  }, [username, fetchStatus, fetchRepos]);

  const disconnect = useCallback(async () => {
    if (!username) return;

    try {
      setError(null);

      const response = await fetch(
        `${process.env.NEXT_PUBLIC_API_URL}/api/v1/github/disconnect?username=${username}`,
        { method: 'DELETE' }
      );

      if (!response.ok) {
        throw new Error('Disconnect failed');
      }

      // Reset state
      setStatus(null);
      setRepos([]);
      setState('idle');
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Disconnect failed');
      setState('error');
    }
  }, [username]);

  return {
    // State
    state,
    status,
    repos,
    error,
    loading,

    // Actions
    connect,
    sync,
    disconnect,
    fetchStatus,
    fetchRepos,
  };
}
