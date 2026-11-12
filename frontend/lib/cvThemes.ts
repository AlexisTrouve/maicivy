// ============================================================================
// Profils (thèmes) du CV — défaut, ordre d'affichage, profils connus de l'interface.
// ----------------------------------------------------------------------------
// QUOI : la source unique, côté front, de « quel profil par défaut » et « quels profils ont un
// libellé traduit » (messages › cv.themes.<id>.name / .description).
//
// POURQUOI le défaut est `ai` : le site est repositionné « Agent Orchestrator — Technomancer ». Le
// backend sert déjà ce profil (`GET /cv/themes` le liste en tête, `/cv?theme=ai` et
// `/cv/export?theme=ai` répondent) : aucune modification serveur n'est nécessaire, seul le repli
// du front, qui valait `fullstack`, contredisait le positionnement.
//
// POURQUOI des libellés traduits plutôt que `name`/`description` de l'API : l'API renvoie un
// mélange d'anglais et de français (« Backend Developer » / « Focus sur développement backend »).
// Un visiteur sur /en lisait du français dans le menu. Un profil INCONNU de cette liste (ajouté au
// backend plus tard) reste affiché avec le nom de l'API : il apparaît, simplement non traduit.
// ============================================================================

/** Profil servi quand l'URL ne précise pas `?theme=`. */
export const DEFAULT_CV_THEME = 'ai';

/** Profils dont l'interface porte un nom et une description traduits, dans l'ordre du menu. */
export const KNOWN_CV_THEMES = ['ai', 'backend', 'cpp', 'devops', 'fullstack', 'artistique'] as const;
export type KnownCvTheme = (typeof KNOWN_CV_THEMES)[number];

export function isKnownCvTheme(id: string): id is KnownCvTheme {
  return (KNOWN_CV_THEMES as readonly string[]).includes(id);
}

/**
 * Ordonne les profils renvoyés par l'API : les profils connus dans l'ordre de KNOWN_CV_THEMES (le
 * défaut en tête), puis les inconnus dans l'ordre de l'API. Tri STABLE : l'ordre serveur reste le
 * départage, on ne dépend donc pas de lui pour placer le profil par défaut en premier.
 */
export function orderCvThemes<T extends { id: string }>(themes: T[]): T[] {
  const rank = (id: string) => {
    const i = (KNOWN_CV_THEMES as readonly string[]).indexOf(id);
    return i === -1 ? KNOWN_CV_THEMES.length : i;
  };
  return [...themes].sort((a, b) => rank(a.id) - rank(b.id));
}
