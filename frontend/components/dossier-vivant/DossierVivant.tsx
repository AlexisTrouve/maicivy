'use client';

import { useCallback, useEffect, useLayoutEffect, useMemo, useReducer, useRef, useState, type ReactNode } from 'react';
import { useLocale, useTranslations } from 'next-intl';

import { clamp01, easeIO, easeOut, lerp, prog } from '@/lib/dossier-vivant/anim';
import { CH } from '@/lib/dossier-vivant/choreography';
import { SECTION_IDS, SUB_AGENTS, type StampedEvent, type SubAgentId } from '@/lib/dossier-vivant/events';
import { ReplaySource, RunClock } from '@/lib/dossier-vivant/eventSource';
import { LIVE_PDF_PATH, REPLAY_PDF_PATH, SKILLS_PATH } from '@/lib/dossier-vivant/pdfExport';
import { LIVE_RUN_PATH, SseSource } from '@/lib/dossier-vivant/sseSource';
import { applyEvent, elapsedAt, initialRunState, totalsAt, type RunState } from '@/lib/dossier-vivant/runState';
import { G, anchorOf, buildScene, introFoot, layoutFor, stageGeometry, type Rect, type Scene, type SceneInput, type SceneNode, type StageGeometry } from '@/lib/dossier-vivant/scene';
import { tesseraScenario, type ReplayContext } from '@/lib/dossier-vivant/tessera';

import { AgentCursors } from './AgentCursors';
import { AgentTeam, fanHeight } from './AgentTeam';
import { DesignDossier, type ProjectFacts } from './DesignDossier';
import { KeptProjects } from './KeptProjects';
import { MountainScene, type MountainMotion } from './MountainScene';
import { ReplayBar, type BarRun } from './ReplayBar';
import { VisitorComposer } from './VisitorComposer';
import styles from './dossier-vivant.module.css';
import { useProjectFlower, type FlowerProjects } from './useProjectFlower';

/** Mesures prises dans le DOM (px de la scène) ; tout le reste se calcule. */
interface Measures {
  W: number;
  H: number;
  stage: StageGeometry;
  /** Le champ du besoin : point de départ du pliage en tuile. */
  need: { x: number; y: number; w: number; h: number };
  orchH: number;
  colsTop: number;
  colsH: number;
  /** Feuille du dossier : sous les colonnes pendant le travail, sous leurs en-têtes ensuite. */
  sheet: { work: number; mid: number };
}

type Action = { type: 'reset' } | { type: 'batch'; events: readonly StampedEvent[] };
function reducer(s: RunState, a: Action): RunState {
  return a.type === 'reset' ? initialRunState() : a.events.reduce(applyEvent, s);
}

/** Position d'un élément dans un ancêtre positionné, sans dépendre des transforms en cours. */
function offsetWithin(el: HTMLElement, ancestor: HTMLElement): { x: number; y: number } {
  let x = 0;
  let y = 0;
  let cur: HTMLElement | null = el;
  while (cur && cur !== ancestor) {
    x += cur.offsetLeft;
    y += cur.offsetTop;
    cur = cur.offsetParent as HTMLElement | null;
  }
  return { x, y };
}

/** Base de l'API côté navigateur (même règle que lib/chat-api.ts). */
const API_BASE = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080';

/**
 * Téléphone : part de la feuille du dossier visible sous les colonnes pendant le travail — son
 * seul en-tête (titre, « en rédaction »). Sur grand écran, jusqu'à 340 px de texte qui s'écrit ;
 * sur ~440 px d'équipe, ce serait autant de moins pour les agents, qui sont le sujet à ce moment.
 */
const COMPACT_SHEET_PEEK = 56;
/**
 * Téléphone couché : la même idée, sur ~220 px de haut — le seul titre de la feuille (44 px), pour
 * laisser aux colonnes de quoi montrer nom, état et appel en cours.
 */
const LANDSCAPE_SHEET_PEEK = 44;

const lerpRect = (a: Rect, b: Rect, k: number): Rect => ({ x: lerp(a.x, b.x, k), y: lerp(a.y, b.y, k), w: lerp(a.w, b.w, k), h: lerp(a.h, b.h, k) });

/**
 * Défile jusqu'où la scène remplit l'écran sous la barre : son bas au bas de la fenêtre.
 * Grand écran, la scène est déjà logée sous l'en-tête du site (--dv-head-h) : c'est le haut de page,
 * l'en-tête reste visible. Téléphone, elle prend tout l'écran sous la barre : on défile juste sous
 * l'en-tête du site (qui reste à un geste au-dessus), et le run se joue comme avant lui.
 */
