'use client';

import { TabsPanel, Tab } from './TabsPanel';

// RightPanel — panel droit (40%) contenant uniquement les onglets.
// Les tips ont été déplacés dans le LeftPanel.
interface RightPanelProps {
  tabs: Tab[];
  activeTabId: string | null;
  onTabClick: (id: string) => void;
  onTabClose: (id: string) => void;
  // Clic sur un projet dans ProjectListFiche → envoie un message dans le chat (même mécanisme que
  // les hints du LeftPanel) pour déclencher l'ouverture de sa fiche détail.
  onProjectClick: (message: string) => void;
}

export function RightPanel({ tabs, activeTabId, onTabClick, onTabClose, onProjectClick }: RightPanelProps) {
  return (
    <div className="flex flex-col h-full overflow-hidden">
      <TabsPanel
        tabs={tabs}
        activeTabId={activeTabId}
        onTabClick={onTabClick}
        onTabClose={onTabClose}
        onProjectClick={onProjectClick}
      />
    </div>
  );
}
