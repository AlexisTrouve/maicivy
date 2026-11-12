'use client';

import { useTranslations } from 'next-intl';
import { DefaultFiche } from './DefaultFiche';
import { ProjectFiche } from './ProjectFiche';
import { ProjectListFiche } from './ProjectListFiche';
import { SkillsFiche } from './SkillsFiche';
import { BlogFiche } from './BlogFiche';
import { BlogListFiche } from './BlogListFiche';

// Un onglet représente une fiche ouverte dans le panel droit.
// panelType correspond aux types de fiches existants.
export interface Tab {
  id: string;           // clé unique : "project:aria", "skills", "experience", "projects", "blog:slug"
  label: string;        // "Aria", "Skills", "Expérience", "Projets", titre article
  panelType: 'project' | 'project_list' | 'skills' | 'experience' | 'blog' | 'blog_list';
  data: unknown;
}

interface TabsPanelProps {
  tabs: Tab[];
  activeTabId: string | null;
  onTabClick: (id: string) => void;
  onTabClose: (id: string) => void;
  onProjectClick: (message: string) => void;
}

// Rendu de la fiche correspondant au panelType de l'onglet actif
function FicheContent({ tab, onProjectClick }: { tab: Tab; onProjectClick: (message: string) => void }) {
  switch (tab.panelType) {
    case 'project':
      // eslint-disable-next-line @typescript-eslint/no-explicit-any
      return <ProjectFiche data={tab.data as any} />;
    case 'project_list':
      // eslint-disable-next-line @typescript-eslint/no-explicit-any
      return <ProjectListFiche data={tab.data as any} onProjectClick={onProjectClick} />;
    case 'skills':
    case 'experience':
      // eslint-disable-next-line @typescript-eslint/no-explicit-any
      return <SkillsFiche data={tab.data as any} />;
    case 'blog':
      // eslint-disable-next-line @typescript-eslint/no-explicit-any
      return <BlogFiche data={tab.data as any} />;
    case 'blog_list':
      // eslint-disable-next-line @typescript-eslint/no-explicit-any
      return <BlogListFiche data={tab.data as any} />;
    default:
      return null;
  }
}

export function TabsPanel({ tabs, activeTabId, onTabClick, onTabClose, onProjectClick }: TabsPanelProps) {
  const t = useTranslations('chat');
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
      <div className="flex items-center border-b bg-card/60 overflow-x-auto shrink-0">
        {tabs.map((tab) => {
          const isActive = tab.id === activeTab.id;
          return (
            <div
              key={tab.id}
              className={`flex items-center gap-1 px-3 py-2 text-sm cursor-pointer shrink-0 border-r
                          transition-colors select-none
                          ${isActive
                            ? 'bg-background text-foreground font-medium border-b border-b-gold -mb-px'
                            : 'text-muted-foreground hover:text-foreground hover:bg-muted/50'
                          }`}
            >
              {/* Clic sur le label → active l'onglet (un vrai bouton : atteignable au clavier).
                  L'onglet actif porte le filet d'or — « où l'on est ». */}
              <button type="button" onClick={() => onTabClick(tab.id)} aria-pressed={isActive} className="py-0.5">
                {tab.label}
              </button>

              {/* Bouton fermer — empêche la propagation pour ne pas activer l'onglet */}
              <button
                onClick={(e) => {
                  e.stopPropagation();
                  onTabClose(tab.id);
                }}
                aria-label={t('closeTab', { label: tab.label })}
                className="ml-1 p-1 text-muted-foreground hover:text-foreground leading-none text-sm"
              >
                ×
              </button>
            </div>
          );
        })}
      </div>

      {/* Contenu de l'onglet actif — key force le remount + animation au changement d'onglet */}
      <div key={activeTab.id} className="flex-1 overflow-y-auto animate-in slide-in-from-right-3 fade-in duration-200">
        <FicheContent tab={activeTab} onProjectClick={onProjectClick} />
      </div>
    </div>
  );
}
