import { gold, ink } from './palette';
import { TEXT_COLUMN_WIDTH } from './PreviewFrame';

/**
 * Pièces typographiques des aperçus de lien, et la colonne d'identité de l'accueil.
 *
 * Hiérarchie pensée pour la TAILLE RÉELLE d'affichage — Discord pose l'image à ~400 px de large,
 * soit un tiers : le titre Cormorant (88 px → ~29 px) doit rester l'élément dominant, la phrase
 * (40 → ~13 px) doit rester lisible, l'accroche dorée et le domaine (→ ~6 px) ne sont plus qu'une
 * texture, ce qui est acceptable puisque les messageries affichent déjà titre et site au-dessus.
 */

/**
 * Colonne de texte posée sur le ciel, à gauche de la montagne : contenu centré verticalement,
 * domaine épinglé en bas — au même endroit sur tous les aperçus du site.
 */
export function TextColumn({ domain, children }: { domain: string; children: React.ReactNode }) {
  return (
    <div
      style={{
        display: 'flex',
        flexDirection: 'column',
        justifyContent: 'center',
        width: TEXT_COLUMN_WIDTH,
        height: '100%',
        padding: '0 0 0 76px',
        position: 'relative',
      }}
    >
      {children}
      <div style={{ position: 'absolute', left: 76, bottom: 44, display: 'flex', fontSize: 18, letterSpacing: 1, color: ink.muted }}>
        {domain}
      </div>
    </div>
  );
}

/** Capitales espacées en or, comme sous le titre de l'accueil : l'accent, jamais le texte courant. */
export function Eyebrow({ children, marginTop = 0 }: { children: React.ReactNode; marginTop?: number }) {
  return (
    <div style={{ display: 'flex', marginTop, fontSize: 19, letterSpacing: 5, textTransform: 'uppercase', color: gold.base }}>
      {children}
    </div>
  );
}

/** Court filet d'or qui sépare le titre de ce qui le précise. */
export function GoldRule() {
  return <div style={{ display: 'flex', width: 56, height: 1, marginTop: 34, marginBottom: 30, background: gold.dim }} />;
}

/** Colonne de l'aperçu principal : nom, accroche dorée, phrase de la voix de l'accueil, domaine. */
export function IdentityBlock({
  name,
  tagline,
  headline,
  domain,
}: {
  name: string;
  tagline: string;
  headline: string;
  domain: string;
}) {
  return (
    <TextColumn domain={domain}>
      <div style={{ display: 'flex', fontFamily: 'Cormorant', fontSize: 92, lineHeight: 1, color: ink.cream }}>{name}</div>
      <Eyebrow marginTop={22}>{tagline}</Eyebrow>
      <GoldRule />
      <div
        style={{
          display: 'flex',
          fontFamily: 'Cormorant',
          fontSize: 42,
          lineHeight: 1.16,
          // Satori n'équilibre pas les lignes (pas de text-wrap: balance) : la largeur est choisie
          // pour que fr et en tombent en deux lignes pleines, sans mot orphelin.
          maxWidth: 560,
          color: ink.cream,
          opacity: 0.88,
        }}
      >
        {headline}
      </div>
    </TextColumn>
  );
}
