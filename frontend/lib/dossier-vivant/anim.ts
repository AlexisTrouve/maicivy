/**
 * anim.ts — les quelques fonctions d'animation du dossier vivant (mêmes courbes que la maquette).
 * Pures : un état d'écran se calcule à partir d'un instant, jamais d'une minuterie.
 */

export const clamp01 = (x: number) => (x < 0 ? 0 : x > 1 ? 1 : x);

/** Avancement de 0 à 1 d'une transition qui commence à `a` et dure `d`. */
export const prog = (t: number, a: number, d: number) => (d <= 0 ? (t >= a ? 1 : 0) : clamp01((t - a) / d));

export const lerp = (a: number, b: number, k: number) => a + (b - a) * k;

/** Entrée et sortie douces (cubique). */
export const easeIO = (x: number) => (x < 0.5 ? 4 * x * x * x : 1 - Math.pow(-2 * x + 2, 3) / 2);

/** Sortie douce (cubique). */
export const easeOut = (x: number) => 1 - Math.pow(1 - x, 3);

/** Apparition d'un élément (fondu + petite montée) commencée à `at`. */
export function reveal(now: number, at: number, d = 0.45, rise = 6): { opacity: number; transform: string; visibility: 'visible' | 'hidden' } {
  const e = easeOut(prog(now, at, d));
  return { opacity: e, transform: `translateY(${((1 - e) * rise).toFixed(2)}px)`, visibility: e > 0 ? 'visible' : 'hidden' };
}
