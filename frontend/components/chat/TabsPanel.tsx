'use client';

import { DefaultFiche } from './DefaultFiche';
import { ProjectFiche } from './ProjectFiche';
import { ProjectListFiche } from './ProjectListFiche';
import { SkillsFiche } from './SkillsFiche';

// Un onglet représente une fiche ouverte dans le panel droit.
// panelType correspond aux types de fiches existants.
export interface Tab {
  id: string;           // clé unique : "project:aria", "skills", "experience", "projects"
  label: string;        // "Aria", "Skills", "Expérience", "Projets"
  panelType: 'project' | 'project_list' | 'skills' | 'experience';
  data: unknown;
}

interface TabsPanelProps {
  tabs: Tab[];
  activeTabId: string | null;
  onTabClick: (id: string) => void;
  onTabClose: (id: string) => void;
}

// Rendu de la fiche correspondant au panelType de l'onglet actif
function FicheContent({ tab }: { tab: Tab }) {
  switch (tab.panelType) {
    case 'project':
      // eslint-disable-next-line @typescript-eslint/no-explicit-any
      return <ProjectFiche data={tab.data as any} />;
    case 'project_list':
      // eslint-disable-next-line @typescript-eslint/no-explicit-any
      return <ProjectListFiche data={tab.data as any} />;
    case 'skills':
    case 'experience':
      // eslint-disable-next-line @typescript-eslint/no-explicit-any
      return <SkillsFiche data={tab.data as any} />;
    default:
      return null;
  }
}

export function TabsPanel({ tabs, activeTabId, onTabClick, onTabClose }: TabsPanelProps) {
  // Aucun onglet → fiche par défaut
  if (tabs.length === 0) {
    return (
      <div className="h-full overflow-y-auto">
        <DefaultFiche />
      </div>
    );
  }

  const activeTab = tabs.find((t) => t.id === activeTabId) ?? tabs[0];

  return (
    <div className="flex flex-col h-full overflow-hidden">
      {/* Barre d'onglets — scrollable horizontalement si débordement */}
      <div className="flex items-center border-b bg-muted/30 overflow-x-auto shrink-0">
        {tabs.map((tab) => {
          const isActive = tab.id === activeTab.id;
          return (
            <div
              key={tab.id}
              className={`flex items-center gap-1 px-3 py-2 text-sm cursor-pointer shrink-0 border-r
                          transition-colors select-none
                          ${isActive
                            ? 'bg-background text-foreground font-medium border-b-2 border-b-primary -mb-px'
                            : 'text-muted-foreground hover:text-foreground hover:bg-muted/50'
                          }`}
            >
              {/* Clic sur le label → active l'onglet */}
              <span onClick={() => onTabClick(tab.id)}>{tab.label}</span>

              {/* Bouton fermer — empêche la propagation pour ne pas activer l'onglet */}
              <button
                onClick={(e) => {
                  e.stopPropagation();
                  onTabClose(tab.id);
                }}
                aria-label={`Fermer ${tab.label}`}
                className="ml-1 text-muted-foreground hover:text-foreground leading-none text-xs"
              >
                ×
              </button>
            </div>
          );
        })}
      </div>

      {/* Contenu de l'onglet actif */}
      <div className="flex-1 overflow-y-auto">
        <FicheContent tab={activeTab} />
      </div>
    </div>
  );
}
