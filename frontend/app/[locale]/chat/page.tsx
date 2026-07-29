'use client';

import { useState, useCallback } from 'react';
import { useTranslations } from 'next-intl';
import { ChatPanel } from '@/components/chat/ChatPanel';
import { LeftPanel, Tip } from '@/components/chat/LeftPanel';
import { RightPanel } from '@/components/chat/RightPanel';
import { Tab } from '@/components/chat/TabsPanel';

// Force dynamic rendering (pas de SSR — page interactive)
export const dynamic = 'force-dynamic';

// MAX_TABS : on garde seulement 4 onglets, le plus ancien est éjecté si overflow
const MAX_TABS = 4;
// MAX_TIPS : on garde 3 tips max dans le LeftPanel (FIFO)
const MAX_TIPS = 3;

// Calcule le label d'un onglet depuis le panelType + les données.
// `t` = traducteur du namespace 'chat'. Les libellés statiques (projets/skills/expérience/blog/fiche)
// sont i18n ; les onglets project/blog gardent le nom/titre issu des données (non traduisible).
function tabLabel(panelType: Tab['panelType'], data: unknown, t: (key: string) => string): string {
  if (panelType === 'project' && data && typeof data === 'object' && 'name' in data) {
    return String((data as { name: string }).name);
  }
  if (panelType === 'project_list') return t('tabs.projects');
  if (panelType === 'skills') return t('tabs.skills');
  if (panelType === 'experience') return t('tabs.experience');
  if (panelType === 'blog' && data && typeof data === 'object' && 'title' in data) {
    // Truncate le titre pour l'onglet
    const title = String((data as { title: string }).title);
    return title.length > 20 ? title.slice(0, 20) + '…' : title;
  }
  if (panelType === 'blog_list') return t('tabs.blog');
  return t('tabs.default');
}

export default function ChatPage() {
  const t = useTranslations('chat');
  const [tabs, setTabs] = useState<Tab[]>([]);
  const [activeTabId, setActiveTabId] = useState<string | null>(null);
  const [tips, setTips] = useState<Tip[]>([]);
  // Questions de relance contextuelles (tool suggest_followups) — remplacent le pool statique de
  // hints dans le LeftPanel tant que le LLM en propose de nouvelles. Écrasées (pas accumulées) à
  // chaque nouvelle suggestion : les anciennes ne sont plus pertinentes une fois le sujet dépassé.
  const [followups, setFollowups] = useState<string[]>([]);

  // Message externe déclenché depuis LeftPanel (hint click)
  const [externalMessage, setExternalMessage] = useState<string | null>(null);

  // pushTab : ouvre ou met à jour un onglet selon sa clé.
  // Si la clé existe déjà → update les data en place.
  // Si nouvelle → prepend + cap à MAX_TABS (supprime le dernier si overflow).
  const pushTab = useCallback((key: string, panelType: Tab['panelType'], data: unknown) => {
    const label = tabLabel(panelType, data, t);
    setTabs((prev) => {
      const existingIdx = prev.findIndex((t) => t.id === key);
      if (existingIdx !== -1) {
        const updated = [...prev];
        updated[existingIdx] = { ...updated[existingIdx], data, label };
        return updated;
      }
      const newTab: Tab = { id: key, label, panelType, data };
      return [newTab, ...prev].slice(0, MAX_TABS);
    });
    setActiveTabId(key);
  }, [t]);

  // handleToolResult — appelé par ChatPanel à chaque tool_result reçu en SSE
  const handleToolResult = useCallback((toolName: string, data: unknown) => {
    switch (toolName) {
      // --- Fiches projet ---
      case 'get_project':
      case 'show_project': {
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

      // --- Article de blog ---
      case 'show_blog_article': {
        const blogData = data as { slug?: string; title?: string } | null;
        const slug = blogData?.slug ?? 'article';
        pushTab(`blog:${slug}`, 'blog', data);
        break;
      }

      // --- Liste des articles de blog ---
      case 'show_blog_list':
        pushTab('blog_list', 'blog_list', data);
        break;

      // --- Tips latéraux dans LeftPanel (FIFO max 3) ---
      case 'add_tip': {
        const tipData = data as { text?: string; icon?: string } | null;
        if (!tipData?.text) break;
        const newTip: Tip = {
          id: `tip-${Date.now()}`,
          text: tipData.text,
          icon: tipData.icon,
        };
        setTips((prev) => [...prev, newTip].slice(-MAX_TIPS));
        break;
      }

      // --- Questions de relance contextuelles (remplacent les hints statiques dans le LeftPanel) ---
      case 'suggest_followups': {
        const followupData = data as { questions?: string[] } | null;
        if (!followupData?.questions?.length) break;
        setFollowups(followupData.questions);
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

  // Clic sur un hint → déclenche l'envoi dans ChatPanel via externalMessage
  const handleHintClick = useCallback((message: string) => {
    setExternalMessage(message);
  }, []);

  // Reset externalMessage après envoi (callback depuis ChatPanel)
  const handleExternalMessageSent = useCallback(() => {
    setExternalMessage(null);
  }, []);

  return (
    // Hauteur = 100vh - hauteur header (64px = h-16)
    <div className="flex h-[calc(100vh-64px)] overflow-hidden">
      {/* LeftPanel : tips + hints (20%) */}
      <div className="w-[20%] shrink-0">
        <LeftPanel
          tips={tips}
          onTipClose={handleTipClose}
          onHintClick={handleHintClick}
          followups={followups}
        />
      </div>

      {/* ChatPanel au centre (40%) */}
      <div className="w-[40%] border-x flex flex-col shrink-0">
        <ChatPanel
          onToolResult={handleToolResult}
          externalMessage={externalMessage}
          onExternalMessageSent={handleExternalMessageSent}
        />
      </div>

      {/* RightPanel : onglets fiches (40%) */}
      <div className="w-[40%] border-l">
        <RightPanel
          tabs={tabs}
          activeTabId={activeTabId}
          onTabClick={handleTabClick}
          onTabClose={handleTabClose}
          onProjectClick={handleHintClick}
        />
      </div>
    </div>
  );
}
