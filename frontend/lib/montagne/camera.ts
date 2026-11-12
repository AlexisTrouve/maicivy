/**
 * camera.ts — la caméra orbitale de la montagne et sa projection 3D → écran. Pur calcul, aucun
 * DOM : testable en isolation et partagé par le rendu et la détection des clics.
 *
 * Repère monde (celui du backend, montagne_layout.go) : y vers le haut, de 0 (pied) au sommet
 * (`shape.apex_y`) ; (x, z) le plan horizontal. Les projets reposent sur un cône d'axe vertical.
 *
 * La caméra orbite autour d'un point de visée (0, targetY, 0) : `yaw` tourne autour de l'axe
 * vertical, `pitch` lève le regard (0 = à l'horizontale, positif = au-dessus), `distance` recule.
 *
 * CADRAGE (refonte du 2026-10-30) : la vue d'ensemble, la rotation libre et la vue d'une
 * strate sont AJUSTÉES sur les nœuds projetés (fitCamera), pas sur une boîte englobante théorique.
 * Mesuré sur la première version : un cadrage par « rayon × hauteur » laissait la strate du bas
 * sortir de l'écran, et le survol zoomé coupait des nœuds aux bords. Ici, chaque nœud (icône +
 * une ligne d'étiquette) tient dans la zone sûre — la scène moins le rail et l'aide.
 */

export interface Camera {
  yaw: number;
  pitch: number;
  distance: number;
  targetY: number;
}

export interface Viewport {
  width: number;
  height: number;
}

/** Marges (px) de la scène occupées par des contrôles : rien d'utile ne doit s'y dessiner. */
export interface Insets {
  top: number;
  right: number;
  bottom: number;
  left: number;
}

/** Ce que la caméra cadre : la taille de la scène et sa zone sûre. */
export interface Frame {
  vp: Viewport;
  safe: Insets;
}

export interface Point3 {
  x: number;
  y: number;
  z: number;
}

export interface Projected {
  /** Coordonnées écran en px CSS. */
  x: number;
  y: number;
  /** Facteur de perspective (px par unité monde à cette profondeur). */
  scale: number;
  /** Distance le long de l'axe de visée : plus grand = plus loin. */
  depth: number;
  /** Faux si le point est derrière la caméra (ou trop près pour être dessiné). */
  visible: boolean;
}

/** Bornes de la caméra libre : on ne passe ni sous terre, ni dans la montagne, ni dans l'espace. */
export const CAMERA_LIMITS = {
  pitchMin: -0.15,
  pitchMax: 1.25,
  distanceMin: 4,
  distanceMax: 90,
} as const;

/** Où se trouve, verticalement, le point visé à l'écran. Le cadrage recentre de toute façon. */
const HORIZON_RATIO = 0.5;

/** Focale en px : proportionnelle au plus petit côté, pour un rendu comparable portrait/paysage. */
export function focalLength(vp: Viewport): number {
  return Math.min(vp.width, vp.height) * 1.1;
}

/** Taille d'icône (px) à une échelle de perspective donnée — partagée par le rendu et le cadrage. */
export function iconSize(scale: number): number {
  return Math.min(56, Math.max(14, scale * 0.5));
}

/** Taille de police d'étiquette (px) à une échelle donnée. */
export function labelFontSize(scale: number): number {
  return Math.min(14, Math.max(10, scale * 0.24));
}

/** Bornes de navigation libre ; `apexY` = hauteur du sommet (shape.apex_y du backend). */
export function clampCamera(c: Camera, apexY: number): Camera {
  const L = CAMERA_LIMITS;
  return {
    yaw: c.yaw,
    pitch: Math.min(L.pitchMax, Math.max(L.pitchMin, c.pitch)),
    distance: Math.min(L.distanceMax, Math.max(L.distanceMin, c.distance)),
    targetY: Math.min(apexY, Math.max(0, c.targetY)),
  };
}

/**
 * Renvoie une fonction de projection pour cette caméra et ce viewport. Les trigonométries sont
 * calculées une fois par image, pas par point (75 nœuds + liens + contours à chaque frame).
 *
 * Base caméra : droite = (cos yaw, 0, −sin yaw), haut = (−sin pitch·sin yaw, cos pitch,
 * −sin pitch·cos yaw), avant = −(cos pitch·sin yaw, sin pitch, cos pitch·cos yaw). Les trois sont
 * orthonormés ; yaw = pitch = 0 place la caméra sur +z, regardant vers −z, x vers la droite.
 */
