'use client';

import { useState, useEffect } from 'react';
import {
  LineChart, Line, XAxis, YAxis, Tooltip, ResponsiveContainer,
  CartesianGrid, Area, AreaChart,
} from 'recharts';
import { GitDayStat, GitStatsResponse } from '@/lib/types';
import { motion } from 'framer-motion';

const API_URL = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080';

// Agrège les stats par semaine pour un graphique lisible sur 6 mois
function aggregateWeekly(daily: GitDayStat[]) {
  const weeks: { week: string; commits: number; additions: number; deletions: number }[] = [];
  let current = { week: '', commits: 0, additions: 0, deletions: 0 };

  for (const day of daily) {
    const d = new Date(day.date);
    // Lundi de la semaine
    const monday = new Date(d);
    monday.setDate(d.getDate() - ((d.getDay() + 6) % 7));
    const weekLabel = monday.toLocaleDateString('fr-FR', { day: '2-digit', month: 'short' });

    if (weekLabel !== current.week) {
      if (current.week) weeks.push({ ...current });
      current = { week: weekLabel, commits: 0, additions: 0, deletions: 0 };
    }
    current.commits += day.commits;
    current.additions += day.additions;
    current.deletions += day.deletions;
  }
  if (current.week) weeks.push(current);

  return weeks;
}

// Stat card en haut
function StatCard({ label, value, color }: { label: string; value: string | number; color: string }) {
  return (
    <motion.div
      initial={{ opacity: 0, y: 20 }}
      animate={{ opacity: 1, y: 0 }}
      className="bg-white dark:bg-gray-800 rounded-xl p-5 shadow-sm border border-gray-200 dark:border-gray-700"
    >
      <div className={`text-3xl font-bold ${color}`}>{value.toLocaleString()}</div>
      <div className="text-sm text-gray-500 dark:text-gray-400 mt-1">{label}</div>
    </motion.div>
  );
}

