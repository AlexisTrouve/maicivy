'use client';

import { useRef, useState, useCallback, useEffect } from 'react';
import { useTranslations } from 'next-intl';
import { Button } from '@/components/ui/button';
import { Send, Square } from 'lucide-react';
import { streamChat, ChatMessage, ChatErrorCode, ChatErrorDetail, RateLimitInfo } from '@/lib/chat-api';
import { MessageBubble, ToolBadge } from './MessageBubble';

// Représente un item dans la liste d'affichage (message ou badge tool)
type DisplayItem =
  | { kind: 'message'; role: 'user' | 'assistant'; content: string; id: string }
  | { kind: 'tool'; name: string; input: unknown; id: string };

// TEXTAREA_MAX_HEIGHT : plafond de l'auto-resize (px) — au-delà, scroll interne plutôt que de manger
// tout l'espace du panel. ~8 lignes à text-sm.
const TEXTAREA_MAX_HEIGHT = 160;

// chatErrorMessage — traduit un ChatErrorCode dans la langue courante. Le backend ne renvoie qu'un
// code stable (jamais de texte), donc TOUTE la traduction se fait ici, comme le reste de l'UI.
function chatErrorMessage(
  t: (key: string, values?: Record<string, string | number>) => string,
  code: ChatErrorCode,
  detail?: ChatErrorDetail,
): string {
  switch (code) {
    case 'network':
      return t('errorNetwork');
    case 'rate_limited':
      return detail?.retryAfterSeconds
        ? t('errorRateLimited', { seconds: detail.retryAfterSeconds })
        : t('errorRateLimitedGeneric');
    case 'server_error':
      return t('errorServer');
    case 'context_too_long':
      return t('errorContextTooLong');
    default:
      return t('errorGeneric');
  }
}

interface ChatPanelProps {
  // Remonte les données tool_result pour mettre à jour le panel droit
  onToolResult: (toolName: string, data: unknown) => void;
  // Message déclenché depuis l'extérieur (ex: clic sur un hint)
  externalMessage?: string | null;
  // Callback appelé après envoi du message externe pour reset le state parent
  onExternalMessageSent?: () => void;
}