function scrollToStage(stage: HTMLElement | null) {
  if (!stage) return;
  const r = stage.getBoundingClientRect();
  window.scrollTo(0, Math.max(0, Math.round(r.top + window.scrollY + r.height - window.innerHeight)));
}

const fmtElapsed = (s: number) => `${Math.floor(s / 60)}:${String(Math.floor(s % 60)).padStart(2, '0')}`;

/**
 * DossierVivant — la scène complète : un besoin devient une tuile de la montagne, une équipe
 * d'agents le traite en parallèle, un dossier de conception s'écrit avec ses preuves chiffrées.
 *
 * QUOI : chef d'orchestre client. Il tient l'HORLOGE du run (RunClock), branche la SOURCE
 * d'événements (ReplaySource dans cette étape ; SSE à la suivante, sans autre changement), replie
 * le flux en état (runState.ts) et rend, à chaque image, la scène comme fonction de (état, instant).
 * Les composants ne possèdent aucune minuterie : pause, rejouer et `?t=` figent tout l'écran d'un coup.
 * COMMENT les mouvements d'ensemble (pliage du champ, glissement de la montagne, entrée de
 * l'équipe) partent de `run.started` (choreography.ts) ; tout le reste part de l'événement qui le
 * cause. Les positions viennent de scene.ts, à partir de quelques mesures DOM prises au montage et
 * au redimensionnement — jamais pendant une image.
 * DEUX SOURCES : par défaut, TOUT visiteur — sans connexion — écrit son besoin (pré-rempli par
 * l'exemple Tessera) et « Lancer l'équipe » part sur la source RÉELLE (SseSource → route publique
 * du run → vrais agents). La démo enregistrée (ReplaySource, Tessera) reste à un clic : « Voir la
 * démo » dans la barre, et `?t=` pour les captures.
 * TROIS DISPOSITIONS, une seule histoire (scene.ts › layoutFor, stageGeometry) : grand écran
 * (montagne à gauche, équipe à droite), téléphone en portrait sous 760 px (montagne en haut, bandeau
 * des projets retenus, équipe dessous) et téléphone COUCHÉ (côte à côte, densité du téléphone) ; sur
 * téléphone, à la remise le dossier prend tout l'écran et se lit en défilant. Le CSS bascule aux
 * mêmes mesures de fenêtre. Mêmes événements, même horloge, mêmes composants : seuls placements et
 * densité changent.
 * ÉCRAN + PAGE : l'en-tête du site au-dessus (il défile avec la page), puis la barre (collante),
 * puis la scène au premier écran (`.stage`, hauteur de la fenêtre moins la barre, et moins l'en-tête
 * sur grand écran — c'est elle que `measure` lit, jamais le document) ; la page continue dessous,
 * avec le pied de page (`footer`, rendu côté serveur). À l'OUVERTURE, la zone de la scène (`.page`)
 * s'allonge jusqu'au pied de la montagne, qu'on voit entière en défilant. À la remise, une fois la
 * feuille en place (`unrolled`), elle grandit à la hauteur de la feuille : le dossier se lit en
 * défilant la PAGE, une seule copie, sans ascenseur interne. La barre reste collée en haut.
 * FLEUR : un clic sur un projet (tuile ou carte en grand écran, pastille des retenus sur téléphone,
 * ligne de preuve partout) ouvre sa fleur (useProjectFlower) par-dessus la scène, sans toucher au run.
 * E2E : `dv-root` porte `data-phase` (intro | run), `data-run` (idle | running | completed |
 * failed), `data-mode` (replay | live) et `data-layout` (wide | compact | landscape, une fois mesuré) ;
 * `dv-page` porte `data-unrolled` une fois le dossier déroulé.
 */
