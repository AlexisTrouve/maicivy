import { readFileSync } from 'fs';
import http from 'http';
import type { AddressInfo } from 'net';
import path from 'path';

import type { Page, Route } from '@playwright/test';

// ============================================================================
// Helpers E2E du dossier vivant — partagés par la version PC et la version téléphone.
// ----------------------------------------------------------------------------
// QUOI : simuler AU NIVEAU RÉSEAU le flux SSE du run (route publique), pour que la page, sa
// SseSource et tout l'écran tournent pour de vrai ; et rejouer le flux BRUT d'un vrai run,
// enregistré (fixtures/dossier-vivant-schema-run.ndjson). Le run de bout en bout contre le
// backend du banc (agents simulés par sa fausse passerelle) est dans dossier-vivant.spec.ts.
// ============================================================================

/** CORS d'un appel du navigateur vers l'API (autre port en local) avec cookies. */
export function cors(route: Route): Record<string, string> {
  return {
    'Access-Control-Allow-Origin': route.request().headers()['origin'] ?? '*',
    'Access-Control-Allow-Credentials': 'true',
    'Access-Control-Allow-Headers': 'content-type',
    'Access-Control-Allow-Methods': 'POST, GET',
  };
}

/** Des événements du contrat, mis en trames SSE (`event:` + `data:`), tels que le backend les émet. */
export const sse = (events: object[]) => events.map((e) => `event: ${(e as { type: string }).type}\ndata: ${JSON.stringify(e)}\n\n`).join('');

/**
 * Le flux d'un VRAI run (6 agents, Dessinateur compris — « Cabinet Lumen »), tel que le navigateur
 * l'a reçu : une ligne `{ receivedMs, event }` par trame. On n'en garde que les événements.
 */
export function recordedRun(): object[] {
  const file = path.join(__dirname, '../fixtures/dossier-vivant-schema-run.ndjson');
  return readFileSync(file, 'utf-8')
    .trim()
    .split('\n')
    .map((l) => (JSON.parse(l) as { event: object }).event);
}

/**
 * La route PUBLIQUE du run répond `events` en SSE — d'un seul bloc, donc tous reçus au même
 * instant : le cas où des messages partagent leur horodatage (cf. runState › MessageView.id).
 * Aucun cookie : le run est ouvert à tout visiteur.
 */
export async function mockLiveRun(page: Page, events: object[]): Promise<void> {
  await page.route('**/api/v1/dossier-vivant/run', (route) => {
    if (route.request().method() === 'OPTIONS') return route.fulfill({ status: 204, headers: cors(route) });
    return route.fulfill({ status: 200, headers: cors(route), contentType: 'text/event-stream', body: sse(events) });
  });
}

/**
 * La route PUBLIQUE du run, servie EN CONTINU par un petit serveur SSE local : les événements
 * partent un par un (`gapMs` d'écart) et le DERNIER (la remise) attend `release()`.
 * POURQUOI : `route.fulfill` livre un corps d'un bloc — tout le run arrive au même instant et il est
 * « remis » aussitôt ; impossible alors d'agir PENDANT le run. Ici la requête du navigateur est
 * réellement envoyée (`route.continue` vers le serveur local) et son corps arrive au fil de l'eau,
 * comme depuis le backend : le test tient le run ouvert aussi longtemps qu'il le veut.
 */
export async function streamLiveRun(page: Page, events: object[], gapMs = 30): Promise<{ release: () => void; close: () => Promise<void> }> {
  let release!: () => void;
  const released = new Promise<void>((r) => (release = r));
  const server = http.createServer(async (req, res) => {
    res.writeHead(200, {
      'Content-Type': 'text/event-stream',
      'Cache-Control': 'no-cache',
      'Access-Control-Allow-Origin': req.headers.origin ?? '*',
      'Access-Control-Allow-Credentials': 'true',
    });
    for (let i = 0; i < events.length; i++) {
      if (i === events.length - 1) await released;
      res.write(sse([events[i]]));
      await new Promise((r) => setTimeout(r, gapMs));
    }
    res.end();
  });
  await new Promise<void>((r) => server.listen(0, '127.0.0.1', r));
  const { port } = server.address() as AddressInfo;
  await page.route('**/api/v1/dossier-vivant/run', (route) => {
    if (route.request().method() === 'OPTIONS') return route.fulfill({ status: 204, headers: cors(route) });
    return route.continue({ url: `http://127.0.0.1:${port}/api/v1/dossier-vivant/run` });
  });
  return {
    release,
    close: () =>
      new Promise<void>((r) => {
        release();
        server.closeAllConnections();
        server.close(() => r());
      }),
  };
}
