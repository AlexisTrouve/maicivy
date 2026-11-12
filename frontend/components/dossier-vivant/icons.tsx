import styles from './dossier-vivant.module.css';

/**
 * icons.tsx — les glyphes 16 × 16 du dossier vivant (tracés de la maquette validée).
 *
 * POURQUOI pas lucide-react : la maquette dessine des glyphes plus fins et plus petits (14 px,
 * trait 1,8) que lucide, et l'état d'un agent se lit d'abord à son glyphe (cercle = attente, arc
 * qui tourne = travail, coche = terminé) — avant même son libellé. Un glyphe accompagne toujours
 * une couleur d'état (règle du panneau admin : on lit l'état, pas le libellé).
 */
const PATHS = {
  plus: 'M8 3v10M3 8h10',
  x: 'M4.5 4.5l7 7M11.5 4.5l-7 7',
  chev: 'M6 4l4 4-4 4',
  q: 'M6 6.2a2 2 0 1 1 2.6 1.9c-.4.2-.6.5-.6.9v.6M8 11.6v.1',
  check: 'M3.5 8.5l3 3 6-7',
  down: 'M3 4h7a3 3 0 0 1 3 3v5M10 9.5l3 3 3-3',
  up: 'M13 12H6a3 3 0 0 1-3-3V4M0 6.5l3-3 3 3',
  note: 'M3 5h10M3 8h10M3 11h6',
  arc: 'M8 3a5 5 0 1 1-5 5',
  dl: 'M8 2.5v8M4.5 7.5L8 11l3.5-3.5M3 13.5h10',
  mail: 'M2.5 4h11v8h-11zM2.5 4.5L8 9l5.5-4.5',
  pause: 'M5.5 3.5v9M10.5 3.5v9',
  play: 'M5 3.5l8 4.5-8 4.5z',
  replay: 'M3 8a5 5 0 1 0 1.6-3.7M3 2.5v3h3',
  arrow: 'M3 8h10M9 4l4 4-4 4',
} as const;

export type IconName = keyof typeof PATHS | 'wait';

export function Icon({ name, rotate }: { name: IconName; rotate?: number }) {
  return (
    <svg className={styles.ico} viewBox="0 0 16 16" aria-hidden="true" style={rotate !== undefined ? { transform: `rotate(${rotate}deg)` } : undefined}>
      {name === 'wait' ? <circle cx="8" cy="8" r="5" /> : <path d={PATHS[name]} />}
    </svg>
  );
}
