/**
 * navigation.ts — LA source des liens du site : en-tête, menu téléphone et pied de page.
 *
 * QUOI : trois groupes de liens. `PRIMARY_NAV` (l'en-tête et son menu téléphone, et la colonne
 * « Explorer » du pied), `WORKSHOP_NAV` (les coulisses : statistiques, architecture, lettres) et
 * `LEGAL_NAV`. Chaque entrée porte son chemin SANS locale (le Link de @/i18n/navigation la préfixe)
 * et l'identifiant qui sert à la fois de clé de message et de suffixe de data-testid.
 * POURQUOI un seul fichier : avant la refonte, l'en-tête, le pied et le bandeau de la montagne
 * tenaient chacun leur liste — et aucun ne menait au dossier vivant ni à la montagne. Une liste
 * unique ne peut plus diverger.
 * COMMENT : les libellés vivent dans messages/*.json, namespace `nav` (primaire, coulisses) et
 * `footer` (légal). Le test lib/site/__tests__/chromeMessages.test.tsx vérifie les 5 langues.
 */

export interface NavItem {
  /** Clé de message (namespace indiqué par le groupe) et suffixe de data-testid. */
  id: string;
  /** Chemin sans préfixe de locale. */
  href: string;
}

/** En-tête : l'accueil (la montagne), puis les portes principales. */
export const PRIMARY_NAV: readonly NavItem[] = [
  { id: 'home', href: '/' },
  { id: 'dossierVivant', href: '/dossier-vivant' },
  { id: 'portfolio', href: '/portfolio' },
  { id: 'cv', href: '/cv' },
  // Git Stats remonté dans l'en-tête le 09/11/2026 (demande d'Alexis : la preuve d'activité doit
  // se voir sans descendre au pied). Retiré des coulisses en conséquence : le pied affiche aussi
  // PRIMARY_NAV, un doublon donnerait deux data-testid identiques.
  { id: 'gitstats', href: '/gitstats' },
  { id: 'blog', href: '/blog' },
  { id: 'chat', href: '/chat' },
];

/** Pied de page, « coulisses » : ce que le site montre de lui-même. Libellés : namespace `nav`. */
export const WORKSHOP_NAV: readonly NavItem[] = [
  { id: 'analytics', href: '/analytics' },
  { id: 'architecture', href: '/architecture' },
];

/** Pied de page, légal. Libellés : namespace `footer`. */
export const LEGAL_NAV: readonly NavItem[] = [
  { id: 'legalNotice', href: '/legal' },
  { id: 'privacy', href: '/privacy' },
];

/**
 * L'entrée est-elle la page courante ? `pathname` est celui de usePathname (@/i18n/navigation),
 * donc sans locale. L'accueil n'est actif que sur lui-même ; une rubrique l'est aussi sur ses
 * sous-pages (un article de blog garde « Blog » allumé).
 */
export function isActiveHref(pathname: string, href: string): boolean {
  if (href === '/') return pathname === '/';
  return pathname === href || pathname.startsWith(`${href}/`);
}
