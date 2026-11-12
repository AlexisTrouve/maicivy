'use client';

import { Fragment, memo, type CSSProperties } from 'react';
import Image from 'next/image';

import { clamp01, easeOut, lerp, prog } from '@/lib/dossier-vivant/anim';
import { CH } from '@/lib/dossier-vivant/choreography';
import { LABEL, labelFit, labelWidths } from '@/lib/dossier-vivant/labels';
import { projectLook } from '@/lib/dossier-vivant/looks';
import type { RunState } from '@/lib/dossier-vivant/runState';
import { G, anchorOf, boxOf, linkEnd, linkPath, threadPath, type Scene, type SceneNode } from '@/lib/dossier-vivant/scene';
import { BACKDROP, summitCover } from '@/lib/montagne/art';

import { ProjectMark } from './ProjectMark';
import styles from './dossier-vivant.module.css';
import type { OpenProject } from './useProjectFlower';

/** Placement courant de la montagne à l'écran : échelle, translation, ouverture des rangs. */
export interface MountainMotion {
  s: number;
  x: number;
  y: number;
  /** 0 = rangs fermés (avant le visiteur), 1 = un rang de libre sous le sommet. */
  open: number;
}

/** Taille d'un nom de tuile À L'ÉCRAN (px) — défaut « noms petits » de la maquette corrigé. */
const LABEL_SCREEN_PX = 13;

/**
 * MountainScene — la montagne RÉELLE des projets, telle que l'équipe d'agents la parcourt.
 *
 * QUOI : fond peint, filets réels « sert à construire », cartes du sommet, tuiles, tuile du
 * visiteur, filets d'or visiteur → projets retenus, anneaux de ping, ligne de balayage du filtrage.
 * Tout ce qui bouge se déduit de (état du run, instant `now`) : aucune minuterie ici.
 * COMMENT : un repère fixe (géométrie grand écran de pyramid.ts) mis à l'échelle par UN transform
 * (`motion`) ; les tuiles descendent d'un rang quand la montagne s'ouvre pour le visiteur.
 * CLIC (grand écran, `onOpen` fourni) : chaque tuile et carte est un VRAI bouton qui ouvre la
 * fleur du projet ; un seul gestionnaire délégué sur la montagne lit `data-project-id` (comme
 * MontagneExplorer) — pas 75 fermetures recréées à chaque image. La tuile du visiteur n'ouvre rien.
 * Sur téléphone (`onOpen` absent), tuiles de ~12 px : aucune cible tactile honnête possible sans
 * refaire la maquette ; les pastilles des retenus (KeptProjects) et les lignes de preuve en tiennent
 * lieu. La montagne y reste une image (`role="img"`) ; cliquable, elle devient un groupe nommé.
 * E2E : `dv-mountain` ; `dv-tile` / `dv-card` portent `data-project-id` et `data-state`
 * (idle | candidate | reading | lit | dropped) ; `dv-visitor-tile` ; `dv-thread` porte `data-to`.
 */
