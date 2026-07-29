import { streamChat } from '../chat-api';

// Avant ce fix, cliquer "stop" (AbortController.abort()) pendant un streaming faisait remonter
// l'AbortError jusqu'au catch générique → onError('network') affichait "connexion perdue" alors que
// l'utilisateur avait VOLONTAIREMENT arrêté la génération. Ces tests verrouillent le bon aiguillage :
// un abort finalise (onDone) le texte déjà streamé, une vraie coupure réseau reste une erreur.

function makeCallbacks() {
  return {
    onText: jest.fn(),
    onToolCall: jest.fn(),
    onToolResult: jest.fn(),
    onDone: jest.fn(),
    onError: jest.fn(),
  };
}

describe('streamChat — gestion de l\'abort (bouton stop)', () => {
  afterEach(() => {
    jest.restoreAllMocks();
  });

  it('fetch() rejeté par un abort → onDone (PAS onError)', async () => {
    const abortErr = new DOMException('The user aborted a request.', 'AbortError');
    jest.spyOn(global, 'fetch').mockRejectedValue(abortErr);

    const callbacks = makeCallbacks();
    await streamChat('salut', [], callbacks);

    expect(callbacks.onDone).toHaveBeenCalledTimes(1);
    expect(callbacks.onError).not.toHaveBeenCalled();
  });

  it('une vraie coupure réseau (pas un abort) → onError(\'network\'), pas onDone', async () => {
    jest.spyOn(global, 'fetch').mockRejectedValue(new TypeError('Failed to fetch'));

    const callbacks = makeCallbacks();
    await streamChat('salut', [], callbacks);

    expect(callbacks.onError).toHaveBeenCalledWith('network');
    expect(callbacks.onDone).not.toHaveBeenCalled();
  });

  it('reader.read() rejeté par un abort EN COURS DE STREAM → onDone (le texte déjà reçu reste acquis)', async () => {
    const abortErr = new DOMException('The user aborted a request.', 'AbortError');
    let readCalls = 0;
    const encoder = new TextEncoder();

    jest.spyOn(global, 'fetch').mockResolvedValue({
      ok: true,
      headers: new Headers(),
      body: {
        getReader: () => ({
          read: () => {
            readCalls += 1;
            if (readCalls === 1) {
              return Promise.resolve({
                done: false,
                value: encoder.encode(`data: ${JSON.stringify({ type: 'text', delta: 'Bonj' })}\n\n`),
              });
            }
            return Promise.reject(abortErr); // abort déclenché pendant le 2e read
          },
          releaseLock: () => {},
        }),
      },
    } as unknown as Response);

    const callbacks = makeCallbacks();
    await streamChat('salut', [], callbacks);

    expect(callbacks.onText).toHaveBeenCalledWith('Bonj'); // le delta déjà reçu n'est pas perdu
    expect(callbacks.onDone).toHaveBeenCalledTimes(1);
    expect(callbacks.onError).not.toHaveBeenCalled();
  });
});

describe('streamChat — quota du budget dédié chat (X-RateLimit-*)', () => {
  afterEach(() => {
    jest.restoreAllMocks();
  });

  function sseBody(): Response['body'] {
    const encoder = new TextEncoder();
    let done = false;
    return {
      getReader: () => ({
        read: () => {
          if (done) return Promise.resolve({ done: true, value: undefined });
          done = true;
          return Promise.resolve({
            done: false,
            value: encoder.encode(`data: ${JSON.stringify({ type: 'done' })}\n\n`),
          });
        },
        releaseLock: () => {},
      }),
    } as unknown as Response['body'];
  }

  it('lit X-RateLimit-Limit/Remaining sur une réponse 200 et les remonte via onRateLimitInfo', async () => {
    jest.spyOn(global, 'fetch').mockResolvedValue({
      ok: true,
      headers: new Headers({ 'X-RateLimit-Limit': '20', 'X-RateLimit-Remaining': '17' }),
      body: sseBody(),
    } as unknown as Response);

    const callbacks = { ...makeCallbacks(), onRateLimitInfo: jest.fn() };
    await streamChat('salut', [], callbacks);

    expect(callbacks.onRateLimitInfo).toHaveBeenCalledWith({ limit: 20, remaining: 17 });
  });

  it('lit aussi le quota sur un 429 (limite journalière atteinte, remaining=0)', async () => {
    jest.spyOn(global, 'fetch').mockResolvedValue({
      ok: false,
      status: 429,
      headers: new Headers({ 'X-RateLimit-Limit': '20', 'X-RateLimit-Remaining': '0' }),
      json: async () => ({ retry_after: 3600 }),
    } as unknown as Response);

    const callbacks = { ...makeCallbacks(), onRateLimitInfo: jest.fn() };
    await streamChat('salut', [], callbacks);

    expect(callbacks.onRateLimitInfo).toHaveBeenCalledWith({ limit: 20, remaining: 0 });
    expect(callbacks.onError).toHaveBeenCalledWith('rate_limited', { retryAfterSeconds: 3600 });
  });

  it('un 429 SANS ces headers (ex: cooldown) ne casse rien et n\'appelle pas onRateLimitInfo', async () => {
    jest.spyOn(global, 'fetch').mockResolvedValue({
      ok: false,
      status: 429,
      headers: new Headers(),
      json: async () => ({}),
    } as unknown as Response);

    const callbacks = { ...makeCallbacks(), onRateLimitInfo: jest.fn() };
    await streamChat('salut', [], callbacks);

    expect(callbacks.onRateLimitInfo).not.toHaveBeenCalled();
  });
});
