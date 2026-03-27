'use client';

import { cn } from '@/lib/utils';

interface MessageBubbleProps {
  role: 'user' | 'assistant';
  content: string;
}

// Rendu Markdown basique (bold, code inline) sans dépendance externe
function renderMarkdown(text: string): React.ReactNode[] {
  // Découper sur les délimiteurs **bold** et `code`
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
    // Gérer les sauts de ligne
    return part.split('\n').map((line, j) => (
      <span key={`${i}-${j}`}>
        {line}
        {j < part.split('\n').length - 1 && <br />}
      </span>
    ));
  });
}

export function MessageBubble({ role, content }: MessageBubbleProps) {
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
      </div>
    </div>
  );
}

interface ToolBadgeProps {
  toolName: string;
  input?: unknown;
}

// Badge affiché inline entre les messages lors d'un tool_call
export function ToolBadge({ toolName, input }: ToolBadgeProps) {
  // Extraire le paramètre principal pour l'affichage
  let label = toolName;
  if (input && typeof input === 'object') {
    const inp = input as Record<string, unknown>;
    if (inp.name) label = `${toolName} · "${inp.name}"`;
  }

  return (
    <div className="flex justify-center my-1">
      <span className="inline-flex items-center gap-1.5 px-2.5 py-1 rounded-full bg-muted text-muted-foreground text-xs font-medium">
        <span>🔧</span>
        <span>{label}</span>
      </span>
    </div>
  );
}
