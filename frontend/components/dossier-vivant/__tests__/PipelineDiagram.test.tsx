/**
 * Rendu du schéma d'architecture (PipelineDiagram) : à la largeur du panneau mesurée à 1920×1080,
 * le SVG a la taille de son échelle — libellés à taille lisible, jamais écrasés par le CSS — et
 * dessine chaque boîte du schéma avec ses deux lignes de texte.
 */
import { render } from '@testing-library/react';

import { LIVE_PIPELINE, PANEL_WIDTH, tesseraPipeline } from '@/lib/dossier-vivant/__fixtures__/pipelines';
import { CH } from '@/lib/dossier-vivant/choreography';
import { SCREEN_FIT } from '@/lib/dossier-vivant/pipeline';

import { PipelineDiagram } from '../PipelineDiagram';

// jsdom ne fait pas de mise en page : la largeur offerte est celle mesurée dans le vrai panneau.
let width = PANEL_WIDTH;
beforeAll(() => {
  Object.defineProperty(HTMLElement.prototype, 'clientWidth', { configurable: true, get: () => width });
});

/** Taille de police nominale des titres de boîte (CSS .dgT) et des sous-titres (.dgS). */
const TITLE_PX = 13.5;
const SUB_PX = 12;

function draw(p: { nodes: typeof LIVE_PIPELINE.nodes; edges: typeof LIVE_PIPELINE.edges }) {
  // Instant après le dessin complet : tout est visible.
  const { container } = render(<PipelineDiagram nodes={p.nodes} edges={p.edges} journal="Journal" label="Schéma" at={0} now={CH.pipelineD + 1} />);
  const svg = container.querySelector('svg')!;
  const [, , vbW] = svg.getAttribute('viewBox')!.split(' ').map(Number);
  return { container, svg, scale: Number(svg.getAttribute('width')) / vbW };
}

describe('PipelineDiagram', () => {
  beforeEach(() => {
    width = PANEL_WIDTH;
  });

  it('run réel : le SVG fait sa taille nominale — titres 13,5 px, sous-titres 12 px à l’écran', () => {
    const { svg, scale, container } = draw(LIVE_PIPELINE);
    expect(svg.getAttribute('data-orientation')).toBe('vertical');
    expect(scale).toBeCloseTo(1, 2);
    expect(Number(svg.getAttribute('width'))).toBeLessThanOrEqual(PANEL_WIDTH);
    expect(TITLE_PX * scale).toBeGreaterThanOrEqual(13);
    // Toutes les boîtes, chacune avec son titre et son sous-titre.
    const texts = [...container.querySelectorAll('text')].map((t) => t.textContent);
    for (const n of LIVE_PIPELINE.nodes) {
      expect(texts).toContain(n.title);
      expect(texts).toContain(n.subtitle);
    }
    expect(texts).toContain('Journal');
  });

  it('rejeu Tessera : horizontale de la maquette, libellés au-dessus du plancher', () => {
    const { svg, scale } = draw(tesseraPipeline());
    expect(svg.getAttribute('data-orientation')).toBe('horizontal');
    expect(scale).toBeGreaterThanOrEqual(SCREEN_FIT.minScale - 0.01);
    expect(SUB_PX * scale).toBeGreaterThanOrEqual(9.8);
  });

  it('aucune largeur de 100 % : le SVG porte des dimensions en pixels', () => {
    const { svg } = draw(LIVE_PIPELINE);
    expect(svg.getAttribute('width')).toMatch(/^\d+$/);
    expect(svg.getAttribute('height')).toMatch(/^\d+$/);
  });

  it('avant le dessin, rien n’est visible ; les pastilles arrivent en dernier', () => {
    const { container } = render(<PipelineDiagram nodes={LIVE_PIPELINE.nodes} edges={LIVE_PIPELINE.edges} journal="J" label="S" at={10} now={9} />);
    const groups = [...container.querySelectorAll('svg > g')] as SVGGElement[];
    expect(groups.every((g) => g.style.visibility === 'hidden')).toBe(true);
    expect(groups.at(-1)!.querySelectorAll('circle')).toHaveLength(3);
  });
});
