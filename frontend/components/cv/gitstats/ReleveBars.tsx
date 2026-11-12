'use client';

import { useState, type PointerEvent } from 'react';
import { niceMax } from '@/lib/gitstatsReleve';
import { DATA_TINTS } from '../dataPalette';

// ============================================================================
// ReleveBars — barres fines de la maquette A (« Le relevé »), en SVG simple.
// ----------------------------------------------------------------------------
// POURQUOI pas recharts : la maquette veut des barres, une part superposée (GitLab), des barres
// SOUS l'axe (lignes supprimées, même échelle) et un pic annoté. Ce sont quelques rectangles.
// COMMENT, sans mesurer la largeur en JS : le SVG s'étire en X (`preserveAspectRatio="none"`,
// une unité = un jour ou une semaine) et garde ses pixels en Y (viewBox de la hauteur réelle).
// Les traits gardent 1 px (`non-scaling-stroke`) ; tout le TEXTE (graduations, mois, pic) est du
// HTML positionné en pourcentage, que l'étirement ne déforme pas. Hauteur fixe dès le premier
// rendu : rien ne bouge à l'arrivée des données, ni au redimensionnement.
//
// RELEVÉ DE BARRE : survoler (souris) ou toucher / glisser (doigt) une colonne affiche aussitôt sa
// valeur exacte (`BarPoint.title`, déjà traduit) dans la bande haute, à la place du pic, et
// éclaire discrètement la colonne. POURQUOI pas un <title> SVG : l'infobulle native attend ~1 s
// et n'existe pas sur téléphone, l'usage principal du site. COMMENT : un seul gestionnaire de
// pointeur sur le <svg> convertit l'abscisse en indice de barre (le tracé est une grille
// régulière de `n` colonnes) — pas une cible par barre. Le relevé est en position absolue dans
// la boîte de hauteur fixe : il n'ajoute aucun décalage de mise en page.
// ============================================================================

export interface BarPoint {
  date: string;
  up: number;
  /** Part de `up` à superposer (commits GitLab). */
  overlay?: number;
  /** Valeur sous l'axe (lignes supprimées). */
  down?: number;
  /** Relevé de la barre (date + valeur exacte), déjà traduit. */
  title: string;
}

// Marges du tracé : droite = graduations, bas = mois, haut = annotation du pic.
const PAD_RIGHT = 40;
const PAD_BOTTOM = 22;
const PAD_TOP = 26;
// Le pic s'annonce vers la gauche quand il est dans le dernier tiers, sinon il sortirait du cadre.
const PEAK_FLIP = 0.65;
// Un nom de mois trop près du suivant (série qui commence en fin de mois) le chevaucherait.
const MONTH_MIN_GAP = 0.06;

