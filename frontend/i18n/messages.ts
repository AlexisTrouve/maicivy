import fr from '@/messages/fr.json';
import en from '@/messages/en.json';
import de from '@/messages/de.json';
import it from '@/messages/it.json';
import zh from '@/messages/zh.json';
import { defaultLocale } from './config';

// ============================================================================
// loadMessages — chargement des messages i18n compatible Turbopack.
// ----------------------------------------------------------------------------
// POURQUOI : le pattern next-intl par défaut `(await import(`@/messages/${locale}.json`)).default`
//   (import dynamique à template) casse `next dev --turbo` : Turbopack ne peut pas savoir, à la
//   compilation, quels JSON bundler pour un `${locale}` runtime → "Module not found" → TOUT le dev
//   local plante (le layout [locale] est traversé par chaque page). Webpack (next build/prod) gère
//   via un context module, d'où "prod OK / dev mort".
// COMMENT : avec un jeu de locales FIXE (5), on importe les 5 JSON STATIQUEMENT (résolus à la
//   compilation → turbopack-safe) et on indexe par locale. Bonus : typé `typeof fr` (vs `any` du
//   dynamic import) → accès `messages.cv.themes[…]` vérifié par TS.
// ============================================================================

// Type canonique des messages = la structure du FR (source de vérité i18n).
type Messages = typeof fr;

// en/de/it/zh partagent la structure de fr → cast assumé (FR = source, les autres en dérivent).
const MESSAGES = { fr, en, de, it, zh } as Record<string, Messages>;

// Renvoie les messages d'une locale ; fallback sur la locale par défaut si inconnue.
export function loadMessages(locale: string): Messages {
  return MESSAGES[locale] ?? MESSAGES[defaultLocale];
}
