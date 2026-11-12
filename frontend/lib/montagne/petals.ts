/**
 * petals.ts — où poser les pétales de la fleur autour de la carte du projet. Pur calcul.
 *
 * Deux dispositions, choisies par la forme de l'écran :
 *  - PAYSAGE (ou carré) : ellipse autour de la carte, premier pétale en haut, sens horaire. Les
 *    demi-axes sont au moins « demi-carte + demi-pétale + marge » : aucun pétale ne mord la carte.
 *  - PORTRAIT (téléphone) : il n'y a pas la largeur pour tourner autour de la carte ; les pétales
 *    vont en une rangée au-dessus et une rangée en dessous, centrées.
 * La taille des pétales est bornée par la place réellement disponible, pour que tout tienne.
 */

export interface Size {
  width: number;
  height: number;
}

export interface PetalLayout {
  /** Côté d'un pétale (carré), px. */
  size: number;
  /** Centre de chaque pétale, px, dans le repère du calque. */
  positions: { x: number; y: number }[];
}

const GAP = 12;
const MAX_PETAL = 168;
const MIN_PETAL = 64;

export function computePetalLayout(count: number, viewport: Size, card: Size): PetalLayout {
  if (count === 0) return { size: 0, positions: [] };
  const cx = viewport.width / 2;
  const cy = viewport.height / 2;
  const portrait = viewport.height > viewport.width * 1.15;

  if (portrait) {
    const top = Math.ceil(count / 2);
    const bottom = count - top;
    // Place verticale libre au-dessus (et au-dessous) de la carte.
    const free = (viewport.height - card.height) / 2 - 2 * GAP;
    const perRow = Math.max(top, 1);
    const byWidth = (viewport.width - GAP * (perRow + 1)) / perRow;
    const size = clamp(Math.min(byWidth, free, MAX_PETAL), MIN_PETAL / 2, MAX_PETAL);
    const row = (n: number, y: number) =>
      Array.from({ length: n }, (_, i) => ({ x: cx + (i - (n - 1) / 2) * (size + GAP), y }));
    return {
      size,
      positions: [
        ...row(top, cy - card.height / 2 - GAP - size / 2),
        ...row(bottom, cy + card.height / 2 + GAP + size / 2),
      ],
    };
  }

  // Paysage : la taille d'abord, bornée par la place sur les côtés et en haut/bas.
  const side = (viewport.width - card.width) / 2 - 2 * GAP;
  const vertical = (viewport.height - card.height) / 2 - 2 * GAP;
  const size = clamp(Math.min(MAX_PETAL, Math.max(side, vertical), viewport.height / 3.2), MIN_PETAL, MAX_PETAL);
  // Demi-axes minimaux qui dégagent la carte d'un pétale entier.
  const ax = card.width / 2 + size / 2 + GAP * 2;
  const ay = card.height / 2 + size / 2 + GAP * 2;
  const positions = Array.from({ length: count }, (_, i) => {
    const angle = -Math.PI / 2 + (i * 2 * Math.PI) / count;
    const c = Math.cos(angle);
    const s = Math.sin(angle);
    // Une ellipse de demi-axes (ax, ay) MORD la carte en diagonale (à 30°, le pétale n'est ni
    // assez loin en x ni assez loin en y). On pousse chaque pétale jusqu'au rectangle (ax, ay) :
    // facteur 1 / max(|cos|, |sin|) ; au moins une des deux distances est alors dégagée.
    const k = 1 / Math.max(Math.abs(c), Math.abs(s));
    return {
      x: clamp(cx + ax * c * k, size / 2 + GAP, viewport.width - size / 2 - GAP),
      y: clamp(cy + ay * s * k, size / 2 + GAP, viewport.height - size / 2 - GAP),
    };
  });
  return { size, positions };
}

function clamp(v: number, min: number, max: number): number {
  return Math.min(max, Math.max(min, v));
}
