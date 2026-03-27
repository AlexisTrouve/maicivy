// chat-api.ts — Client SSE pour l'endpoint POST /api/v1/chat/stream
// Utilise fetch + ReadableStream (pas EventSource) car on a besoin d'envoyer un POST body.

const API_BASE = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080';

export interface ChatMessage {
  role: 'user' | 'assistant';
  content: string;
}

export type ChatEventType = 'text' | 'tool_call' | 'tool_result' | 'done' | 'error';

export interface ChatEvent {
  type: ChatEventType;
  delta?: string;       // pour type="text"
  name?: string;        // pour tool_call / tool_result
  input?: unknown;      // pour tool_call
  data?: unknown;       // pour tool_result
  message?: string;     // pour error
}

interface StreamChatCallbacks {
  onText: (delta: string) => void;
  onToolCall: (name: string, input: unknown) => void;
  onToolResult: (name: string, data: unknown) => void;
  onDone: () => void;
  onError: (msg: string) => void;
}

/**
 * Envoie un message au backend et stream les events SSE via callbacks.
 * Gère le parsing des lignes "data: {...}\n\n" manuellement.
 */
export async function streamChat(
  message: string,
  history: ChatMessage[],
  callbacks: StreamChatCallbacks,
  signal?: AbortSignal,
): Promise<void> {
  let response: Response;

  try {
    response = await fetch(`${API_BASE}/api/v1/chat/stream`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ message, history }),
      signal,
    });
  } catch (err) {
    callbacks.onError(err instanceof Error ? err.message : 'Network error');
    return;
  }

  if (!response.ok) {
    callbacks.onError(`HTTP ${response.status}: ${response.statusText}`);
    return;
  }

  if (!response.body) {
    callbacks.onError('No response body');
    return;
  }

  const reader = response.body.getReader();
  const decoder = new TextDecoder();
  let buffer = '';

  try {
    while (true) {
      const { done, value } = await reader.read();
      if (done) break;

      buffer += decoder.decode(value, { stream: true });

      // Parser les lignes SSE : chaque event est "data: {...}\n\n"
      const lines = buffer.split('\n');
      buffer = lines.pop() ?? ''; // garder la dernière ligne incomplète en buffer

      for (const line of lines) {
        const trimmed = line.trim();
        if (!trimmed.startsWith('data: ')) continue;

        const raw = trimmed.slice(6); // enlever "data: "
        if (!raw) continue;

        let event: ChatEvent;
        try {
          event = JSON.parse(raw);
        } catch {
          continue; // ligne malformée — ignorer
        }

        switch (event.type) {
          case 'text':
            if (event.delta) callbacks.onText(event.delta);
            break;
          case 'tool_call':
            if (event.name) callbacks.onToolCall(event.name, event.input);
            break;
          case 'tool_result':
            if (event.name) callbacks.onToolResult(event.name, event.data);
            break;
          case 'done':
            callbacks.onDone();
            return;
          case 'error':
            callbacks.onError(event.message ?? 'Unknown error');
            return;
        }
      }
    }
  } finally {
    reader.releaseLock();
  }

  // Stream terminé sans event "done" (cas rare)
  callbacks.onDone();
}