export function makeProjector(cam: Camera, vp: Viewport): (p: Point3) => Projected {
  const cy = Math.cos(cam.yaw);
  const sy = Math.sin(cam.yaw);
  const cp = Math.cos(cam.pitch);
  const sp = Math.sin(cam.pitch);
  const camX = cam.distance * cp * sy;
  const camY = cam.targetY + cam.distance * sp;
  const camZ = cam.distance * cp * cy;
  const focal = focalLength(vp);
  const cx = vp.width / 2;
  const cyScreen = vp.height * HORIZON_RATIO;

  return (p) => {
    const dx = p.x - camX;
    const dy = p.y - camY;
    const dz = p.z - camZ;
    const right = dx * cy - dz * sy;
    const up = -dx * sp * sy + dy * cp - dz * sp * cy;
    const depth = -(dx * cp * sy + dy * sp + dz * cp * cy);
    if (depth < 0.4) {
      return { x: 0, y: 0, scale: 0, depth, visible: false };
    }
    const scale = focal / depth;
    return { x: cx + right * scale, y: cyScreen - up * scale, scale, depth, visible: true };
  };
}

/**
 * Emprise écran d'un nœud projeté : son icône, plus une ligne d'étiquette sous elle. C'est ce que
 * le cadrage doit faire tenir — l'étiquette d'un nœud du bord peut, elle, s'estomper si elle
 * déborde (labels.ts), l'icône jamais.
 */
export function nodeFootprint(p: Projected): { left: number; right: number; top: number; bottom: number } {
  const half = iconSize(p.scale) / 2;
  return {
    left: p.x - half - 2,
    right: p.x + half + 2,
    top: p.y - half - 2,
    bottom: p.y + half + 4 + labelFontSize(p.scale) * 1.25,
  };
}

/**
 * Cadre `points` dans la zone sûre de `frame`, pour un yaw et un pitch donnés : rend la distance
 * et la hauteur de visée qui font tenir toutes leurs emprises (nodeFootprint).
 *
 * COMMENT : la perspective rend le problème non linéaire, on itère. À chaque pas : projeter,
 * prendre la boîte des emprises, multiplier la distance par le rapport « taille de la boîte /
 * taille disponible » (la taille à l'écran varie comme 1/distance), puis déplacer la visée pour
 * centrer la boîte verticalement (un déplacement Δ de la visée décale l'écran de Δ·cos(pitch)·
 * échelle). Horizontalement la montagne est symétrique autour de son axe : on borne par le plus
 * grand des deux débords autour du centre. 14 pas suffisent (mesuré : < 0,5 px d'écart au-delà
 * de 8) ; un dernier facteur 1,01 absorbe l'erreur résiduelle.
 */
export function fitCamera(points: Point3[], frame: Frame, yaw: number, pitch: number): Camera {
  const { vp, safe } = frame;
  const availW = Math.max(40, vp.width - safe.left - safe.right);
  const availH = Math.max(40, vp.height - safe.top - safe.bottom);
  const centerX = safe.left + availW / 2;
  const centerY = safe.top + availH / 2;
  const ys = points.map((p) => p.y);
  let cam: Camera = {
    yaw,
    pitch,
    distance: 30,
    targetY: (Math.min(...ys) + Math.max(...ys)) / 2,
  };
  for (let i = 0; i < 14; i++) {
    const project = makeProjector(cam, vp);
    let left = Infinity;
    let right = -Infinity;
    let top = Infinity;
    let bottom = -Infinity;
    let scaleSum = 0;
    for (const pt of points) {
      const p = project(pt);
      if (!p.visible) continue;
      const f = nodeFootprint(p);
      left = Math.min(left, f.left);
      right = Math.max(right, f.right);
      top = Math.min(top, f.top);
      bottom = Math.max(bottom, f.bottom);
      scaleSum += p.scale;
    }
    if (!Number.isFinite(left)) {
      cam = { ...cam, distance: cam.distance * 2 };
      continue;
    }
    const halfW = Math.max(centerX - left, right - centerX);
    const ratio = Math.max((2 * halfW) / availW, (bottom - top) / availH);
    const meanScale = scaleSum / points.length;
    const shiftPx = (top + bottom) / 2 - centerY;
    cam = {
      ...cam,
      targetY: cam.targetY - shiftPx / (meanScale * Math.cos(pitch)),
      distance: cam.distance * ratio,
    };
  }
  return { ...cam, distance: cam.distance * 1.01 };
}

