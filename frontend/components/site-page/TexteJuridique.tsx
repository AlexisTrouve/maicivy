/*
 * TexteJuridique — gabarit commun des mentions légales et de la politique de confidentialité.
 *
 * QUOI : un titre, puis des sections numérotées (titre serif + paragraphe) dans une colonne de
 *   lecture de ≈ 65 caractères.
 * POURQUOI un gabarit partagé : les deux pages ont exactement la même forme ; les écrire deux fois,
 *   c'est les laisser diverger. Le texte reste dans son namespace next-intl, le gabarit ne décide
 *   que de la mise en page.
 * COMMENT : le texte courant est en ivoire atténué (pas de gris trop pâle : un texte juridique se
 *   LIT, il ne se survole pas) ; le numéro de section est en chiffres tabulaires, sans couleur.
 */
import { PageIntro } from '@/components/site-page/kit';

export interface SectionJuridique {
  id: string;
  titre: string;
  texte: string;
}

export function TexteJuridique({
  surtitre,
  titre,
  sections,
}: {
  surtitre: string;
  titre: string;
  sections: SectionJuridique[];
}) {
  return (
    <div className="container max-w-3xl px-5 py-14 md:px-8 md:py-20">
      <PageIntro surtitre={surtitre} titre={titre} />

      <ol className="mt-12 space-y-10 border-t border-border pt-10">
        {sections.map((section, index) => (
          <li key={section.id} className="grid gap-2 md:grid-cols-[3rem_minmax(0,1fr)] md:gap-4">
            <span aria-hidden="true" className="font-heading text-lg tabular-nums text-muted-foreground">
              {String(index + 1).padStart(2, '0')}
            </span>
            <section aria-labelledby={`juridique-${section.id}`}>
              <h2 id={`juridique-${section.id}`} className="text-xl font-medium leading-snug text-foreground md:text-2xl">
                {section.titre}
              </h2>
              <p className="mt-3 max-w-[65ch] text-base leading-relaxed text-foreground/80">{section.texte}</p>
            </section>
          </li>
        ))}
      </ol>
    </div>
  );
}
