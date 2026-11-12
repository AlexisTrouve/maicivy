/*
 * kit.tsx — vocabulaire partagé des pages de contenu du site public (architecture, analytics,
 * portfolio, mentions légales, confidentialité, 404, erreur).
 *
 * QUOI : l'en-tête de page, le titre de section, la « feuille » (panneau posé sur la nuit) et les
 *   deux classes de bouton (action principale en or, action secondaire sobre).
 * POURQUOI : ces pages reprennent la matière de la feuille du dossier vivant (dossier-vivant.module.css
 *   › .dossier / .sec / .proofs) — titres serif, filets fins, beaucoup d'air — et doivent la
 *   reprendre À L'IDENTIQUE d'une page à l'autre. Un seul endroit évite que deux pages voisines
 *   dérivent (une marge ici, un rayon là).
 * COMMENT : uniquement des variables du thème (globals.css : background, card, border, foreground,
 *   muted-foreground, primary) et la palette `gold` de tailwind.config.ts. Doctrine de l'or : il
 *   n'apparaît ici que comme FILET (tête de feuille, trait sous le titre) et sur l'action principale
 *   (`boutonOr`) — une par écran au plus, jamais en texte courant ni en aplat large.
 *   Composants sans hook : utilisables depuis un Server Component comme depuis un Client Component.
 */
import type { ReactNode } from 'react';

import { cn } from '@/lib/utils';

/** Largeurs de lecture : `texte` pour la prose (≈ 65 caractères), `large` pour les grilles. */
export const LARGEUR = {
  texte: 'max-w-3xl',
  large: 'max-w-6xl',
} as const;

/**
 * Action principale de l'écran — l'or plein. UNE par écran au maximum.
 * Encre `primary-foreground` (#0b0d12, la même que --dv-gold-ink) : contraste 9,9:1 sur l'or.
 * Hauteur 44 px mini : cible tactile.
 */
export const boutonOr =
  'inline-flex min-h-[44px] items-center justify-center gap-2 rounded-md bg-gradient-to-b from-gold-bright to-gold-deep px-5 text-sm font-semibold text-primary-foreground shadow-gold transition-[filter] hover:brightness-110 focus:outline-none focus-visible:ring-2 focus-visible:ring-gold-soft focus-visible:ring-offset-2 focus-visible:ring-offset-background disabled:cursor-not-allowed disabled:opacity-50';

/** Action secondaire — un trait, pas de matière. */
export const boutonSobre =
  'inline-flex min-h-[44px] items-center justify-center gap-2 rounded-md border border-border bg-card/40 px-5 text-sm text-foreground transition-colors hover:border-muted-foreground focus:outline-none focus-visible:ring-2 focus-visible:ring-ring disabled:cursor-not-allowed disabled:opacity-50';

/** Étiquette de section (petites capitales espacées, en gris) — jamais en or : c'est du texte. */
export const etiquette = 'text-xs font-medium uppercase tracking-[0.18em] text-muted-foreground';

/**
 * En-tête de page : surtitre discret, titre serif, chapeau. Le court filet d'or sous le surtitre
 * rappelle la signature de la montagne sans colorer le texte.
 */
export function PageIntro({
  surtitre,
  titre,
  chapeau,
  children,
  className,
}: {
  surtitre?: string;
  titre: ReactNode;
  chapeau?: ReactNode;
  children?: ReactNode;
  className?: string;
}) {
  return (
    <header className={cn('max-w-3xl', className)}>
      {surtitre && (
        <p className={etiquette}>
          {surtitre}
          <span aria-hidden="true" className="mt-3 block h-px w-12 bg-gold/70" />
        </p>
      )}
      <h1 className={cn('text-4xl font-medium leading-[1.08] tracking-tight text-foreground md:text-5xl', surtitre && 'mt-5')}>
        {titre}
      </h1>
      {chapeau && <p className="mt-5 text-base leading-relaxed text-muted-foreground md:text-lg">{chapeau}</p>}
      {children}
    </header>
  );
}

/** Titre de section : serif, avec une note facultative à droite (comme .sec h3 small). */
export function TitreSection({
  children,
  note,
  id,
  className,
}: {
  children: ReactNode;
  note?: ReactNode;
  id?: string;
  className?: string;
}) {
  return (
    <div className={cn('mb-6 flex flex-wrap items-baseline gap-x-4 gap-y-1 border-b border-border pb-3', className)}>
      <h2 id={id} className="text-2xl font-medium leading-tight text-foreground md:text-[1.75rem]">
        {children}
      </h2>
      {note && <p className="text-sm text-muted-foreground">{note}</p>}
    </div>
  );
}

/**
 * La feuille : un panneau opaque posé sur le ciel, filet d'or en tête quand `signal` est vrai (le
 * panneau qui porte l'essentiel de la page — un seul, comme .dossier[data-ready]).
 */
export function Feuille({
  children,
  signal = false,
  className,
  as: Balise = 'div',
  ...rest
}: {
  children: ReactNode;
  signal?: boolean;
  className?: string;
  as?: 'div' | 'section' | 'article' | 'aside' | 'figure';
} & Record<`data-${string}`, string | undefined>) {
  return (
    <Balise
      {...rest}
      className={cn(
        'rounded-[10px] border border-border bg-card/90 p-5 md:p-7',
        signal && 'border-t-gold/60',
        className,
      )}
    >
      {children}
    </Balise>
  );
}
