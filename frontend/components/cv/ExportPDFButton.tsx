'use client';

import { useState } from 'react';
import { useTranslations, useLocale } from 'next-intl';
import { Download, Loader2 } from 'lucide-react';

// Langues dans lesquelles le backend sait rendre un PDF (cf. /cv/export : fr, en, zh).
// POURQUOI : le site a 5 locales, le PDF 3. Envoyer « de » ou « it » tel quel valait un 400
// → le bouton affichait une erreur à tout visiteur allemand ou italien. Repli ANGLAIS, jamais
// français (même règle que le contenu maiProFiles, cf. CLAUDE.md).
const PDF_LANGS = ['fr', 'en', 'zh'];
export function pdfLang(locale: string): string {
  return PDF_LANGS.includes(locale) ? locale : 'en';
}

interface ExportPDFButtonProps {
  theme: string;
}

// Bouton d'export du CV — L'action principale de la page /cv, donc le SEUL aplat d'or de l'écran
// (doctrine de l'or : une action principale par écran). Encre sombre sur or : mêmes valeurs que
// `.btnGold` du dossier vivant (contraste 9,9:1, survol 12,6:1).
export default function ExportPDFButton({ theme }: ExportPDFButtonProps) {
  const [loading, setLoading] = useState(false);
  const [failed, setFailed] = useState(false);
  const t = useTranslations('cv.export');
  const locale = useLocale();

  const handleExport = async () => {
    setLoading(true);
    setFailed(false);

    try {
      const response = await fetch(
        `${process.env.NEXT_PUBLIC_API_URL}/api/v1/cv/export?theme=${theme}&format=pdf&lang=${pdfLang(locale)}`
      );
      if (!response.ok) throw new Error(`export ${response.status}`);

      // Get filename from Content-Disposition header
      const contentDisposition = response.headers.get('Content-Disposition');
      const filenameMatch = contentDisposition?.match(/filename="(.+)"/);
      const filename = filenameMatch ? filenameMatch[1] : `CV_${theme}.pdf`;

      // Download file
      const blob = await response.blob();
      const url = window.URL.createObjectURL(blob);
      const a = document.createElement('a');
      a.href = url;
      a.download = filename;
      document.body.appendChild(a);
      a.click();
      window.URL.revokeObjectURL(url);
      document.body.removeChild(a);
    } catch (err) {
      // Le visiteur lit TOUJOURS le message traduit : le texte technique de l'erreur (« Failed to
      // fetch », statut HTTP) est en anglais brut et ne lui dit rien. Il reste dans la console.
      console.error('CV export failed:', err);
      setFailed(true);
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="flex flex-col gap-2">
      <button
        type="button"
        onClick={handleExport}
        disabled={loading}
        aria-busy={loading}
        data-testid="cv-export-pdf"
        className="inline-flex h-11 items-center justify-center gap-2 whitespace-nowrap rounded-lg border border-gold bg-gold px-6 text-sm font-semibold text-primary-foreground transition-colors hover:bg-gold-bright focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 focus-visible:ring-offset-background active:scale-[0.98] disabled:cursor-wait disabled:opacity-70"
      >
        {loading ? (
          <>
            <Loader2 className="h-4 w-4 animate-spin" aria-hidden="true" />
            {t('downloading')}
          </>
        ) : (
          <>
            <Download className="h-4 w-4" aria-hidden="true" />
            {t('download')}
          </>
        )}
      </button>
      {/* L'état d'échec se lit au glyphe ✕ autant qu'à la couleur. */}
      {failed && (
        <p role="alert" className="text-sm text-destructive">
          <span aria-hidden="true">✕ </span>
          {t('error')}
        </p>
      )}
    </div>
  );
}