export function DossierVivant({
  sceneInput,
  facts,
  replay,
  seek,
  speed,
  projects,
  footer,
}: {
  sceneInput: SceneInput;
  facts: Readonly<Record<string, ProjectFacts>>;
  replay: ReplayContext;
  /** Instant du run où figer l'écran (captures) ; null = parcours normal. */
  seek: number | null;
  /** Vitesse de l'horloge du rejeu (E2E). */
  speed: number;
  /** Les projets complets de la montagne, pour leur fleur. */
  projects: FlowerProjects;
  /** Pied de page, rendu côté serveur, posé sous la scène. */
  footer: ReactNode;
}) {
  const t = useTranslations('dossierVivant');
  const locale = useLocale();
  const scene = useMemo(() => buildScene(sceneInput), [sceneInput]);
  const nodes = useMemo(() => new Map<string, SceneNode>(sceneInput.nodes.map((n) => [n.id, n])), [sceneInput]);
  const scenario = useMemo(() => tesseraScenario(replay), [replay]);
  const visitor = scenario.visitor;

  const clock = useRef(new RunClock(speed));
  const stopSource = useRef<(() => void) | null>(null);
  const [state, dispatch] = useReducer(reducer, undefined, initialRunState);
  const [phase, setPhase] = useState<'intro' | 'run'>('intro');
  const [now, setNow] = useState(0);
  const [playing, setPlaying] = useState(false);
  const [K, setK] = useState<Measures | null>(null);
  // Besoin et présentation éditables, pré-remplis par le scénario (un exemple). `mode` = la source
  // du dernier départ ; avant tout départ, rien ne tourne (phase `intro`).
  const [mode, setMode] = useState<'replay' | 'live'>('replay');
  const modeRef = useRef(mode);
  modeRef.current = mode;
  const [draftNeed, setDraftNeed] = useState(visitor.need);
  const [draftWho, setDraftWho] = useState(`${visitor.name}, ${visitor.affiliation}`);

  const rootRef = useRef<HTMLDivElement>(null);
  const stageRef = useRef<HTMLDivElement>(null);
  const needRef = useRef<HTMLTextAreaElement>(null);
  const flower = useProjectFlower(projects);
  /** Hauteur de la feuille déroulée (px), mesurée : la page grandit d'autant. */
  const [sheetH, setSheetH] = useState(0);

  /** Lance (ou relance) le run à l'instant `from`, horloge figée si `freeze`. */
  const launch = useCallback(
    (from = 0, freeze = false) => {
      stopSource.current?.();
      dispatch({ type: 'reset' });
      // La scène est au premier écran : un départ (ou « Rejouer » depuis le bas du dossier) y ramène.
      scrollToStage(stageRef.current);
      if (freeze) clock.current.freeze(from);
      else clock.current.start(from);
      stopSource.current = new ReplaySource(scenario.events, clock.current).open((events) => dispatch({ type: 'batch', events }));
      setMode('replay');
      setPhase('run');
      setNow(clock.current.now());
      setPlaying(clock.current.playing);
    },
    [scenario],
  );

  /** Run RÉEL : le besoin du visiteur part au backend, les événements arrivent au fil du travail. */
  const launchLive = useCallback(() => {
    stopSource.current?.();
    dispatch({ type: 'reset' });
    scrollToStage(stageRef.current);
    clock.current.start(0);
    const request = { need: draftNeed, who: draftWho, lang: locale };
    stopSource.current = new SseSource(`${API_BASE}${LIVE_RUN_PATH}`, request, clock.current).open((events) => dispatch({ type: 'batch', events }));
    setMode('live');
    setPhase('run');
    setPlaying(true);
  }, [draftNeed, draftWho, locale]);

  /** La démo enregistrée. Mouvement réduit : pas de chorégraphie, le dossier remis s'affiche directement. */
  const launchDemo = useCallback(() => {
    if (window.matchMedia('(prefers-reduced-motion: reduce)').matches) launch(scenario.duration, true);
    else launch(0);
  }, [launch, scenario.duration]);

  // `?t=` : écran figé à un instant du run (captures de comparaison avec la maquette).
  useEffect(() => {
    if (seek !== null) launch(seek, true);
    return () => stopSource.current?.();
  }, [seek, launch]);

  // Une seule boucle d'images : elle avance l'horloge du run, et arrête le rejeu à sa fin.
  useEffect(() => {
    let frame = 0;
    const tick = () => {
      const c = clock.current;
      // Le rejeu s'arrête à la fin de son scénario ; un run réel dure ce qu'il dure.
      if (modeRef.current === 'replay' && c.playing && c.now() >= scenario.duration) {
        c.freeze(scenario.duration);
        setPlaying(false);
      }
      setNow(c.now());
      frame = requestAnimationFrame(tick);
    };
    frame = requestAnimationFrame(tick);
    return () => cancelAnimationFrame(frame);
  }, [scenario.duration]);

  // Mesures : au montage et au redimensionnement.
  useLayoutEffect(() => {
    const measure = () => {
      const stage = stageRef.current;
      const root = rootRef.current;
      const needEl = needRef.current;
      if (!stage || !root || !needEl) return;
      const W = stage.clientWidth;
      const H = stage.clientHeight;
      const hero = root.querySelector<HTMLElement>('[data-testid="dv-hero"]')!;
      // La disposition se décide sur la FENÊTRE, comme les media queries du CSS (même règle) :
      // la géométrie et la matière basculent ensemble.
      const geo = stageGeometry(W, H, hero.offsetTop + hero.offsetHeight, scene, layoutFor(window.innerWidth, window.innerHeight));
      const needPos = offsetWithin(needEl, stage);
      const orch = root.querySelector<HTMLElement>('[data-testid="dv-orchestrator"]');
      const orchH = orch?.offsetHeight ?? 128;
      const dosWorkVisible =
        geo.layout === 'landscape' ? LANDSCAPE_SHEET_PEEK : geo.compact ? COMPACT_SHEET_PEEK : Math.round(Math.min(340, geo.team.h * 0.36));
      const fanH = fanHeight(geo.compact);
      const colsTop = orchH + fanH;
      const colsH = geo.team.h - orchH - fanH - 12 - dosWorkVisible;
      const states = [...root.querySelectorAll<HTMLElement>('[data-testid="dv-agent-state"]')];
      // Grand écran : la feuille monte en laissant voir les en-têtes des colonnes. Téléphone : les
      // colonnes sont sur deux rangées, elle les couvre toutes et garde l'orchestrateur en vue.
      const headH = geo.compact ? 0 : Math.max(0, ...states.map((el) => el.offsetTop + el.offsetHeight)) + 12;
      setK({
        W,
        H,
        stage: geo,
        need: { x: needPos.x, y: needPos.y, w: needEl.offsetWidth, h: needEl.offsetHeight },
        orchH,
        colsTop,
        colsH,
        sheet: { work: colsTop + colsH + 12, mid: colsTop + headH },
      });
    };
    measure();
    // Les polices changent les hauteurs (titre, en-têtes de colonnes) : on remesure une fois chargées.
    document.fonts?.ready.then(measure);
    window.addEventListener('resize', measure);
    return () => window.removeEventListener('resize', measure);
  }, [scene]);

  const started = state.startedAt !== null;
  const rt = started ? now - state.startedAt! : -1;

  // Mouvements d'ensemble, depuis run.started.
  const ho = easeIO(prog(rt, CH.fold, CH.heroOut));
  const mv = easeIO(prog(rt, CH.move, CH.moveD));
  const open = easeIO(prog(rt, CH.open, CH.openD));
  const ti = easeOut(prog(rt, CH.team, CH.teamD));
  const motion: MountainMotion | null = K
    ? {
        s: lerp(K.stage.intro.s, K.stage.work.s, mv),
        x: lerp(K.stage.intro.x, K.stage.work.x, mv),
        y: lerp(K.stage.intro.y, K.stage.work.y, mv),
        open,
      }
    : null;

  // Boîte de l'équipe : fixe sur grand écran ; sur téléphone, elle s'étend à toute la scène à la
  // remise, au rythme où la feuille du dossier monte (CH.fullD) — le dossier se lit alors en plein écran.
  const fullK = state.completedAt === null ? 0 : easeIO(prog(now, state.completedAt, CH.fullD));
  const teamBox = K ? lerpRect(K.stage.team, K.stage.full, fullK) : null;
  const compact = K?.stage.compact ?? false;
  const layout = K?.stage.layout ?? 'wide';
  // Déroulé : la remise est jouée jusqu'au bout (feuille en haut, boîte à sa taille finale) — rien de
  // la chorégraphie ne bouge plus. Même instant que le retour en haut de la feuille (DesignDossier).
  const unrolled = state.completedAt !== null && now >= state.completedAt + CH.fullD + 0.1;

  // Feuille déroulée : sa hauteur suit son contenu (compétences arrivées après coup, images, police).
  useLayoutEffect(() => {
    const sheet = unrolled ? rootRef.current?.querySelector<HTMLElement>('[data-testid="dv-dossier"]') : null;
    if (!sheet) {
      setSheetH(0);
      return;
    }
    const read = () => setSheetH(sheet.offsetHeight);
    read();
    const ro = new ResizeObserver(read);
    ro.observe(sheet);
    return () => ro.disconnect();
  }, [unrolled]);
  // Hauteur de la page : l'écran, sauf deux cas où elle s'allonge — à l'ouverture, jusqu'au pied de
  // la montagne (on la voit entière en défilant, jamais réduite pour tenir) ; déroulée, jusqu'au bas
  // de la feuille et sa marge.
  const pageH = !K
    ? undefined
    : phase === 'intro'
      ? Math.max(K.H, Math.ceil(introFoot(scene, K.stage.intro) + K.stage.pad))
      : teamBox && unrolled && sheetH > 0
        ? Math.max(K.H, Math.ceil(teamBox.y + sheetH + K.stage.pad))
        : undefined;

  const usage = totalsAt(state, now);
  const elapsedLabel = fmtElapsed(elapsedAt(state, now));
  const costLabel = new Intl.NumberFormat(locale, { minimumFractionDigits: 3, maximumFractionDigits: 3 }).format(usage.costUsd);
  const shortNames = Object.fromEntries(SUB_AGENTS.map((id) => [id, t(`team.agents.${id}.short`)])) as Record<SubAgentId, string>;

  const onToggle = () => {
    // Avant tout départ, le bouton de lecture de la barre lance la démo (« Voir la démo »).
    if (phase === 'intro') {
      launchDemo();
      return;
    }
    const c = clock.current;
    if (c.playing) c.pause();
    else if (c.now() >= scenario.duration) {
      launch(0);
      return;
    } else c.resume();
    setPlaying(c.playing);
  };

  const runStatus =
    state.completedAt !== null && now >= state.completedAt ? 'completed' : state.failure ? 'failed' : started ? 'running' : 'idle';
  const live = mode === 'live';
  // Avancement d'un run réel : sections écrites sur cinq (sa durée n'est pas connue d'avance).
  const liveProgress = state.completedAt !== null ? 1 : state.dossier ? SECTION_IDS.filter((id) => state.dossier!.sections[id].status === 'done').length / SECTION_IDS.length : 0;
  // Refus avant le départ (saisie hors bornes, service indisponible) : le champ reste là, l'erreur dessous.
  const preStartError = live && !started ? (state.failure?.reason ?? null) : null;
  // Pendant le rejeu, le champ montre le besoin du SCÉNARIO (en lecture seule) ; sinon celui du visiteur.
  const replaying = phase === 'run' && !live;
  const shownNeed = replaying ? visitor.need : (state.visitor?.need ?? draftNeed);
  const bar: BarRun =
    phase === 'intro'
      ? { kind: 'idle' }
      : live
        ? { kind: 'live', model: state.agents.orchestrator.model, failure: state.failure?.reason ?? null, over: runStatus === 'completed' || runStatus === 'failed' }
        : { kind: 'replay' };
  const shownInitials = state.visitor?.initials ?? visitor.initials;

  return (
    <div
      ref={rootRef}
      className={styles.root}
      data-testid="dv-root"
      data-phase={phase}
      data-run={runStatus}
      data-mode={mode}
      data-layout={K ? layout : undefined}
    >
      <ReplayBar
        progress={phase !== 'run' ? 0 : live ? liveProgress : clamp01(now / scenario.duration)}
        playing={live ? false : playing}
        onToggle={onToggle}
        // Rejeu : la démo repart du début. Direct : un nouveau round réel sur le même besoin
        // (launchLive remet l'état à zéro et rouvre le flux) — la barre ne l'autorise qu'une fois
        // le round précédent fini.
        onReplay={live ? launchLive : () => launch(0)}
        run={bar}
      />
      <main className={styles.page} style={{ height: pageH }} data-testid="dv-page" data-unrolled={unrolled ? '' : undefined}>
        {/* Le premier écran : la scène, à la hauteur de la fenêtre (mesurée par `measure`). */}
        <div ref={stageRef} className={styles.stage}>
          <VisitorComposer
            ref={needRef}
            need={shownNeed}
            who={replaying ? `${visitor.name}, ${visitor.affiliation}` : draftWho}
            edit={replaying ? undefined : { onNeed: setDraftNeed, onWho: setDraftWho, error: preStartError }}
            onLaunch={launchLive}
            launchFocusable={ho < 0.5}
            style={{ transform: `translateY(${(-50 * ho).toFixed(1)}px)`, opacity: 1 - ho, visibility: ho < 1 ? 'visible' : 'hidden' }}
          />

          {K && motion && (
            <>
              {/* Tuiles cliquables en grand écran seulement : sur téléphone, ~12 px (cf. MountainScene). */}
              <MountainScene scene={scene} nodes={nodes} state={state} now={now} motion={motion} label={t('mountain.label')} onOpen={compact ? undefined : flower.open} />
              <AgentCursors scene={scene} state={state} now={now} motion={motion} names={shortNames} compact={compact} />
              <Ghost K={K} rt={rt} visitorNeed={shownNeed} initials={shownInitials} scene={scene} />
              {K.stage.kept && phase === 'run' && (
                <KeptProjects state={state} now={now} nodes={nodes} box={K.stage.kept} label={t('mountain.kept')} onOpen={flower.open} canOpen={flower.canOpen} />
              )}
            </>
          )}
        </div>

        {/* L'équipe et la feuille : HORS du cadre de la scène (qui rogne à l'écran), aux mêmes
            coordonnées — déroulée, la feuille doit pouvoir descendre sous le premier écran. */}
        <aside
          className={styles.team}
          aria-label={t('team.label')}
          data-testid="dv-team"
          data-unrolled={unrolled ? '' : undefined}
          style={
            teamBox
              ? {
                  left: teamBox.x,
                  top: teamBox.y,
                  width: teamBox.w,
                  height: teamBox.h,
                  transform: `translateX(${((1 - ti) * 40).toFixed(1)}px)`,
                  opacity: ti,
                  visibility: ti > 0 ? 'visible' : 'hidden',
                }
              : { visibility: 'hidden' }
          }
        >
          <AgentTeam
            state={state}
            now={now}
            width={K?.stage.team.w ?? 800}
            orchHeight={K?.orchH ?? 128}
            colsHeight={K?.colsH ?? 400}
            totals={{ usage, elapsedLabel, costLabel }}
            layout={layout}
          />
          {K && (
            <DesignDossier
              state={state}
              now={now}
              stops={K.sheet}
              nodes={nodes}
              facts={facts}
              elapsedLabel={elapsedLabel}
              costLabel={costLabel}
              // Deux routes publiques : le PDF d'un run réel (sans mention) et celui du rejeu (avec).
              pdfEndpoint={`${API_BASE}${live ? LIVE_PDF_PATH : REPLAY_PDF_PATH}`}
              skillsEndpoint={`${API_BASE}${SKILLS_PATH}`}
              unrolled={unrolled}
              onOpen={flower.open}
              canOpen={flower.canOpen}
            />
          )}
        </aside>
      </main>
      {footer}
      {flower.flower}
    </div>
  );
}

