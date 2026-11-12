import { NextRequest } from 'next/server';
import { hasLocale } from 'next-intl';

import { defaultLocale, locales } from '@/i18n/config';
import { loadMessages } from '@/i18n/messages';

import { IdentityBlock } from './_brand/IdentityBlock';
import { renderPreview, siteDomain } from './_brand/PreviewFrame';

export const runtime = 'edge';

/**
 * Aperçu de lien du site (Discord, Slack, X, LinkedIn) — 1200 × 630, « montagne peinte ».
 *
 * QUOI : la colonne d'identité (nom, accroche, phrase de l'accueil, domaine) sur le ciel, le fond
 *   peint de l'accueil à droite. Image par défaut de TOUTES les pages publiques (lib/seo/social.ts).
 * POURQUOI ce visuel : c'est la même image que voit le visiteur en arrivant — l'aperçu promet
 *   exactement ce qu'on ouvre (retenu face à la variante « tuiles » vectorielle).
 * COMMENT : AUCUNE phrase en dur — nom, accroche et phrase sont ceux de l'accueil
 *   (messages/*.json › nav.brand, montagne.voice), donc une retouche de la voix du site s'y propage.
 *   Une locale inconnue retombe sur la locale par défaut, comme le layout. Le chinois n'a pas de
 *   glyphes dans Cormorant : satori va alors chercher une police Noto CJK sur Google Fonts
 *   (comportement intégré de next/og).
 */
export async function GET(request: NextRequest) {
  const requested = new URL(request.url).searchParams.get('locale');
  const locale = hasLocale(locales, requested) ? requested : defaultLocale;
  const messages = loadMessages(locale);

  return renderPreview(
    <IdentityBlock
      name={messages.nav.brand}
      tagline={messages.montagne.voice.tagline}
      headline={messages.montagne.voice.title}
      domain={siteDomain()}
    />,
  );
}
