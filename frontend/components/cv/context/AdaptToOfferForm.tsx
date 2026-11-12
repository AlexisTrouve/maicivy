'use client';

import { useId, useState } from 'react';
import { useLocale, useTranslations } from 'next-intl';
import { Loader2 } from 'lucide-react';
import { adaptedLang, generateUrl, saveResponseAsFile } from '@/lib/cvContext';

// ============================================================================
// Lien discret « l'adapter à une offre » — le flux EXISTANT de POST /api/v1/cv/generate (format
// pdf), rien de nouveau côté serveur : l'offre (texte ou adresse) part, le PDF adapté revient.
// ----------------------------------------------------------------------------
// Replié par défaut : la page ne pose aucune question (décision d'Alexi). Une génération coûte
// ~20 s et tire sur le quota IA du visiteur : le libellé d'attente le dit, et un refus de quota
// (429/503) a son propre message plutôt qu'une erreur générique.
// ============================================================================

type Phase = 'idle' | 'sending' | 'error' | 'limit';

export function AdaptToOfferForm() {
  const t = useTranslations('cv.context.offer');
  const locale = useLocale();
  const fieldId = useId();
  const [open, setOpen] = useState(false);
  const [offer, setOffer] = useState('');
  const [phase, setPhase] = useState<Phase>('idle');

  const submit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!offer.trim()) return;
    setPhase('sending');
    try {
      const res = await fetch(generateUrl(), {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        credentials: 'include',
        body: JSON.stringify({ offer: offer.trim(), lang: adaptedLang(locale), format: 'pdf' }),
      });
      if (res.status === 429 || res.status === 503) {
        setPhase('limit');
        return;
      }
      if (!res.ok) throw new Error(`generate ${res.status}`);
      await saveResponseAsFile(res, 'CV_offre.pdf');
      setPhase('idle');
    } catch (err) {
      console.error('CV generation from offer failed:', err);
      setPhase('error');
    }
  };

  if (!open) {
    return (
      <button
        type="button"
        onClick={() => setOpen(true)}
        data-testid="cv-offer-toggle"
        className="mt-4 text-sm text-muted-foreground underline decoration-border underline-offset-4 transition-colors hover:text-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
      >
        {t('toggle')}
      </button>
    );
  }

  const sending = phase === 'sending';
  return (
    <form onSubmit={submit} className="mt-4 max-w-2xl space-y-3" data-testid="cv-offer-form">
      <label
        htmlFor={fieldId}
        className="block text-xs font-medium uppercase tracking-[0.18em] text-muted-foreground"
      >
        {t('label')}
      </label>
      <textarea
        id={fieldId}
        value={offer}
        onChange={(e) => setOffer(e.target.value)}
        placeholder={t('placeholder')}
        rows={5}
        disabled={sending}
        data-testid="cv-offer-input"
        className="w-full rounded-lg border border-border bg-surface-inset p-3 text-sm text-foreground placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
      />
      <div className="flex flex-wrap items-center gap-3">
        <button
          type="submit"
          disabled={sending || !offer.trim()}
          aria-busy={sending}
          data-testid="cv-offer-submit"
          className="inline-flex h-9 items-center gap-2 rounded-md border border-border px-3 text-sm text-foreground transition-colors hover:border-gold/70 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring disabled:cursor-not-allowed disabled:opacity-60"
        >
          {sending && <Loader2 className="h-4 w-4 animate-spin" aria-hidden="true" />}
          {sending ? t('submitting') : t('submit')}
        </button>
        <button
          type="button"
          onClick={() => setOpen(false)}
          disabled={sending}
          className="text-sm text-muted-foreground underline-offset-4 hover:underline"
        >
          {t('cancel')}
        </button>
      </div>
      {(phase === 'error' || phase === 'limit') && (
        <p role="alert" className="text-sm text-destructive">
          <span aria-hidden="true">✕ </span>
          {phase === 'limit' ? t('limit') : t('error')}
        </p>
      )}
    </form>
  );
}
