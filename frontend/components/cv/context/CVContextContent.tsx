'use client';

import type { ReactNode } from 'react';
import { useTranslations } from 'next-intl';
import ExperienceTimeline from '@/components/cv/ExperienceTimeline';
import SkillsCloud from '@/components/cv/SkillsCloud';
import ProjectsGrid from '@/components/cv/ProjectsGrid';
import { SectionHeading } from '@/components/cv/PageHeading';
import { useCVContext } from './CVContextProvider';

// ============================================================================
// Corps du CV : le standard (rendu serveur, reçu en `children`) ou l'adapté (JSON de /cv/context).
// ----------------------------------------------------------------------------
// Le standard reste MONTÉ quand l'adapté est affiché (simplement masqué) : revenir au standard est
// instantané et ne relance ni ses Suspense ni ses fetchs. Le CV adapté a la même forme que celui de
// GET /cv (AdaptiveCVResponse), il passe donc par les mêmes composants — mêmes sections, même
// apparence, seuls l'ordre et les accroches changent. Les LOC par langage (enrichissement de la
// fiche d'un skill) ne sont pas rechargées pour lui : la fiche s'affiche sans ce bloc.
// ============================================================================

export function CVContextContent({ children }: { children: ReactNode }) {
  const { view, shown } = useCVContext();
  const t = useTranslations('cv');
  const cv = shown === 'adapted' ? view.cv : undefined;

  return (
    <>
      <div hidden={!!cv} data-testid="cv-standard">
        {children}
      </div>
      {cv && (
        <main className="space-y-20" data-testid="cv-adapted">
          <section id="experiences">
            <SectionHeading>{t('sections.experiences')}</SectionHeading>
            <ExperienceTimeline experiences={cv.experiences} />
          </section>
          <section id="skills">
            <SectionHeading>{t('sections.skills')}</SectionHeading>
            <SkillsCloud
              skills={cv.skills}
              projects={cv.projects}
              experiences={cv.experiences}
              langStats={null}
            />
          </section>
          <section id="projects">
            <SectionHeading>{t('sections.projects')}</SectionHeading>
            <ProjectsGrid projects={cv.projects} />
          </section>
        </main>
      )}
    </>
  );
}
