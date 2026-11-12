import { ReadableStream } from 'stream/web';
import { TextDecoder, TextEncoder } from 'util';

import type { AgentEvent, StampedEvent } from '../events';
import { SseParser, SseSource } from '../sseSource';

/**
 * SseSource — le parsing du flux SSE du backend et l'horodatage À LA RÉCEPTION.
 * Le flux est servi par une fausse réponse fetch dont le test pousse les paquets un à un, en
 * avançant l'horloge du run entre deux : on vérifie ainsi que chaque événement prend l'instant
 * où sa trame s'est COMPLÉTÉE, pas celui où elle a commencé, ni une heure du serveur.
 */

// jsdom n'expose ni TextDecoder ni TextEncoder : ceux de Node, comme dans un navigateur.
Object.assign(globalThis, { TextDecoder, TextEncoder });

const frame = (e: AgentEvent) => `event: ${e.type}\ndata: ${JSON.stringify(e)}\n\n`;
const started: AgentEvent = {
  type: 'run.started',
  visitor: { name: 'Inès Morel', affiliation: 'CTO chez Tessera', organisation: 'Tessera', initials: 'IM', need: 'Trier 1 400 tickets.' },
  agents: [{ id: 'orchestrator', model: 'claude-haiku-5-5' }],
};
const usage: AgentEvent = { type: 'usage', agent: 'reader', inputTokens: 900, outputTokens: 120, costUsd: 0.00015 };

/** Une réponse fetch dont le corps est alimenté à la main. */
function controlledFetch(status = 200) {
  let controller!: ReadableStreamDefaultController<Uint8Array>;
  const body = new ReadableStream<Uint8Array>({ start: (c) => void (controller = c) });
  const calls: { url: string; init: RequestInit }[] = [];
  const fetchImpl = (async (url: string, init: RequestInit) => {
    calls.push({ url, init });
    // Le strict nécessaire d'une Response pour SseSource : statut et corps en flux.
    return { ok: status >= 200 && status < 300, status, body: status === 200 ? body : null };
  }) as unknown as typeof fetch;
  const enc = new TextEncoder();
  return {
    fetchImpl,
    calls,
    push: (s: string) => controller.enqueue(enc.encode(s)),
    pushBytes: (b: Uint8Array) => controller.enqueue(b),
    end: () => controller.close(),
  };
}

const flush = async () => {
  for (let i = 0; i < 5; i++) await new Promise((r) => setTimeout(r, 0));
};

function openSource(status = 200) {
  const clock = { t: 0, now() { return this.t; } };
  const net = controlledFetch(status);
  const got: StampedEvent[] = [];
  const stop = new SseSource('http://api/run', { need: 'besoin', who: 'Inès Morel, CTO chez Tessera', lang: 'fr' }, clock, net.fetchImpl).open((b) => got.push(...b));
  return { clock, net, got, stop };
}

describe('SseParser', () => {
  it('rend les trames complètes, quelle que soit la découpe des paquets', () => {
    const p = new SseParser();
    const text = frame(started) + frame(usage);
    const out: AgentEvent[] = [];
    // Découpe à chaque caractère : le pire cas réseau.
    for (const ch of text) out.push(...p.feed(ch));
    expect(out).toEqual([started, usage]);
  });

  it('accepte les fins de ligne \\r\\n, ignore commentaires, trames illisibles et types hors contrat', () => {
    const p = new SseParser();
    const out = p.feed(
      ': battement\r\n\r\n' +
        'event: usage\r\ndata: {pas du json}\r\n\r\n' +
        'data: {"type":"inconnu"}\r\n\r\n' +
        `event: usage\r\ndata: ${JSON.stringify(usage)}\r` + // \r coupé de son \n
        '\n\r\n',
    );
    expect(out).toEqual([usage]);
  });
});

describe('SseSource', () => {
  it('envoie le besoin en POST, sans exiger de session (route publique)', async () => {
    const { net } = openSource();
    await flush();
    expect(net.calls).toHaveLength(1);
    expect(net.calls[0].init.method).toBe('POST');
    expect(net.calls[0].init.credentials).toBeUndefined();
    expect(JSON.parse(net.calls[0].init.body as string)).toEqual({ need: 'besoin', who: 'Inès Morel, CTO chez Tessera', lang: 'fr' });
  });

  it('horodate chaque événement à la réception de la fin de sa trame, sur l’horloge du run', async () => {
    const { clock, net, got } = openSource();
    await flush();

    clock.t = 0.4;
    net.push(frame(started) + frame(usage).slice(0, 20)); // usage arrive à moitié
    await flush();
    expect(got).toEqual([{ at: 0.4, event: started }]);

    clock.t = 2.5;
    net.push(frame(usage).slice(20)); // … et se complète plus tard : c'est CET instant qui compte
    await flush();
    expect(got[1]).toEqual({ at: 2.5, event: usage });

    // Un caractère multi-octets coupé entre deux paquets n'est pas abîmé.
    const msg: AgentEvent = { type: 'message', from: 'orchestrator', to: 'writer', text: 'l’architecture est prête — 架构' };
    const bytes = new TextEncoder().encode(frame(msg));
    const mid = bytes.indexOf(0xe2) + 1; // au milieu du « ’ »
    clock.t = 3;
    net.pushBytes(bytes.slice(0, mid));
    net.pushBytes(bytes.slice(mid));
    net.push(frame({ type: 'run.completed' }));
    net.end();
    await flush();
    expect(got.slice(2)).toEqual([
      { at: 3, event: msg },
      { at: 3, event: { type: 'run.completed' } },
    ]);
  });

  it('flux coupé avant la remise → run.failed « network »', async () => {
    const { clock, net, got } = openSource();
    await flush();
    net.push(frame(started));
    clock.t = 7;
    net.end();
    await flush();
    expect(got.at(-1)).toEqual({ at: 7, event: { type: 'run.failed', reason: 'network' } });
  });

  it('un arrêt nommé par le serveur reste le seul arrêt', async () => {
    const { net, got } = openSource();
    await flush();
    net.push(frame(started) + frame({ type: 'run.failed', reason: 'model' }));
    net.end();
    await flush();
    expect(got.map((s) => s.event)).toEqual([started, { type: 'run.failed', reason: 'model' }]);
  });

  it.each(['budget', 'timeout', 'inconnue'])('raison hors contrat « %s » → arrêt « upstream », jamais sans libellé', async (reason) => {
    const { net, got } = openSource();
    await flush();
    net.push(frame(started) + `event: run.failed\ndata: ${JSON.stringify({ type: 'run.failed', reason })}\n\n`);
    net.end();
    await flush();
    expect(got.map((s) => s.event)).toEqual([started, { type: 'run.failed', reason: 'upstream' }]);
  });

  it.each([
    [400, 'invalid'],
    [429, 'upstream'],
    [502, 'upstream'],
  ])('refus HTTP %i → run.failed « %s »', async (status, reason) => {
    const { got } = openSource(status);
    await flush();
    expect(got).toEqual([{ at: 0, event: { type: 'run.failed', reason } }]);
  });

  it('arrêtée par l’écran (stop), elle ne signale rien', async () => {
    const { net, got, stop } = openSource();
    await flush();
    stop();
    net.end();
    await flush();
    expect(got).toEqual([]);
  });
});
