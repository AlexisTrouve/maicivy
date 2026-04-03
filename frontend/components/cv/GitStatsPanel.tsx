'use client';

import { useState, useEffect } from 'react';
import {
  XAxis, YAxis, Tooltip, ResponsiveContainer,
  CartesianGrid, Area, AreaChart,
} from 'recharts';
import { GitDayStat, GitStatsResponse } from '@/lib/types';
import { motion } from 'framer-motion';

const API_URL = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080';

// Filtre les 6 derniers mois et formate les dates pour l'affichage
function filterLast6Months(daily: GitDayStat[]) {
  const sixMonthsAgo = new Date();
  sixMonthsAgo.setMonth(sixMonthsAgo.getMonth() - 6);

  return daily
    .filter((d) => new Date(d.date) >= sixMonthsAgo)
    .map((d) => ({
      ...d,
      label: new Date(d.date).toLocaleDateString('fr-FR', { day: '2-digit', month: 'short' }),
    }));
}

// Animation variants pour les sections qui apparaissent au scroll
const sectionVariants = {
  hidden: { opacity: 0, y: 40 },
  visible: { opacity: 1, y: 0 },
};

// Stat card avec stagger animation
function StatCard({ label, value, color, index }: { label: string; value: string | number; color: string; index: number }) {
  return (
    <motion.div
      variants={sectionVariants}
      initial="hidden"
      whileInView="visible"
      viewport={{ once: true, margin: '-50px' }}
      transition={{ duration: 0.5, delay: index * 0.1, ease: 'easeOut' }}
      className="bg-white dark:bg-gray-800 rounded-xl p-5 shadow-sm border border-gray-200 dark:border-gray-700"
    >
      <div className={`text-3xl font-bold ${color}`}>{value.toLocaleString()}</div>
      <div className="text-sm text-gray-500 dark:text-gray-400 mt-1">{label}</div>
    </motion.div>
  );
}

// Wrapper pour les charts — apparaît au scroll avec slide-up
function ChartSection({ children, delay = 0 }: { children: React.ReactNode; delay?: number }) {
  return (
    <motion.div
      variants={sectionVariants}
      initial="hidden"
      whileInView="visible"
      viewport={{ once: true, margin: '-80px' }}
      transition={{ duration: 0.6, delay, ease: 'easeOut' }}
      className="bg-white dark:bg-gray-800 rounded-xl p-6 shadow-sm border border-gray-200 dark:border-gray-700"
    >
      {children}
    </motion.div>
  );
}

