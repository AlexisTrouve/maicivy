'use client';

import { useRef, useState, useCallback, useEffect } from 'react';
import { Button } from '@/components/ui/button';
import { Send } from 'lucide-react';
import { streamChat, ChatMessage } from '@/lib/chat-api';
import { MessageBubble, ToolBadge } from './MessageBubble';

// Représente un item dans la liste d'affichage (message ou badge tool)
type DisplayItem =
  | { kind: 'message'; role: 'user' | 'assistant'; content: string; id: string }
  | { kind: 'tool'; name: string; input: unknown; id: string };

interface ChatPanelProps {
  // Remonte les données tool_result pour mettre à jour le panel droit
  onToolResult: (toolName: string, data: unknown) => void;
}

export function ChatPanel({ onToolResult }: ChatPanelProps) {
  const [items, setItems] = useState<DisplayItem[]>([]);
  const [history, setHistory] = useState<ChatMessage[]>([]);
  const [input, setInput] = useState('');
  const [isStreaming, setIsStreaming] = useState(false);
  // Accumulateur pour le texte assistant en cours de streaming
  const [streamingText, setStreamingText] = useState('');

  const messagesRef = useRef<HTMLDivElement>(null);
  const textareaRef = useRef<HTMLTextAreaElement>(null);
  const abortRef = useRef<AbortController | null>(null);

  // Scroll le container du chat vers le bas — sans toucher à la window
  useEffect(() => {
    const el = messagesRef.current;
    if (el) el.scrollTop = el.scrollHeight;
  }, [items, streamingText]);

  const sendMessage = useCallback(async () => {
    const msg = input.trim();
    if (!msg || isStreaming) return;

    setInput('');
    setIsStreaming(true);
    setStreamingText('');

    // Ajouter le message user dans l'affichage
    const userItemId = `user-${Date.now()}`;
    setItems((prev) => [...prev, { kind: 'message', role: 'user', content: msg, id: userItemId }]);

    // Préparer un emplacement pour la réponse assistant (id stable pour mise à jour)
    const assistantItemId = `assistant-${Date.now()}`;
    let accumulatedText = '';

    abortRef.current = new AbortController();

    await streamChat(
      msg,
      history,
      {
        onText: (delta) => {
          accumulatedText += delta;
          setStreamingText(accumulatedText);
        },
        onToolCall: (name, toolInput) => {
          // Ajouter un badge tool inline
          setItems((prev) => [
            ...prev,
            { kind: 'tool', name, input: toolInput, id: `tool-${Date.now()}` },
          ]);
        },
        onToolResult: (name, data) => {
          // Remonter au parent pour update du panel droit
          onToolResult(name, data);
        },
        onDone: () => {
          // Finaliser : ajouter le message assistant comme item permanent
          if (accumulatedText) {
            setItems((prev) => [
              ...prev,
              { kind: 'message', role: 'assistant', content: accumulatedText, id: assistantItemId },
            ]);
          }
          // Mettre à jour l'historique pour le prochain tour
          setHistory((prev) => [
            ...prev,
            { role: 'user', content: msg },
            { role: 'assistant', content: accumulatedText },
          ]);
          setStreamingText('');
          setIsStreaming(false);
        },
        onError: (errorMsg) => {
          setItems((prev) => [
            ...prev,
            {
              kind: 'message',
              role: 'assistant',
              content: `❌ Erreur : ${errorMsg}`,
              id: `error-${Date.now()}`,
            },
          ]);
          setStreamingText('');
          setIsStreaming(false);
        },
      },
      abortRef.current.signal,
    );
  }, [input, isStreaming, history, onToolResult]);

  // Envoyer avec Cmd/Ctrl+Entrée ou Entrée seul (Shift+Entrée = saut de ligne)
  const handleKeyDown = (e: React.KeyboardEvent<HTMLTextAreaElement>) => {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault();
      sendMessage();
    }
  };

  return (
    <div className="flex flex-col h-full">
      {/* Zone messages */}
      <div ref={messagesRef} className="flex-1 overflow-y-auto p-4 space-y-3">
        {items.length === 0 && (
          <div className="flex flex-col items-center justify-center h-full text-center text-muted-foreground">
            <p className="text-lg font-medium mb-2">👋 Bonjour !</p>
            <p className="text-sm max-w-xs">
              Posez-moi des questions sur mes projets, compétences ou expériences.
            </p>
          </div>
        )}

        {items.map((item) =>
          item.kind === 'message' ? (
            <MessageBubble key={item.id} role={item.role} content={item.content} />
          ) : (
            <ToolBadge key={item.id} toolName={item.name} input={item.input} />
          ),
        )}

        {/* Message assistant en cours de streaming */}
        {isStreaming && streamingText && (
          <MessageBubble role="assistant" content={streamingText + '▋'} />
        )}

        {/* Indicateur de chargement quand Claude réfléchit (avant le premier texte) */}
        {isStreaming && !streamingText && (
          <div className="flex justify-start">
            <div className="bg-card border rounded-2xl rounded-bl-sm px-4 py-2.5">
              <span className="text-muted-foreground text-sm animate-pulse">...</span>
            </div>
          </div>
        )}

      </div>

      {/* Zone input */}
      <div className="border-t p-4">
        <div className="flex gap-2 items-end">
          <textarea
            ref={textareaRef}
            value={input}
            onChange={(e) => setInput(e.target.value)}
            onKeyDown={handleKeyDown}
            placeholder="Posez votre question... (Entrée pour envoyer)"
            rows={2}
            disabled={isStreaming}
            className="flex-1 resize-none rounded-lg border bg-background px-3 py-2 text-sm placeholder:text-muted-foreground focus:outline-none focus:ring-2 focus:ring-ring disabled:opacity-50"
          />
          <Button
            onClick={sendMessage}
            disabled={isStreaming || !input.trim()}
            size="icon"
            className="shrink-0"
          >
            <Send className="h-4 w-4" />
          </Button>
        </div>
      </div>
    </div>
  );
}
