'use client';

import { TipBar, Tip } from './TipBar';
import { TabsPanel, Tab } from './TabsPanel';

// RightPanel = barre de tips (max 2) + zone d'onglets (max 4)
// Les tips compressent la zone de fiches — shrink-0 vs flex-1
interface RightPanelProps {
  tips: Tip[];
  tabs: Tab[];
  activeTabId: string | null;
  onTipClose: (id: string) => void;
  onTabClick: (id: string) => void;
  onTabClose: (id: string) => void;
}

export function RightPanel({ tips, tabs, activeTabId, onTipClose, onTabClick, onTabClose }: RightPanelProps) {
  return (
    <div className="flex flex-col h-full overflow-hidden">
      {/* Tips en haut — shrink-0, compressent la zone de fiches */}
      {tips.map((tip) => (
        <TipBar key={tip.id} tip={tip} onClose={onTipClose} />
      ))}

      {/* Zone d'onglets — prend le reste de l'espace */}
      <div className="flex-1 overflow-hidden">
        <TabsPanel
          tabs={tabs}
          activeTabId={activeTabId}
          onTabClick={onTabClick}
          onTabClose={onTabClose}
        />
      </div>
    </div>
  );
}
