'use client';

import { useEffect, useRef, useState } from 'react';
import { useTranslations } from 'next-intl';
import { streamChat, type ChatMessage } from '@/lib/chat-api';

const API = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080';

interface ConvSummary {
  id: string;
  title: string;
  updated_at: string;
}

// Agent de chat du panneau admin. Conversations PERSISTÉES (mémoire durable, CRUD via /admin/chat),
// streaming via /chat/stream (cookie owner → Opus + tools maiProFiles : projects/skills/experiences/
// blog/search). Sidebar liste + thread + input.
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
      onError: (msg) => {
        acc += `\n[${msg}]`;
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
      <div className="flex w-56 shrink-0 flex-col rounded-lg border border-slate-800 bg-slate-900">
        <button
          onClick={() => newConv()}
          data-testid="chat-new"
          className="m-2 rounded-md bg-blue-600 px-3 py-2 text-sm font-medium hover:bg-blue-500"
        >
          + {t('chat.newConversation')}
        </button>
        <ul className="flex-1 overflow-auto px-2 pb-2 text-sm" data-testid="chat-conversations">
          {conversations.length === 0 && (
            <li className="px-2 py-2 text-xs text-slate-500">{t('chat.noConversations')}</li>
          )}
          {conversations.map((c) => (
            <li
              key={c.id}
              onClick={() => selectConv(c.id)}
              className={`group flex cursor-pointer items-center justify-between gap-1 rounded-md px-2 py-2 ${
                activeId === c.id ? 'bg-slate-800 text-white' : 'text-slate-400 hover:bg-slate-800/50'
              }`}
            >
              <span className="truncate">{c.title || '…'}</span>
              <button
                onClick={(e) => deleteConv(c.id, e)}
                className="shrink-0 text-slate-600 opacity-0 hover:text-red-400 group-hover:opacity-100"
                aria-label="delete"
              >
                ✕
              </button>
            </li>
          ))}
        </ul>
      </div>

      {/* Thread + input */}
      <div className="flex flex-1 flex-col rounded-lg border border-slate-800 bg-slate-900">
        <div className="border-b border-slate-800 p-3">
          <h1 className="text-lg font-bold">{t('chat.title')}</h1>
          <p className="text-xs text-slate-400">{t('chat.help')}</p>
        </div>

        <div className="flex-1 space-y-3 overflow-auto p-4" data-testid="chat-thread">
          {messages.length === 0 && !busy && (
            <div className="flex h-full items-center justify-center text-center text-sm text-slate-500">
              {t('chat.empty')}
            </div>
          )}
          {messages.map((m, i) => (
            <div key={i} className={`flex ${m.role === 'user' ? 'justify-end' : 'justify-start'}`}>
              <div
                className={`max-w-[80%] whitespace-pre-wrap rounded-lg px-3 py-2 text-sm ${
                  m.role === 'user' ? 'bg-blue-600' : 'bg-slate-800'
                }`}
              >
                {m.content}
              </div>
            </div>
          ))}
          {/* Réponse en cours de streaming */}
          {busy && (
            <div className="flex justify-start" data-testid="chat-streaming">
              <div className="max-w-[80%] rounded-lg bg-slate-800 px-3 py-2 text-sm">
                {tools.length > 0 && (
                  <div className="mb-1 flex flex-wrap gap-1">
                    {tools.map((name, i) => (
                      <span key={i} className="rounded bg-slate-700 px-1.5 py-0.5 text-[10px] text-slate-300">
                        🔧 {name}
                      </span>
                    ))}
                  </div>
                )}
                <span className="whitespace-pre-wrap">{streamText || <span className="text-slate-500">{t('chat.thinking')}</span>}</span>
              </div>
            </div>
          )}
          <div ref={bottomRef} />
        </div>

        <form onSubmit={send} className="flex gap-2 border-t border-slate-800 p-3">
          <input
            value={input}
            onChange={(e) => setInput(e.target.value)}
            placeholder={t('chat.placeholder')}
            disabled={busy}
            data-testid="chat-input"
            className="flex-1 rounded-md border border-slate-700 bg-slate-950 px-3 py-2 text-sm outline-none focus:border-blue-500 disabled:opacity-50"
          />
          <button
            type="submit"
            disabled={busy || !input.trim()}
            data-testid="chat-send"
            className="rounded-md bg-blue-600 px-4 py-2 text-sm font-medium hover:bg-blue-500 disabled:opacity-50"
          >
            {t('chat.send')}
          </button>
        </form>
      </div>
    </div>
  );
}
