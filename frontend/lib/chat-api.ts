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
  code?: string;        // pour type="error" — identifiant stable ("generic" | "context_too_long")
}

// ChatErrorCode : catégories d'erreur distinguées côté UI (chaque appelant de streamChat traduit
// dans la langue du visiteur — jamais de texte en dur ici, cf. règle i18n du projet).
// - 'network' : fetch a échoué avant même d'atteindre le serveur (coupure, DNS, CORS...).
// - 'rate_limited' : 429 du middleware AIRateLimit (cooldown ou limite journalière — potentiellement
//   déclenchée par une génération CV/lettre PRÉCÉDENTE de la même session, pas forcément par le chat).
// - 'server_error' : réponse HTTP non-2xx autre que 429 (5xx, etc.).
// - 'context_too_long' : erreur SSE mi-stream, conversation trop longue pour le modèle/proxy.
// - 'generic' : toute autre erreur SSE mi-stream, ou corps de réponse absent.
export type ChatErrorCode = 'network' | 'rate_limited' | 'server_error' | 'context_too_long' | 'generic';

export interface ChatErrorDetail {
  retryAfterSeconds?: number; // renseigné pour 'rate_limited' quand le serveur l'a fourni
}

// RateLimitInfo — quota du budget dédié chat (cf. chatRateLimitMW backend, 20/jour par défaut),
// lu depuis les headers X-RateLimit-Limit/X-RateLimit-Remaining déjà renvoyés par le middleware
// (jusque-là jamais exploités côté frontend — l'utilisateur découvrait le mur sans prévenir).
export interface RateLimitInfo {
  limit: number;
  remaining: number;
}

interface StreamChatCallbacks {
  onText: (delta: string) => void;
  onToolCall: (name: string, input: unknown) => void;
  onToolResult: (name: string, data: unknown) => void;
  onDone: () => void;
  onError: (code: ChatErrorCode, detail?: ChatErrorDetail) => void;
  // Optionnel : pas toutes les réponses ne portent ces headers (ex: 429 cooldown/in-flight, qui ne
  // remplissent pas X-RateLimit-* — seuls la limite journalière et le succès le font).
  onRateLimitInfo?: (info: RateLimitInfo) => void;
}

// readRateLimitInfo extrait le quota des headers, si présents et cohérents. Headers.get() est
// insensible à la casse (spec Fetch) — peu importe que le serveur envoie "X-Ratelimit-Limit" (Go
// canonicalise ainsi) ou "X-RateLimit-Limit".
function readRateLimitInfo(response: Response): RateLimitInfo | undefined {
  const limit = Number(response.headers.get('X-RateLimit-Limit'));
  const remaining = Number(response.headers.get('X-RateLimit-Remaining'));
  if (!Number.isFinite(limit) || limit <= 0 || !Number.isFinite(remaining) || remaining < 0) {
    return undefined;
  }
  return { limit, remaining };
}

// isAbortError — vrai si l'exception vient d'un AbortController.abort() (bouton "stop" utilisateur),
// PAS d'une vraie coupure réseau. Sur abort volontaire on finalise (onDone) le texte déjà streamé
// au lieu d'afficher un message d'erreur — l'utilisateur sait qu'il a arrêté, ce n'est pas une panne.
function isAbortError(err: unknown): boolean {
  return err instanceof DOMException && err.name === 'AbortError';
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
      // include : envoie les cookies (session visiteur + cookie admin owner → tier Opus côté backend).
      credentials: 'include',
      body: JSON.stringify({ message, history }),
      signal,
    });
  } catch (err) {
    if (isAbortError(err)) {
      callbacks.onDone();
      return;
    }
    callbacks.onError('network');
    return;
  }

  if (!response.ok) {
    if (response.status === 429) {
      const rateLimitInfo = readRateLimitInfo(response);
      if (rateLimitInfo) callbacks.onRateLimitInfo?.(rateLimitInfo);
      // Corps JSON du middleware AIRateLimit : { retry_after: <secondes>, ... } — best-effort, un
      // corps non-JSON/absent ne doit pas faire planter le parsing, juste omettre le détail.
      let retryAfterSeconds: number | undefined;
      try {
        const body = await response.json();
        if (typeof body?.retry_after === 'number') retryAfterSeconds = body.retry_after;
      } catch {
        /* corps non-JSON — pas de détail, le message générique de rate-limit suffit */
      }
      callbacks.onError('rate_limited', { retryAfterSeconds });
    } else {
      callbacks.onError('server_error');
    }
    return;
  }

  const rateLimitInfo = readRateLimitInfo(response);
  if (rateLimitInfo) callbacks.onRateLimitInfo?.(rateLimitInfo);

  if (!response.body) {
    callbacks.onError('generic');
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
            callbacks.onError(event.code === 'context_too_long' ? 'context_too_long' : 'generic');
            return;
        }
      }
    }
  } catch (err) {
    if (isAbortError(err)) {
      callbacks.onDone();
    } else {
      callbacks.onError('generic');
    }
    return;
  } finally {
    reader.releaseLock();
  }

  // Stream terminé sans event "done" (cas rare)
  callbacks.onDone();
}
