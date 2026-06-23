'use client';

import { useState } from 'react';
import { useTranslations } from 'next-intl';

const API = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080';

// Outil « Messages plateforme » du panneau admin. Colle une mission (Malt/LinkedIn/Upwork) → le
// backend (POST /messages/generate, sync ~5s) renvoie un message prêt à envoyer. credentials:include
// → cookie admin = owner → modèle Opus, pas de rate-limit (vs version publique Haiku plafonnée).
export default function AdminMessagesTool() {
  const t = useTranslations('admin');
  const [mission, setMission] = useState('');
  const [platform, setPlatform] = useState('malt');
  const [tjm, setTjm] = useState('');
  const [lang, setLang] = useState('fr');
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');
  const [content, setContent] = useState('');
  const [copied, setCopied] = useState(false);

  async function generate(e: React.FormEvent) {
    e.preventDefault();
    setLoading(true);
    setError('');
    setContent('');
    setCopied(false);
    try {
      const body: Record<string, unknown> = { mission, platform, lang };
      if (tjm) body.tjm = parseInt(tjm, 10); // optionnel — omis si vide (backend: 50–5000)
      const res = await fetch(`${API}/api/v1/messages/generate`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        credentials: 'include',
        body: JSON.stringify(body),
      });
      if (!res.ok) {
        let msg = `${t('errorPrefix')} ${res.status}`;
        try {
          const j = await res.json();
          if (j?.error) msg = j.error;
        } catch {
          /* corps non-JSON */
        }
        setError(msg);
        return;
      }
      const j = await res.json();
      setContent(j.content || '');
    } catch {
      setError(t('login.networkError'));
    } finally {
      setLoading(false);
    }
  }

  async function copy() {
    await navigator.clipboard.writeText(content);
    setCopied(true);
  }

  return (
    <div className="grid h-full gap-6 lg:grid-cols-2" data-testid="admin-messages-tool">
      {/* Saisie */}
      <form onSubmit={generate} className="flex flex-col gap-3">
        <h1 className="text-2xl font-bold">{t('messages.title')}</h1>
        <p className="text-sm text-slate-400" data-testid="msg-help">{t('messages.help')}</p>
        <textarea
          value={mission}
          onChange={(e) => setMission(e.target.value)}
          placeholder={t('messages.missionPlaceholder')}
          rows={12}
          data-testid="msg-mission"
          className="w-full resize-y rounded-lg border border-slate-700 bg-slate-950 p-3 text-sm outline-none focus:border-blue-500"
        />
        <div className="flex flex-wrap items-center gap-3">
          <select
            value={platform}
            onChange={(e) => setPlatform(e.target.value)}
            data-testid="msg-platform"
            className="rounded-md border border-slate-700 bg-slate-950 px-3 py-2 text-sm"
          >
            <option value="malt">Malt</option>
            <option value="linkedin">LinkedIn</option>
            <option value="upwork">Upwork</option>
          </select>
          <input
            type="number"
            value={tjm}
            onChange={(e) => setTjm(e.target.value)}
            placeholder={t('messages.tjmPlaceholder')}
            data-testid="msg-tjm"
            className="w-24 rounded-md border border-slate-700 bg-slate-950 px-3 py-2 text-sm"
          />
          <select
            value={lang}
            onChange={(e) => setLang(e.target.value)}
            data-testid="msg-lang"
            className="rounded-md border border-slate-700 bg-slate-950 px-3 py-2 text-sm"
          >
            <option value="fr">Français</option>
            <option value="en">English</option>
          </select>
          <button
            type="submit"
            disabled={loading || mission.trim().length < 20}
            data-testid="msg-generate"
            className="rounded-md bg-blue-600 px-4 py-2 text-sm font-medium hover:bg-blue-500 disabled:opacity-50"
          >
            {loading ? t('messages.generating') : t('messages.generate')}
          </button>
        </div>
        {error && (
          <p data-testid="msg-error" className="text-sm text-red-400">
            {error}
          </p>
        )}
      </form>

      {/* Résultat */}
      <div className="flex min-h-[300px] flex-col rounded-lg border border-slate-800 bg-slate-900">
        {content ? (
          <>
            <div className="flex items-center justify-between border-b border-slate-800 px-4 py-2">
              <span className="text-sm text-slate-400">{t('messages.resultTitle')}</span>
              <button
                onClick={copy}
                data-testid="msg-copy"
                className="rounded-md bg-green-600 px-3 py-1.5 text-sm font-medium hover:bg-green-500"
              >
                {copied ? t('messages.copied') : t('messages.copy')}
              </button>
            </div>
            <pre data-testid="msg-result" className="flex-1 overflow-auto whitespace-pre-wrap p-4 text-sm text-slate-200">
              {content}
            </pre>
          </>
        ) : (
          <div className="flex flex-1 items-center justify-center p-8 text-center text-sm text-slate-500">
            {loading ? t('messages.loading') : t('messages.empty')}
          </div>
        )}
      </div>
    </div>
  );
}
