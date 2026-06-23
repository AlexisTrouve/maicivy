'use client';

import { useState } from 'react';
import { useTranslations } from 'next-intl';

const API = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080';

// Outil « CV depuis offre » du panneau admin. Colle une offre (texte ou URL) → le backend
// (POST /cv/generate, format pdf) génère un CV tailoré + couche stealth ATS → on l'affiche inline
// et on l'offre au téléchargement. credentials:include → le cookie admin donne les privilèges owner.
export default function AdminCVTool() {
  const t = useTranslations('admin');
  const [offer, setOffer] = useState('');
  const [lang, setLang] = useState('fr');
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');
  const [pdfUrl, setPdfUrl] = useState<string | null>(null);

  async function generate(e: React.FormEvent) {
    e.preventDefault();
    setLoading(true);
    setError('');
    if (pdfUrl) {
      URL.revokeObjectURL(pdfUrl);
      setPdfUrl(null);
    }
    try {
      const res = await fetch(`${API}/api/v1/cv/generate`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        credentials: 'include',
        body: JSON.stringify({ offer, lang, format: 'pdf' }),
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
      const blob = await res.blob();
      setPdfUrl(URL.createObjectURL(blob));
    } catch {
      setError(t('login.networkError'));
    } finally {
      setLoading(false);
    }
  }

  return (
    <div className="grid h-full gap-6 lg:grid-cols-2" data-testid="admin-cv-tool">
      <form onSubmit={generate} className="flex flex-col gap-3">
        <h1 className="text-2xl font-bold">{t('cv.title')}</h1>
        <p className="text-sm text-slate-400" data-testid="cv-help">{t('cv.help')}</p>
        <textarea
          value={offer}
          onChange={(e) => setOffer(e.target.value)}
          placeholder={t('cv.offerPlaceholder')}
          rows={13}
          data-testid="cv-offer"
          className="w-full resize-y rounded-lg border border-slate-700 bg-slate-950 p-3 text-sm outline-none focus:border-blue-500"
        />
        <div className="flex items-center gap-3">
          <select
            value={lang}
            onChange={(e) => setLang(e.target.value)}
            data-testid="cv-lang"
            className="rounded-md border border-slate-700 bg-slate-950 px-3 py-2 text-sm"
          >
            <option value="fr">Français</option>
            <option value="en">English</option>
          </select>
          <button
            type="submit"
            disabled={loading || offer.trim().length < 20}
            data-testid="cv-generate"
            className="rounded-md bg-blue-600 px-4 py-2 text-sm font-medium hover:bg-blue-500 disabled:opacity-50"
          >
            {loading ? t('cv.generating') : t('cv.generate')}
          </button>
        </div>
        {error && (
          <p data-testid="cv-error" className="text-sm text-red-400">
            {error}
          </p>
        )}
      </form>

      <div className="flex min-h-[400px] flex-col rounded-lg border border-slate-800 bg-slate-900">
        {pdfUrl ? (
          <>
            <div className="flex items-center justify-between border-b border-slate-800 px-4 py-2">
              <span className="text-sm text-slate-400">{t('cv.previewTitle')}</span>
              <a
                href={pdfUrl}
                download={`CV_${lang}.pdf`}
                data-testid="cv-download"
                className="rounded-md bg-green-600 px-3 py-1.5 text-sm font-medium hover:bg-green-500"
              >
                {t('cv.download')}
              </a>
            </div>
            <iframe src={pdfUrl} title={t('cv.previewTitle')} className="flex-1 rounded-b-lg" data-testid="cv-preview" />
          </>
        ) : (
          <div className="flex flex-1 items-center justify-center p-8 text-center text-sm text-slate-500">
            {loading ? t('cv.loading') : t('cv.empty')}
          </div>
        )}
      </div>
    </div>
  );
}
