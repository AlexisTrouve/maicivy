import { Cormorant_Garamond } from 'next/font/google';

/**
 * fonts.ts — la serif de la montagne : la « voix », les noms des œuvres du sommet, et depuis la
 * refonte (2026-11) les titres h1–h3 de TOUT le site public (`font-heading`, globals.css).
 *
 * POURQUOI cette serif : la maquette retenue (AC-2) pose une voix éditoriale, en grandes capitales
 * serif ; une grotesque la rendrait générique. Le site entier a pris l'univers de la montagne.
 * Chargée par app/[locale]/layout.tsx (variable posée sur <body>) ; next/font l'auto-héberge au
 * build (aucune requête vers Google chez le visiteur).
 * Le chinois n'est pas dans ses glyphes : il retombe sur la serif du système (cf. .serif dans
 * montagne.module.css).
 */
export const montagneSerif = Cormorant_Garamond({
  subsets: ['latin'],
  weight: ['500', '600'],
  variable: '--font-montagne-serif',
  display: 'swap',
});
