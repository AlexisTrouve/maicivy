import type { Page } from '@playwright/test';

// ============================================================================
// Helper E2E partagé — collecte des VRAIES exceptions JS de l'app, sans le bruit réseau.
// ----------------------------------------------------------------------------
// POURQUOI : tous les specs vérifient « zéro exception JS » via `page.on('pageerror')` puis
// `expect(errors).toHaveLength(0)`. Or WebKit/Safari remontent aussi, via ce MÊME event, les requêtes
// cross-origin bloquées (CORS/CSP) — beacon Cloudflare RUM (`/cdn-cgi/rum`), profil maiProFiles (autre
// sous-domaine), prefetch RSC Next avorté à la navigation, heartbeat — avec le message
// « ...due to access control checks. ». Ce ne sont PAS des bugs applicatifs : c'est du bruit
// environnemental propre à WebKit. Sans filtre, les specs flakent sur webkit/Mobile Safari alors que
// l'app va bien. On centralise ici la définition de « ce qui compte comme une erreur » → source unique
// de vérité, plutôt qu'un filtre copié dans 11 fichiers.
// ============================================================================

// isEnvironmentalPageError — vrai si le message est du bruit réseau cross-origin WebKit (à ignorer),
// PAS une exception JS de l'app. Ciblé sur la formule exacte de WebKit (« access control checks ») :
// une vraie erreur applicative (TypeError, ReferenceError…) ne contient jamais cette phrase, donc on ne
// risque pas de masquer un vrai bug.
export function isEnvironmentalPageError(message: string): boolean {
  return /access control checks/i.test(message);
}

// trackPageErrors — branche un collecteur d'exceptions JS sur la page et renvoie le tableau (vivant) des
// VRAIES erreurs applicatives (le bruit cross-origin WebKit est filtré à la source). À asserter en fin
// de test : `expect(errors, 'exceptions JS:\n' + errors.join('\n')).toHaveLength(0)`.
export function trackPageErrors(page: Page): string[] {
  const errors: string[] = [];
  page.on('pageerror', (e) => {
    if (isEnvironmentalPageError(e.message)) return;
    errors.push(e.message);
  });
  return errors;
}
