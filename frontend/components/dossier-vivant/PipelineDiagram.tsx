'use client';

import { Fragment, useLayoutEffect, useRef, useState } from 'react';

import { reveal } from '@/lib/dossier-vivant/anim';
import { CH } from '@/lib/dossier-vivant/choreography';
import type { PipelineEdge, PipelineNode } from '@/lib/dossier-vivant/events';
import { layoutPipeline, PIPE, SCREEN_FIT } from '@/lib/dossier-vivant/pipeline';

import styles from './dossier-vivant.module.css';

/**
 * PipelineDiagram — le schéma d'architecture du dossier, dessiné élément par élément sur
 * `CH.pipelineD` secondes, à une échelle qui garde ses libellés LISIBLES.
 *
 * QUOI : mesure la largeur que lui laisse sa section, puis demande à layoutPipeline la mise en page
 * qui y tient sans descendre sous `SCREEN_FIT.minScale` — horizontale (maquette) quand elle entre,
 * verticale sinon (cf. pipeline.ts). Le SVG prend la taille de cette échelle, en pixels : il n'est
 * jamais étiré au-delà de sa taille nominale, ni comprimé en douce par le CSS.
 * POURQUOI mesurer ici et pas en CSS : le choix d'orientation dépend de la largeur réelle, et un
 * `width:100%` laissait le navigateur écraser un schéma de 742 unités dans 399 px (texte de 7 px).
 * COMMENT : une mesure au montage puis à chaque redimensionnement (ResizeObserver) — jamais par
 * image. Avant la première mesure, rien n'est dessiné (l'effet passe avant la peinture).
 * E2E : `dv-diagram` porte `data-orientation` et `data-scale`.
 */
export function PipelineDiagram({ nodes, edges, journal, label, at, now }: { nodes: PipelineNode[]; edges: PipelineEdge[]; journal: string; label: string; at: number; now: number }) {
  const boxRef = useRef<HTMLDivElement>(null);
  const [avail, setAvail] = useState(0);

  useLayoutEffect(() => {
    const el = boxRef.current;
    if (!el) return;
    const measure = () => setAvail(el.clientWidth);
    measure();
    if (typeof ResizeObserver === 'undefined') return;
    const ro = new ResizeObserver(measure);
    ro.observe(el);
    return () => ro.disconnect();
  }, []);

  return (
    <div ref={boxRef} className={styles.diagramBox}>
      {avail > 0 && <Drawing nodes={nodes} edges={edges} journal={journal} label={label} at={at} now={now} maxWidth={avail} />}
    </div>
  );
}

function Drawing({
  nodes,
  edges,
  journal,
  label,
  at,
  now,
  maxWidth,
}: {
  nodes: PipelineNode[];
  edges: PipelineEdge[];
  journal: string;
  label: string;
  at: number;
  now: number;
  maxWidth: number;
}) {
  const L = layoutPipeline(nodes, edges, { maxWidth, minScale: SCREEN_FIT.minScale });
  const m = PIPE.margin;
  const vbW = L.width + 2 * m;
  const vbH = L.height + 2 * m;
  // Les pastilles se posent toutes ensemble, en dernier : elles numérotent les garde-fous.
  const steps = L.parts.flatMap((p) => p.edges).flatMap((e) => (e.step ? [e.step] : []));
  const partAt = (i: number) => at + (CH.pipelineD * i) / Math.max(1, L.parts.length - 1);
  return (
    <svg
      className={styles.diagram}
      viewBox={`${-m} ${-m} ${vbW} ${vbH}`}
      width={Math.round(vbW * L.scale)}
      height={Math.round(vbH * L.scale)}
      role="img"
      aria-label={label}
      data-testid="dv-diagram"
      data-orientation={L.orientation}
      data-scale={L.scale.toFixed(2)}
    >
      {L.parts.map((p, i) => {
        const r = reveal(now, partAt(i), CH.blockReveal, 0);
        return (
          <g key={i} style={{ opacity: r.opacity, visibility: r.visibility }}>
            {p.edges.map((e) => (
              <Fragment key={e.key}>
                <path className={styles.dgEdge} d={e.d} data-dashed={e.dashed ? '' : undefined} />
                <path className={styles.dgHead} d={e.head} />
              </Fragment>
            ))}
            {p.boxes.map((b) => (
              <g key={b.id}>
                <rect className={styles.dgBox} data-human={b.human ? '' : undefined} x={b.x} y={b.y} width={b.w} height={PIPE.boxH} rx={6} />
                <text className={styles.dgT} x={b.x + b.w / 2} y={b.y + 17} textAnchor="middle">
                  {b.title}
                </text>
                <text className={styles.dgS} x={b.x + b.w / 2} y={b.y + 32} textAnchor="middle">
                  {b.subtitle}
                </text>
              </g>
            ))}
            {p.journal && (
              <>
                <rect className={styles.dgBox} x={0} y={L.journalY} width={L.width} height={PIPE.journalH} rx={5} />
                <text className={styles.dgS} x={L.width / 2} y={L.journalY + 16} textAnchor="middle">
                  {journal}
                </text>
              </>
            )}
            {p.steps &&
              steps.map((s) => (
                <g key={s.n}>
                  <circle className={styles.dgRing} cx={s.x} cy={s.y} r={8} />
                  <text className={styles.dgN} x={s.x} y={s.y + 4} textAnchor="middle">
                    {s.n}
                  </text>
                </g>
              ))}
          </g>
        );
      })}
    </svg>
  );
}
