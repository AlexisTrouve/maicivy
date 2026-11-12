import type { Page } from '@playwright/test';

// ============================================================================
// Helper E2E — les avertissements React de CLÉ EN DOUBLE, relevés sur la console.
// ----------------------------------------------------------------------------
// POURQUOI : une clé en double n'est pas cosmétique — React peut fusionner ou perdre les deux lignes
// concernées (un message du fil remplacé par un autre). Le défaut ne se voit qu'en dev, dans la
// console, et seulement quand deux éléments tombent sur la même clé : c'est arrivé sur le rejeu d'un
// run réel (`0.032-orchestrator-orchestrator`, deux messages du même paquet SSE), jamais sur le rejeu
// Tessera. Un test ne le voit donc que s'il ÉCOUTE la console pendant le parcours entier.
// COMMENT : React (dev) écrit « Encountered two children with the same key, `…` » en console.error.
// On garde la phrase entière : elle porte la clé fautive, de quoi trouver la liste en cause.
// ============================================================================

const DUPLICATE_KEY = /same key/i;

/** Branche le relevé sur la page ; renvoie le tableau (vivant) des avertissements de clé en double. */
export function trackDuplicateKeys(page: Page): string[] {
  const found: string[] = [];
  page.on('console', (m) => {
    if (m.type() === 'error' && DUPLICATE_KEY.test(m.text())) found.push(m.text().slice(0, 300));
  });
  return found;
}
