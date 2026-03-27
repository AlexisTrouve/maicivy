'use client';

import { useState, useCallback } from 'react';
import { ChatPanel } from '@/components/chat/ChatPanel';
import { CenterZone } from '@/components/chat/CenterZone';
import { RightPanel, ActivePanel } from '@/components/chat/RightPanel';

// Force dynamic rendering (pas de SSR — page interactive)
export const dynamic = 'force-dynamic';

export default function ChatPage() {
  const [activePanel, setActivePanel] = useState<ActivePanel>('default');
  const [panelData, setPanelData] = useState<unknown>(null);

  // Callback appelé par ChatPanel quand un tool_result est reçu
  // Met à jour le panel droit selon le tool utilisé
  const handleToolResult = useCallback((toolName: string, data: unknown) => {
    switch (toolName) {
      // Tools de données — mettent aussi à jour le panel
      case 'get_project':
      case 'show_project':
        setActivePanel('project');
        setPanelData(data);
        break;
      case 'list_projects':
      case 'show_projects':
        setActivePanel('project_list');
        setPanelData(data);
        break;
      case 'list_skills':
      case 'show_skills':
        setActivePanel('skills');
        setPanelData(data);
        break;
      case 'get_experience':
      case 'show_experience':
        setActivePanel('experience');
        setPanelData(data);
        break;
    }
  }, []);

  return (
    // Hauteur = 100vh - hauteur header (64px = h-16)
    <div className="flex h-[calc(100vh-64px)] overflow-hidden">
      {/* Panel gauche : chat (40%) */}
      <div className="w-[40%] border-r flex flex-col">
        <ChatPanel onToolResult={handleToolResult} />
      </div>

      {/* Zone centrale réservée (20%) */}
      <CenterZone />

      {/* Panel droit : fiche contextuelle (40%) */}
      <div className="w-[40%] border-l">
        <RightPanel activePanel={activePanel} data={panelData} />
      </div>
    </div>
  );
}
