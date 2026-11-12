import { getTranslations } from 'next-intl/server';

import { BACKDROP } from '@/lib/montagne/art';

import styles from './montagne.module.css';

/**
 * MontagneBackdrop — la montagne nocturne dorée peinte (Eikon) derrière la voix, le sommet et
 * les tuiles. Rendu côté serveur, posé DANS le conteneur de la montagne (MontagnePeak).
 *
 * QUOI : un <picture> à deux cadrages — portrait sous 768 px, paysage au-delà (le seuil de la
 * géométrie téléphone, lib/montagne/art.ts) — et un voile qui garde lisibles le titre, les noms
 * des tuiles et les filets.
 * POURQUOI <picture> et pas next/image : next/image ne choisit pas une SOURCE par media query
 * (seulement une taille) ; or le téléphone a besoin d'un autre cadrage, pas d'une version réduite
 * du paysage. Les fichiers sont déjà des webp dimensionnés (scripts/montagne-art.mjs) :
 * l'optimiseur n'apporterait rien.
 * COMMENT le cadrage suit la montagne des tuiles, quelle que soit sa hauteur (elle varie avec le
 * nombre de projets et la géométrie) : montagne.module.css › .backdrop calcule, à partir des
 * variables posées par MontagnePeak (--h hauteur de la montagne, --peak bas des cartes), une
 * hauteur d'image telle que la POINTE peinte tombe sous les cartes et le PIED peint au bas du
 * dernier rang — zoom plafonné, pour que la pointe tienne à l'écran d'un téléphone. Le dessin
 * de la roche n'existe donc qu'une fois, dans l'illustration.
 * Illustration d'ambiance, pas une preuve : son `alt` (next-intl) la décrit comme telle.
 * E2E : `montagne-backdrop` (montagne.spec.ts vérifie que l'image est décodée).
 */
export async function MontagneBackdrop({ locale }: { locale: string }) {
  const t = await getTranslations({ locale, namespace: 'montagne.art' });
  return (
    <>
      {/* Prolongement : roche et brume dorée sous l'illustration, jusqu'au dernier rang (sur
          téléphone, la montagne des tuiles est bien plus haute que l'illustration). AVANT elle
          dans le DOM : dans la zone de relais, la peinture passe devant. Aucune image : dégradés
          et bruit SVG (montagne.module.css › .foothills). */}
      <div className={styles.foothills} data-testid="montagne-foothills" aria-hidden="true" />
      <div className={styles.backdrop} data-testid="montagne-backdrop">
        <picture>
          <source media={BACKDROP.desktopMedia} srcSet={BACKDROP.desktop} />
          {/* eslint-disable-next-line @next/next/no-img-element -- <picture> à deux cadrages, déjà en webp optimisé : next/image ne choisit pas une source par media query */}
          <img
            src={BACKDROP.mobile}
            alt={t('backdrop')}
            // Visible dès le premier écran sur téléphone (derrière le titre) : chargée tout de suite.
            fetchPriority="high"
            decoding="async"
            className={styles.backdropImage}
          />
        </picture>
        {/* Voile : nuit en haut (le titre se lit sur un ciel sombre), assombrissement léger sur la
            roche (les noms des tuiles), fondu vers la nuit en bas (la liste suit sans couture). */}
        <div className={styles.backdropVeil} aria-hidden="true" />
      </div>
    </>
  );
}
