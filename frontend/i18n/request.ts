import { getRequestConfig } from 'next-intl/server';
import { hasLocale } from 'next-intl';
import { locales, defaultLocale, type Locale } from './config';

// QUOI : fournit la locale + les messages à TOUT rendu côté serveur qui passe par next-intl
//   (Server Components via useTranslations/getTranslations sans locale explicite).
// POURQUOI : next-intl v4 a RETIRÉ le paramètre `locale` de getRequestConfig au profit de
//   `requestLocale` (une Promise). L'ancien code lisait `{ locale }` → toujours `undefined` en v4
//   → fallback systématique sur defaultLocale='fr'. Conséquence : /en (et /de, /it, /zh) rendaient
//   tous leurs Server Components en FRANÇAIS, alors que le provider client (layout via
//   getMessages({locale})) recevait la bonne langue → mismatch serveur/client visible (ex: bouton
//   "Voir mon CV" rendu serveur + messages "View my CV" côté client dans le même HTML). C'est la
//   cause du "/en affiché en FR" remonté en prod.
// COMMENT :
//   1. await requestLocale → la locale du segment [locale] de l'URL (peut être undefined hors route).
//   2. hasLocale (même garde que le layout) valide l'appartenance à la liste blanche `locales`.
//   3. fallback sur defaultLocale uniquement si la locale est absente/invalide (vraie 404 gérée ailleurs).
//   4. on RETOURNE `locale` (obligatoire en v4) + les messages importés pour cette locale.
export default getRequestConfig(async ({ requestLocale }) => {
  const requested = await requestLocale;
  const locale: Locale = hasLocale(locales, requested) ? (requested as Locale) : defaultLocale;

  return {
    locale,
    messages: (await import(`../messages/${locale}.json`)).default
  };
});
