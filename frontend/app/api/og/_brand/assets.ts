/**
 * Ressources binaires des images générées, chargées une fois par isolat.
 *
 * POURQUOI `fetch(new URL('./…', import.meta.url))` : c'est la seule forme que le bundler de Next
 * sait embarquer dans une route edge (pas de `fs`). Le chemin DOIT rester littéral et relatif à CE
 * fichier, sinon l'asset n'est pas copié dans le build et le fetch échoue en production seulement.
 *
 * POURQUOI la police est chargée explicitement : sans elle, satori retombe EN SILENCE sur sa
 * Noto Sans embarquée — l'aperçu sortirait en sans-serif, sans erreur. Cormorant Garamond SemiBold
 * est le TTF du backend (backend/fonts, OFL — licence copiée à côté), seule graisse utilisée.
 *
 * Promesses mémorisées au niveau du module : chaque aperçu servi re-lirait sinon 774 kB de TTF.
 */
let cormorant: Promise<ArrayBuffer> | undefined;
let montagneArt: Promise<ArrayBuffer> | undefined;

export function loadCormorant(): Promise<ArrayBuffer> {
  cormorant ??= fetch(new URL('./CormorantGaramond-SemiBold.ttf', import.meta.url)).then((r) => r.arrayBuffer());
  return cormorant;
}

/**
 * Fond peint de l'accueil (public/montagne/art/fond-desktop.webp), ramené à 1100 px de large en
 * JPEG q82 (66 kB) : satori ne décode pas le WebP, et la source 1536 px ne ferait que ralentir.
 */
export function loadMontagneArt(): Promise<ArrayBuffer> {
  montagneArt ??= fetch(new URL('./montagne-og.jpg', import.meta.url)).then((r) => r.arrayBuffer());
  return montagneArt;
}

/** Déclaration de police commune à toutes les ImageResponse de la marque. */
export async function brandFonts() {
  return [{ name: 'Cormorant', data: await loadCormorant(), weight: 600 as const, style: 'normal' as const }];
}
