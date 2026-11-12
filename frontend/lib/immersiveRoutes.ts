/**
 * immersiveRoutes.ts — les pages qui portent leur PROPRE barre collante sous l'en-tête du site.
 *
 * QUOI : sur ces routes, l'en-tête du site (components/layout/Header) se rend comme partout, mais il
 * ne colle PAS au haut de l'écran : il défile avec la page, et c'est la barre de la page qui colle.
 * POURQUOI : /dossier-vivant a sa barre de rejeu (pause, rejouer, progression), qui doit rester à
 * portée pendant qu'on lit le dossier. Deux barres collantes empilées prendraient 120 px (140 sur
 * téléphone) à toute la lecture ; et un en-tête collant au-dessus d'une barre collante finirait par
 * la recouvrir. L'en-tête reste donc dans le flux : visible en haut de page, puis il laisse la place.
 * Seule exception : téléphone COUCHÉ (≤ 500 px de haut), où ses 64 px empêcheraient l'ouverture de
 * tenir — il y est masqué, la barre de la page (avec le nom d'Alexis) en tient lieu.
 * COMMENT : chemins SANS préfixe de locale, tels que les rend `usePathname` de next-intl.
 */
const IMMERSIVE_ROUTES = ['/dossier-vivant'];

export function isImmersiveRoute(pathname: string): boolean {
  return IMMERSIVE_ROUTES.includes(pathname);
}
