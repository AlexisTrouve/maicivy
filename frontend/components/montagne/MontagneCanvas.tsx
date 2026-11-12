'use client';

import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { useTranslations } from 'next-intl';

import type { Montagne, MontagneNode, MontagneStratum, StratumId } from '@/lib/montagne-api';
import {
  type Camera,
  type Frame,
  type Insets,
  type Point3,
  type Projected,
  clampCamera,
  facingYaw,
  iconSize,
  labelFontSize,
  makeProjector,
  orbit,
  overviewCamera,
  overviewPitch,
  stratumCamera,
} from '@/lib/montagne/camera';
import { type LabelCandidate, type Rect, chooseLabels, intersects, wrapLabel } from '@/lib/montagne/labels';
import { nodeIconSrc } from '@/lib/montagne/monogram';
import { MontagneFlower } from './MontagneFlower';

/**
 * MontagneCanvas — la « vue réseau » : la montagne en relief, canvas 2D, projection 3D maison.
 *
 * Vue SECONDAIRE depuis la refonte du 2026-10-31 : la page s'ouvre sur la montagne à plat rendue
 * côté serveur (MontagnePeak) ; ce canvas n'est téléchargé qu'au clic sur « vue réseau »
 * (MontagneExplorer) et s'ouvre directement en vue d'ensemble — le survol guidé d'arrivée a été
 * retiré avec son rôle de première vue.
 *
 * QUOI :
 *  - dessine la silhouette du cône, ses courbes de niveau (un anneau par rangée de projets), les
 *    liens « sert à construire » et les nœuds (icône + nom) à partir des positions PRÉCALCULÉES
 *    par le backend — aucune position n'est inventée ici ;
 *  - glisser = tourner, molette / pincer = zoomer, Maj+glisser ou rail des strates = monter ;
 *  - un clic sur un nœud ouvre sa fleur (MontagneFlower).
 *
 * POURQUOI un canvas 2D et pas three.js : 75 nœuds et ~20 liens tiennent largement dans un dessin
 * 2D projeté, sans embarquer un moteur 3D pour une seule page (exigence de la spécification).
 *
 * COMMENT :
 *  - CAMÉRA dans une ref (pas d'état React) : elle change à chaque image, re-rendre React à 60 Hz
 *    serait du gaspillage. Le rendu est À LA DEMANDE (`requestDraw`) hors survol, transitions et
 *    fondus d'étiquettes.
 *  - CADRAGE sur la zone sûre (camera.ts › fitCamera) : la scène moins le rail et l'aide, mesurés
 *    dans le DOM — sur téléphone le rail est une rangée en bas, sur grand écran une colonne à gauche.
 *  - ÉTIQUETTES arbitrées à chaque image (labels.ts › chooseLabels) : une étiquette qui en
 *    chevaucherait une autre, ou recouvrirait une icône, s'estompe (fondu ~150 ms) ; l'icône reste.
 *  - CALQUE DE CLIC : un <button> transparent par nœud, repositionné à chaque image par écriture
 *    directe du style. Il porte le clic réel (souris, doigt), le focus clavier et le nom accessible.
 *  - E2E : sur la scène `data-camera` (pose arrondie), `data-ready` et `data-labels-settled` (aucun fondu en cours) ; sur chaque nœud `data-node-id`, `data-label`
 *    (shown | hidden), `data-label-box` et `data-icon-box` (« gauche,haut,droite,bas » en px de
 *    la scène) — le test mesure chevauchements et cadrage sur ce que le dessin a réellement fait.
 */

/** Couleur des liens au repos et en sélection (l'or ne sert QUE pour la sélection). */
const LINK_REST = '226, 220, 205';
const GOLD = '217, 180, 95';
/** Seuil de déplacement (px) au-delà duquel un geste est un glisser et non un clic. */
const DRAG_THRESHOLD = 6;
/** Durée des transitions de caméra déclenchées par le rail des strates. */
const RAIL_TWEEN_MS = 700;
/** Durée d'un fondu d'étiquette (apparition comme disparition). */
const LABEL_FADE_MS = 150;
/** Largeur maximale d'une ligne d'étiquette avant passage à la ligne (px). */
const LABEL_MAX_WIDTH = 150;
/** Point de bascule du rail (colonne à gauche / rangée en bas) : le `sm` de Tailwind. */
const WIDE_QUERY = '(min-width: 640px)';

interface Tween {
  from: Camera;
  to: Camera;
  start: number;
}

/** Arrondi d'un rectangle en attribut `data-*` : « gauche,haut,droite,bas ». */
function boxAttr(r: Rect): string {
  return [r.left, r.top, r.right, r.bottom].map((v) => v.toFixed(1)).join(',');
}

