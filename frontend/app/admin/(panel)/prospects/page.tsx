'use client';

import { useCallback, useEffect, useState } from 'react';
import { useTranslations } from 'next-intl';
import { Panel, PageTitle, SectionLabel, btnNeutral, btnPrimary, fieldInline } from '@/components/admin/ui';

const API = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080';

type ProspectRow = {
  id: string;
  name: string;
  ref: string;
  active: boolean;
  pending: boolean;
  visits: number;
  cvs: number;
};

// Écran owner des prospecteurs (apporteurs d'affaires). Alexi ne gère d'eux QUE l'accès : inviter,
// révoquer, réactiver — et voir ce qu'ils rapportent (visites arrivées par leur ?ref=, CV générés).
// Owner-only : le garde serveur refuse /admin/prospects aux prospecteurs, et la nav ne le leur montre pas.
export default function ProspectsPage() {
  const t = useTranslations('admin.prospects');
  const [rows, setRows] = useState<ProspectRow[]>([]);
  const [name, setName] = useState('');
  const [email, setEmail] = useState('');
  const [link, setLink] = useState('');
  const [copied, setCopied] = useState(false);
  const [error, setError] = useState('');

  const load = useCallback(async () => {
    const res = await fetch(`${API}/api/v1/admin/prospects`, { credentials: 'include' });
    if (res.ok) setRows((await res.json()).prospects ?? []);
  }, []);

  useEffect(() => {
    void load();
  }, [load]);

  async function invite(e: React.FormEvent) {
    e.preventDefault();
    setError('');
    setLink('');
    setCopied(false);
    const res = await fetch(`${API}/api/v1/admin/prospects/invitations`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      credentials: 'include',
      body: JSON.stringify({ name, email }),
    });
    if (!res.ok) {
      setError(res.status === 409 ? t('errorExists') : t('errorGeneric'));
      return;
    }
    const j = await res.json();
    // Le jeton n'est renvoyé qu'UNE fois (seule son empreinte est stockée) : le lien s'affiche ici
    // pour être copié, il ne pourra pas être relu ensuite.
    setLink(`${window.location.origin}/admin/invite/${j.token}`);
    setName('');
    setEmail('');
    void load();
  }

  async function toggle(row: ProspectRow) {
    await fetch(`${API}/api/v1/admin/prospects/${row.id}/${row.active ? 'revoke' : 'activate'}`, {
      method: 'POST',
      credentials: 'include',
    });
    void load();
  }

  return (
    <div className="mx-auto max-w-5xl space-y-6" data-testid="admin-prospects">
      <PageTitle sub={t('sub')}>{t('title')}</PageTitle>

      <Panel hairline className="p-6">
        <form onSubmit={invite} className="flex flex-wrap items-center gap-3">
          <input
            value={name}
            onChange={(e) => setName(e.target.value)}
            placeholder={t('inviteName')}
            data-testid="prospect-invite-name"
            className={`${fieldInline} flex-1`}
          />
          <input
            value={email}
            onChange={(e) => setEmail(e.target.value)}
            placeholder={t('inviteEmail')}
            data-testid="prospect-invite-email"
            className={`${fieldInline} flex-1`}
          />
          {/* L'action principale de l'écran — la seule en or. */}
          <button type="submit" disabled={!name.trim()} data-testid="prospect-invite-submit" className={btnPrimary}>
            {t('invite')}
          </button>
        </form>
        {error && <p className="mt-3 text-sm text-rose-400">✕ {error}</p>}
        {link && (
          <div className="mt-4 space-y-2">
            <SectionLabel>{t('inviteLink')}</SectionLabel>
            <div className="flex items-center gap-2">
              <input readOnly value={link} data-testid="prospect-invite-link" className={`${fieldInline} flex-1 font-mono text-xs`} />
              <button
                type="button"
                className={btnNeutral}
                onClick={() => {
                  void navigator.clipboard?.writeText(link);
                  setCopied(true);
                }}
              >
                {copied ? `✓ ${t('copied')}` : t('copy')}
              </button>
            </div>
          </div>
        )}
      </Panel>

      <Panel className="overflow-x-auto p-6">
        {rows.length === 0 ? (
          <p className="text-sm text-slate-500">{t('empty')}</p>
        ) : (
          <table className="w-full text-sm" data-testid="prospect-table">
            <thead className="text-left text-xs uppercase tracking-wider text-slate-500">
              <tr>
                <th className="py-2">{t('colName')}</th>
                <th>{t('colRef')}</th>
                <th>{t('colStatus')}</th>
                <th className="text-right">{t('colVisits')}</th>
                <th className="text-right">{t('colCVs')}</th>
                <th className="text-right">{t('colActions')}</th>
              </tr>
            </thead>
            <tbody>
              {rows.map((r) => (
                <tr key={r.id} className="border-t border-slate-800" data-testid={`prospect-row-${r.name}`}>
                  <td className="py-2 text-slate-200">{r.name}</td>
                  <td className="font-mono text-slate-400">{r.ref}</td>
                  <td>
                    {/* Glyphe + couleur : l'état se lit sans le libellé. */}
                    {r.pending ? (
                      <span className="text-slate-400">– {t('statusPending')}</span>
                    ) : r.active ? (
                      <span className="text-emerald-400">✓ {t('statusActive')}</span>
                    ) : (
                      <span className="text-rose-400">✕ {t('statusRevoked')}</span>
                    )}
                  </td>
                  <td className="text-right tabular-nums">{r.visits}</td>
                  <td className="text-right tabular-nums">{r.cvs}</td>
                  <td className="text-right">
                    {!r.pending || r.active ? (
                      <button type="button" onClick={() => toggle(r)} data-testid={`prospect-toggle-${r.name}`} className={btnNeutral}>
                        {r.active ? t('revoke') : t('activate')}
                      </button>
                    ) : null}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </Panel>
    </div>
  );
}
