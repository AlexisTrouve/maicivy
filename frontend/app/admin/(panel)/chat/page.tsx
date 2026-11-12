'use client';

import { useEffect, useRef, useState } from 'react';
import { useTranslations } from 'next-intl';
import { streamChat, type ChatMessage } from '@/lib/chat-api';
import { Panel, btnPrimary, field } from '@/components/admin/ui';

const API = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080';

interface ConvSummary {
  id: string;
  title: string;
  updated_at: string;
}

// Agent de chat du panneau admin. Conversations PERSISTÉES (mémoire durable, CRUD via /admin/chat),
// streaming via /chat/stream (cookie owner → Opus + tools maiProFiles : projects/skills/experiences/
// blog/search). Sidebar liste + thread + input.
// Ouvert aussi aux prospecteurs : le backend cloisonne leurs conversations et leur donne le modèle
// visiteur + le toolset public (cf. ChatHandler.turnPolicy). L'écran est identique pour les deux rôles.
export default function AdminChatTool() {
  const t = useTranslations('admin');
  const [conversations, setConversations] = useState<ConvSummary[]>([]);
  const [activeId, setActiveId] = useState<string | null>(null);
  const [messages, setMessages] = useState<ChatMessage[]>([]);
  const [input, setInput] = useState('');
  const [streamText, setStreamText] = useState('');
  const [tools, setTools] = useState<string[]>([]);
  const [busy, setBusy] = useState(false);
  const bottomRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    loadConversations();
  }, []);
  useEffect(() => {
    // ?.scrollIntoView?.() : no-op si l'API n'existe pas (jsdom en test), marche en browser.
    bottomRef.current?.scrollIntoView?.({ behavior: 'smooth' });
  }, [messages, streamText]);

  async function loadConversations() {
    const res = await fetch(`${API}/api/v1/admin/chat/conversations`, { credentials: 'include' });
    if (!res.ok) return;
    const data = await res.json();
    setConversations(data.conversations || []);
  }

  async function selectConv(id: string) {
    setActiveId(id);
    setStreamText('');
    setTools([]);
    const res = await fetch(`${API}/api/v1/admin/chat/conversations/${id}`, { credentials: 'include' });
    if (!res.ok) {
      setMessages([]);
      return;
    }
    const data = await res.json();
    setMessages(data.messages || []);
  }

  async function newConv(): Promise<string | null> {
    const res = await fetch(`${API}/api/v1/admin/chat/conversations`, { method: 'POST', credentials: 'include' });
    if (!res.ok) return null;
    const { id } = await res.json();
    await loadConversations();
    setActiveId(id);
    setMessages([]);
    setStreamText('');
    setTools([]);
    return id;
  }

  async function deleteConv(id: string, e: React.MouseEvent) {
    e.stopPropagation();
    await fetch(`${API}/api/v1/admin/chat/conversations/${id}`, { method: 'DELETE', credentials: 'include' });
    if (activeId === id) {
      setActiveId(null);
      setMessages([]);
    }
    loadConversations();
  }

  // Persiste la conversation (titre auto depuis le 1er message user).
  async function save(id: string, msgs: ChatMessage[]) {
    const firstUser = msgs.find((m) => m.role === 'user');
    await fetch(`${API}/api/v1/admin/chat/conversations/${id}`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      credentials: 'include',
      body: JSON.stringify({ title: firstUser ? firstUser.content.slice(0, 60) : '', messages: msgs }),
    });
    loadConversations();
  }

  async function send(e: React.FormEvent) {
    e.preventDefault();
    if (busy || !input.trim()) return;
    let id = activeId;
    if (!id) {
      id = await newConv();
      if (!id) return;
    }
    const userMsg: ChatMessage = { role: 'user', content: input.trim() };
    const history = messages;
    const withUser = [...messages, userMsg];
    setMessages(withUser);
    setInput('');
    setBusy(true);
    setStreamText('');
    setTools([]);

    let acc = '';
    await streamChat(userMsg.content, history, {
      onText: (d) => {
        acc += d;
        setStreamText(acc);
      },
      onToolCall: (name) => setTools((ts) => [...ts, name]),
      onToolResult: () => {},
      onDone: () => {},
      onError: (code, detail) => {
        // Outil interne (owner) — pas de traduction, juste le code brut pour debug.
        acc += `\n[${code}${detail?.retryAfterSeconds ? `:${detail.retryAfterSeconds}s` : ''}]`;
        setStreamText(acc);
      },
    });

    const finalMessages: ChatMessage[] = [...withUser, { role: 'assistant', content: acc }];
    setMessages(finalMessages);
    setStreamText('');
    setTools([]);
    setBusy(false);
    await save(id, finalMessages);
  }

  return (
    <div className="flex h-[calc(100vh-4rem)] gap-4" data-testid="admin-chat-tool">
      {/* Liste des conversations */}
      <Panel className="flex w-60 shrink-0 flex-col overflow-hidden">
        <div className="p-2">
          <button onClick={() => newConv()} data-testid="chat-new" className={`${btnPrimary} w-full`}>
            + {t('chat.newConversation')}
          </button>
        </div>
        <ul className="flex-1 overflow-auto px-2 pb-2 text-sm" data-testid="chat-conversations">
          {conversations.length === 0 && (
            <li className="px-2 py-2 text-xs text-slate-500">{t('chat.noConversations')}</li>
          )}
          {conversations.map((c) => (
            <li
              key={c.id}
              onClick={() => selectConv(c.id)}
              // Barre dorée sur la conversation ouverte — même code que la navigation de la sidebar :
              // « où je suis » se signale partout de la même façon, sinon il faut l'apprendre écran
              // par écran.
              className={`group relative flex cursor-pointer items-center justify-between gap-1 rounded-lg py-2 pl-3 pr-2 transition-colors ${
                activeId === c.id ? 'bg-slate-800/60 text-slate-50' : 'text-slate-400 hover:bg-slate-800/40 hover:text-slate-200'
              }`}
            >
              {activeId === c.id && (
                <span aria-hidden="true" className="absolute inset-y-1.5 left-0 w-[3px] rounded-full bg-gold" />
              )}
              <span className="truncate">{c.title || '…'}</span>
              <button
                onClick={(e) => deleteConv(c.id, e)}
                className="shrink-0 text-slate-600 opacity-0 transition hover:text-rose-400 focus-visible:opacity-100 group-hover:opacity-100"
                aria-label={t('chat.deleteConversation')}
              >
                ✕
              </button>
            </li>
          ))}
        </ul>
      </Panel>

      {/* Thread + input */}
      <Panel hairline className="flex min-w-0 flex-1 flex-col overflow-hidden">
        <div className="flex items-baseline gap-3 border-b border-slate-800/80 px-4 py-3">
          <h1 className="font-heading text-base font-semibold text-slate-100">{t('chat.title')}</h1>
          <p className="truncate text-xs text-slate-500">{t('chat.help')}</p>
        </div>

        <div className="flex-1 space-y-3 overflow-auto p-4" data-testid="chat-thread">
          {messages.length === 0 && !busy && (
            <div className="flex h-full items-center justify-center text-center text-sm text-slate-500">
              {t('chat.empty')}
            </div>
          )}
          {messages.map((m, i) => (
            <div key={i} className={`flex ${m.role === 'user' ? 'justify-end' : 'justify-start'}`}>
              {/* Bleu pour ce que dit l'humain, ardoise pour ce que répond la machine — le même
                  code que l'écran compose. Deux conventions différentes entre deux chats du même
                  panneau obligeraient à réapprendre qui parle à chaque changement d'outil. */}
              <div
                className={`max-w-[80%] whitespace-pre-wrap rounded-xl px-3.5 py-2.5 text-sm leading-relaxed ${
                  m.role === 'user'
                    ? 'border border-sky-800/50 bg-sky-900/40 text-sky-50'
                    : 'border border-slate-800 bg-slate-800/60 text-slate-200'
                }`}
              >
                {m.content}
              </div>
            </div>
          ))}
          {/* Réponse en cours de streaming */}
          {busy && (
            <div className="flex justify-start" data-testid="chat-streaming">
              <div className="max-w-[80%] rounded-xl border border-slate-800 bg-slate-800/60 px-3.5 py-2.5 text-sm leading-relaxed text-slate-200">
                {tools.length > 0 && (
                  <div className="mb-1.5 flex flex-wrap gap-1">
                    {tools.map((name, i) => (
                      <span
                        key={i}
                        className="rounded-md border border-sky-900/60 bg-sky-950/40 px-1.5 py-0.5 font-mono text-[10px] text-sky-300"
                      >
                        {name}
                      </span>
                    ))}
                  </div>
                )}
                <span className="whitespace-pre-wrap">
                  {streamText || <span className="animate-pulse text-slate-500">{t('chat.thinking')}</span>}
                </span>
              </div>
            </div>
          )}
          <div ref={bottomRef} />
        </div>

        <form onSubmit={send} className="flex gap-2 border-t border-slate-800/80 p-3">
          <input
            value={input}
            onChange={(e) => setInput(e.target.value)}
            placeholder={t('chat.placeholder')}
            disabled={busy}
            data-testid="chat-input"
            className={`${field} flex-1 disabled:opacity-50`}
          />
          <button type="submit" disabled={busy || !input.trim()} data-testid="chat-send" className={btnPrimary}>
            {t('chat.send')}
          </button>
        </form>
      </Panel>
    </div>
  );
}
