'use client';

import { useEffect, useRef, useState } from 'react';
import { useTranslations } from 'next-intl';

const API = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080';

// Outil « Lettres » du panneau admin. Génère une PAIRE (motivation + anti-motivation) de façon
// ASYNC : enqueue → polling du job → liens PDF. credentials:include → cookie admin = owner → Opus.
//
// Subtilité : /letters/generate exige un cookie maicivy_session (la lettre est rattachée au visiteur
// de la session ; le PDF download vérifie ce rattachement). On s'assure donc d'abord d'une session
// (GET /api/v1/ pose le cookie) — pas de changement backend, le flux d'ownership existant marche.

interface JobStatus {
  status: 'queued' | 'processing' | 'completed' | 'failed';
  progress: number;
  letter_motivation_id?: string;
  letter_anti_motivation_id?: string;
  error?: string;
}

export default function AdminLettersTool() {
  const t = useTranslations('admin');
  const [company, setCompany] = useState('');
  const [offer, setOffer] = useState('');
  const [lang, setLang] = useState('fr');
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');
  const [job, setJob] = useState<JobStatus | null>(null);
  const pollRef = useRef<ReturnType<typeof setInterval> | null>(null);

  // Nettoyage du polling au démontage.
  useEffect(() => () => stopPolling(), []);

  function stopPolling() {
    if (pollRef.current) {
      clearInterval(pollRef.current);
      pollRef.current = null;
    }
  }

  async function generate(e: React.FormEvent) {
    e.preventDefault();
    setLoading(true);
    setError('');
    setJob(null);
    stopPolling();
    try {
      // 1. S'assurer d'une session visiteur (cookie maicivy_session) — requis par /letters/generate.
      await fetch(`${API}/api/v1/`, { credentials: 'include' }).catch(() => {});

      // 2. Enqueue le job (cookie admin → Opus).
      const res = await fetch(`${API}/api/v1/letters/generate`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        credentials: 'include',
        body: JSON.stringify({ company_name: company, job_offer: offer, lang }),
      });
      if (res.status !== 202 && !res.ok) {
        let msg = `${t('errorPrefix')} ${res.status}`;
        try {
          const j = await res.json();
          if (j?.error) msg = j.error;
        } catch {
          /* non-JSON */
        }
        setError(msg);
        setLoading(false);
        return;
      }
      const { job_id: jobId } = await res.json();
      // 3. Poll le statut du job toutes les 2s.
      pollRef.current = setInterval(() => pollJob(jobId), 2000);
      pollJob(jobId); // 1er poll immédiat
    } catch {
      setError(t('login.networkError'));
      setLoading(false);
    }
  }

  async function pollJob(jobId: string) {
    try {
      const res = await fetch(`${API}/api/v1/letters/job/${jobId}`, { credentials: 'include' });
      if (!res.ok) return; // job pas encore visible / transient → on retentera
      const j: JobStatus = await res.json();
      setJob(j);
      if (j.status === 'completed' || j.status === 'failed') {
        stopPolling();
        setLoading(false);
        if (j.status === 'failed') setError(j.error || t('letters.failed'));
      }
    } catch {
      /* transient → prochain tick */
    }
  }

  const pdfUrl = (id?: string) => `${API}/api/v1/letters/${id}/pdf`;

  return (
    <div className="grid h-full gap-6 lg:grid-cols-2" data-testid="admin-letters-tool">
      {/* Saisie */}
      <form onSubmit={generate} className="flex flex-col gap-3">
        <h1 className="text-2xl font-bold">{t('letters.title')}</h1>
        <p className="text-sm text-slate-400" data-testid="lt-help">{t('letters.help')}</p>
        <input
          value={company}
          onChange={(e) => setCompany(e.target.value)}
          placeholder={t('letters.companyPlaceholder')}
          data-testid="lt-company"
          className="w-full rounded-md border border-slate-700 bg-slate-950 px-3 py-2 text-sm outline-none focus:border-blue-500"
        />
        <textarea
          value={offer}
          onChange={(e) => setOffer(e.target.value)}
          placeholder={t('letters.offerPlaceholder')}
          rows={11}
          data-testid="lt-offer"
          className="w-full resize-y rounded-lg border border-slate-700 bg-slate-950 p-3 text-sm outline-none focus:border-blue-500"
        />
        <div className="flex items-center gap-3">
          <select
            value={lang}
            onChange={(e) => setLang(e.target.value)}
            data-testid="lt-lang"
            className="rounded-md border border-slate-700 bg-slate-950 px-3 py-2 text-sm"
          >
            <option value="fr">Français</option>
            <option value="en">English</option>
          </select>
          <button
            type="submit"
            disabled={loading || company.trim().length < 2}
            data-testid="lt-generate"
            className="rounded-md bg-blue-600 px-4 py-2 text-sm font-medium hover:bg-blue-500 disabled:opacity-50"
          >
            {loading ? t('letters.generating') : t('letters.generate')}
          </button>
        </div>
        {error && (
          <p data-testid="lt-error" className="text-sm text-red-400">
            {error}
          </p>
        )}
      </form>

      {/* Statut + résultat */}
      <div className="flex min-h-[300px] flex-col gap-3 rounded-lg border border-slate-800 bg-slate-900 p-5">
        {!job && !loading && (
          <div className="flex flex-1 items-center justify-center text-center text-sm text-slate-500">
            {t('letters.empty')}
          </div>
        )}

        {job && job.status !== 'completed' && (
          <div data-testid="lt-progress" className="flex flex-col gap-2">
            <span className="text-sm text-slate-400">
              {job.status === 'failed' ? t('letters.failed') : t('letters.progress')} ({job.progress}%)
            </span>
            <div className="h-2 w-full overflow-hidden rounded-full bg-slate-800">
              <div className="h-full rounded-full bg-blue-500 transition-all" style={{ width: `${job.progress}%` }} />
            </div>
          </div>
        )}

        {job?.status === 'completed' && (
          <div data-testid="lt-result" className="flex flex-col gap-3">
            <span className="text-sm text-green-400">{t('letters.done')}</span>
            <a
              href={pdfUrl(job.letter_motivation_id)}
              download
              data-testid="lt-pdf-motivation"
              className="rounded-md bg-green-600 px-3 py-2 text-center text-sm font-medium hover:bg-green-500"
            >
              {t('letters.downloadMotivation')}
            </a>
            <a
              href={pdfUrl(job.letter_anti_motivation_id)}
              download
              data-testid="lt-pdf-anti"
              className="rounded-md bg-slate-700 px-3 py-2 text-center text-sm font-medium hover:bg-slate-600"
            >
              {t('letters.downloadAnti')}
            </a>
          </div>
        )}
      </div>
    </div>
  );
}
