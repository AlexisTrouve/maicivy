/**
 * Les COPIES SERVEUR de ce que le front sait faire — verrouillées ici, relues par le backend Go.
 *
 * QUOI : le PDF public du dossier n'imprime que du contenu produit par le serveur (le navigateur
 * n'envoie plus qu'un identifiant et une langue). Deux choses que le front définit doivent donc
 * exister côté serveur, à l'identique :
 *  1. le dossier du rejeu Tessera (tessera.ts) → backend/internal/services/dossier_tessera.json ;
 *  2. la mise en page imprimée du schéma (pdfExport.ts › printDiagram) → portée en Go
 *     (dossier_print_layout.go), comparée sur la golden testdata/dossier_print_diagrams.json.
 * POURQUOI ici : c'est le front qui fait foi. Modifier le scénario ou la mise en page sans
 * régénérer les copies fait échouer CE test ; le test Go relit les mêmes fichiers.
 * COMMENT régénérer : `DV_WRITE_SERVER_COPIES=1 npx jest serverCopies`.
 */
import { existsSync, readFileSync, writeFileSync } from 'fs';
import path from 'path';

import { SECTION_IDS, type PipelineEdge, type PipelineNode } from '../events';
import { printDiagram } from '../pdfExport';
import { elapsedAt, foldEvents, totalsAt } from '../runState';
import { tesseraScenario } from '../tessera';
import { LIVE_PIPELINE, SCHEMA_PIPELINE, tesseraPipeline } from '../__fixtures__/pipelines';

const SERVICES = path.join(__dirname, '../../../../backend/internal/services');
const WRITE = process.env.DV_WRITE_SERVER_COPIES === '1';

/** Jetons remplacés par le serveur au moment d'imprimer (dossier_tessera.go). */
const LOCALE_TOKEN = '{locale}';
const VIDEO_TOKEN = '{melodicode-video}';

function lock(file: string, value: unknown) {
  const text = JSON.stringify(value, null, 1) + '\n';
  if (WRITE) writeFileSync(file, text);
  expect(existsSync(file)).toBe(true);
  expect(JSON.parse(readFileSync(file, 'utf-8'))).toEqual(JSON.parse(text));
}

describe('copies serveur', () => {
  it('le dossier du rejeu Tessera, replié comme à l’écran', () => {
    const s = tesseraScenario({ projectCount: 0, siteUrl: `/${LOCALE_TOKEN}`, melodicodeVideoUrl: VIDEO_TOKEN });
    const state = foldEvents(s.events);
    const end = Number.MAX_SAFE_INTEGER;
    lock(path.join(SERVICES, 'dossier_tessera.json'), {
      visitor: { name: s.visitor.name, affiliation: s.visitor.affiliation, organisation: s.visitor.organisation },
      agents: state.team.length + 1,
      elapsedSeconds: elapsedAt(state, end),
      costUsd: totalsAt(state, end).costUsd,
      sections: SECTION_IDS.map((id) => ({ id, title: state.dossier!.sections[id].title, blocks: state.dossier!.sections[id].blocks.map((b) => b.block) })),
    });
  });

  it('la mise en page imprimée du schéma (golden partagée avec le port Go)', () => {
    const node = (id: string, col: number, row: number, title: string, subtitle: string, human = false): PipelineNode => ({ id, title, subtitle, col, row, ...(human ? { human } : {}) });
    // Une chaîne longue (7 étages) avec un retour : bascule à la verticale, couloir de retour à droite.
    const chain = Array.from({ length: 7 }, (_, i) => node(`n${i}`, i, 0, `Étape numéro ${i}`, 'sous-titre assez long'));
    const chainEdges: PipelineEdge[] = [
      ...chain.slice(1).map((n, i) => ({ from: `n${i}`, to: n.id, ...(i % 2 ? { step: i } : {}) })),
      { from: 'n6', to: 'n2', dashed: true, step: 9 },
    ];
    // Verticale avec deux couloirs : arêtes de même étage, dans les deux sens, et descente courbe.
    const lanes = [...chain.map((n) => ({ ...n, row: 0 })), node('h', 3, 1, 'Humain 🙂', 'reprend', true)];
    const lanesEdges: PipelineEdge[] = [...chainEdges, { from: 'n3', to: 'h', dashed: true }, { from: 'h', to: 'n3' }, { from: 'n2', to: 'h', step: 4 }];
    const cases = [
      { name: 'tessera', ...tesseraPipeline() },
      { name: 'live-2026-10', ...LIVE_PIPELINE },
      { name: 'schema-2026-11', ...SCHEMA_PIPELINE },
      { name: 'chaine-verticale', nodes: chain, edges: chainEdges },
      { name: 'couloirs-verticaux', nodes: lanes, edges: lanesEdges },
    ].map((c) => ({ ...c, journal: `Journal ${c.name}`, label: `Schéma ${c.name}` }));
    lock(
      path.join(SERVICES, 'testdata/dossier_print_diagrams.json'),
      cases.map((c) => ({ name: c.name, nodes: c.nodes, edges: c.edges, journal: c.journal, label: c.label, diagram: printDiagram(c) })),
    );
  });
});
