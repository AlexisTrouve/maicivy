import type { CVData } from '@/lib/types';

// ============================================================================
// CV adapté au besoin du visiteur — contrat de GET /api/v1/cv/context (backend/internal/api/
// cv_context.go) et petites fonctions réseau partagées par CVContextSwitch et AdaptToOfferForm.
// ----------------------------------------------------------------------------
// Le backend garde un RÉSUMÉ du besoin exprimé dans le chat public ou un dossier vivant (cookie de
// session), prépare en fond un CV adapté, et le rend ici. Les états sont des identifiants stables :
// le front les traduit, il n'affiche jamais un texte venu du serveur hormis le résumé lui-même.
// ============================================================================

export type CVContextStatus = 'none' | 'pending' | 'ready' | 'failed' | 'limited';
export type CVContextSource = 'chat' | 'dossier';

export interface CVContextView {
  status: CVContextStatus;
  need?: string;
  source?: CVContextSource;
  lang?: string;
  cv?: CVData;
}

const API_BASE = process.env.NEXT_PUBLIC_API_URL || '';

// Langues du CV adapté (comme le PDF) : fr/en/zh natifs, le reste en anglais — même repli que
// ExportPDFButton.pdfLang et que TailoredLanguage côté Go, jamais le français par défaut.
const ADAPTED_LANGS = ['fr', 'en', 'zh'];
export function adaptedLang(locale: string): string {
  return ADAPTED_LANGS.includes(locale) ? locale : 'en';
}

export const contextUrl = (locale: string) =>
  `${API_BASE}/api/v1/cv/context?lang=${adaptedLang(locale)}`;
export const contextPdfUrl = (locale: string) =>
  `${API_BASE}/api/v1/cv/context/pdf?lang=${adaptedLang(locale)}`;
export const generateUrl = () => `${API_BASE}/api/v1/cv/generate`;

/**
 * Lit l'état du CV adapté. `credentials: 'include'` : c'est le cookie de session qui porte le
 * contexte. Toute réponse illisible ou en erreur vaut « none » : la page garde alors le CV standard,
 * qui est de toute façon déjà affiché — rien n'est perdu pour le visiteur.
 */
export async function fetchCVContext(locale: string, signal?: AbortSignal): Promise<CVContextView> {
  try {
    const res = await fetch(contextUrl(locale), {
      credentials: 'include',
      cache: 'no-store',
      signal,
    });
    if (!res.ok) return { status: 'none' };
    const view = (await res.json()) as CVContextView;
    return view && typeof view.status === 'string' ? view : { status: 'none' };
  } catch {
    return { status: 'none' };
  }
}

/** Déclenche le téléchargement d'un PDF reçu (même recette que ExportPDFButton). */
export async function saveResponseAsFile(res: Response, fallbackName: string): Promise<void> {
  const disposition = res.headers.get('Content-Disposition');
  const match = disposition?.match(/filename="?([^";]+)"?/);
  const blob = await res.blob();
  const url = window.URL.createObjectURL(blob);
  const a = document.createElement('a');
  a.href = url;
  a.download = match ? match[1] : fallbackName;
  document.body.appendChild(a);
  a.click();
  window.URL.revokeObjectURL(url);
  document.body.removeChild(a);
}