/**
 * Ghost — le passage : le champ du besoin se replie en tuile et rejoint sa place sous le sommet.
 * Le texte s'efface pendant le repli, les initiales apparaissent à la fin ; puis la vraie tuile
 * du visiteur (MountainScene) prend le relais à l'atterrissage.
 */
function Ghost({ K, rt, visitorNeed, initials, scene }: { K: Measures; rt: number; visitorNeed: string; initials: string; scene: Scene }) {
  if (rt < CH.ghostA || rt >= CH.ghostB + 0.12) return null;
  const gp = easeIO(prog(rt, CH.ghostA, CH.ghostB - CH.ghostA));
  const { s, x: mx, y: my } = K.stage.work;
  const [ax, ay] = anchorOf(scene, 'visitor');
  const tw = G.tile * s;
  const vx = mx + ax * s - tw / 2;
  const vy = my + ay * s - tw / 2;
  return (
    <div
      className={styles.ghost}
      aria-hidden="true"
      data-testid="dv-ghost"
      style={{
        transform: `translate(${lerp(K.need.x, vx, gp).toFixed(1)}px, ${lerp(K.need.y, vy, gp).toFixed(1)}px)`,
        width: lerp(K.need.w, tw, gp),
        height: lerp(K.need.h, tw, gp),
        borderRadius: lerp(8, 8 * s, gp),
      }}
    >
      <p className={styles.ghostText} style={{ opacity: 1 - prog(gp, 0, 0.35) }}>
        {visitorNeed}
      </p>
      <span className={styles.ghostMono} style={{ opacity: prog(gp, 0.75, 0.25), fontSize: 22 * s }}>
        {initials}
      </span>
    </div>
  );
}
