'use client';

import { useEffect, useState } from 'react';
import { useLocale } from 'next-intl';

import { absoluteDemoUrl, fetchDossierSkills, pdfSections, type DossierSkills } from '@/lib/dossier-vivant/pdfExport';
import type { RunState } from '@/lib/dossier-vivant/runState';

export type SkillsState = { status: 'idle' | 'loading' | 'failed' } | { status: 'ready'; data: DossierSkills };

/**
 * useDossierSkills — la section « Compétences mobilisées » du dossier à l'écran.
 *
 * QUOI : dès que la section des preuves est ÉCRITE (`done`), envoie les sections du dossier à la
 * route des compétences (pdfExport.ts › SKILLS_PATH) et rend la sélection du backend.
 * POURQUOI le backend choisit : c'est la même fonction qui compose la section du PDF
 * (services/dossier_skills.go) — l'écran et le document ne peuvent pas montrer deux listes. Le
 * front ne choisit, ne classe ni ne complète rien.
 * COMMENT : une seule requête par dossier — la clé est la liste des preuves ; le besoin et
 * l'architecture sont écrits avant les preuves (ordre des sections), ils ne changent donc plus.
 */
export function useDossierSkills(state: RunState, endpoint: string): SkillsState {
  const locale = useLocale();
  const [result, setResult] = useState<{ key: string; state: SkillsState }>({ key: '', state: { status: 'idle' } });

  const proofs = state.dossier?.sections.proofs;
  const ready = proofs?.status === 'done';
  const key = ready ? proofs.blocks.flatMap(({ block }) => (block.type === 'proof' ? [block.projectId] : [])).join(',') : '';

  useEffect(() => {
    if (!key) return;
    const ctl = new AbortController();
    setResult({ key, state: { status: 'loading' } });
    const sections = pdfSections(state, (d) => absoluteDemoUrl(d, window.location.origin));
    fetchDossierSkills(endpoint, { lang: locale, sections }, ctl.signal)
      .then((data) => setResult({ key, state: { status: 'ready', data } }))
      .catch(() => {
        if (!ctl.signal.aborted) setResult({ key, state: { status: 'failed' } });
      });
    return () => ctl.abort();
    // `state` change à chaque événement : la requête ne dépend que de la liste des preuves.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [key, endpoint, locale]);

  return result.key === key && key ? result.state : { status: 'idle' };
}