export default function GitStatsPanel() {
  const [stats, setStats] = useState<GitStatsResponse | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    fetch(`${API_URL}/api/v1/cv/gitstats`)
      .then(res => {
        if (!res.ok) throw new Error('Failed to fetch git stats');
        return res.json();
      })
      .then(setStats)
      .catch(err => setError(err.message))
      .finally(() => setLoading(false));
  }, []);

  if (loading) {
    return (
      <div className="space-y-4">
        <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
          {[...Array(4)].map((_, i) => (
            <div key={i} className="h-24 bg-gray-200 dark:bg-gray-700 rounded-xl animate-pulse" />
          ))}
        </div>
        <div className="h-64 bg-gray-200 dark:bg-gray-700 rounded-xl animate-pulse" />
      </div>
    );
  }

  if (error || !stats) {
    return (
      <div className="text-center text-gray-500 py-12">
        <p>Stats Git indisponibles</p>
        {error && <p className="text-sm mt-2 text-red-400">{error}</p>}
      </div>
    );
  }

  const weekly = aggregateWeekly(stats.daily);

  // Tooltip custom
  const ChartTooltip = ({ active, payload, label }: any) => {
    if (!active || !payload?.length) return null;
    return (
      <div className="bg-gray-900 text-white px-3 py-2 rounded-lg text-sm shadow-lg">
        <div className="font-medium mb-1">{label}</div>
        {payload.map((p: any) => (
          <div key={p.dataKey} className="flex items-center gap-2">
            <span className="w-2 h-2 rounded-full" style={{ backgroundColor: p.color }} />
            <span>{p.name}: {p.value.toLocaleString()}</span>
          </div>
        ))}
      </div>
    );
  };

  return (
    <div className="space-y-8">
      {/* KPI cards */}
      <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
        <StatCard label="Commits (6 mois)" value={stats.totalCommits} color="text-blue-600" />
        <StatCard label="Lignes ajoutées" value={stats.totalAdded} color="text-green-600" />
        <StatCard label="Lignes supprimées" value={stats.totalDeleted} color="text-red-500" />
        <StatCard label="Repos actifs" value={stats.activeRepos} color="text-purple-600" />
      </div>

      {/* Commits par semaine — line chart */}
      <motion.div
        initial={{ opacity: 0 }}
        animate={{ opacity: 1 }}
        transition={{ delay: 0.2 }}
        className="bg-white dark:bg-gray-800 rounded-xl p-6 shadow-sm border border-gray-200 dark:border-gray-700"
      >
        <h3 className="text-lg font-semibold mb-4">Commits par semaine</h3>
        <ResponsiveContainer width="100%" height={280}>
          <AreaChart data={weekly}>
            <defs>
              <linearGradient id="commitGrad" x1="0" y1="0" x2="0" y2="1">
                <stop offset="0%" stopColor="#3b82f6" stopOpacity={0.3} />
                <stop offset="100%" stopColor="#3b82f6" stopOpacity={0} />
              </linearGradient>
            </defs>
            <CartesianGrid strokeDasharray="3 3" stroke="#374151" opacity={0.3} />
            <XAxis dataKey="week" tick={{ fontSize: 12 }} stroke="#9ca3af" />
            <YAxis tick={{ fontSize: 12 }} stroke="#9ca3af" />
            <Tooltip content={<ChartTooltip />} />
            <Area
              type="monotone"
              dataKey="commits"
              name="Commits"
              stroke="#3b82f6"
              strokeWidth={2.5}
              fill="url(#commitGrad)"
              dot={{ r: 3, fill: '#3b82f6' }}
              activeDot={{ r: 5 }}
            />
          </AreaChart>
        </ResponsiveContainer>
      </motion.div>

      {/* Lignes ajoutées/supprimées par semaine */}
      <motion.div
        initial={{ opacity: 0 }}
        animate={{ opacity: 1 }}
        transition={{ delay: 0.3 }}
        className="bg-white dark:bg-gray-800 rounded-xl p-6 shadow-sm border border-gray-200 dark:border-gray-700"
      >
        <h3 className="text-lg font-semibold mb-4">Lignes de code par semaine</h3>
        <ResponsiveContainer width="100%" height={280}>
          <LineChart data={weekly}>
            <CartesianGrid strokeDasharray="3 3" stroke="#374151" opacity={0.3} />
            <XAxis dataKey="week" tick={{ fontSize: 12 }} stroke="#9ca3af" />
            <YAxis tick={{ fontSize: 12 }} stroke="#9ca3af" />
            <Tooltip content={<ChartTooltip />} />
            <Line
              type="monotone"
              dataKey="additions"
              name="Ajoutées"
              stroke="#22c55e"
              strokeWidth={2.5}
              dot={{ r: 3, fill: '#22c55e' }}
              activeDot={{ r: 5 }}
            />
            <Line
              type="monotone"
              dataKey="deletions"
              name="Supprimées"
              stroke="#ef4444"
              strokeWidth={2.5}
              dot={{ r: 3, fill: '#ef4444' }}
              activeDot={{ r: 5 }}
            />
          </LineChart>
        </ResponsiveContainer>
      </motion.div>

      {/* Repos actifs — liste compacte */}
      <motion.div
        initial={{ opacity: 0 }}
        animate={{ opacity: 1 }}
        transition={{ delay: 0.4 }}
        className="bg-white dark:bg-gray-800 rounded-xl p-6 shadow-sm border border-gray-200 dark:border-gray-700"
      >
        <h3 className="text-lg font-semibold mb-4">
          Repositories ({stats.repos.length})
        </h3>
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-3">
          {stats.repos.map((repo) => (
            <div
              key={repo.name}
              className="flex items-center gap-3 p-3 rounded-lg bg-gray-50 dark:bg-gray-700/50 border border-gray-100 dark:border-gray-600"
            >
              <div className="flex-1 min-w-0">
                <div className="font-medium text-sm truncate">{repo.name}</div>
                {repo.language && (
                  <span className="text-xs px-2 py-0.5 rounded-full bg-blue-100 dark:bg-blue-900/30 text-blue-700 dark:text-blue-300">
                    {repo.language}
                  </span>
                )}
              </div>
              {repo.stars > 0 && (
                <span className="text-xs text-yellow-500">★ {repo.stars}</span>
              )}
            </div>
          ))}
        </div>
      </motion.div>
    </div>
  );
}
