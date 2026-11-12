import { night } from './palette';

/**
 * Visuel des aperçus de lien : le fond peint de l'accueil, la montagne au sommet doré.
 *
 * Calé à droite pour que le texte tienne sur le ciel ; un dégradé vers la couleur de page le
 * fond dans la colonne d'identité — sans lui, les nuages dorés du pied gauche passeraient sous
 * le texte. C'est la même image que voit le visiteur en arrivant : l'aperçu promet ce qu'il ouvre.
 */
const ART_W = 1100;
const ART_H = 733;

export function PaintedMountain({ art, width, height }: { art: ArrayBuffer; width: number; height: number }) {
  // Sommet de la source à ~50 % / 31 % : on le place aux deux tiers de la largeur, haut dans le cadre.
  const left = width - ART_W + 230;
  const top = -70;
  return (
    <div style={{ display: 'flex', position: 'absolute', left: 0, top: 0, width, height }}>
      {/* <img> brut : satori ne connaît pas next/image. Décoratif (alt vide) — un aperçu de lien n'a
          pas de lecteur d'écran, le texte de la colonne dit déjà tout. */}
      {/* eslint-disable-next-line @next/next/no-img-element */}
      <img
        alt=""
        // satori accepte un ArrayBuffer comme source ; le typage DOM ne connaît que les chaînes.
        src={art as unknown as string}
        width={ART_W}
        height={ART_H}
        style={{ position: 'absolute', left, top }}
      />
      <div
        style={{
          position: 'absolute',
          left: 0,
          top: 0,
          width,
          height,
          display: 'flex',
          background: `linear-gradient(90deg, ${night.page} 0%, ${night.page} 30%, rgba(6,9,17,0.75) 46%, rgba(6,9,17,0) 66%)`,
        }}
      />
      {/* Bas assombri : le domaine, en bas à gauche, reste lisible sur les nuages. */}
      <div
        style={{
          position: 'absolute',
          left: 0,
          bottom: 0,
          width,
          height: 160,
          display: 'flex',
          background: 'linear-gradient(180deg, rgba(6,9,17,0) 0%, rgba(6,9,17,0.55) 100%)',
        }}
      />
    </div>
  );
}
