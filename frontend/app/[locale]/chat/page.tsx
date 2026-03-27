'use client';

import { useState, useCallback } from 'react';
import { ChatPanel } from '@/components/chat/ChatPanel';
import { CenterZone } from '@/components/chat/CenterZone';
import { RightPanel } from '@/components/chat/RightPanel';
import { Tab } from '@/components/chat/TabsPanel';
import { Tip } from '@/components/chat/TipBar';

// Force dynamic rendering (pas de SSR — page interactive)
export const dynamic = 'force-dynamic';

// MAX_TABS : on garde seulement 4 onglets, le plus ancien est éjecté si overflow
const MAX_TABS = 4;
// MAX_TIPS : on garde 2 tips max (FIFO)
const MAX_TIPS = 2;

// Calcule le label d'un onglet depuis le panelType + les données
function tabLabel(panelType: Tab['panelType'], data: unknown): string {
  if (panelType === 'project' && data && typeof data === 'object' && 'name' in data) {
    return String((data as { name: string }).name);
  }
  if (panelType === 'project_list') return 'Projets';
  if (panelType === 'skills') return 'Skills';
  if (panelType === 'experience') return 'Expérience';
  return 'Fiche';
}

export default function ChatPage() {
  const [tabs, setTabs] = useState<Tab[]>([]);
  const [activeTabId, setActiveTabId] = useState<string | null>(null);
  const [tips, setTips] = useState<Tip[]>([]);

  // pushTab : ouvre ou met à jour un onglet selon sa clé.
  // Si la clé existe déjà → update les data en place.
  // Si nouvelle → prepend + cap à MAX_TABS (supprime le dernier si overflow).
  const pushTab = useCallback((key: string, panelType: Tab['panelType'], data: unknown) => {
    const label = tabLabel(panelType, data);
    setTabs((prev) => {
      const existingIdx = prev.findIndex((t) => t.id === key);
      if (existingIdx !== -1) {
        // Update les data de l'onglet existant (ex : réponse plus fraîche)
        const updated = [...prev];
        updated[existingIdx] = { ...updated[existingIdx], data, label };
        return updated;
      }
      // Nouvel onglet — prepend et cap à MAX_TABS
      const newTab: Tab = { id: key, label, panelType, data };
      return [newTab, ...prev].slice(0, MAX_TABS);
    });
    setActiveTabId(key);
  }, []);

  // handleToolResult — appelé par ChatPanel à chaque tool_result reçu en SSE
  const handleToolResult = useCallback((toolName: string, data: unknown) => {
    switch (toolName) {
      // --- Fiches projet ---
      case 'get_project':
      case 'show_project': {
        // Clé unique par projet (ex: "project:aria")
        const projectName =
          data && typeof data === 'object' && 'name' in data
            ? String((data as { name: string }).name).toLowerCase()
            : 'projet';
        pushTab(`project:${projectName}`, 'project', data);
        break;
      }

      // --- Liste des projets ---
      case 'list_projects':
      case 'show_projects':
        pushTab('projects', 'project_list', data);
        break;

      // --- Skills ---
      case 'list_skills':
      case 'show_skills':
        pushTab('skills', 'skills', data);
        break;

      // --- Expérience ---
      case 'get_experience':
      case 'show_experience':
        pushTab('experience', 'experience', data);
        break;

      // --- Tips latéraux (FIFO max 2) ---
      case 'add_tip': {
        const tipData = data as { text?: string; icon?: string } | null;
        if (!tipData?.text) break;
        const newTip: Tip = {
          id: `tip-${Date.now()}`,
          text: tipData.text,
          icon: tipData.icon,
        };
        // FIFO : on garde les MAX_TIPS derniers
        setTips((prev) => [...prev, newTip].slice(-MAX_TIPS));
        break;
      }
    }
  }, [pushTab]);

  // Fermer un tip par id
  const handleTipClose = useCallback((id: string) => {
    setTips((prev) => prev.filter((t) => t.id !== id));
  }, []);

  // Activer un onglet
  const handleTabClick = useCallback((id: string) => {
    setActiveTabId(id);
  }, []);

  // Fermer un onglet — active l'onglet suivant ou null si plus rien
  const handleTabClose = useCallback((id: string) => {
    setTabs((prev) => {
      const idx = prev.findIndex((t) => t.id === id);
      const next = prev.filter((t) => t.id !== id);
      setActiveTabId(next.length > 0 ? (next[Math.min(idx, next.length - 1)]?.id ?? null) : null);
      return next;
    });
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

      {/* Panel droit : tips + onglets (40%) */}
      <div className="w-[40%] border-l">
        <RightPanel
          tips={tips}
          tabs={tabs}
          activeTabId={activeTabId}
          onTipClose={handleTipClose}
          onTabClick={handleTabClick}
          onTabClose={handleTabClose}
        />
      </div>
    </div>
  );
}
