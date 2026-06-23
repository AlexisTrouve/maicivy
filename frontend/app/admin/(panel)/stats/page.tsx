'use client';

import { useEffect, useState } from 'react';
import { useTranslations } from 'next-intl';

const API = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080';

// Dashboard « Stats privées » owner-only. Charge GET /admin/stats (credentials:include → cookie admin)
// et rend 3 sections : coûts/usage IA, abus/sécurité (IPs sus), analytics détaillées (profils, referrers).
// Toutes ces données existent déjà côté backend — ce panneau les agrège pour l'owner.

interface StatsData {
  ai: {
    generations_today: number;
    letters_this_month: { model: string; count: number; tokens: number; cost_eur: number }[];
    letters_tokens: number;
    letters_cost_eur: number;
  };
  security: { flagged_ips: { ip: string; score: number; paths: string[] }[]; flagged_count: number };
  analytics: {
    by_profile: { profile: string; count: number }[];
    top_referrers: { referrer: string; count: number }[];
  };
}

function Card({ title, children }: { title: string; children: React.ReactNode }) {
  return (
    <section className="rounded-xl border border-slate-800 bg-slate-900 p-5">
      <h2 className="mb-4 text-sm font-semibold uppercase tracking-wide text-slate-400">{title}</h2>
      {children}
    </section>
  );
}

export default function AdminStats() {
  const t = useTranslations('admin.stats');
  const [data, setData] = useState<StatsData | null>(null);
  const [error, setError] = useState(false);

  useEffect(() => {
    fetch(`${API}/api/v1/admin/stats`, { credentials: 'include' })
      .then((r) => (r.ok ? r.json() : Promise.reject()))
      .then(setData)
      .catch(() => setError(true));
  }, []);

  if (error) return <p data-testid="stats-error" className="text-sm text-red-400">{t('error')}</p>;
  if (!data) return <p className="text-sm text-slate-500">{t('loading')}</p>;

  const eur = (n: number) => `${n.toFixed(2)} €`;

  return (
    <div className="flex flex-col gap-6" data-testid="admin-stats">
      <h1 className="text-2xl font-bold">{t('title')}</h1>

      {/* Section 1 — Coûts / usage IA */}
      <Card title={t('aiTitle')}>
        <div className="mb-4 flex gap-8">
          <div>
            <div className="text-2xl font-bold">{data.ai.generations_today}</div>
            <div className="text-xs text-slate-500">{t('genToday')}</div>
          </div>
          <div>
            <div className="text-2xl font-bold">{eur(data.ai.letters_cost_eur)}</div>
            <div className="text-xs text-slate-500">{t('lettersMonth', { tokens: data.ai.letters_tokens.toLocaleString('fr') })}</div>
          </div>
        </div>
        {data.ai.letters_this_month.length > 0 ? (
          <table className="w-full text-sm" data-testid="stats-ai-table">
            <thead className="text-left text-xs text-slate-500">
              <tr><th className="pb-1">{t('model')}</th><th>{t('lettersCount')}</th><th>{t('tokens')}</th><th className="text-right">{t('cost')}</th></tr>
            </thead>
            <tbody>
              {data.ai.letters_this_month.map((r) => (
                <tr key={r.model} className="border-t border-slate-800">
                  <td className="py-1 font-mono text-xs">{r.model}</td>
                  <td>{r.count}</td>
                  <td>{r.tokens.toLocaleString('fr')}</td>
                  <td className="text-right">{eur(r.cost_eur)}</td>
                </tr>
              ))}
            </tbody>
          </table>
        ) : (
          <p className="text-sm text-slate-500">{t('noLetters')}</p>
        )}
        <p className="mt-3 text-[11px] text-slate-600">{t('aiNote')}</p>
      </Card>

      {/* Section 2 — Abus / sécurité */}
      <Card title={t('securityTitle', { count: data.security.flagged_count })}>
        {data.security.flagged_ips.length > 0 ? (
          <ul className="space-y-2 text-sm" data-testid="stats-sus-list">
            {data.security.flagged_ips.map((s) => (
              <li key={s.ip} className="flex items-start justify-between gap-3 border-t border-slate-800 pt-2">
                <div className="min-w-0">
                  <span className="font-mono text-xs text-slate-300">{s.ip}</span>
                  {s.paths.length > 0 && (
                    <div className="truncate text-[11px] text-slate-500">{s.paths.join(' · ')}</div>
                  )}
                </div>
                <span className="shrink-0 rounded-full bg-red-900/40 px-2 py-0.5 text-xs font-medium text-red-300" title="score">
                  {s.score.toFixed(1)}
                </span>
              </li>
            ))}
          </ul>
        ) : (
          <p className="text-sm text-slate-500">{t('noSus')}</p>
        )}
      </Card>

      {/* Section 3 — Analytics détaillées */}
      <Card title={t('analyticsTitle')}>
        <div className="grid gap-6 sm:grid-cols-2">
          <div>
            <h3 className="mb-2 text-xs font-medium text-slate-400">{t('byProfile')}</h3>
            {data.analytics.by_profile.length > 0 ? (
              <ul className="space-y-1 text-sm" data-testid="stats-profiles">
                {data.analytics.by_profile.map((p) => (
                  <li key={p.profile} className="flex justify-between">
                    <span className="text-slate-300">{p.profile}</span>
                    <span className="text-slate-500">{p.count}</span>
                  </li>
                ))}
              </ul>
            ) : (
              <p className="text-sm text-slate-500">—</p>
            )}
          </div>
          <div>
            <h3 className="mb-2 text-xs font-medium text-slate-400">{t('topReferrers')}</h3>
            {data.analytics.top_referrers.length > 0 ? (
              <ul className="space-y-1 text-sm" data-testid="stats-referrers">
                {data.analytics.top_referrers.map((r) => (
                  <li key={r.referrer} className="flex justify-between gap-2">
                    <span className="truncate text-slate-300">{r.referrer}</span>
                    <span className="shrink-0 text-slate-500">{r.count}</span>
                  </li>
                ))}
              </ul>
            ) : (
              <p className="text-sm text-slate-500">{t('noReferrers')}</p>
            )}
          </div>
        </div>
      </Card>
    </div>
  );
}
