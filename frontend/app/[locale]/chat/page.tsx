'use client';

import { useState, useCallback } from 'react';
import { useTranslations } from 'next-intl';
import { ChatPanel } from '@/components/chat/ChatPanel';
import { LeftPanel, Tip } from '@/components/chat/LeftPanel';
import { RightPanel } from '@/components/chat/RightPanel';
import { Tab } from '@/components/chat/TabsPanel';
import { cn } from '@/lib/utils';

// Force dynamic rendering (pas de SSR — page interactive)
export const dynamic = 'force-dynamic';

// MAX_TABS : on garde seulement 4 onglets, le plus ancien est éjecté si overflow
const MAX_TABS = 4;
// MAX_TIPS : on garde 3 tips max dans le LeftPanel (FIFO)
const MAX_TIPS = 3;

// Vues du téléphone. QUOI : sous `lg`, les trois panneaux (questions, conversation, fiches) ne
// tiennent pas côte à côte — 20 % de 390 px = 78 px de questions illisibles. On en montre UN, choisi
// par une barre d'onglets. POURQUOI tous restent montés (masqués par CSS) : la conversation en cours,
// le flux SSE et les fiches ouvertes survivent au changement de vue.
type VueMobile = 'questions' | 'conversation' | 'fiches';
const VUES_MOBILES: VueMobile[] = ['questions', 'conversation', 'fiches'];

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

  // Vue affichée sur téléphone (ignorée à partir de `lg`, où les trois panneaux sont visibles).
  const [vue, setVue] = useState<VueMobile>('conversation');
  // Une fiche s'est ouverte pendant qu'on lisait ailleurs : point sur l'onglet « Fiches », sans
  // arracher le lecteur à la réponse qu'il est en train de lire.
  const [ficheNonVue, setFicheNonVue] = useState(false);

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
    setFicheNonVue(true);
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

  // Clic sur un hint → déclenche l'envoi dans ChatPanel via externalMessage. Sur téléphone, on
  // bascule sur la conversation : c'est là que la réponse arrive.
  const handleHintClick = useCallback((message: string) => {
    setExternalMessage(message);
    setVue('conversation');
  }, []);

  const choisirVue = useCallback((v: VueMobile) => {
    setVue(v);
    if (v === 'fiches') setFicheNonVue(false);
  }, []);

  // Reset externalMessage après envoi (callback depuis ChatPanel)
  const handleExternalMessageSent = useCallback(() => {
    setExternalMessage(null);
  }, []);

  // Libellé d'un onglet mobile (« Fiches » porte le nombre de fiches ouvertes).
  const libelleVue = (v: VueMobile) =>
    v === 'fiches' && tabs.length > 0 ? t('views.sheetsCount', { count: tabs.length }) : t(`views.${v}`);

  return (
    // Hauteur = l'écran moins l'en-tête (h-16, 64 px — Header.tsx). `dvh` quand il existe : sur
    // téléphone, la barre d'adresse repliée ne doit pas pousser le champ de saisie hors de l'écran.
    <div className="flex h-[calc(100vh-64px)] flex-col overflow-hidden supports-[height:100dvh]:h-[calc(100dvh-64px)] lg:flex-row">
      {/* Barre de vues — téléphone et tablette seulement. L'onglet actif porte le filet d'or
          (« où l'on est »), comme l'entrée active de l'en-tête. */}
      <div role="tablist" aria-label={t('views.label')} className="flex shrink-0 border-b border-border bg-background/80 lg:hidden">
        {VUES_MOBILES.map((v) => {
          const actif = vue === v;
          return (
            <button
              key={v}
              type="button"
              role="tab"
              id={`chat-vue-${v}`}
              aria-selected={actif}
              aria-controls={`chat-panneau-${v}`}
              onClick={() => choisirVue(v)}
              className={cn(
                'relative flex min-h-[44px] flex-1 items-center justify-center gap-1.5 text-sm transition-colors',
                actif ? 'text-foreground' : 'text-muted-foreground hover:text-foreground',
              )}
            >
              {libelleVue(v)}
              {v === 'fiches' && ficheNonVue && !actif && (
                <>
                  <span aria-hidden="true" className="h-1.5 w-1.5 rounded-full bg-gold" />
                  <span className="sr-only">{t('views.newSheet')}</span>
                </>
              )}
              {actif && <span aria-hidden="true" className="absolute inset-x-4 bottom-0 h-px bg-gold" />}
            </button>
          );
        })}
      </div>

      {/* Questions : tips + relances + hints (22 % sur grand écran) */}
      <div
        id="chat-panneau-questions"
        role="tabpanel"
        aria-labelledby="chat-vue-questions"
        className={cn('min-h-0 lg:block lg:w-[22%] lg:shrink-0', vue === 'questions' ? 'flex-1' : 'hidden')}
      >
        <LeftPanel
          tips={tips}
          onTipClose={handleTipClose}
          onHintClick={handleHintClick}
          followups={followups}
        />
      </div>

      {/* Conversation au centre (39 %) */}
      <div
        id="chat-panneau-conversation"
        role="tabpanel"
        aria-labelledby="chat-vue-conversation"
        className={cn(
          'min-h-0 flex-col lg:flex lg:w-[39%] lg:shrink-0 lg:border-x lg:border-border',
          vue === 'conversation' ? 'flex flex-1' : 'hidden',
        )}
      >
        <ChatPanel
          onToolResult={handleToolResult}
          externalMessage={externalMessage}
          onExternalMessageSent={handleExternalMessageSent}
        />
      </div>

      {/* Fiches : onglets (39 %) */}
      <div
        id="chat-panneau-fiches"
        role="tabpanel"
        aria-labelledby="chat-vue-fiches"
        className={cn('min-h-0 lg:block lg:flex-1', vue === 'fiches' ? 'flex-1' : 'hidden')}
      >
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
