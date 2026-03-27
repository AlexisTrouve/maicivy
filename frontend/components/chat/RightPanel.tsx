'use client';

import { DefaultFiche } from './DefaultFiche';
import { ProjectFiche } from './ProjectFiche';
import { ProjectListFiche } from './ProjectListFiche';
import { SkillsFiche } from './SkillsFiche';

export type ActivePanel = 'default' | 'project' | 'project_list' | 'skills' | 'experience';

interface RightPanelProps {
  activePanel: ActivePanel;
  data: unknown;
}

// RightPanel affiche la fiche contextuelle selon l'activePanel
// La transition entre fiches est gérée par un simple re-render (pas de Framer Motion pour éviter la dépendance)
export function RightPanel({ activePanel, data }: RightPanelProps) {
  return (
    <div className="h-full overflow-y-auto">
      {activePanel === 'default' && <DefaultFiche />}

      {/* eslint-disable-next-line @typescript-eslint/no-explicit-any */}
      {activePanel === 'project' && !!data && <ProjectFiche data={data as any} />}

      {/* eslint-disable-next-line @typescript-eslint/no-explicit-any */}
      {activePanel === 'project_list' && !!data && <ProjectListFiche data={data as any} />}

      {/* eslint-disable-next-line @typescript-eslint/no-explicit-any */}
      {(activePanel === 'skills' || activePanel === 'experience') && !!data && <SkillsFiche data={data as any} />}
    </div>
  );
}
