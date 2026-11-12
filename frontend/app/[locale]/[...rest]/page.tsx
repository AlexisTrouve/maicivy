import { notFound } from 'next/navigation';

/*
 * QUOI : attrape toute URL inconnue SOUS une locale (`/fr/xyz`, `/en/a/b`) et déclenche le 404.
 * POURQUOI : sans cette route, Next ne trouve aucun segment correspondant et sert le `not-found`
 *   RACINE (`app/not-found.tsx`), hors du layout localisé : ni en-tête, ni pied, ni langue du
 *   visiteur. Appeler `notFound()` ICI fait remonter au `[locale]/not-found.tsx`, rendu dans la
 *   chrome du site et traduit. C'est le motif documenté par next-intl.
 * COMMENT : les routes explicites (blog, cv…) restent prioritaires sur un catch-all, donc rien
 *   d'existant n'est intercepté.
 */
export default function CatchAll() {
  notFound();
}
