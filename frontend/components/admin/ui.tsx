// Vocabulaire visuel du panneau admin — source UNIQUE.
//
// POURQUOI ce fichier existe : les neuf écrans du panneau avaient chacun leur propre chaîne de
// classes pour dire « un panneau », « un bouton », « un champ ». Neuf définitions divergentes de la
// même intention, donc neuf endroits à corriger pour un changement d'apparence, et une dérive
// garantie dès le dixième écran. Ici, une intention = un nom.
//
// POURQUOI des CONSTANTES de classes pour les éléments interactifs, et des COMPOSANTS pour les
// conteneurs : boutons, champs et liens portent des attributs contractuels (`data-testid`, `type`,
// `aria-label`, `disabled`, `href`) que la suite E2E interroge. Les envelopper obligerait à
// retranscrire ces props une par une, avec le risque d'en perdre une en route. Une constante ne
// change QUE l'apparence — la structure du DOM reste exactement celle que les tests connaissent.
// Les conteneurs, eux, n'ont pas d'attribut contractuel : ils gagnent à être des composants.
//
// DOCTRINE DE L'OR — l'or est un SIGNAL, pas une décoration. Il marque :
//   1. la surface owner (filet en tête des panneaux structurants),
//   2. l'endroit où l'on est (entrée de navigation active),
//   3. l'action principale d'un écran — une seule par écran.
// Il ne sert JAMAIS de couleur de texte courant ni d'aplat large : au-delà d'un trait, il cesse
// d'attirer l'œil vers quelque chose et devient un simple thème, ce qui annule sa fonction.
// Le bleu (sky) reste réservé au SYSTÈME — agent, appels de tools, informations machine — pour que
// « ce que la machine fait » et « ce que je dois faire » ne se confondent jamais.

import type { ReactNode } from 'react';

// --- Éléments interactifs (constantes de classes) ---------------------------

// Socle commun. L'anneau de focus est doré et passe par `focus-visible` : au clavier il guide, à la
// souris il n'apparaît pas — un anneau qui s'allume à chaque clic devient du bruit qu'on apprend à
// ignorer, y compris quand il porte une information.
const BTN_BASE =
  'inline-flex items-center justify-center gap-1.5 rounded-lg text-sm font-medium transition-colors ' +
  'focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-gold/40 ' +
  'disabled:cursor-not-allowed disabled:opacity-40';

// Action PRINCIPALE — une par écran, en APLAT DORÉ PLEIN.
//
// Le dégradé va du clair au foncé de haut en bas et s'accompagne d'une arête interne claire : la
// lumière vient d'en haut, le bouton a une épaisseur. Un aplat uni serait une étiquette, pas un
// bouton. Texte en `slate-950` — sur de l'or, seul du texte très sombre reste lisible.
//
// ARBITRAGE : le contour doré discret d'abord retenu se lisait comme « bordure un peu chaude »
// plutôt que comme un signal. Choix d'Alexi d'assumer l'or. Contrepartie acceptée : plus l'or
// couvre de surface, moins il signale — d'où la règle qui devient stricte, UNE seule action
// principale par écran.
// L'état DÉSACTIVÉ neutralise l'or au lieu de le rendre transparent : de l'or à 40 % d'opacité sur
// une surface bleu nuit donne un brun sale, et tous les formulaires s'ouvrent désactivés — c'était
// donc l'état le plus souvent à l'écran. Gris franc, sans relief : « pas encore », pas « cassé ».
export const btnPrimary =
  `${BTN_BASE} border border-gold-deep/60 bg-gradient-to-b from-gold-bright to-gold-deep px-4 py-2 ` +
  `font-semibold text-slate-950 shadow-gold hover:from-gold-soft hover:to-gold ` +
  `disabled:border-slate-700 disabled:from-slate-800 disabled:to-slate-800 disabled:text-slate-500 ` +
  `disabled:opacity-100 disabled:shadow-none`;

// Action secondaire — neutre, elle s'efface devant la principale.
export const btnNeutral =
  `${BTN_BASE} border border-slate-700 bg-slate-800/70 px-3 py-2 text-slate-300 shadow-sm shadow-black/40 hover:border-slate-600 hover:bg-slate-700/70 hover:text-white`;

// Action tertiaire — pas de cadre au repos, le cadre n'apparaît qu'au survol. Pour ce qui doit
// rester atteignable sans réclamer d'attention (pagination, bascules d'affichage).
export const btnGhost =
  `${BTN_BASE} px-2.5 py-1.5 text-slate-400 hover:bg-slate-800/60 hover:text-slate-200`;

// Aboutissement d'une tâche : télécharger le PDF, copier le message. Vert et non or — l'or dit
// « c'est ici qu'on agit », le vert dit « c'est fait, prends-le ». Deux messages distincts.
export const btnSuccess =
  `${BTN_BASE} border border-emerald-600/50 bg-emerald-500/15 px-3 py-2 text-emerald-200 hover:bg-emerald-500/25`;

