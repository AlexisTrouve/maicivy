'use client';

import { useState } from 'react';
import { useTranslations } from 'next-intl';
import { Copy, Check } from 'lucide-react';
import { cn } from '@/lib/utils';

interface MessageBubbleProps {
  role: 'user' | 'assistant';
  content: string;
  isStreaming?: boolean; // active le curseur clignotant en fin de texte
}

// CopyButton — copie le texte BRUT (markdown non rendu) de la réponse assistant. Réutilise les
// clés i18n common.copy/copied déjà établies ailleurs sur le site (pas de nouvelles clés).
function CopyButton({ text }: { text: string }) {
  const t = useTranslations('common');
  const [copied, setCopied] = useState(false);

  const handleCopy = async () => {
    try {
      await navigator.clipboard.writeText(text);
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    } catch {
      /* clipboard indisponible (contexte non sécurisé / permission refusée) — pas de fallback bruyant */
    }
  };

  return (
    <button
      type="button"
      onClick={handleCopy}
      aria-label={copied ? t('copied') : t('copy')}
      title={copied ? t('copied') : t('copy')}
      className="mt-0.5 opacity-0 group-hover:opacity-100 focus-visible:opacity-100 transition-opacity
                 text-muted-foreground/60 hover:text-foreground p-1"
    >
      {copied ? <Check className="h-3 w-3" /> : <Copy className="h-3 w-3" />}
    </button>
  );
}

// Rendu Markdown basique (gras, code inline, LIENS, LISTES à puces) sans dépendance externe.
// POURQUOI pas de lib markdown : le chat ne produit que des réponses courtes et conversationnelles
// (pas de tableaux/imbrication) — une regex maison suffit et évite d'ajouter une dépendance pour ça.
// Avant l'ajout des liens/listes : un lien "[GitHub](url)" ou une liste "- Go\n- React" s'affichait
// avec les crochets/tirets bruts visibles au lieu d'un vrai rendu — pas cassé, juste moche.

// renderInline applique le formatage EN LIGNE (gras/code/lien) à un fragment de texte — pas de
// notion de bloc ici (une ligne, ou le texte d'un item de liste).
function renderInline(text: string, keyPrefix: string): React.ReactNode[] {
  const parts = text.split(/(\*\*[^*]+\*\*|`[^`]+`|\[[^\]]+\]\([^)]+\))/g);
  return parts.map((part, i) => {
    const key = `${keyPrefix}-${i}`;
    if (part.startsWith('**') && part.endsWith('**')) {
      return <strong key={key}>{part.slice(2, -2)}</strong>;
    }
    if (part.startsWith('`') && part.endsWith('`')) {
      return (
        <code key={key} className="px-1 py-0.5 rounded bg-muted text-sm font-mono">
          {part.slice(1, -1)}
        </code>
      );
    }
    const link = part.match(/^\[([^\]]+)\]\(([^)]+)\)$/);
    if (link) {
      return (
        <a
          key={key}
          href={link[2]}
          target="_blank"
          rel="noopener noreferrer"
          className="underline underline-offset-2 hover:text-primary"
        >
          {link[1]}
        </a>
      );
    }
    return <span key={key}>{part}</span>;
  });
}

// bulletLine : ligne de liste à puces ("- item" / "* item", indentation tolérée). Les puces
// CONSÉCUTIVES sont groupées dans un seul <ul> — pas de listes imbriquées, pas le besoin ici.
const BULLET_LINE = /^\s*[-*]\s+(.*)$/;

function renderMarkdown(text: string): React.ReactNode[] {
  const lines = text.split('\n');
  const blocks: React.ReactNode[] = [];
  let bulletBuffer: string[] = [];

  const flushBullets = (key: string) => {
    if (bulletBuffer.length === 0) return;
    blocks.push(
      <ul key={key} className="list-disc pl-4 my-1">
        {bulletBuffer.map((item, i) => (
          <li key={i}>{renderInline(item, `${key}-li-${i}`)}</li>
        ))}
      </ul>,
    );
    bulletBuffer = [];
  };

  lines.forEach((line, i) => {
    const bullet = line.match(BULLET_LINE);
    if (bullet) {
      bulletBuffer.push(bullet[1]);
      return;
    }
    flushBullets(`ul-${i}`);
    blocks.push(
      <span key={`line-${i}`}>
        {renderInline(line, `line-${i}`)}
        {i < lines.length - 1 && <br />}
      </span>,
    );
  });
  flushBullets('ul-end');

  return blocks;
}

export function MessageBubble({ role, content, isStreaming }: MessageBubbleProps) {
  const isUser = role === 'user';
  // Copie affichée seulement pour une réponse assistant FINALISÉE — pas pendant le streaming
  // (texte encore partiel), pas pour les messages user (peu d'intérêt à copier son propre message).
  const showCopy = !isUser && !isStreaming;

  return (
    <div className={cn('flex w-full flex-col group', isUser ? 'items-end' : 'items-start')}>
      <div
        className={cn(
          'max-w-[80%] rounded-2xl px-4 py-2.5 text-sm leading-relaxed',
          isUser
            ? 'bg-primary text-primary-foreground rounded-br-sm'
            : 'bg-card border rounded-bl-sm text-card-foreground',
        )}
      >
        {isUser ? content : renderMarkdown(content)}
        {/* Curseur clignotant pendant le streaming */}
        {isStreaming && (
          <span className="inline-block w-0.5 h-3.5 bg-current ml-0.5 align-middle animate-[cursor-blink_0.8s_ease-in-out_infinite]" />
        )}
      </div>
      {showCopy && <CopyButton text={content} />}
    </div>
  );
}

interface ToolBadgeProps {
  toolName: string;
  input?: unknown;
}

// Mapping tool → clé i18n (chat.toolLabels.*). Les libellés eux-mêmes sont traduits.
const TOOL_LABEL_KEYS: Record<string, string> = {
  show_project:      'project',
  get_project:       'project',
  show_projects:     'projects',
  list_projects:     'projects',
  show_skills:       'skills',
  list_skills:       'skills',
  show_experience:   'experience',
  get_experience:    'experience',
  show_blog_article: 'article',
  show_blog_list:    'blog',
  add_tip:           'tip',
};

// Badge compact affiché inline entre les messages lors d'un tool_call
export function ToolBadge({ toolName, input }: ToolBadgeProps) {
  const t = useTranslations('chat.toolLabels');
  // Clé connue → libellé traduit ; sinon on retombe sur le nom brut de l'outil.
  const key = TOOL_LABEL_KEYS[toolName];
  const base = key ? t(key) : toolName;

  // Paramètre principal si dispo (name ou slug)
  let param = '';
  if (input && typeof input === 'object') {
    const inp = input as Record<string, unknown>;
    const val = inp.name ?? inp.slug ?? inp.text;
    if (val && typeof val === 'string') param = val;
  }

  return (
    <div className="flex items-center gap-1.5 my-0.5 px-1">
      {/* Ligne de connexion verticale */}
      <div className="w-px h-3 bg-border ml-3" />
      <span className="text-[11px] text-muted-foreground/60 font-mono">
        {base}{param ? ` "${param}"` : ''}
      </span>
    </div>
  );
}
