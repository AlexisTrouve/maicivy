'use client';

import React, { useState } from 'react';

interface GitHubConnectProps {
  onConnectSuccess?: (username: string) => void;
  onConnectError?: (error: string) => void;
}

export function GitHubConnect({ onConnectSuccess, onConnectError }: GitHubConnectProps) {
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const handleConnect = async () => {
    try {
      setLoading(true);
      setError(null);

      // Récupérer l'URL d'authentification GitHub
      const response = await fetch(`${process.env.NEXT_PUBLIC_API_URL}/api/v1/github/auth-url`);
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

      // Écouter le callback (via postMessage ou polling)
      const checkPopup = setInterval(() => {
        if (!popup || popup.closed) {
          clearInterval(checkPopup);
          setLoading(false);

          // Vérifier si connexion réussie (via localStorage ou autre méthode)
          const connected = localStorage.getItem('github_connected');
          if (connected) {
            const username = localStorage.getItem('github_username');
            localStorage.removeItem('github_connected');
            if (username && onConnectSuccess) {
              onConnectSuccess(username);
            }
          }
        }
      }, 500);

    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : 'Unknown error';
      setError(errorMessage);
      if (onConnectError) {
        onConnectError(errorMessage);
      }
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="github-connect">
      <button
        onClick={handleConnect}
        disabled={loading}
        className={`
          flex items-center gap-2 px-6 py-3 rounded-lg font-medium
          transition-all duration-200
          ${loading
            ? 'bg-gray-300 text-gray-500 cursor-not-allowed'
            : 'bg-gray-900 text-white hover:bg-gray-800 active:scale-95'
          }
        `}
      >
        {loading ? (
          <>
            <svg className="animate-spin h-5 w-5" viewBox="0 0 24 24">
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
            <span>Connexion...</span>
          </>
        ) : (
          <>
            <svg className="h-5 w-5" fill="currentColor" viewBox="0 0 24 24">
              <path
                fillRule="evenodd"
                d="M12 2C6.477 2 2 6.477 2 12c0 4.42 2.865 8.17 6.839 9.49.5.092.682-.217.682-.482 0-.237-.008-.866-.013-1.7-2.782.603-3.369-1.34-3.369-1.34-.454-1.156-1.11-1.463-1.11-1.463-.908-.62.069-.608.069-.608 1.003.07 1.531 1.03 1.531 1.03.892 1.529 2.341 1.087 2.91.832.092-.647.35-1.088.636-1.338-2.22-.253-4.555-1.11-4.555-4.943 0-1.091.39-1.984 1.029-2.683-.103-.253-.446-1.27.098-2.647 0 0 .84-.269 2.75 1.025A9.578 9.578 0 0112 6.836c.85.004 1.705.114 2.504.336 1.909-1.294 2.747-1.025 2.747-1.025.546 1.377.203 2.394.1 2.647.64.699 1.028 1.592 1.028 2.683 0 3.842-2.339 4.687-4.566 4.935.359.309.678.919.678 1.852 0 1.336-.012 2.415-.012 2.743 0 .267.18.578.688.48C19.138 20.167 22 16.418 22 12c0-5.523-4.477-10-10-10z"
                clipRule="evenodd"
              />
            </svg>
            <span>Connecter GitHub</span>
          </>
        )}
      </button>

      {error && (
        <div className="mt-2 text-sm text-red-600">
          Erreur: {error}
        </div>
      )}
    </div>
  );
}