export default function GitStatsPanel() {
  const [stats, setStats] = useState<GitStatsResponse | null>(null);
  const [loading, setLoading] = useState(true);
  const [refreshing, setRefreshing] = useState(false);
  const [error, setError] = useState<string | null>(null);

  // Fetch stats — force=true pour ignorer le cache Redis
  const fetchStats = (force = false) => {
    const query = force ? '?force=true' : '';
    if (force) setRefreshing(true);

    fetch(`${API_URL}/api/v1/cv/gitstats${query}`)
      .then(res => {
        if (!res.ok) throw new Error('Failed to fetch git stats');
        return res.json();
      })
      .then(setStats)
      .catch(err => setError(err.message))
      .finally(() => {
        setLoading(false);
        setRefreshing(false);
      });
  };

  useEffect(() => { fetchStats(); }, []);

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

  const daily = filterLast6Months(stats.daily);

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

  const kpis = [
    { label: 'Commits (6 mois)', value: stats.totalCommits, color: 'text-blue-600' },
    { label: 'Lignes ajoutées', value: stats.totalAdded, color: 'text-green-600' },
    { label: 'Lignes supprimées', value: stats.totalDeleted, color: 'text-red-500' },
    { label: 'Repos actifs', value: stats.activeRepos, color: 'text-purple-600' },
  ];

  return (
    <div className="space-y-8">
      {/* Header avec bouton refresh */}
      <div className="flex justify-end">
        <button
          onClick={() => fetchStats(true)}
          disabled={refreshing}
          className="flex items-center gap-2 px-3 py-1.5 text-sm rounded-lg bg-gray-100 dark:bg-gray-700 hover:bg-gray-200 dark:hover:bg-gray-600 text-gray-600 dark:text-gray-300 transition-colors disabled:opacity-50"
        >
          <svg
            className={`w-4 h-4 ${refreshing ? 'animate-spin' : ''}`}
            fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}
          >
            <path strokeLinecap="round" strokeLinejoin="round" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
          </svg>
          {refreshing ? 'Refresh...' : 'Refresh'}
        </button>
      </div>

      {/* KPI cards — stagger animation */}
      <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
        {kpis.map((kpi, i) => (
          <StatCard key={kpi.label} index={i} {...kpi} />
        ))}
      </div>

      {/* Commits par jour — area chart */}
      <ChartSection>
        <h3 className="text-lg font-semibold mb-4">Commits par jour</h3>
        <ResponsiveContainer width="100%" height={280}>
          <AreaChart data={daily}>
            <defs>
              <linearGradient id="commitGrad" x1="0" y1="0" x2="0" y2="1">
                <stop offset="0%" stopColor="#3b82f6" stopOpacity={0.3} />
                <stop offset="100%" stopColor="#3b82f6" stopOpacity={0} />
              </linearGradient>
            </defs>
            <CartesianGrid strokeDasharray="3 3" stroke="#374151" opacity={0.3} />
            <XAxis dataKey="label" tick={{ fontSize: 11 }} stroke="#9ca3af" interval="preserveStartEnd" />
            <YAxis tick={{ fontSize: 12 }} stroke="#9ca3af" />
            <Tooltip content={<ChartTooltip />} />
            <Area
              type="monotone"
              dataKey="commits"
              name="Commits"
              stroke="#3b82f6"
              strokeWidth={2}
              fill="url(#commitGrad)"
              dot={false}
              activeDot={{ r: 4 }}
            />
          </AreaChart>
        </ResponsiveContainer>
      </ChartSection>

      {/* Lignes ajoutées/supprimées par jour */}
      <ChartSection delay={0.1}>
        <h3 className="text-lg font-semibold mb-4">Lignes de code par jour</h3>
        <ResponsiveContainer width="100%" height={280}>
          <AreaChart data={daily}>
            <defs>
              <linearGradient id="addGrad" x1="0" y1="0" x2="0" y2="1">
                <stop offset="0%" stopColor="#22c55e" stopOpacity={0.3} />
                <stop offset="100%" stopColor="#22c55e" stopOpacity={0} />
              </linearGradient>
              <linearGradient id="delGrad" x1="0" y1="0" x2="0" y2="1">
                <stop offset="0%" stopColor="#ef4444" stopOpacity={0.3} />
                <stop offset="100%" stopColor="#ef4444" stopOpacity={0} />
              </linearGradient>
            </defs>
            <CartesianGrid strokeDasharray="3 3" stroke="#374151" opacity={0.3} />
            <XAxis dataKey="label" tick={{ fontSize: 11 }} stroke="#9ca3af" interval="preserveStartEnd" />
            <YAxis tick={{ fontSize: 12 }} stroke="#9ca3af" />
            <Tooltip content={<ChartTooltip />} />
            <Area
              type="monotone"
              dataKey="additions"
              name="Ajoutées"
              stroke="#22c55e"
              strokeWidth={2}
              fill="url(#addGrad)"
              dot={false}
              activeDot={{ r: 4 }}
            />
            <Area
              type="monotone"
              dataKey="deletions"
              name="Supprimées"
              stroke="#ef4444"
              strokeWidth={2}
              fill="url(#delGrad)"
              dot={false}
              activeDot={{ r: 4 }}
            />
          </AreaChart>
        </ResponsiveContainer>
      </ChartSection>

      {/* Repos actifs — liste compacte */}
      <ChartSection delay={0.15}>
        <h3 className="text-lg font-semibold mb-4">
          Repositories ({stats.repos.length})
        </h3>
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-3">
          {stats.repos.map((repo, i) => (
            <motion.div
              key={repo.name}
              initial={{ opacity: 0, scale: 0.95 }}
              whileInView={{ opacity: 1, scale: 1 }}
              viewport={{ once: true }}
              transition={{ duration: 0.3, delay: i * 0.03 }}
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
            </motion.div>
          ))}
        </div>
      </ChartSection>
    </div>
  );
}
