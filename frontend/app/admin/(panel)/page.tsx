'use client';

import { useState } from 'react';
import { useTranslations } from 'next-intl';
import { Panel, PageTitle, SectionLabel, btnPrimary, btnSuccess, fieldArea, fieldInline } from '@/components/admin/ui';

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
          // Plafond quotidien du prospecteur : message traduit, pas le libellé technique du backend.
          if (j?.code === 'PROSPECT_DAILY_LIMIT') msg = t('prospects.dailyLimit');
          else if (j?.error) msg = j.error;
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
    <div className="mx-auto grid h-full w-full max-w-6xl gap-8 lg:grid-cols-2" data-testid="admin-cv-tool">
      <form onSubmit={generate} className="flex flex-col gap-4">
        <PageTitle sub={<span data-testid="cv-help">{t('cv.help')}</span>}>{t('cv.title')}</PageTitle>
        <textarea
          value={offer}
          onChange={(e) => setOffer(e.target.value)}
          placeholder={t('cv.offerPlaceholder')}
          rows={13}
          data-testid="cv-offer"
          // flex-1 : la zone de saisie occupe toute la hauteur restante de la colonne.
          // POURQUOI : à hauteur fixe, le formulaire se tassait en haut d'un écran de 1000 px et
          // laissait la moitié basse vide. En laissant les deux colonnes descendre jusqu'en bas,
          // l'écran se lit comme une composition — et on gagne de la place pour coller une offre
          // longue, ce qui est exactement l'usage.
          className={`${fieldArea} flex-1`}
        />
        <div className="flex items-center gap-3">
          <select value={lang} onChange={(e) => setLang(e.target.value)} data-testid="cv-lang" className={`${fieldInline} cursor-pointer`}>
            <option value="fr">{t('outputLang.fr')}</option>
            <option value="en">{t('outputLang.en')}</option>
            {/* zh : PDF tailoré natif en chinois (libellés, texte ATS, prompt) côté backend */}
            <option value="zh">{t('outputLang.zh')}</option>
          </select>
          <button type="submit" disabled={loading || offer.trim().length < 20} data-testid="cv-generate" className={btnPrimary}>
            {loading ? t('cv.generating') : t('cv.generate')}
          </button>
        </div>
        {error && (
          <p data-testid="cv-error" className="text-sm text-rose-400">
            {error}
          </p>
        )}
      </form>

      {/* Le PDF est le LIVRABLE de cet écran : filet doré, comme les panneaux qui portent un
          résultat et non une saisie. C'est ce qu'on vient chercher ici. */}
      {/* Pleine hauteur, et non plus rétréci à son contenu : la colonne de saisie descend désormais
          jusqu'en bas, donc un panneau court créerait un décrochage au lieu d'une paire équilibrée.
          Et pour le PDF, chaque pixel de hauteur se lit. */}
      <Panel hairline className="flex min-h-[400px] flex-col overflow-hidden">
        {pdfUrl ? (
          <>
            <div className="flex items-center justify-between gap-3 border-b border-slate-800/80 px-4 py-2.5">
              <SectionLabel>{t('cv.previewTitle')}</SectionLabel>
              <a href={pdfUrl} download={`CV_${lang}.pdf`} data-testid="cv-download" className={btnSuccess}>
                {t('cv.download')}
              </a>
            </div>
            {/* Fond blanc explicite : un PDF s'affiche sur du blanc, et sans ça le cadre du visualiseur
                se découpait en gris sale sur le fond sombre du panneau. */}
            <iframe src={pdfUrl} title={t('cv.previewTitle')} className="flex-1 bg-white" data-testid="cv-preview" />
          </>
        ) : (
          <div className="flex flex-1 items-center justify-center p-8 text-center text-sm text-slate-500">
            {loading ? t('cv.loading') : t('cv.empty')}
          </div>
        )}
      </Panel>
    </div>
  );
}