export function ReleveBars({
  points,
  height,
  upColor,
  overlayColor,
  downColor,
  label,
  peakLabel,
  formatShort,
  formatMonth,
  className = '',
}: {
  points: BarPoint[];
  height: number;
  upColor: string;
  overlayColor?: string;
  downColor?: string;
  /** Résumé lu par les lecteurs d'écran (le graphe est une image pour eux). */
  label: string;
  peakLabel: (point: BarPoint) => string;
  formatShort: (n: number) => string;
  formatMonth: (iso: string) => string;
  className?: string;
}) {
  const n = points.length;
  const plotH = height - PAD_TOP - PAD_BOTTOM;
  const max = niceMax(Math.max(0, ...points.map((p) => p.up)));
  const maxDown = downColor ? Math.max(0, ...points.map((p) => p.down ?? 0)) : 0;
  // Même échelle au-dessus et au-dessous : l'axe se place au prorata des deux maxima.
  const scale = plotH / (max + maxDown);
  const axisY = PAD_TOP + max * scale;
  const barW = n > 60 ? 0.62 : 0.7;
  const plotWidth = `calc(100% - ${PAD_RIGHT}px)`;

  const months: { i: number; text: string }[] = [];
  points.forEach((p, i) => {
    const prev = points[i - 1];
    if (!prev || prev.date.slice(0, 7) !== p.date.slice(0, 7)) months.push({ i, text: formatMonth(p.date) });
  });
  if (months.length > 1 && (months[1].i - months[0].i) / n < MONTH_MIN_GAP) months.shift();

  const peak = points.reduce((best, p, i) => (p.up > points[best].up ? i : best), 0);
  const hasPeak = n > 0 && points[peak].up > 0;
  const peakX = peak + 0.5;
  const peakFlipped = peakX / n > PEAK_FLIP;

  // Indice de la colonne relevée, `null` = aucun relevé (le pic annoté reprend sa place).
  const [active, setActive] = useState<number | null>(null);
  // Abscisse du pointeur → colonne. Le <svg> s'étire sur toute la largeur du tracé et chaque
  // colonne y vaut une unité : la proportion suffit, bornée pour les bords exacts.
  const pick = (e: PointerEvent<SVGSVGElement>) => {
    if (n === 0) return;
    const r = e.currentTarget.getBoundingClientRect();
    setActive(Math.min(n - 1, Math.max(0, Math.floor(((e.clientX - r.left) / r.width) * n))));
  };
  const shown = active !== null && active < n ? active : null;
  const readoutFlipped = shown !== null && (shown + 0.5) / n > PEAK_FLIP;

  return (
    <div role="img" aria-label={hasPeak ? `${label} ${peakLabel(points[peak])}` : label} className={`relative ${className}`} style={{ height }}>
      <svg
        aria-hidden="true"
        className="absolute inset-y-0 left-0 h-full overflow-visible"
        // `pan-y` : le défilement vertical de la page reste au doigt, le glissé horizontal
        // parcourt les barres au lieu de déclencher un geste du navigateur.
        style={{ width: plotWidth, touchAction: 'pan-y' }}
        viewBox={`0 0 ${Math.max(n, 1)} ${height}`}
        preserveAspectRatio="none"
        onPointerDown={pick}
        onPointerMove={pick}
        // Souris : quitter le graphe retire le relevé. Doigt : il reste lisible après le
        // lever (le pointeur « quitte » à chaque fin de toucher), jusqu'au prochain toucher.
        onPointerLeave={(e) => { if (e.pointerType === 'mouse') setActive(null); }}
      >
        {/* Fond de capture : le pointeur est reçu partout dans le tracé, même entre les barres. */}
        <rect x={0} y={0} width={Math.max(n, 1)} height={height} fill="transparent" />
        {/* Colonne relevée : un voile clair sous les barres, sur toute la hauteur du tracé. */}
        {shown !== null && (
          <rect data-testid="gitstats-bar-highlight" x={shown} y={PAD_TOP} width={1} height={plotH}
            fill="hsl(var(--foreground))" fillOpacity={0.08} />
        )}
        {/* Grille : deux paliers ronds au-dessus de l'axe. */}
        {[0.5, 1].map((f) => (
          <line key={f} x1={0} x2={n} y1={axisY - max * f * scale} y2={axisY - max * f * scale}
            stroke="hsl(var(--border))" strokeOpacity={0.6} vectorEffect="non-scaling-stroke" />
        ))}
        {points.map((p, i) => {
          const x = i + (1 - barW) / 2;
          const h = p.up * scale;
          const g = (p.overlay ?? 0) * scale;
          const d = (p.down ?? 0) * scale;
          return (
            <g key={p.date}>
              {p.up > 0 && <rect x={x} y={axisY - h} width={barW} height={Math.max(h, 1)} fill={upColor} opacity={downColor ? 0.9 : 0.82} />}
              {overlayColor && g > 0 && <rect x={x} y={axisY - g} width={barW} height={Math.max(g, 1)} fill={overlayColor} />}
              {downColor && d > 0 && <rect x={x} y={axisY + 1} width={barW} height={Math.max(d, 1)} fill={downColor} opacity={0.9} />}
            </g>
          );
        })}
        {months.filter((m) => m.i > 0).map((m) => (
          <line key={m.i} x1={m.i} x2={m.i} y1={axisY} y2={axisY + 4} stroke={DATA_TINTS.rule} vectorEffect="non-scaling-stroke" />
        ))}
        <line x1={0} x2={n} y1={axisY + 0.5} y2={axisY + 0.5} stroke={DATA_TINTS.rule} vectorEffect="non-scaling-stroke" />
        {hasPeak && (
          <line x1={peakX} x2={peakX} y1={axisY - points[peak].up * scale - 4} y2={PAD_TOP - 8}
            stroke={DATA_TINTS.rule} vectorEffect="non-scaling-stroke" />
        )}
      </svg>

      {/* Graduations, à droite du tracé. */}
      {[0.5, 1].map((f) => (
        <span key={f} aria-hidden="true"
          className="absolute right-0 -translate-y-1/2 text-[11px] tabular-nums leading-none text-muted-foreground"
          style={{ top: axisY - max * f * scale }}>
          {formatShort(Math.round(max * f))}
        </span>
      ))}

      {/* Mois, sous l'axe. Transparents au pointeur : le glissé continue sous les libellés. */}
      <div aria-hidden="true" className="pointer-events-none absolute bottom-0 left-0" style={{ width: plotWidth, height: PAD_BOTTOM }}>
        {months.map((m) => (
          <span key={m.i} className="absolute bottom-1 text-[11px] leading-none text-muted-foreground" style={{ left: `${(m.i / n) * 100}%` }}>
            {m.text}
          </span>
        ))}
      </div>

      {/* Relevé de la colonne active, dans la bande haute (celle du pic, qu'il remplace le temps
          du relevé). Fond opaque discret pour rester lisible par-dessus le filet du pic ; pas d'or,
          réservé à l'action principale. aria-hidden : doublon visuel, le résumé reste sur role="img". */}
      {shown !== null && (
        <div aria-hidden="true" className="pointer-events-none absolute left-0 top-0" style={{ width: plotWidth, height: PAD_TOP }}>
          <span
            data-testid="gitstats-bar-readout"
            className="absolute top-0 whitespace-nowrap rounded-sm bg-background/90 px-1.5 py-0.5 text-xs tabular-nums text-foreground"
            style={{
              left: `${((shown + 0.5) / n) * 100}%`,
              transform: readoutFlipped ? 'translateX(calc(-100% - 6px))' : 'translateX(6px)',
            }}
          >
            {points[shown].title}
          </span>
        </div>
      )}

      {/* Pic annoté : le chiffre et sa date, au-dessus de la barre. Masqué pendant un relevé, qui
          occupe la même bande. */}
      {hasPeak && shown === null && (
        <div aria-hidden="true" className="pointer-events-none absolute left-0 top-0" style={{ width: plotWidth, height: PAD_TOP }}>
          <span
            className="absolute top-0.5 whitespace-nowrap text-xs tabular-nums text-foreground/80"
            style={{
              left: `${(peakX / n) * 100}%`,
              transform: peakFlipped ? 'translateX(calc(-100% - 6px))' : 'translateX(6px)',
            }}
          >
            {peakLabel(points[peak])}
          </span>
        </div>
      )}
    </div>
  );
}