export function ChatPanel({ onToolResult, externalMessage, onExternalMessageSent }: ChatPanelProps) {
  const t = useTranslations('chat');
  const [items, setItems] = useState<DisplayItem[]>([]);
  const [history, setHistory] = useState<ChatMessage[]>([]);
  const [input, setInput] = useState('');
  const [isStreaming, setIsStreaming] = useState(false);
  // Accumulateur pour le texte assistant en cours de streaming
  const [streamingText, setStreamingText] = useState('');
  // Quota du budget dédié chat (X-RateLimit-* du backend) — null tant qu'aucune réponse n'est arrivée.
  const [rateLimitInfo, setRateLimitInfo] = useState<RateLimitInfo | null>(null);

  const messagesRef = useRef<HTMLDivElement>(null);
  const textareaRef = useRef<HTMLTextAreaElement>(null);
  const abortRef = useRef<AbortController | null>(null);

  // Scroll le container du chat vers le bas — sans toucher à la window
  useEffect(() => {
    const el = messagesRef.current;
    if (el) el.scrollTop = el.scrollHeight;
  }, [items, streamingText]);

  // Auto-resize du textarea : hauteur = contenu, plafonnée à TEXTAREA_MAX_HEIGHT (au-delà, scroll
  // interne). Remplace le rows=2 fixe qui rendait un message un peu long minuscule et peu lisible.
  useEffect(() => {
    const ta = textareaRef.current;
    if (!ta) return;
    ta.style.height = 'auto'; // reset avant mesure — sinon scrollHeight ne rétrécit jamais
    ta.style.height = `${Math.min(ta.scrollHeight, TEXTAREA_MAX_HEIGHT)}px`;
  }, [input]);

  // sendMessageWithText — logique d'envoi découplée de l'état input.
  // Appelée par sendMessage (via input) et par le trigger externe (hint click).
  const sendMessageWithText = useCallback(async (msg: string) => {
    if (!msg || isStreaming) return;

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
        onRateLimitInfo: setRateLimitInfo,
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
        onError: (code, detail) => {
          setItems((prev) => [
            ...prev,
            {
              kind: 'message',
              role: 'assistant',
              content: chatErrorMessage(t, code, detail),
              id: `error-${Date.now()}`,
            },
          ]);
          setStreamingText('');
          setIsStreaming(false);
        },
      },
      abortRef.current.signal,
    );
  }, [isStreaming, history, onToolResult]);

  // handleStop — interrompt la génération en cours. streamChat/chat-api.ts détecte l'AbortError et
  // finalise (onDone) le texte déjà streamé au lieu d'afficher une erreur réseau.
  const handleStop = useCallback(() => {
    abortRef.current?.abort();
  }, []);

  // sendMessage — wrapper qui lit input et le réinitialise après envoi
  const sendMessage = useCallback(() => {
    const msg = input.trim();
    if (!msg) return;
    setInput('');
    sendMessageWithText(msg);
  }, [input, sendMessageWithText]);

  // Trigger externe : hint click → envoie un message dans le chat
  useEffect(() => {
    if (externalMessage && !isStreaming) {
      sendMessageWithText(externalMessage);
      onExternalMessageSent?.();
    }
    // On ne dépend pas de isStreaming pour éviter un double envoi
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [externalMessage]);

  // Envoyer avec Entrée seul (Shift+Entrée = saut de ligne)
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
            <p className="text-lg font-medium mb-2">{t('greeting')}</p>
            <p className="text-sm max-w-xs">
              {t('greetingSubtitle')}
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

        {/* Message assistant en cours de streaming — curseur clignotant */}
        {isStreaming && streamingText && (
          <MessageBubble role="assistant" content={streamingText} isStreaming />
        )}

        {/* Indicateur de chargement — 3 points qui rebondissent en décalé */}
        {isStreaming && !streamingText && (
          <div className="flex justify-start">
            <div className="bg-card border rounded-2xl rounded-bl-sm px-4 py-3 flex items-center gap-1">
              <span className="w-1.5 h-1.5 rounded-full bg-muted-foreground/60 animate-bounce [animation-delay:0ms]" />
              <span className="w-1.5 h-1.5 rounded-full bg-muted-foreground/60 animate-bounce [animation-delay:150ms]" />
              <span className="w-1.5 h-1.5 rounded-full bg-muted-foreground/60 animate-bounce [animation-delay:300ms]" />
            </div>
          </div>
        )}

      </div>

      {/* Zone input */}
      <div className="border-t p-4">
        {/* Quota restant — évite de découvrir le mur du rate-limit sans prévenir */}
        {rateLimitInfo && (
          <div className="text-xs text-muted-foreground/60 text-right mb-1.5">
            {t('remainingQuota', { remaining: rateLimitInfo.remaining, limit: rateLimitInfo.limit })}
          </div>
        )}
        <div className="flex gap-2 items-end">
          <textarea
            ref={textareaRef}
            value={input}
            onChange={(e) => setInput(e.target.value)}
            onKeyDown={handleKeyDown}
            placeholder={t('inputPlaceholder')}
            rows={1}
            disabled={isStreaming}
            style={{ maxHeight: TEXTAREA_MAX_HEIGHT }}
            className="flex-1 resize-none overflow-y-auto rounded-lg border bg-background px-3 py-2 text-sm placeholder:text-muted-foreground focus:outline-none focus:ring-2 focus:ring-ring disabled:opacity-50"
          />
          {isStreaming ? (
            <Button
              onClick={handleStop}
              size="icon"
              variant="secondary"
              className="shrink-0"
              aria-label={t('stopGenerating')}
              title={t('stopGenerating')}
            >
              <Square className="h-4 w-4" />
            </Button>
          ) : (
            <Button
              onClick={sendMessage}
              disabled={!input.trim()}
              size="icon"
              className="shrink-0"
            >
              <Send className="h-4 w-4" />
            </Button>
          )}
        </div>
      </div>
    </div>
  );
}
