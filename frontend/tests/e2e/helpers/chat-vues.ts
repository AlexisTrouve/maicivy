import type { Page } from '@playwright/test';

/**
 * Ouvre une vue du chat sur téléphone (barre d'onglets sous `lg` : Questions / Conversation / Fiches).
 *
 * QUOI : clique l'onglet `#chat-vue-<vue>` s'il est affiché ; sur grand écran (barre masquée, les
 *   trois panneaux visibles côte à côte), ne fait rien.
 * POURQUOI : depuis la refonte passe 2, un téléphone ne montre qu'UN panneau à la fois. Les
 *   assertions sur une fiche ou une relance restent les mêmes, il faut seulement regarder au bon
 *   endroit — comme le ferait le visiteur.
 * COMMENT : sélecteur par id (indépendant de la langue de la page).
 */
export async function ouvrirVueChat(page: Page, vue: 'questions' | 'conversation' | 'fiches') {
  const onglet = page.locator(`#chat-vue-${vue}`);
  if (await onglet.isVisible()) await onglet.click();
}
