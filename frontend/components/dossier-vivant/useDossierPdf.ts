'use client';

import { useCallback, useEffect, useState } from 'react';
import { useLocale, useTranslations } from 'next-intl';

import { DossierPdfError, fetchDossierPdf, pdfFileName, type DossierPdfQuery } from '@/lib/dossier-vivant/pdfExport';
import type { RunState } from '@/lib/dossier-vivant/runState';

/**
 * `expired` : le serveur ne connaît pas (ou plus) ce dossier — DOSSIER_NOT_FOUND, conservé 24 h.
 * Distinct de `failed` : réessayer ne servira à rien, il faut relancer l'équipe.
 */
export type PdfDownloadState = 'idle' | 'busy' | 'failed' | 'expired';

/**
 * useDossierPdf — « Télécharger le dossier » : un VRAI PDF, mis en page A4 par le backend.
 *
 * QUOI : au clic, demande le PDF à `endpoint` avec la langue de l'interface et, pour un run réel,
 * l'id de la copie serveur du dossier (`state.dossierId`, reçu dans run.completed) ; remet le
 * fichier reçu au navigateur comme un téléchargement (lien `download` sur un blob — l'événement
 * `download` que l'E2E attend).
 * POURQUOI rien d'autre ne part : le backend imprime SA copie du dossier (celle du run qu'il a
 * exécuté, ou le scénario Tessera qu'il connaît) — un corps forgé ne fait rien imprimer.
 * POURQUOI deux états d'échec : un clic sans effet laisserait croire au visiteur que le PDF est
 * parti. L'échec se dit sur le bouton, glyphe ✕ compris ; `failed` invite à réessayer, `expired`
 * dit que le dossier n'est plus disponible.
 */
export function useDossierPdf(state: RunState, endpoint: string) {
  const t = useTranslations('dossierVivant.dossier');
  const locale = useLocale();
  const [status, setStatus] = useState<PdfDownloadState>('idle');

  // Un nouveau round (« Rejouer ») remet un autre dossier : l'échec ou l'expiration du précédent
  // ne le concerne pas. Le composant reste monté d'un round à l'autre, d'où la remise à zéro ici.
  useEffect(() => {
    setStatus('idle');
  }, [state.dossierId]);

  const download = useCallback(async () => {
    if (!state.dossier || !state.visitor || status === 'busy') return;
    const query: DossierPdfQuery = state.dossierId ? { lang: locale, dossierId: state.dossierId } : { lang: locale };
    setStatus('busy');
    try {
      const blob = await fetchDossierPdf(endpoint, query);
      const href = URL.createObjectURL(blob);
      const a = document.createElement('a');
      a.href = href;
      a.download = pdfFileName(t('pdf.filename', { organisation: state.visitor.organisation }));
      document.body.appendChild(a);
      a.click();
      a.remove();
      // Révocation différée : le navigateur lit le blob après le clic, pas pendant.
      setTimeout(() => URL.revokeObjectURL(href), 30_000);
      setStatus('idle');
    } catch (err) {
      setStatus(err instanceof DossierPdfError && err.code === 'DOSSIER_NOT_FOUND' ? 'expired' : 'failed');
    }
  }, [state, endpoint, locale, status, t]);

  return { status, download };
}