export default function MontagneCanvas({ montagne }: { montagne: Montagne }) {
  const t = useTranslations('montagne');

  const stageRef = useRef<HTMLDivElement>(null);
  const canvasRef = useRef<HTMLCanvasElement>(null);
  const railRef = useRef<HTMLElement>(null);
  const hintRef = useRef<HTMLParagraphElement>(null);
  const buttonsRef = useRef(new Map<string, HTMLButtonElement>());
  const imagesRef = useRef(new Map<string, HTMLImageElement>());

  const frameRef = useRef<Frame>({ vp: { width: 1, height: 1 }, safe: { top: 0, right: 0, bottom: 0, left: 0 } });
  const cameraRef = useRef<Camera>({ yaw: 0, pitch: 0.3, distance: 30, targetY: 5 });
  /** Pitch de la vue d'ensemble pour le cadre courant (camera.ts › overviewPitch), recalculé au
   * redimensionnement : trop coûteux pour chaque image. */
  const overviewPitchRef = useRef(0.26);
  const tweenRef = useRef<Tween | null>(null);
  const rafRef = useRef(0);
  const hoverRef = useRef<string | null>(null);
  const selectedRef = useRef<string | null>(null);
  /** Opacité courante de chaque étiquette (0 → 1), pour les fondus ; et l'instant du dernier dessin. */
  const labelAlphaRef = useRef(new Map<string, number>());
  const lastDrawRef = useRef(0);
  /** Lignes d'étiquette déjà coupées, par nœud et taille de police (wrapLabel mesure du texte). */
  const wrapCacheRef = useRef(new Map<string, string[]>());

  const [selected, setSelected] = useState<MontagneNode | null>(null);
  const [ready, setReady] = useState(false);
  const readyRef = useRef(false);

  const strata = useMemo(
    () => [...montagne.strata].filter((s) => s.count > 0).sort((a, b) => a.level - b.level),
    [montagne.strata],
  );
  const nodeById = useMemo(() => new Map(montagne.nodes.map((n) => [n.id, n])), [montagne.nodes]);
  const points = useMemo<Point3[]>(() => montagne.nodes.map((n) => ({ x: n.x, y: n.y, z: n.z })), [montagne.nodes]);
  /** Degré de chaque nœud (liens entrants + sortants) : départage des étiquettes (labels.ts). */
  const degree = useMemo(() => {
    const d = new Map<string, number>();
    for (const l of montagne.links) {
      d.set(l.from, (d.get(l.from) ?? 0) + 1);
      d.set(l.to, (d.get(l.to) ?? 0) + 1);
    }
    return d;
  }, [montagne.links]);
  /** Hauteurs des courbes de niveau (une par rangée de projets), avec leur strate. */
  const rings = useMemo(() => {
    const byY = new Map<number, StratumId>();
    for (const n of montagne.nodes) byY.set(n.y, n.stratum);
    return [...byY.entries()].map(([y, stratum]) => ({ y, stratum })).sort((a, b) => a.y - b.y);
  }, [montagne.nodes]);
  /**
   * Yaw de la vue d'ensemble (pose d'ouverture) : face au projet le plus relié (à degré égal, l'id).
   * POURQUOI : ses liens « sert à construire » montent alors la face visible de la montagne — le
   * réseau se lit de face au lieu de s'écraser sur un flanc. Déterministe : il ne dépend que des
   * données.
   */
  const heroYaw = useMemo(() => {
    let hero: MontagneNode | null = null;
    for (const n of montagne.nodes) {
      const d = degree.get(n.id) ?? 0;
      const best = hero ? (degree.get(hero.id) ?? 0) : -1;
      if (d > best || (d === best && hero && n.id < hero.id)) hero = n;
    }
    return hero ? facingYaw(hero) : 0;
  }, [degree, montagne.nodes]);
  const strataLabels = useMemo(() => {
    const out = {} as Record<StratumId, string>;
    for (const s of montagne.strata) out[s.id] = t(`strata.${s.id}`);
    return out;
  }, [montagne.strata, t]);
  const { apex_y: apexY, slope } = montagne.shape;
  const baseRadius = slope * apexY;
  const coneRadius = useCallback((y: number) => slope * Math.max(0, apexY - y), [apexY, slope]);

  // ------------------------------------------------------------------------------------------
  // Dessin d'une image
  // ------------------------------------------------------------------------------------------
  const draw = useCallback(
    (now: number) => {
      const canvas = canvasRef.current;
      const stage = stageRef.current;
      if (!canvas || !stage) return false;
      const ctx = canvas.getContext('2d');
      if (!ctx) return false;
      const { vp, safe } = frameRef.current;
      const cam = cameraRef.current;
      const dpr = Math.min(window.devicePixelRatio || 1, 2);
      ctx.setTransform(dpr, 0, 0, dpr, 0, 0);

      // Fond : nuit profonde, un halo à peine plus clair derrière la montagne.
      const bg = ctx.createRadialGradient(vp.width / 2, vp.height * 0.55, 0, vp.width / 2, vp.height * 0.55, Math.max(vp.width, vp.height) * 0.75);
      bg.addColorStop(0, '#0d1424');
      bg.addColorStop(1, '#04060b');
      ctx.fillStyle = bg;
      ctx.fillRect(0, 0, vp.width, vp.height);

      const project = makeProjector(cam, vp);
      // Plage de profondeur de la montagne vue d'ici : sert à graduer pâleur et taille.
      const nearDepth = cam.distance - baseRadius;
      const farDepth = cam.distance + baseRadius;
      const nearness = (depth: number) => Math.min(1, Math.max(0, (farDepth - depth) / (farDepth - nearDepth)));

      // 1. Silhouette : enveloppe convexe du sommet et du cercle de pied projetés — c'est
      // exactement le contour d'un cône vu de n'importe où. Un voile très léger, un filet discret.
      const outline: { x: number; y: number }[] = [];
      const apex = project({ x: 0, y: apexY, z: 0 });
      if (apex.visible) outline.push(apex);
      for (let i = 0; i < 72; i++) {
        const a = (i / 72) * Math.PI * 2;
        const p = project({ x: baseRadius * Math.cos(a), y: 0, z: baseRadius * Math.sin(a) });
        if (p.visible) outline.push(p);
      }
      const hull = convexHull(outline);
      if (hull.length > 2) {
        const top = Math.min(...hull.map((p) => p.y));
        const bottom = Math.max(...hull.map((p) => p.y));
        const veil = ctx.createLinearGradient(0, top, 0, bottom);
        veil.addColorStop(0, 'rgba(148, 163, 184, 0.07)');
        veil.addColorStop(1, 'rgba(148, 163, 184, 0.025)');
        ctx.beginPath();
        hull.forEach((p, i) => (i ? ctx.lineTo(p.x, p.y) : ctx.moveTo(p.x, p.y)));
        ctx.closePath();
        ctx.fillStyle = veil;
        ctx.fill();
        ctx.strokeStyle = 'rgba(148, 163, 184, 0.16)';
        ctx.lineWidth = 1;
        ctx.stroke();
      }

      // 2. Courbes de niveau : un anneau par rangée de projets.
      for (const ring of rings) {
        const r = coneRadius(ring.y);
        if (r <= 0) continue;
        ctx.beginPath();
        let started = false;
        for (let i = 0; i <= 72; i++) {
          const a = (i / 72) * Math.PI * 2;
          const p = project({ x: r * Math.cos(a), y: ring.y, z: r * Math.sin(a) });
          if (!p.visible) {
            started = false;
            continue;
          }
          if (!started) ctx.moveTo(p.x, p.y);
          else ctx.lineTo(p.x, p.y);
          started = true;
        }
        ctx.strokeStyle = 'rgba(148, 163, 184, 0.1)';
        ctx.lineWidth = 1;
        ctx.stroke();
      }

      // 3. Projection des nœuds.
      const projected = new Map<string, Projected>();
      for (const n of montagne.nodes) projected.set(n.id, project(n));
      const focus = selectedRef.current ?? hoverRef.current;

      // 4. Noms des strates, sur le flanc gauche à mi-hauteur de leur bande. Réservés AVANT les
      // étiquettes des nœuds : ce sont eux qui disent l'ascension. Un nom qui sortirait du cadre
      // ou couvrirait une icône n'est pas dessiné.
      ctx.font = '600 11px Poppins, Inter, system-ui, sans-serif';
      const icons = new Map<string, Rect>();
      for (const n of montagne.nodes) {
        const p = projected.get(n.id)!;
        const half = iconSize(p.scale) / 2;
        icons.set(n.id, { left: p.x - half, top: p.y - half, right: p.x + half, bottom: p.y + half });
      }
      const bounds: Rect = { left: safe.left, top: safe.top, right: vp.width - safe.right, bottom: vp.height - safe.bottom };
      const reserved: { rect: Rect; text: string }[] = [];
      for (const s of strata) {
        const y = (s.y_min + s.y_max) / 2;
        const r = coneRadius(y);
        // Point du cercle le plus à gauche à l'écran : −r·(axe « droite » de la caméra).
        const p = project({ x: -Math.cos(cam.yaw) * r, y, z: Math.sin(cam.yaw) * r });
        if (!p.visible) continue;
        const text = strataLabels[s.id].toUpperCase();
        const w = ctx.measureText(text).width;
        const rect = { left: p.x - 16 - w, top: p.y - 7, right: p.x - 16, bottom: p.y + 7 };
        const inside = rect.left >= bounds.left && rect.right <= bounds.right && rect.top >= bounds.top && rect.bottom <= bounds.bottom;
        if (!inside || [...icons.values()].some((ic) => rect.left < ic.right && ic.left < rect.right && rect.top < ic.bottom && ic.top < rect.bottom)) continue;
        reserved.push({ rect, text });
      }
      for (const r of reserved) {
        ctx.fillStyle = 'rgba(148, 163, 184, 0.55)';
        ctx.textAlign = 'right';
        ctx.textBaseline = 'middle';
        ctx.fillText(r.text, r.rect.right, (r.rect.top + r.rect.bottom) / 2);
      }

      // 5. Liens : B → A. Sélection = or ; repos = ivoire, plus présent au premier plan. Un chevron
      // aux deux tiers indique le sens « sert à construire ».
      for (const l of montagne.links) {
        const a = projected.get(l.from);
        const b = projected.get(l.to);
        if (!a?.visible || !b?.visible) continue;
        const lit = focus !== null && (l.from === focus || l.to === focus);
        const near = (nearness(a.depth) + nearness(b.depth)) / 2;
        ctx.strokeStyle = lit ? `rgba(${GOLD}, 0.9)` : `rgba(${LINK_REST}, ${0.08 + 0.22 * near})`;
        ctx.lineWidth = lit ? 1.8 : 1;
        ctx.beginPath();
        ctx.moveTo(a.x, a.y);
        ctx.lineTo(b.x, b.y);
        ctx.stroke();
        const mx = a.x + (b.x - a.x) * 0.66;
        const my = a.y + (b.y - a.y) * 0.66;
        const ang = Math.atan2(b.y - a.y, b.x - a.x);
        const head = lit ? 7 : 5;
        ctx.beginPath();
        ctx.moveTo(mx - head * Math.cos(ang - 0.45), my - head * Math.sin(ang - 0.45));
        ctx.lineTo(mx, my);
        ctx.lineTo(mx - head * Math.cos(ang + 0.45), my - head * Math.sin(ang + 0.45));
        ctx.stroke();
      }

      // 6. Étiquettes candidates : lignes coupées, rectangle sous l'icône.
      const labelLines = new Map<string, { lines: string[]; font: number }>();
      const candidates: LabelCandidate[] = [];
      for (const n of montagne.nodes) {
        const p = projected.get(n.id)!;
        if (!p.visible) continue;
        const font = Math.round(labelFontSize(p.scale) * 2) / 2;
        ctx.font = `500 ${font}px Inter, system-ui, sans-serif`;
        const key = `${n.id}|${font}`;
        let lines = wrapCacheRef.current.get(key);
        if (!lines) {
          lines = wrapLabel(n.name, LABEL_MAX_WIDTH, (s) => ctx.measureText(s).width);
          wrapCacheRef.current.set(key, lines);
        }
        const width = Math.max(...lines.map((s) => ctx.measureText(s).width));
        const icon = icons.get(n.id)!;
        const top = icon.bottom + 3;
        labelLines.set(n.id, { lines, font });
        candidates.push({
          id: n.id,
          label: { left: p.x - width / 2, top, right: p.x + width / 2, bottom: top + lines.length * font * 1.2 },
          icon,
          depth: p.depth,
          degree: degree.get(n.id) ?? 0,
          focus: n.id === focus,
        });
      }
      const shown = chooseLabels(
        candidates,
        bounds,
        reserved.map((r) => r.rect),
      );
      const candidateOf = new Map(candidates.map((c) => [c.id, c]));

      // Fondus : chaque étiquette glisse vers 1 (affichée) ou 0 (estompée). Exception : une
      // étiquette qui s'efface en CROISANT une étiquette affichée disparaît d'un coup — mesuré
      // pendant une rotation sur mobile, un fondu de 150 ms suffisait à superposer deux noms à mi-opacité
      // sur la moitié des images. Le fondu ne sert qu'aux étiquettes qui ne gênent personne.
      const dt = lastDrawRef.current ? Math.min(100, now - lastDrawRef.current) : LABEL_FADE_MS;
      lastDrawRef.current = now;
      const shownRects = candidates.filter((c) => shown.has(c.id)).map((c) => c.label);
      let fading = false;
      for (const n of montagne.nodes) {
        const target = shown.has(n.id) ? 1 : 0;
        const cur = labelAlphaRef.current.get(n.id) ?? 0;
        const step = dt / LABEL_FADE_MS;
        const c = candidateOf.get(n.id);
        const collides = target === 0 && c !== undefined && shownRects.some((r) => intersects(c.label, r));
        const next = collides ? 0 : target > cur ? Math.min(target, cur + step) : Math.max(target, cur - step);
        labelAlphaRef.current.set(n.id, next);
        if (next !== target) fading = true;
      }

      // 7. Nœuds, du plus lointain au plus proche (le proche se peint par-dessus).
      const order = [...montagne.nodes].sort((x, y) => projected.get(y.id)!.depth - projected.get(x.id)!.depth);
      ctx.textAlign = 'center';
      ctx.textBaseline = 'top';
      order.forEach((n, rank) => {
        const p = projected.get(n.id)!;
        const button = buttonsRef.current.get(n.id);
        if (!p.visible) {
          if (button) button.style.visibility = 'hidden';
          return;
        }
        const k = nearness(p.depth);
        const isFocus = n.id === focus;
        const icon = iconSize(p.scale);
        const alpha = isFocus ? 1 : 0.35 + 0.65 * k;

        const img = imagesRef.current.get(n.id);
        ctx.globalAlpha = alpha;
        if (img && img.complete && img.naturalWidth > 0) {
          ctx.drawImage(img, p.x - icon / 2, p.y - icon / 2, icon, icon);
        } else {
          ctx.fillStyle = '#0b111c';
          ctx.beginPath();
          ctx.arc(p.x, p.y, icon / 2, 0, Math.PI * 2);
          ctx.fill();
        }
        if (isFocus) {
          ctx.strokeStyle = `rgba(${GOLD}, 1)`;
          ctx.lineWidth = 2;
          ctx.beginPath();
          ctx.arc(p.x, p.y, icon / 2 + 3, 0, Math.PI * 2);
          ctx.stroke();
        }
        ctx.globalAlpha = 1;

        // Étiquette : contour sombre puis texte, lisible quel que soit ce qui passe derrière. Son
        // opacité = fondu × pâleur de profondeur (le fond reste en retrait, jamais illisible).
        const c = candidateOf.get(n.id)!;
        const { lines, font } = labelLines.get(n.id)!;
        const fade = labelAlphaRef.current.get(n.id) ?? 0;
        const labelAlpha = fade * (isFocus ? 1 : 0.55 + 0.45 * k);
        if (labelAlpha > 0.01) {
          ctx.font = `500 ${font}px Inter, system-ui, sans-serif`;
          ctx.lineWidth = 3;
          ctx.strokeStyle = `rgba(4, 6, 11, ${0.85 * labelAlpha})`;
          ctx.fillStyle = isFocus ? `rgba(${GOLD}, 1)` : `rgba(239, 233, 220, ${labelAlpha})`;
          lines.forEach((line, i) => {
            ctx.strokeText(line, p.x, c.label.top + i * font * 1.2);
            ctx.fillText(line, p.x, c.label.top + i * font * 1.2);
          });
        }

        // Cible de clic : icône, plus l'étiquette quand elle est affichée ; z-index vers l'avant.
        if (button) {
          const showing = shown.has(n.id);
          const box: Rect = showing
            ? { left: Math.min(c.icon.left, c.label.left), top: c.icon.top, right: Math.max(c.icon.right, c.label.right), bottom: c.label.bottom }
            : c.icon;
          button.style.visibility = 'visible';
          button.style.width = `${(box.right - box.left).toFixed(1)}px`;
          button.style.height = `${(box.bottom - box.top).toFixed(1)}px`;
          button.style.transform = `translate(${box.left.toFixed(1)}px, ${box.top.toFixed(1)}px)`;
          button.style.zIndex = String(rank + 1);
          button.dataset.label = showing ? 'shown' : 'hidden';
          button.dataset.labelBox = boxAttr(c.label);
          button.dataset.iconBox = boxAttr(c.icon);
        }
      });

      stage.dataset.camera = [cam.yaw, cam.pitch, cam.distance, cam.targetY].map((v) => v.toFixed(3)).join('|');
      stage.dataset.labelsSettled = fading ? 'false' : 'true';
      // `data-ready` = la première image est DESSINÉE, pas seulement le composant monté : un
      // `data-ready` posé au montage annonçait une scène vide (mesuré sur la première version).
      if (!readyRef.current) {
        readyRef.current = true;
        setReady(true);
      }
      return fading;
    },
    [apexY, baseRadius, coneRadius, degree, montagne.links, montagne.nodes, rings, strata, strataLabels],
  );

  // ------------------------------------------------------------------------------------------
  // Boucle : transitions du rail, fondus d'étiquettes, ou une seule image à la demande
  // ------------------------------------------------------------------------------------------
  const tick = useCallback(
    (now: number) => {
      rafRef.current = 0;
      let again = false;
      if (tweenRef.current) {
        const tw = tweenRef.current;
        const u = Math.min(1, (now - tw.start) / RAIL_TWEEN_MS);
        const e = u * u * (3 - 2 * u);
        cameraRef.current = {
          yaw: tw.from.yaw + (tw.to.yaw - tw.from.yaw) * e,
          pitch: tw.from.pitch + (tw.to.pitch - tw.from.pitch) * e,
          distance: tw.from.distance + (tw.to.distance - tw.from.distance) * e,
          targetY: tw.from.targetY + (tw.to.targetY - tw.from.targetY) * e,
        };
        if (u >= 1) tweenRef.current = null;
        else again = true;
      }
      if (draw(now)) again = true;
      if (again) rafRef.current = requestAnimationFrame(tick);
    },
    [draw],
  );

  const requestDraw = useCallback(() => {
    if (!rafRef.current) rafRef.current = requestAnimationFrame(tick);
  }, [tick]);

  /** Applique une modification de caméra (bornée), puis redessine. Un geste annule la transition
   * du rail en cours : la main est à l'utilisateur. */
  const moveCamera = useCallback(
    (change: (c: Camera) => Camera) => {
      tweenRef.current = null;
      cameraRef.current = clampCamera(change(cameraRef.current), apexY);
      requestDraw();
    },
    [apexY, requestDraw],
  );

  /** Tourner / incliner : la montagne garde sa place dans le cadre (camera.ts › orbit). */
  const rotateCamera = useCallback(
    (dYaw: number, dPitch: number) =>
      moveCamera((c) => {
        const pitch = clampCamera({ ...c, pitch: c.pitch + dPitch }, apexY).pitch;
        return orbit(points, frameRef.current, c, c.yaw + dYaw, pitch);
      }),
    [apexY, moveCamera, points],
  );

  // ------------------------------------------------------------------------------------------
  // Montage : taille, zone sûre, images, pose d'ouverture
  // ------------------------------------------------------------------------------------------
  useEffect(() => {
    const stage = stageRef.current;
    const canvas = canvasRef.current;
    if (!stage || !canvas) return;

    /**
     * Zone sûre = la scène moins ce que couvrent les contrôles. Grand écran : rail en colonne à
     * gauche, aide en bas à droite. Téléphone : aide en haut, rail en rangée en bas. Mesurée dans
     * le DOM plutôt que supposée : un libellé plus long (allemand) élargit le rail, le cadrage suit.
     */
    const measureInsets = (): Insets => {
      const s = stage.getBoundingClientRect();
      const rail = railRef.current?.getBoundingClientRect();
      const hint = hintRef.current?.getBoundingClientRect();
      const pad = 10;
      if (window.matchMedia(WIDE_QUERY).matches) {
        return {
          top: pad,
          right: pad,
          bottom: hint ? Math.max(pad, s.bottom - hint.top + 6) : pad,
          left: rail ? Math.max(pad, rail.right - s.left + 14) : pad,
        };
      }
      return {
        top: hint ? Math.max(pad, hint.bottom - s.top + 6) : pad,
        right: pad,
        bottom: rail ? Math.max(pad, s.bottom - rail.top + 8) : pad,
        left: pad,
      };
    };

    // Taille du canvas = taille de la scène × DPR (plafonné à 2 : au-delà, coût sans gain visible).
    const resize = () => {
      const w = stage.clientWidth;
      const h = stage.clientHeight;
      const dpr = Math.min(window.devicePixelRatio || 1, 2);
      frameRef.current = { vp: { width: w, height: h }, safe: measureInsets() };
      overviewPitchRef.current = overviewPitch(points, frameRef.current, heroYaw);
      canvas.width = Math.round(w * dpr);
      canvas.height = Math.round(h * dpr);
      canvas.style.width = `${w}px`;
      canvas.style.height = `${h}px`;
      requestDraw();
    };
    resize();
    const ro = new ResizeObserver(resize);
    ro.observe(stage);
    if (railRef.current) ro.observe(railRef.current);

    // Icônes : une image par nœud (monogramme SVG en data-URL, ou icône Eikon). Chaque chargement
    // redemande une image : les icônes apparaissent dès qu'elles sont prêtes.
    for (const n of montagne.nodes) {
      if (imagesRef.current.has(n.id)) continue;
      const img = new Image();
      img.decoding = 'async';
      img.onload = () => requestDraw();
      img.src = nodeIconSrc(n);
      imagesRef.current.set(n.id, img);
    }

    // Pose d'ouverture : la vue d'ensemble, face au projet le plus relié.
    cameraRef.current = overviewCamera(points, frameRef.current, heroYaw, overviewPitchRef.current);
    requestDraw();

    return () => {
      ro.disconnect();
      if (rafRef.current) cancelAnimationFrame(rafRef.current);
      rafRef.current = 0;
    };
  }, [heroYaw, montagne.nodes, points, requestDraw]);

  // ------------------------------------------------------------------------------------------
  // Gestes : glisser (souris, doigt), pincer, molette, clavier
  // ------------------------------------------------------------------------------------------
  const pointersRef = useRef(new Map<number, { x: number; y: number }>());
  const dragRef = useRef({ startX: 0, startY: 0, moved: false, pinch: 0 });

  // Pas de setPointerCapture : la capture redirigerait le `click` vers la scène et les boutons des
  // nœuds ne le recevraient plus. Le suivi se fait sur window le temps du geste.
  const onPointerDown = (e: React.PointerEvent) => {
    if (e.button !== 0 && e.pointerType === 'mouse') return;
    pointersRef.current.set(e.pointerId, { x: e.clientX, y: e.clientY });
    if (pointersRef.current.size === 1) {
      dragRef.current = { startX: e.clientX, startY: e.clientY, moved: false, pinch: 0 };
    }
    if (pointersRef.current.size === 2) {
      const [p1, p2] = [...pointersRef.current.values()];
      dragRef.current.pinch = Math.hypot(p1.x - p2.x, p1.y - p2.y);
      dragRef.current.moved = true; // un pincement n'est jamais un clic
    }

    const onMove = (ev: PointerEvent) => {
      const prev = pointersRef.current.get(ev.pointerId);
      if (!prev) return;
      const dx = ev.clientX - prev.x;
      const dy = ev.clientY - prev.y;
      pointersRef.current.set(ev.pointerId, { x: ev.clientX, y: ev.clientY });
      const d = dragRef.current;
      if (Math.hypot(ev.clientX - d.startX, ev.clientY - d.startY) > DRAG_THRESHOLD) d.moved = true;
      if (!d.moved) return;

      if (pointersRef.current.size >= 2) {
        // Pincer : le rapport des écarts entre les doigts règle la distance.
        const [p1, p2] = [...pointersRef.current.values()];
        const gap = Math.hypot(p1.x - p2.x, p1.y - p2.y);
        if (d.pinch > 0 && gap > 0) {
          const ratio = d.pinch / gap;
          moveCamera((c) => ({ ...c, distance: c.distance * ratio }));
        }
        d.pinch = gap;
        return;
      }
      if (ev.shiftKey) {
        // Maj + glisser : monter ou descendre le long de la montagne.
        moveCamera((c) => ({ ...c, targetY: c.targetY + dy * 0.02 }));
      } else {
        rotateCamera(-dx * 0.007, dy * 0.004);
      }
    };
    const onUp = (ev: PointerEvent) => {
      pointersRef.current.delete(ev.pointerId);
      if (pointersRef.current.size === 0) {
        window.removeEventListener('pointermove', onMove);
        window.removeEventListener('pointerup', onUp);
        window.removeEventListener('pointercancel', onUp);
      }
    };
    window.addEventListener('pointermove', onMove);
    window.addEventListener('pointerup', onUp);
    window.addEventListener('pointercancel', onUp);
  };

  // Molette : listener natif non passif (React attache `wheel` en passif → preventDefault ignoré,
  // et la page défilerait en même temps que l'on zoome).
  useEffect(() => {
    const stage = stageRef.current;
    if (!stage) return;
    const onWheel = (e: WheelEvent) => {
      // Dans la fleur ouverte, la molette appartient à la fleur, pas à la caméra derrière.
      if ((e.target as Element).closest('[data-testid="montagne-flower"]')) return;
      e.preventDefault();
      if (e.shiftKey) moveCamera((c) => ({ ...c, targetY: c.targetY - e.deltaY * 0.004 }));
      else moveCamera((c) => ({ ...c, distance: c.distance * Math.exp(e.deltaY * 0.0012) }));
    };
    stage.addEventListener('wheel', onWheel, { passive: false });
    return () => stage.removeEventListener('wheel', onWheel);
  }, [moveCamera]);

  const onKeyDown = (e: React.KeyboardEvent) => {
    const rotations: Record<string, [number, number]> = {
      ArrowLeft: [0.12, 0],
      ArrowRight: [-0.12, 0],
      ArrowUp: [0, 0.06],
      ArrowDown: [0, -0.06],
    };
    // Seulement quand la scène elle-même a le focus : les flèches restent aux boutons sinon.
    if (e.target !== e.currentTarget) return;
    const rotation = rotations[e.key];
    if (rotation) {
      e.preventDefault();
      rotateCamera(...rotation);
      return;
    }
    const actions: Record<string, (c: Camera) => Camera> = {
      PageUp: (c) => ({ ...c, targetY: c.targetY + 0.6 }),
      PageDown: (c) => ({ ...c, targetY: c.targetY - 0.6 }),
      '+': (c) => ({ ...c, distance: c.distance * 0.88 }),
      '=': (c) => ({ ...c, distance: c.distance * 0.88 }),
      '-': (c) => ({ ...c, distance: c.distance / 0.88 }),
    };
    const action = actions[e.key];
    if (action) {
      e.preventDefault();
      moveCamera(action);
    }
  };

  /** Rail des strates : transition douce vers la strate (ou la vue d'ensemble), yaw conservé. */
  const flyTo = (stratum: MontagneStratum | null) => {
    const frame = frameRef.current;
    const yaw = cameraRef.current.yaw;
    const target = stratum
      ? stratumCamera(
          montagne.nodes.filter((n) => n.stratum === stratum.id),
          frame,
          yaw,
        )
      : overviewCamera(points, frame, yaw, overviewPitchRef.current);
    tweenRef.current = { from: cameraRef.current, to: target, start: performance.now() };
    requestDraw();
  };

  const openNode = (node: MontagneNode, e: React.MouseEvent) => {
    // Un glisser commencé sur un nœud ne l'ouvre pas. `detail === 0` : activation clavier
    // (Entrée, Espace), jamais un glisser — le drapeau du dernier geste ne compte pas.
    if (e.detail !== 0 && dragRef.current.moved) return;
    selectedRef.current = node.id;
    setSelected(node);
    requestDraw();
  };

  const closeFlower = useCallback(() => {
    const id = selectedRef.current;
    selectedRef.current = null;
    setSelected(null);
    requestDraw();
    // Le focus revient au nœud qui a ouvert la fleur (navigation clavier).
    if (id) buttonsRef.current.get(id)?.focus();
  }, [requestDraw]);

  return (
    <div
      ref={stageRef}
      data-testid="montagne-stage"
      data-ready={ready ? 'true' : 'false'}
      role="application"
      aria-label={t('stage.label')}
      tabIndex={0}
      onPointerDown={onPointerDown}
      onKeyDown={onKeyDown}
      className="relative h-full w-full touch-none select-none overflow-hidden outline-none focus-visible:ring-1 focus-visible:ring-inset focus-visible:ring-gold/40"
    >
      <canvas ref={canvasRef} aria-hidden="true" data-testid="montagne-canvas" className="absolute inset-0 block" />

      {/* Calque de clic : un bouton transparent par nœud, positionné par draw(). `z-10` en fait un
          contexte d'empilement : les z-index des nœuds (rang de profondeur) restent sous le rail
          (z-200) et la fleur (z-300). */}
      <div className="absolute inset-0 z-10">
        {montagne.nodes.map((n) => (
          <button
            key={n.id}
            type="button"
            ref={(el) => {
              if (el) buttonsRef.current.set(n.id, el);
              else buttonsRef.current.delete(n.id);
            }}
            data-testid="montagne-node"
            data-node-id={n.id}
            aria-label={n.name}
            onClick={(e) => openNode(n, e)}
            onPointerEnter={() => {
              hoverRef.current = n.id;
              requestDraw();
            }}
            onPointerLeave={() => {
              if (hoverRef.current === n.id) hoverRef.current = null;
              requestDraw();
            }}
            style={{ visibility: 'hidden' }}
            className="absolute left-0 top-0 cursor-pointer rounded-md bg-transparent focus:outline-none focus-visible:ring-1 focus-visible:ring-gold/80"
          />
        ))}
      </div>

      {/* Rail des strates : naviguer sans geste fin (téléphone, clavier). Rangée défilante en bas
          sur téléphone (une colonne y mangerait le tiers de la hauteur), colonne à gauche ailleurs. */}
      <nav
        ref={railRef}
        aria-label={t('stage.label')}
        className="absolute inset-x-2 bottom-2 z-[200] flex flex-row gap-1 overflow-x-auto sm:inset-x-auto sm:bottom-auto sm:left-4 sm:top-1/2 sm:-translate-y-1/2 sm:flex-col sm:overflow-visible"
      >
        {[...strata].reverse().map((s) => (
          <button
            key={s.id}
            type="button"
            data-testid="montagne-rail"
            data-stratum={s.id}
            onClick={() => flyTo(s)}
            aria-label={t('stage.goTo', { stratum: strataLabels[s.id] })}
            className="shrink-0 rounded-md border border-white/10 bg-[#0b111c]/80 px-2 py-1 text-left text-[11px] sm:px-2.5 sm:text-xs text-slate-300 backdrop-blur transition-colors hover:border-white/25 hover:text-white focus:outline-none focus-visible:ring-1 focus-visible:ring-gold/70"
          >
            {strataLabels[s.id]}
          </button>
        ))}
        <button
          type="button"
          data-testid="montagne-rail-overview"
          onClick={() => flyTo(null)}
          className="shrink-0 rounded-md border border-white/10 bg-[#0b111c]/80 px-2 py-1 text-left text-[11px] sm:px-2.5 sm:text-xs text-slate-400 backdrop-blur transition-colors hover:border-white/25 hover:text-white focus:outline-none focus-visible:ring-1 focus-visible:ring-gold/70"
        >
          {t('stage.overview')}
        </button>
      </nav>

      {/* Aide : les gestes possibles. En haut sur téléphone (le rail occupe le bas), en bas à
          droite ailleurs. */}
      <p
        ref={hintRef}
        data-testid="montagne-hint"
        className="pointer-events-none absolute right-3 top-2 z-[200] max-w-[90%] text-right text-[11px] text-slate-500 sm:bottom-3 sm:top-auto sm:max-w-[60%]"
      >
        {t('stage.hint')}
      </p>

      {selected && nodeById.has(selected.id) && <MontagneFlower node={selected} onClose={closeFlower} />}
    </div>
  );
}

/**
 * Enveloppe convexe (chaîne monotone d'Andrew), sens trigonométrique. Sert au contour du cône :
 * la projection d'un cône est exactement l'enveloppe de son sommet et de son cercle de base.
 */
function convexHull(points: { x: number; y: number }[]): { x: number; y: number }[] {
  const pts = [...points].sort((a, b) => a.x - b.x || a.y - b.y);
  if (pts.length < 3) return pts;
  const cross = (o: { x: number; y: number }, a: { x: number; y: number }, b: { x: number; y: number }) =>
    (a.x - o.x) * (b.y - o.y) - (a.y - o.y) * (b.x - o.x);
  const lower: { x: number; y: number }[] = [];
  for (const p of pts) {
    while (lower.length >= 2 && cross(lower[lower.length - 2], lower[lower.length - 1], p) <= 0) lower.pop();
    lower.push(p);
  }
  const upper: { x: number; y: number }[] = [];
  for (let i = pts.length - 1; i >= 0; i--) {
    const p = pts[i];
    while (upper.length >= 2 && cross(upper[upper.length - 2], upper[upper.length - 1], p) <= 0) upper.pop();
    upper.push(p);
  }
  return [...lower.slice(0, -1), ...upper.slice(0, -1)];
}
