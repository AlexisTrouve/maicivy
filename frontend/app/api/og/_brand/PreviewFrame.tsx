import { ImageResponse } from 'next/og';

import { brandFonts, loadMontagneArt } from './assets';
import { PaintedMountain } from './PaintedMountain';
import { gold, night } from './palette';

/** Format des aperçus de lien : celui qu'attendent Open Graph et la carte large de X. */
export const PREVIEW_WIDTH = 1200;
export const PREVIEW_HEIGHT = 630;
/** Largeur de la colonne de texte ; la montagne peinte occupe le reste, à droite. */
export const TEXT_COLUMN_WIDTH = 690;

/** Domaine affiché en pied d'aperçu — celui du site servi, pas une constante recopiée. */
export function siteDomain(): string {
  return new URL(process.env.NEXT_PUBLIC_BASE_URL || 'https://maicivy.etheryale.com').host;
}

/**
 * Rend un aperçu de lien « montagne peinte » (1200 × 630) autour d'une colonne de texte.
 *
 * QUOI : le cadre commun à TOUS les aperçus du site (accueil, blog, activité) — ciel de nuit,
 *   fond peint de l'accueil calé à droite, filet d'or en tête — dans lequel chaque route ne fournit
 *   que sa colonne de texte.
 * POURQUOI : un lien partagé depuis n'importe quelle page doit se reconnaître comme le même site.
 *   Quand chaque route dessinait son propre fond, le blog et l'activité étaient restés en bleu et
 *   violet, avec un monogramme « AT » que le site n'utilise plus.
 * COMMENT : polices et peinture sont chargées une fois par isolat (cf. assets.ts) ; la colonne est
 *   posée APRÈS la peinture pour se peindre au-dessus de son dégradé.
 */
export async function renderPreview(column: React.ReactNode): Promise<ImageResponse> {
  const [fonts, art] = await Promise.all([brandFonts(), loadMontagneArt()]);
  return new ImageResponse(
    (
      <div
        style={{
          width: '100%',
          height: '100%',
          display: 'flex',
          position: 'relative',
          background: `radial-gradient(ellipse at 78% 28%, ${night.band} 0%, ${night.page} 70%)`,
        }}
      >
        <PaintedMountain art={art} width={PREVIEW_WIDTH} height={PREVIEW_HEIGHT} />
        {column}
        {/* Filet d'or en tête, comme l'en-tête du site : 2 px, car 1 px disparaît au tiers. */}
        <div
          style={{
            position: 'absolute',
            left: 0,
            top: 0,
            width: PREVIEW_WIDTH,
            height: 2,
            display: 'flex',
            background: `linear-gradient(90deg, rgba(217,180,95,0) 0%, ${gold.base} 50%, rgba(217,180,95,0) 100%)`,
            opacity: 0.7,
          }}
        />
      </div>
    ),
    { width: PREVIEW_WIDTH, height: PREVIEW_HEIGHT, fonts },
  );
}
