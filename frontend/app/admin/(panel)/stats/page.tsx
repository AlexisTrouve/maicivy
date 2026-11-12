'use client';

import { useEffect, useState } from 'react';
import { useLocale, useTranslations } from 'next-intl';
import { Panel, PageTitle, SectionLabel } from '@/components/admin/ui';

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

// Card — section du tableau de bord. S'appuie sur la primitive partagée du panneau (Panel) au lieu
// de redéfinir sa propre chaîne de classes : l'apparence d'une section se décide à un seul endroit.
function Card({ title, children }: { title: string; children: React.ReactNode }) {
  return (
    <Panel className="p-5">
      <SectionLabel className="mb-4">{title}</SectionLabel>
      {children}
    </Panel>
  );
}

// Kpi — un chiffre-clé et son libellé. Le chiffre est en or et en chasse tabulaire : c'est ce qu'on
// vient lire sur cet écran, et l'alignement des chiffres permet de comparer deux valeurs sans les
// relire caractère par caractère.
function Kpi({ value, label }: { value: string; label: string }) {
  return (
    <div>
      <div className="font-heading text-2xl font-bold tabular-nums text-gold-soft">{value}</div>
      <div className="mt-0.5 text-xs text-slate-500">{label}</div>
    </div>
  );
}

export default function AdminStats() {
  const t = useTranslations('admin.stats');
  const locale = useLocale();
  const [data, setData] = useState<StatsData | null>(null);
  const [error, setError] = useState(false);

  useEffect(() => {
    fetch(`${API}/api/v1/admin/stats`, { credentials: 'include' })
      .then((r) => (r.ok ? r.json() : Promise.reject()))
      .then(setData)
      .catch(() => setError(true));
  }, []);

  if (error) return <p data-testid="stats-error" className="text-sm text-rose-400">{t('error')}</p>;
  if (!data) return <p className="text-sm text-slate-500">{t('loading')}</p>;

  const eur = (n: number) => `${n.toFixed(2)} €`;

  return (
    <div className="mx-auto flex w-full max-w-5xl flex-col gap-6" data-testid="admin-stats">
      <PageTitle>{t('title')}</PageTitle>

      {/* Section 1 — Coûts / usage IA */}
      <Card title={t('aiTitle')}>
        <div className="mb-5 flex gap-10">
          <Kpi value={String(data.ai.generations_today)} label={t('genToday')} />
          <Kpi
            value={eur(data.ai.letters_cost_eur)}
            label={t('lettersMonth', { tokens: data.ai.letters_tokens.toLocaleString(locale) })}
          />
        </div>
        {data.ai.letters_this_month.length > 0 ? (
          <table className="w-full text-sm" data-testid="stats-ai-table">
            <thead className="text-left text-[11px] uppercase tracking-wider text-slate-600">
              <tr><th className="pb-2 font-medium">{t('model')}</th><th className="font-medium">{t('lettersCount')}</th><th className="font-medium">{t('tokens')}</th><th className="text-right font-medium">{t('cost')}</th></tr>
            </thead>
            <tbody>
              {data.ai.letters_this_month.map((r) => (
                <tr key={r.model} className="border-t border-slate-800/70">
                  <td className="py-1.5 font-mono text-xs text-slate-300">{r.model}</td>
                  <td className="tabular-nums text-slate-300">{r.count}</td>
                  <td className="tabular-nums text-slate-400">{r.tokens.toLocaleString(locale)}</td>
                  <td className="text-right tabular-nums text-slate-200">{eur(r.cost_eur)}</td>
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
              <li key={s.ip} className="flex items-start justify-between gap-3 border-t border-slate-800/70 pt-2">
                <div className="min-w-0">
                  <span className="font-mono text-xs text-slate-300">{s.ip}</span>
                  {s.paths.length > 0 && (
                    <div className="truncate text-[11px] text-slate-500">{s.paths.join(' · ')}</div>
                  )}
                </div>
                <span
                  className="shrink-0 rounded-full border border-rose-800/60 bg-rose-950/50 px-2 py-0.5 text-xs font-medium tabular-nums text-rose-300"
                  title={t('scoreTitle')}
                >
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
            <h3 className="mb-2 text-[11px] font-medium uppercase tracking-wider text-slate-600">{t('byProfile')}</h3>
            {data.analytics.by_profile.length > 0 ? (
              <ul className="space-y-1 text-sm" data-testid="stats-profiles">
                {data.analytics.by_profile.map((p) => (
                  <li key={p.profile} className="flex justify-between border-b border-slate-800/50 py-1">
                    <span className="text-slate-300">{p.profile}</span>
                    <span className="tabular-nums text-slate-500">{p.count}</span>
                  </li>
                ))}
              </ul>
            ) : (
              <p className="text-sm text-slate-500">—</p>
            )}
          </div>
          <div>
            <h3 className="mb-2 text-[11px] font-medium uppercase tracking-wider text-slate-600">{t('topReferrers')}</h3>
            {data.analytics.top_referrers.length > 0 ? (
              <ul className="space-y-1 text-sm" data-testid="stats-referrers">
                {data.analytics.top_referrers.map((r) => (
                  <li key={r.referrer} className="flex justify-between gap-2 border-b border-slate-800/50 py-1">
                    <span className="truncate text-slate-300">{r.referrer}</span>
                    <span className="shrink-0 tabular-nums text-slate-500">{r.count}</span>
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
