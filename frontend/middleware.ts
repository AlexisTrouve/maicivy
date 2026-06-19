import createMiddleware from 'next-intl/middleware';
import { NextRequest, NextResponse } from 'next/server';
import { locales, defaultLocale } from './i18n/config';

// Nom du cookie de préférence de langue (écrit par next-intl à chaque visite d'une route préfixée,
// ex: aller sur /en pose NEXT_LOCALE=en). C'est notre source de vérité du choix explicite du visiteur.
const LOCALE_COOKIE = 'NEXT_LOCALE';

// Middleware next-intl standard.
// localeDetection:false (volontaire) → on ne devine PAS la langue depuis l'Accept-Language du
// navigateur : un PREMIER visiteur sans préférence atterrit en FR (défaut assumé d'Alexi). La
// détection « intelligente » par langue du navigateur n'est pas souhaitée ici.
const intlMiddleware = createMiddleware({
  locales,
  defaultLocale,
  localePrefix: 'always',
  localeDetection: false,
});

// QUOI : aiguille chaque requête vers la bonne locale, en respectant le choix explicite du visiteur.
// POURQUOI : avec localeDetection:false seul, next-intl redirige TOUTE entrée sans préfixe (`/`,
//   `/cv`, …) vers defaultLocale='fr' — y compris pour quelqu'un qui avait EXPLICITEMENT choisi
//   l'anglais (cookie NEXT_LOCALE=en), et il réécrivait même ce cookie en 'fr'. Conséquence concrète :
//   impossible de « rester » en anglais en revenant par la racine (ex: rouvrir le lien partagé) → le
//   choix de langue ne collait pas. C'est le bug remonté (femme d'Alexi bloquée en FR malgré EN choisi).
// COMMENT :
//   1. URL déjà préfixée par une locale valide (/en, /fr/cv, …) → on ne touche à rien : l'URL explicite
//      prime (taper /fr doit donner FR même si le cookie dit 'en'). next-intl gère + maintient le cookie.
//   2. Entrée SANS préfixe + cookie NEXT_LOCALE valide → redirection vers CETTE locale (le choix colle).
//      On ne regarde toujours PAS l'Accept-Language : FR reste le défaut des nouveaux visiteurs.
//   3. Entrée sans préfixe + pas de cookie → next-intl applique le défaut FR (comportement inchangé).
export default function middleware(request: NextRequest) {
  const { pathname } = request.nextUrl;

  // 1. La requête vise-t-elle déjà une route préfixée par une locale connue ?
  const hasLocalePrefix = locales.some(
    (l) => pathname === `/${l}` || pathname.startsWith(`/${l}/`)
  );

  if (!hasLocalePrefix) {
    // 2. Pas de préfixe → on respecte le choix explicite stocké en cookie, s'il est valide.
    const cookieLocale = request.cookies.get(LOCALE_COOKIE)?.value;
    if (cookieLocale && (locales as readonly string[]).includes(cookieLocale)) {
      const url = request.nextUrl.clone();
      // racine '/' → '/<locale>' ; '/cv' → '/<locale>/cv' (pas de double slash sur la racine)
      url.pathname = `/${cookieLocale}${pathname === '/' ? '' : pathname}`;
      // 307 (temporaire) : dépend du cookie, pas une redirection permanente.
      return NextResponse.redirect(url);
    }
  }

  // 3. Cas par défaut : on délègue à next-intl (préfixe présent, ou pas de cookie → défaut FR).
  return intlMiddleware(request);
}

export const config = {
  // Exclude api, _next, _vercel, static files, and special routes (api-test)
  matcher: ['/((?!api|_next|_vercel|api-test|.*\\..*).*)']
};