/**
 * Pitch de la vue d'ensemble : assez en plongée pour que les courbes de niveau s'ouvrent (l'avant
 * d'un anneau passe sous l'arrière), pas trop pour que l'ellipse du pied ne mange pas la hauteur —
 * c'est elle qui limite le cadrage en paysage.
 */
export const OVERVIEW_PITCH = 0.26;
/** Pitch maximal de la vue d'ensemble : au-delà, on regarde la montagne d'avion, plus de face. */
const OVERVIEW_PITCH_MAX = 0.85;

/**
 * Pitch de la vue d'ensemble pour ce cadre. Paysage : OVERVIEW_PITCH. Portrait : la montagne est
 * limitée par sa LARGEUR et laisse du vide en haut et en bas (mesuré sur 393×851 : ~300 px vides,
 * 9 étiquettes affichées). On lève alors le regard tant que le cadrage ne recule pas (distance
 * ≤ 1,02 × celle du pitch de base) : les courbes de niveau s'ouvrent dans la hauteur libre,
 * l'avant et l'arrière d'un anneau se séparent, et leurs étiquettes trouvent leur place — sans
 * rien perdre en taille. Calcul coûteux (une quinzaine de cadrages) : à faire au redimensionnement,
 * pas à chaque image.
 */
export function overviewPitch(points: Point3[], frame: Frame, yaw: number): number {
  const base = fitCamera(points, frame, yaw, OVERVIEW_PITCH).distance;
  let best = OVERVIEW_PITCH;
  for (let pitch = OVERVIEW_PITCH + 0.04; pitch <= OVERVIEW_PITCH_MAX + 1e-9; pitch += 0.04) {
    if (fitCamera(points, frame, yaw, pitch).distance > base * 1.02) break;
    best = pitch;
  }
  return best;
}

/** Vue d'ensemble : toute la montagne, cadrée sur tous ses nœuds — la pose d'ouverture. */
export function overviewCamera(points: Point3[], frame: Frame, yaw: number, pitch = OVERVIEW_PITCH): Camera {
  return fitCamera(points, frame, yaw, pitch);
}

/**
 * Tourner sans perdre le cadre : la pose après une rotation (yaw, pitch) garde le MÊME rapport au
 * cadrage ajusté que la pose de départ — même facteur de zoom (distance / distance ajustée) et
 * même décalage de visée. POURQUOI : tourner ou incliner une montagne déplace son emprise à
 * l'écran (en plongée, l'ellipse du pied descend) ; mesuré sur la première version, un simple
 * glisser suffisait à faire sortir la strate du bas. Ici, sans zoom de l'utilisateur, toute la
 * montagne reste dans le cadre quel que soit l'angle ; zoomée, elle le reste dans la même mesure.
 */
export function orbit(points: Point3[], frame: Frame, from: Camera, yaw: number, pitch: number): Camera {
  const before = fitCamera(points, frame, from.yaw, from.pitch);
  const after = fitCamera(points, frame, yaw, pitch);
  return {
    yaw,
    pitch,
    distance: after.distance * (from.distance / before.distance),
    targetY: after.targetY + (from.targetY - before.targetY),
  };
}

/**
 * Yaw qui met un nœud face à la caméra. La caméra est en (sin yaw, ·, cos yaw)·d ; un nœud d'angle
 * φ = atan2(z, x) lui fait face quand (cos φ, sin φ) ∥ (sin yaw, cos yaw), soit yaw = π/2 − φ.
 */
export function facingYaw(p: Point3): number {
  return Math.PI / 2 - Math.atan2(p.z, p.x);
}

/** Vue d'une strate (rail) : cadrée sur les seuls nœuds de cette strate. */
export function stratumCamera(points: Point3[], frame: Frame, yaw: number): Camera {
  return fitCamera(points, frame, yaw, 0.22);
}