export function MountainScene({
  scene,
  nodes,
  state,
  now,
  motion,
  label,
  onOpen,
}: {
  scene: Scene;
  nodes: ReadonlyMap<string, SceneNode>;
  state: RunState;
  now: number;
  motion: MountainMotion;
  label: string;
  /** Ouvre la fleur d'un projet ; absent = montagne non cliquable (téléphone). */
  onOpen?: OpenProject;
}) {
  const started = state.startedAt;
  const t = started === null ? -1 : now - started;
  // Les filets réels s'effacent quand le visiteur arrive : les siens prennent le relais.
  const linksOpacity = lerp(1, 0.32, prog(t, CH.dim, CH.dimD));
  const font = Math.max(12, LABEL_SCREEN_PX / motion.s);
  const lbl = `${font.toFixed(2)}px`;

  // Position de chaque projet dans le balayage (0 en haut, 1 en bas) : il se teinte quand la ligne le touche.
  const sweepSpan = scene.fitHeight - scene.tilesTop + 20;
  const sweepPos = (id: string) => clamp01((anchorOf(scene, id)[1] - scene.tilesTop + 20) / sweepSpan);

  const filter = state.filter;
  const sweepK = filter ? prog(now, filter.at, CH.sweepD) : 0;
  const scanY = lerp(scene.tilesTop - 20, scene.fitHeight, sweepK);
  const scanOpacity = filter && now >= filter.at && now <= filter.at + CH.sweepD + 0.15 ? 1 - prog(now, filter.at + CH.sweepD, 0.15) : 0;

  const landed = started !== null && t >= CH.ghostB;
  const visitor = state.visitor;

  // Pings : ceux du flux (retenu, cité, remise) et l'arrivée du visiteur.
  const pings = new Map(state.pings);

  // Apparence de chaque tuile, puis la largeur offerte à son nom : seuls les retenus (allumés) sont
  // nommés en permanence, et deux noms voisins se partagent l'espace entre eux (labels.ts).
  const looks = new Map(scene.tiles.map((b) => [b.id, projectLook(state, b.id, now, sweepPos(b.id))]));
  const widths = labelWidths(scene.tiles, new Set(scene.tiles.filter((b) => looks.get(b.id)!.look === 'lit').map((b) => b.id)));
  const visitorLines = labelFit(LABEL.maxWidth, font);
  if (started !== null) pings.set('visitor', [started + CH.ghostB, ...(state.pings.get('visitor') ?? [])]);

  const mtnStyle: CSSProperties = {
    width: scene.width,
    height: scene.height,
    transform: `translate(${motion.x.toFixed(2)}px, ${motion.y.toFixed(2)}px) scale(${motion.s.toFixed(4)})`,
    ['--lbl' as string]: lbl,
    // Inverse de l'échelle : sur téléphone (échelle ~0,2), le CSS compact s'en sert pour garder des
    // traits et des bordures d'au moins un pixel À L'ÉCRAN. Le grand écran ne le lit pas.
    ['--inv' as string]: (1 / motion.s).toFixed(3),
  };

  // Tuiles et cartes : boutons quand la montagne est cliquable, simples blocs sinon. Le transtypage
  // ne sert qu'au typage JSX (`type` n'existe pas sur un div) ; `type` reste indéfini pour un div.
  const Hit = (onOpen ? 'button' : 'div') as 'button';
  const hitType = onOpen ? ('button' as const) : undefined;
  const onClick = onOpen
    ? (e: React.MouseEvent) => {
        const el = (e.target as Element).closest<HTMLElement>('[data-project-id]');
        if (el && el.dataset.projectId) onOpen(el.dataset.projectId, el);
      }
    : undefined;

  return (
    <div className={styles.mtn} style={mtnStyle} role={onOpen ? 'group' : 'img'} aria-label={label} data-testid="dv-mountain" onClick={onClick}>
      <Backdrop scene={scene} />
      <svg className={styles.links} width={scene.width} height={scene.height} viewBox={`0 0 ${scene.width} ${scene.height}`} aria-hidden="true">
        <RealLinks scene={scene} open={motion.open} opacity={linksOpacity} />
        {/* Filets du visiteur SOUS les tuiles, comme les liens réels : la tuile reste un nœud opaque. */}
        <g>
          {[...state.threads].map(([id, at]) => {
            const k = easeOut(prog(now, at - 0.05, CH.threadD));
            const [x2, y2] = anchorOf(scene, id);
            // Dernière respiration : le filet s'éclaire à la remise puis retombe.
            const breath = state.completedAt !== null ? 1 - prog(now, state.completedAt + 0.5, 1.2) : 0;
            return (
              <g key={id} data-testid="dv-thread" data-to={id}>
                <path
                  className={styles.thread}
                  d={threadPath(scene, id)}
                  pathLength={1}
                  strokeDasharray="1"
                  strokeDashoffset={(1 - k).toFixed(4)}
                  strokeOpacity={k > 0 ? 0.72 + 0.28 * (state.completedAt !== null && now >= state.completedAt ? breath : 0) : 0}
                />
                <circle className={styles.threadEnd} cx={x2} cy={y2} r={2.6} opacity={prog(now, at + 0.6, 0.2)} />
              </g>
            );
          })}
        </g>
        <line className={styles.scan} x1={0} x2={scene.width} y1={scanY} y2={scanY} opacity={scanOpacity} />
      </svg>

      {scene.cards.map((c) => {
        const n = nodes.get(c.id);
        const { look, quiet } = projectLook(state, c.id, now, sweepPos(c.id));
        const cover = summitCover(c.id);
        return (
          <Hit
            key={c.id}
            type={hitType}
            className={styles.card}
            style={{ left: c.x, top: c.y, width: c.w, height: c.h, opacity: 1 - 0.5 * quiet }}
            aria-label={onOpen ? n?.name : undefined}
            aria-haspopup={onOpen ? 'dialog' : undefined}
            data-testid="dv-card"
            data-project-id={c.id}
            data-state={look}
          >
            <span className={styles.cardCover}>
              {cover ? (
                <Image src={cover} alt="" fill sizes={`${G.cardWidth}px`} className="object-cover" priority />
              ) : (
                <ProjectMark node={n} monoClass={styles.cardMono} iconClass={styles.cardIcon} />
              )}
            </span>
            <span className={styles.cardName}>{n?.name}</span>
          </Hit>
        );
      })}

      {scene.tiles.map((b) => {
        const n = nodes.get(b.id);
        if (!n) return null;
        const { look, quiet } = looks.get(b.id)!;
        const width = widths.get(b.id)!;
        return (
          <Hit
            key={b.id}
            type={hitType}
            className={styles.tile}
            style={{
              left: b.x,
              top: b.y,
              width: b.w,
              height: b.h,
              // `translate` et non `transform` : sur téléphone, une tuile allumée est agrandie par la
              // propriété `scale` (CSS), qui s'appliquerait AUSSI à un translateY de `transform`.
              translate: motion.open ? `0 ${(motion.open * G.rowHeight).toFixed(2)}px` : undefined,
              opacity: 1 - 0.5 * quiet,
            }}
            title={n.name}
            // Le nom sous la tuile est masqué hors survol (labels.ts) : le bouton porte le sien.
            aria-label={onOpen ? n.name : undefined}
            aria-haspopup={onOpen ? 'dialog' : undefined}
            data-testid="dv-tile"
            data-project-id={b.id}
            data-stratum={n.stratum}
            data-state={look}
          >
            <ProjectMark node={n} monoClass={styles.mono} iconClass={styles.markIcon} />
            {/* Nom entier, sans point de coupure dans un mot : trop long, il finit par « … » (le
                `title` de la tuile le donne en entier). Muet sauf allumé ou survolé (labels.ts). */}
            <span
              className={styles.tileName}
              style={{ ['--lw' as string]: `${width.toFixed(1)}px`, ['--lines' as string]: labelFit(width, font) }}
              data-lines={labelFit(width, font)}
              data-testid="dv-tile-name"
            >
              {n.name}
            </span>
          </Hit>
        );
      })}

      {visitor && (
        <div
          className={styles.tile}
          style={{ left: scene.visitor.x, top: scene.visitor.y, width: G.tile, height: G.tile, opacity: landed ? 1 : 0, visibility: landed ? 'visible' : 'hidden' }}
          data-visitor=""
          data-stratum="works"
          data-testid="dv-visitor-tile"
        >
          <span className={styles.mono}>{visitor.initials}</span>
          {/* Seul sur son rang : la largeur maximale, deux lignes voulues (nom, organisation). */}
          <span
            className={styles.tileName}
            style={{ ['--lw' as string]: `${LABEL.maxWidth}px`, ['--lines' as string]: visitorLines }}
            data-lines={visitorLines}
            data-testid="dv-tile-name"
          >
            {visitorLines === 2 ? (
              <>
                {visitor.name}
                <br />
                {visitor.organisation}
              </>
            ) : (
              `${visitor.name} · ${visitor.organisation}`
            )}
          </span>
        </div>
      )}

      {[...pings].map(([id, times]) => {
        const box = boxOf(scene, id);
        const last = times.filter((x) => x <= now).pop();
        const age = last === undefined ? Infinity : now - last;
        if (!box || age >= CH.pingD) return null;
        const k = age / CH.pingD;
        return (
          <div
            key={id}
            className={styles.ping}
            aria-hidden="true"
            style={{ left: box.x, top: box.y, width: box.w, height: box.h, opacity: 0.9 * (1 - k), transform: `scale(${1 + 0.45 * easeOut(k)})` }}
          />
        );
      })}
    </div>
  );
}

/** Le fond peint, cadré comme sur /montagne (pointe sous les cartes). Figé : mémoïsé. */
const Backdrop = memo(function Backdrop({ scene }: { scene: Scene }) {
  return (
    <div className={styles.art} style={{ top: scene.art.top, width: scene.art.width, height: scene.art.height }} aria-hidden="true">
      {/* eslint-disable-next-line @next/next/no-img-element -- webp déjà dimensionné (scripts/montagne-art.mjs), posé dans un repère mis à l'échelle par transform : l'optimiseur n'apporterait rien */}
      <img src={BACKDROP.desktop} alt="" fetchPriority="high" />
    </div>
  );
});

/** Filets réels « sert à construire ». Ne change qu'à l'ouverture des rangs : mémoïsé. */
const RealLinks = memo(function RealLinks({ scene, open, opacity }: { scene: Scene; open: number; opacity: number }) {
  return (
    <g style={{ opacity }}>
      {scene.links.map((l) => {
        const [ex, ey] = linkEnd(l, open);
        return (
          <Fragment key={l.key}>
            <path className={styles.link} d={linkPath(l, open)} />
            <circle className={styles.linkEnd} cx={ex} cy={ey} r={2} />
          </Fragment>
        );
      })}
    </g>
  );
});