// Champs de saisie. La bordure passe à l'or au focus : le même code couleur que l'action principale,
// donc « là où je travaille » se lit d'un coup d'œil sans lire les libellés.
//
// POURQUOI la largeur est SÉPARÉE du socle : `w-full` posé dans le socle ne peut pas être annulé en
// ajoutant `w-auto` à l'appel. Entre deux utilitaires de même spécificité, c'est l'ordre dans la
// FEUILLE DE STYLE qui tranche, pas l'ordre dans l'attribut class — et Tailwind émet `w-full` après
// `w-auto`. Un `select` marqué `w-auto` s'étirait donc quand même sur toute la ligne et écrasait le
// bouton d'à côté sur deux lignes. Deux constantes explicites, aucune règle à contourner.
// `bg-surface-inset` + ombre interne : un champ est un CREUX dans le panneau. Sans l'ombre portée
// vers l'intérieur, un rectangle plus sombre se lit aussi bien comme un trou que comme une carte —
// c'est l'ombre en haut du champ qui tranche.
const FIELD_BASE =
  'rounded-lg border border-slate-700/60 bg-surface-inset px-3 py-2 text-sm text-slate-200 ' +
  'shadow-[inset_0_1px_3px_rgba(0,0,0,0.6)] placeholder:text-slate-600 outline-none transition-colors ' +
  'focus:border-gold/70 focus:ring-1 focus:ring-gold/25';

// Champ pleine largeur — le cas courant (saisie empilée dans une colonne).
export const field = `${FIELD_BASE} w-full`;

// Champ dimensionné à son contenu — pour les commandes alignées sur une ligne (listes déroulantes,
// petits nombres) où le champ ne doit PAS manger la place des boutons voisins.
export const fieldInline = FIELD_BASE;

// Zone de texte : même traitement, plus un interligne confortable — on y colle des annonces
// entières, pas des mots isolés.
export const fieldArea = `${field} resize-y leading-relaxed`;

// --- Conteneurs (composants) ------------------------------------------------

// Panel — la surface de base du panneau admin.
//
// `relative` est indispensable et non négociable : le filet doré est un pseudo-élément positionné.
// Il est posé ici plutôt que laissé à l'appelant, sinon un oubli fait disparaître le filet en
// silence, sans erreur, et personne ne comprend pourquoi ce panneau-là n'a pas le trait.
//
// `hairline` distingue les panneaux STRUCTURANTS des panneaux secondaires. Tout marquer reviendrait
// à ne rien marquer : si chaque panneau porte le filet, il ne hiérarchise plus rien.
export function Panel({
  children,
  className = '',
  hairline = false,
  testid,
}: {
  children: ReactNode;
  className?: string;
  hairline?: boolean;
  testid?: string;
}) {
  return (
    <div
      data-testid={testid}
      // Surface OPAQUE (`surface-raised`) et non plus translucide : sur un fond sombre, un panneau
      // à 50 % d'opacité prenait la couleur de ce qu'il y avait derrière et se détachait à peine.
      // L'ombre fait le reste du travail — c'est elle, pas la bordure, qui dit « ceci est posé
      // au-dessus ». Les panneaux à filet reçoivent en plus le liseré et la lueur dorés.
      className={`relative rounded-xl border border-slate-800 bg-surface-raised ${
        hairline ? 'admin-hairline shadow-panel-gold' : 'shadow-panel'
      } ${className}`}
    >
      {children}
    </div>
  );
}

// SectionLabel — micro-titre de section : capitales, interlettrage large, faible contraste.
// POURQUOI si discret : il nomme une zone, il ne la met pas en avant. Un titre de section plus
// visible que son contenu inverse la hiérarchie et fatigue à la lecture répétée — or ces écrans
// sont vus des dizaines de fois par jour.
export function SectionLabel({ children, className = '' }: { children: ReactNode; className?: string }) {
  return (
    <h2 className={`text-[11px] font-semibold uppercase tracking-[0.14em] text-slate-500 ${className}`}>
      {children}
    </h2>
  );
}

// PageTitle — en-tête d'un outil. Barre dorée verticale + titre en Poppins (la typo de titrage du
// site public : le panneau appartient au même produit, il n'a pas à s'en distinguer).
// Le sous-titre est optionnel — plusieurs écrans n'ont rien d'utile à ajouter, et une ligne
// d'explication qui paraphrase le titre n'apprend rien.
export function PageTitle({ children, sub }: { children: ReactNode; sub?: ReactNode }) {
  return (
    <div className="flex items-start gap-3.5">
      {/* Barre d'accent : dégradé et lueur plutôt qu'un trait uni. Elle marque le début de l'écran
          comme une enluminure marque le début d'un chapitre. */}
      <span
        aria-hidden="true"
        className="mt-1.5 h-8 w-[3px] shrink-0 rounded-full bg-gradient-to-b from-gold-bright to-gold-deep shadow-[0_0_12px_rgba(217,180,95,0.5)]"
      />
      <div className="min-w-0">
        {/* Le titre glisse du blanc vers l'or sur sa fin. POURQUOI pas tout doré : un titre
            entièrement en or perd en lisibilité et se met à concurrencer l'action principale, qui
            est dorée elle aussi. Le dégradé donne la chaleur sans prendre le rôle. */}
        <h1 className="bg-gradient-to-r from-slate-50 via-slate-100 to-gold-soft bg-clip-text font-heading text-[26px] font-bold leading-tight tracking-tight text-transparent">
          {children}
        </h1>
        {/* Nettement plus effacé que le titre : sans cet écart, titre et sous-titre se lisaient
            comme deux lignes de même importance. */}
        {sub ? <p className="mt-1.5 max-w-xl text-sm leading-relaxed text-slate-500">{sub}</p> : null}
      </div>
    </div>
  );
}
