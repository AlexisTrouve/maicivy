'use client';

import { cn } from '@/lib/utils';

interface MessageBubbleProps {
  role: 'user' | 'assistant';
  content: string;
  isStreaming?: boolean; // active le curseur clignotant en fin de texte
}

// Rendu Markdown basique (bold, code inline) sans dépendance externe
function renderMarkdown(text: string): React.ReactNode[] {
  const parts = text.split(/(\*\*[^*]+\*\*|`[^`]+`)/g);
  return parts.map((part, i) => {
    if (part.startsWith('**') && part.endsWith('**')) {
      return <strong key={i}>{part.slice(2, -2)}</strong>;
    }
    if (part.startsWith('`') && part.endsWith('`')) {
      return (
        <code key={i} className="px-1 py-0.5 rounded bg-muted text-sm font-mono">
          {part.slice(1, -1)}
        </code>
      );
    }
    return part.split('\n').map((line, j) => (
      <span key={`${i}-${j}`}>
        {line}
        {j < part.split('\n').length - 1 && <br />}
      </span>
    ));
  });
}

export function MessageBubble({ role, content, isStreaming }: MessageBubbleProps) {
  const isUser = role === 'user';

  return (
    <div className={cn('flex w-full', isUser ? 'justify-end' : 'justify-start')}>
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
    </div>
  );
}

interface ToolBadgeProps {
  toolName: string;
  input?: unknown;
}

// Mapping tool → label court lisible
const TOOL_LABELS: Record<string, string> = {
  show_project:      'projet',
  get_project:       'projet',
  show_projects:     'projets',
  list_projects:     'projets',
  show_skills:       'skills',
  list_skills:       'skills',
  show_experience:   'expérience',
  get_experience:    'expérience',
  show_blog_article: 'article',
  show_blog_list:    'blog',
  add_tip:           'tip',
};

// Badge compact affiché inline entre les messages lors d'un tool_call
export function ToolBadge({ toolName, input }: ToolBadgeProps) {
  const base = TOOL_LABELS[toolName] ?? toolName;

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
