'use client';

import { TabsPanel, Tab } from './TabsPanel';

// RightPanel — panel droit (40%) contenant uniquement les onglets.
// Les tips ont été déplacés dans le LeftPanel.
interface RightPanelProps {
  tabs: Tab[];
  activeTabId: string | null;
  onTabClick: (id: string) => void;
  onTabClose: (id: string) => void;
}

export function RightPanel({ tabs, activeTabId, onTabClick, onTabClose }: RightPanelProps) {
  return (
    <div className="flex flex-col h-full overflow-hidden">
      <TabsPanel
        tabs={tabs}
        activeTabId={activeTabId}
        onTabClick={onTabClick}
        onTabClose={onTabClose}
      />
    </div>
  );
}
