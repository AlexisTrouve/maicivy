'use client';

import React, { useState, useEffect } from 'react';
import { ActivityProject, ActivityStats } from '@/lib/types';
import { activityApi } from '@/lib/api';

interface RecentActivityProps {
  showcaseOnly?: boolean;
  maxProjects?: number;
  showStats?: boolean;
}

export function RecentActivity({
  showcaseOnly = true,
  maxProjects = 5,
  showStats = true,
}: RecentActivityProps) {
  const [projects, setProjects] = useState<ActivityProject[]>([]);
  const [stats, setStats] = useState<ActivityStats | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    fetchActivity();
  }, [showcaseOnly]);

  const fetchActivity = async () => {
    try {
      setLoading(true);
      setError(null);

      const feed = await activityApi.getFeed(showcaseOnly);
      setProjects(feed.projects.slice(0, maxProjects));
      setStats(feed.stats);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load activity');
    } finally {
      setLoading(false);
    }
  };

  const formatDate = (dateString: string) => {
    const date = new Date(dateString);
    const now = new Date();
    const diffMs = now.getTime() - date.getTime();
    const diffDays = Math.floor(diffMs / (1000 * 60 * 60 * 24));

    if (diffDays === 0) return "Aujourd'hui";
    if (diffDays === 1) return 'Hier';
    if (diffDays < 7) return `Il y a ${diffDays} jours`;
    if (diffDays < 30) return `Il y a ${Math.floor(diffDays / 7)} semaines`;
    return date.toLocaleDateString('fr-FR', {
      year: 'numeric',
      month: 'short',
      day: 'numeric',
    });
  };

  const getLanguageColor = (language: string) => {
    const colors: Record<string, string> = {
      JavaScript: 'bg-yellow-100 text-yellow-800 dark:bg-yellow-900 dark:text-yellow-200',
      TypeScript: 'bg-blue-100 text-blue-800 dark:bg-blue-900 dark:text-blue-200',
      Python: 'bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-200',
      Go: 'bg-cyan-100 text-cyan-800 dark:bg-cyan-900 dark:text-cyan-200',
      Rust: 'bg-orange-100 text-orange-800 dark:bg-orange-900 dark:text-orange-200',
      Java: 'bg-red-100 text-red-800 dark:bg-red-900 dark:text-red-200',
      Shell: 'bg-gray-100 text-gray-800 dark:bg-gray-700 dark:text-gray-200',
      Dockerfile: 'bg-indigo-100 text-indigo-800 dark:bg-indigo-900 dark:text-indigo-200',
    };
    return colors[language] || 'bg-gray-100 text-gray-800 dark:bg-gray-700 dark:text-gray-200';
  };

  const getCategoryBadge = (category: string) => {
    const badges: Record<string, string> = {
      WIP: 'bg-amber-100 text-amber-800 dark:bg-amber-900 dark:text-amber-200',
      Production: 'bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-200',
      Archive: 'bg-gray-100 text-gray-600 dark:bg-gray-700 dark:text-gray-400',
    };
    return badges[category] || 'bg-gray-100 text-gray-800 dark:bg-gray-700 dark:text-gray-200';
  };

  if (loading) {
    return (
      <div className="space-y-4">
        {showStats && (
          <div className="grid grid-cols-3 gap-4 mb-6">
            {[1, 2, 3].map((i) => (
              <div key={i} className="bg-gray-50 dark:bg-gray-800 rounded-lg p-4 animate-pulse">
                <div className="h-8 bg-gray-200 dark:bg-gray-700 rounded w-1/2 mb-2"></div>
                <div className="h-4 bg-gray-200 dark:bg-gray-700 rounded w-3/4"></div>
              </div>
            ))}
          </div>
        )}
        {[1, 2, 3].map((i) => (
          <div key={i} className="bg-gray-50 dark:bg-gray-800 rounded-lg p-5 animate-pulse">
            <div className="h-5 bg-gray-200 dark:bg-gray-700 rounded w-1/3 mb-3"></div>
            <div className="h-4 bg-gray-200 dark:bg-gray-700 rounded w-2/3 mb-2"></div>
            <div className="h-4 bg-gray-200 dark:bg-gray-700 rounded w-1/2"></div>
          </div>
        ))}
      </div>
    );
  }

  if (error) {
    return (
      <div className="bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 rounded-lg p-4">
        <p className="text-sm text-red-600 dark:text-red-400">Erreur: {error}</p>
        <button
          onClick={fetchActivity}
          className="mt-2 text-sm text-red-700 dark:text-red-300 underline hover:no-underline"
        >
          Réessayer
        </button>
      </div>
    );
  }

  if (projects.length === 0) {
    return (
      <div className="bg-gray-50 dark:bg-gray-800 rounded-lg p-8 text-center">
        <p className="text-gray-600 dark:text-gray-400">Aucune activité récente</p>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      {/* Stats Overview */}
      {showStats && stats && (
        <div className="grid grid-cols-1 md:grid-cols-3 gap-4 mb-6">
          <div className="bg-gradient-to-br from-blue-50 to-indigo-50 dark:from-blue-900/20 dark:to-indigo-900/20 rounded-xl p-4 border border-blue-100 dark:border-blue-800">
            <div className="text-3xl font-bold text-blue-600 dark:text-blue-400">
              {stats.total_commits_30d}
            </div>
            <div className="text-sm text-blue-700 dark:text-blue-300">commits (30j)</div>
          </div>
          <div className="bg-gradient-to-br from-green-50 to-emerald-50 dark:from-green-900/20 dark:to-emerald-900/20 rounded-xl p-4 border border-green-100 dark:border-green-800">
            <div className="text-3xl font-bold text-green-600 dark:text-green-400">
              {stats.active_projects}
            </div>
            <div className="text-sm text-green-700 dark:text-green-300">projets actifs</div>
          </div>
          <div className="bg-gradient-to-br from-purple-50 to-violet-50 dark:from-purple-900/20 dark:to-violet-900/20 rounded-xl p-4 border border-purple-100 dark:border-purple-800">
            <div className="flex flex-wrap gap-1">
              {stats.top_languages.slice(0, 3).map((lang) => (
                <span
                  key={lang}
                  className={`text-xs px-2 py-0.5 rounded ${getLanguageColor(lang)}`}
                >
                  {lang}
                </span>
              ))}
            </div>
            <div className="text-sm text-purple-700 dark:text-purple-300 mt-1">top langages</div>
          </div>
        </div>
      )}

      {/* Projects List */}
      <div className="space-y-4">
        {projects.map((project) => (
          <div
            key={project.id}
            className="bg-white dark:bg-gray-800 border border-gray-200 dark:border-gray-700 rounded-xl p-5 hover:shadow-lg transition-all duration-200 hover:border-blue-300 dark:hover:border-blue-600"
          >
            {/* Header */}
            <div className="flex items-start justify-between mb-3">
              <div className="flex-1">
                <div className="flex items-center gap-2 mb-1">
                  <h3 className="text-lg font-semibold text-gray-900 dark:text-white">
                    <a
                      href={project.repo_url}
                      target="_blank"
                      rel="noopener noreferrer"
                      className="hover:text-blue-600 dark:hover:text-blue-400 transition-colors"
                    >
                      {project.name}
                    </a>
                  </h3>
                  <span className={`text-xs px-2 py-0.5 rounded ${getCategoryBadge(project.category)}`}>
                    {project.category}
                  </span>
                </div>
                {project.description && (
                  <p className="text-sm text-gray-600 dark:text-gray-400">{project.description}</p>
                )}
              </div>

              {/* Activity badge */}
              <div className="flex flex-col items-end gap-1">
                <span className="inline-flex items-center gap-1 px-2 py-1 bg-blue-100 dark:bg-blue-900/50 text-blue-700 dark:text-blue-300 text-xs rounded-full">
                  <svg className="w-3 h-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M13 7h8m0 0v8m0-8l-8 8-4-4-6 6" />
                  </svg>
                  {project.commits_7d} cette semaine
                </span>
              </div>
            </div>

            {/* Languages */}
            <div className="flex flex-wrap gap-2 mb-3">
              {project.languages.map((lang) => (
                <span
                  key={lang}
                  className={`text-xs px-2 py-0.5 rounded ${getLanguageColor(lang)}`}
                >
                  {lang}
                </span>
              ))}
            </div>

            {/* Recent commits */}
            {project.recent_commits.length > 0 && (
              <div className="mt-3 pt-3 border-t border-gray-100 dark:border-gray-700">
                <div className="text-xs text-gray-500 dark:text-gray-400 mb-2">
                  Derniers commits :
                </div>
                <div className="space-y-1">
                  {project.recent_commits.slice(0, 3).map((commit) => (
                    <div key={commit.sha} className="flex items-start gap-2 text-sm">
                      <code className="text-xs text-gray-400 dark:text-gray-500 font-mono">
                        {commit.sha.slice(0, 7)}
                      </code>
                      <span className="text-gray-700 dark:text-gray-300 truncate flex-1">
                        {commit.message.split('\n')[0]}
                      </span>
                      <span className="text-xs text-gray-400 dark:text-gray-500 whitespace-nowrap">
                        {formatDate(commit.date)}
                      </span>
                    </div>
                  ))}
                </div>
              </div>
            )}

            {/* Footer */}
            <div className="mt-3 pt-3 border-t border-gray-100 dark:border-gray-700 flex items-center justify-between text-xs text-gray-500 dark:text-gray-400">
              <span>
                {project.commits_30d} commits sur 30 jours
              </span>
              <span>
                Dernière activité : {formatDate(project.last_activity)}
              </span>
            </div>
          </div>
        ))}
      </div>

      {/* Last updated */}
      {stats && (
        <div className="text-center text-xs text-gray-400 dark:text-gray-500">
          Mis à jour : {formatDate(stats.last_updated)}
        </div>
      )}
    </div>
  );
}
