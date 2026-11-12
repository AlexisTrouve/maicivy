'use client';

import { useState } from 'react';
import { useLocale, useTranslations } from 'next-intl';
import { Download, Loader2 } from 'lucide-react';
import { adaptedLang, contextPdfUrl, saveResponseAsFile } from '@/lib/cvContext';
import { useCVContext, type CVView } from './CVContextProvider';

// ============================================================================
// Bandeau « Adapté à ce que vous m'avez dit » — visible seulement si le visiteur a exprimé un besoin
// (chat public ou dossier vivant) dans cette session. Sans contexte : rien, aucune question.
// ----------------------------------------------------------------------------
// États : en préparation (le standard reste affiché) · prêt (bascule adapté/standard + PDF adapté)
// · indisponible (échec ou quota IA du jour atteint — le standard reste, sans alarme).
//
// Doctrine de l'or : l'export standard de l'en-tête reste LA seule action pleine de l'écran ; le PDF
// adapté est un bouton à contour. La bascule reprend l'état actif des filtres de compétences
// (filet or + fond secondaire) et s'expose en `aria-pressed`, lisible sans la couleur.
// ============================================================================

const segment =
  'inline-flex h-9 items-center rounded-md border px-3 text-sm transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring';
const segmentOn = 'border-gold/70 bg-secondary text-foreground';
const segmentOff = 'border-border text-muted-foreground hover:text-foreground';

export function CVContextBanner() {
  const { view, shown, setShown } = useCVContext();
  const t = useTranslations('cv.context');
  const tExport = useTranslations('cv.export');
  const locale = useLocale();
  const [downloading, setDownloading] = useState(false);
  const [failed, setFailed] = useState(false);

  if (view.status === 'none') return null;

  const download = async () => {
    setDownloading(true);
    setFailed(false);
    try {
      const res = await fetch(contextPdfUrl(locale), { credentials: 'include' });
      if (!res.ok) throw new Error(`context pdf ${res.status}`);
      // Nom de repli = celui que le serveur pose : hors même origine, Content-Disposition n'est pas
      // lisible (en-tête non exposé par CORS) et le repli seul nomme le fichier.
      await saveResponseAsFile(res, `cv_adapte_${adaptedLang(locale)}.pdf`);
    } catch (err) {
      console.error('Adapted CV download failed:', err);
      setFailed(true);
    } finally {
      setDownloading(false);
    }
  };

  const toggle = (v: CVView, label: string, testId: string) => (
    <button
      type="button"
      aria-pressed={shown === v}
      onClick={() => setShown(v)}
      data-testid={testId}
      className={`${segment} ${shown === v ? segmentOn : segmentOff}`}
    >
      {label}
    </button>
  );

  return (
    <section
      aria-labelledby="cv-context-title"
      data-testid="cv-context-banner"
      data-status={view.status}
      className="mt-6 rounded-xl border border-border bg-card p-5"
    >
      <p className="text-[11px] font-medium uppercase tracking-[0.18em] text-muted-foreground">
        {view.source === 'dossier' ? t('sourceDossier') : t('sourceChat')}
      </p>
      <h2 id="cv-context-title" className="mt-2 text-lg font-medium text-foreground">
        {t('title')}
      </h2>
      {view.need && (
        <p
          className="mt-2 max-w-2xl text-sm italic text-muted-foreground"
          data-testid="cv-context-need"
        >
          {view.need}
        </p>
      )}

      <div aria-live="polite" className="mt-4">
        {view.status === 'pending' && (
          <p
            role="status"
            className="flex items-center gap-2 text-sm text-muted-foreground"
            data-testid="cv-context-pending"
          >
            <Loader2 className="h-4 w-4 animate-spin" aria-hidden="true" />
            {t('preparing')}
          </p>
        )}
        {(view.status === 'failed' || view.status === 'limited') && (
          <p className="text-sm text-muted-foreground" data-testid="cv-context-unavailable">
            {t('unavailable')}
          </p>
        )}
        {view.status === 'ready' && (
          <div className="flex flex-wrap items-center gap-3">
            <div role="group" aria-label={t('viewLabel')} className="flex gap-2">
              {toggle('adapted', t('showAdapted'), 'cv-view-adapted')}
              {toggle('standard', t('showStandard'), 'cv-view-standard')}
            </div>
            <button
              type="button"
              onClick={download}
              disabled={downloading}
              aria-busy={downloading}
              data-testid="cv-context-download"
              className="inline-flex h-9 items-center gap-2 rounded-md border border-border px-3 text-sm text-foreground transition-colors hover:border-gold/70 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring disabled:cursor-wait disabled:opacity-70"
            >
              {downloading ? (
                <Loader2 className="h-4 w-4 animate-spin" aria-hidden="true" />
              ) : (
                <Download className="h-4 w-4" aria-hidden="true" />
              )}
              {downloading ? tExport('downloading') : t('downloadAdapted')}
            </button>
          </div>
        )}
        {failed && (
          <p role="alert" className="mt-2 text-sm text-destructive">
            <span aria-hidden="true">✕ </span>
            {tExport('error')}
          </p>
        )}
      </div>
    </section>
  );
}
