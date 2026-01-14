'use client';

import React, { useState, useEffect } from 'react';

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

interface RepoListProps {
  username: string;
  showPrivate?: boolean;
}

export function RepoList({ username, showPrivate = false }: RepoListProps) {
  const [repos, setRepos] = useState<GitHubRepo[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    fetchRepos();
  }, [username, showPrivate]);

  const fetchRepos = async () => {
    try {
      setLoading(true);
      setError(null);

      const url = new URL(`${process.env.NEXT_PUBLIC_API_URL}/api/v1/github/repos`);
      url.searchParams.set('username', username);
      url.searchParams.set('include_private', showPrivate.toString());

      const response = await fetch(url.toString());
      const data = await response.json();

      if (!response.ok) {
        throw new Error(data.message || 'Failed to fetch repos');
      }

      setRepos(data.repositories || []);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Unknown error');
    } finally {
      setLoading(false);
    }
  };

  const formatDate = (dateString: string) => {
    const date = new Date(dateString);
    return date.toLocaleDateString('fr-FR', {
      year: 'numeric',
      month: 'short',
      day: 'numeric',
    });
  };

  const getLanguageColor = (language: string) => {
    const colors: Record<string, string> = {
      JavaScript: 'bg-yellow-100 text-yellow-800',
      TypeScript: 'bg-blue-100 text-blue-800',
      Python: 'bg-green-100 text-green-800',
      Go: 'bg-cyan-100 text-cyan-800',
      Rust: 'bg-orange-100 text-orange-800',
      Java: 'bg-red-100 text-red-800',
      C: 'bg-gray-100 text-gray-800',
      'C++': 'bg-purple-100 text-purple-800',
      Ruby: 'bg-red-100 text-red-800',
      PHP: 'bg-indigo-100 text-indigo-800',
    };
    return colors[language] || 'bg-gray-100 text-gray-800';
  };

  if (loading) {
    return (
      <div className="space-y-4">
        {[1, 2, 3].map((i) => (
          <div key={i} className="bg-gray-50 rounded-lg p-6 animate-pulse">
            <div className="h-5 bg-gray-200 rounded w-1/3 mb-3"></div>
            <div className="h-4 bg-gray-200 rounded w-2/3 mb-2"></div>
            <div className="h-4 bg-gray-200 rounded w-1/2"></div>
          </div>
        ))}
      </div>
    );
  }

  if (error) {
    return (
      <div className="bg-red-50 border border-red-200 rounded-lg p-4">
        <p className="text-sm text-red-600">Erreur: {error}</p>
      </div>
    );
  }

  if (repos.length === 0) {
    return (
      <div className="bg-gray-50 rounded-lg p-8 text-center">
        <p className="text-gray-600">Aucun repo trouvé</p>
      </div>
    );
  }

  return (
    <div className="space-y-4">
      {repos.map((repo) => (
        <div
          key={repo.id}
          className="bg-white border border-gray-200 rounded-lg p-6 hover:shadow-md transition-shadow"
        >
          {/* Header */}
          <div className="flex items-start justify-between mb-3">
            <div className="flex-1">
              <h3 className="text-lg font-semibold text-gray-900 mb-1">
                <a
                  href={repo.url}
                  target="_blank"
                  rel="noopener noreferrer"
                  className="hover:text-blue-600 transition-colors"
                >
                  {repo.repo_name}
                </a>
              </h3>
              <p className="text-sm text-gray-500">{repo.full_name}</p>
            </div>

            {/* Badge GitHub */}
            <span className="inline-flex items-center gap-1 px-2 py-1 bg-gray-900 text-white text-xs rounded">
              <svg className="w-3 h-3" fill="currentColor" viewBox="0 0 24 24">
                <path
                  fillRule="evenodd"
                  d="M12 2C6.477 2 2 6.477 2 12c0 4.42 2.865 8.17 6.839 9.49.5.092.682-.217.682-.482 0-.237-.008-.866-.013-1.7-2.782.603-3.369-1.34-3.369-1.34-.454-1.156-1.11-1.463-1.11-1.463-.908-.62.069-.608.069-.608 1.003.07 1.531 1.03 1.531 1.03.892 1.529 2.341 1.087 2.91.832.092-.647.35-1.088.636-1.338-2.22-.253-4.555-1.11-4.555-4.943 0-1.091.39-1.984 1.029-2.683-.103-.253-.446-1.27.098-2.647 0 0 .84-.269 2.75 1.025A9.578 9.578 0 0112 6.836c.85.004 1.705.114 2.504.336 1.909-1.294 2.747-1.025 2.747-1.025.546 1.377.203 2.394.1 2.647.64.699 1.028 1.592 1.028 2.683 0 3.842-2.339 4.687-4.566 4.935.359.309.678.919.678 1.852 0 1.336-.012 2.415-.012 2.743 0 .267.18.578.688.48C19.138 20.167 22 16.418 22 12c0-5.523-4.477-10-10-10z"
                  clipRule="evenodd"
                />
              </svg>
              GitHub
            </span>
          </div>

          {/* Description */}
          {repo.description && (
            <p className="text-sm text-gray-600 mb-4">{repo.description}</p>
          )}

          {/* Stats & Meta */}
          <div className="flex items-center gap-4 mb-3 text-sm text-gray-500">
            {/* Stars */}
            <div className="flex items-center gap-1">
              <svg className="w-4 h-4 text-yellow-500" fill="currentColor" viewBox="0 0 20 20">
                <path d="M9.049 2.927c.3-.921 1.603-.921 1.902 0l1.07 3.292a1 1 0 00.95.69h3.462c.969 0 1.371 1.24.588 1.81l-2.8 2.034a1 1 0 00-.364 1.118l1.07 3.292c.3.921-.755 1.688-1.54 1.118l-2.8-2.034a1 1 0 00-1.175 0l-2.8 2.034c-.784.57-1.838-.197-1.539-1.118l1.07-3.292a1 1 0 00-.364-1.118L2.98 8.72c-.783-.57-.38-1.81.588-1.81h3.461a1 1 0 00.951-.69l1.07-3.292z" />
              </svg>
              <span>{repo.stars}</span>
            </div>

            {/* Language */}
            {repo.language && (
              <span className={`px-2 py-1 rounded text-xs font-medium ${getLanguageColor(repo.language)}`}>
                {repo.language}
              </span>
            )}

            {/* Private badge */}
            {repo.is_private && (
              <span className="px-2 py-1 bg-gray-100 text-gray-700 rounded text-xs font-medium">
                🔒 Privé
              </span>
            )}

            {/* Last push */}
            <span className="ml-auto">Mis à jour {formatDate(repo.pushed_at)}</span>
          </div>

          {/* Topics */}
          {repo.topics && repo.topics.length > 0 && (
            <div className="flex flex-wrap gap-2">
              {repo.topics.map((topic) => (
                <span
                  key={topic}
                  className="inline-block px-2 py-1 bg-blue-50 text-blue-700 text-xs rounded"
                >
                  {topic}
                </span>
              ))}
            </div>
          )}
        </div>
      ))}
    </div>
  );
}
