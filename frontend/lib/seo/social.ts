import type { Metadata } from 'next';

/** Origine publique du site : les URLs d'images d'aperçu doivent être absolues. */
export function siteBaseUrl(): string {
  return process.env.NEXT_PUBLIC_BASE_URL || 'https://maicivy.etheryale.com';
}

/**
 * Version du DESSIN des aperçus, portée dans l'URL de l'image (`v=`).
 *
 * POURQUOI : Discord, Slack, LinkedIn mettent l'image en cache PAR URL (et Next la sert en
 * `immutable, max-age=1 an`). L'image est générée à la volée, mais tant que son URL ne change
 * pas, ils resservent l'ancienne — mesuré le 10/11 : la prod rendait la montagne, Discord
 * montrait toujours l'aperçu bleu/violet. À INCRÉMENTER à chaque refonte du dessin des aperçus.
 */
const PREVIEW_VERSION = '2';

/** Aperçu « montagne peinte » de la locale (app/api/og/route.tsx), image par défaut du site. */
export function brandPreviewImageUrl(locale: string): string {
  return `${siteBaseUrl()}/api/og?locale=${locale}&v=${PREVIEW_VERSION}`;
}

/** Aperçu du blog (app/api/og/blog/route.tsx) : d'un article si `slug`, sinon de la liste. */
export function blogPreviewImageUrl(locale: string, slug?: string): string {
  const query = new URLSearchParams({ locale, ...(slug ? { slug } : {}), v: PREVIEW_VERSION });
  return `${siteBaseUrl()}/api/og/blog?${query}`;
}

/** Locale Open Graph (`langue_TERRITOIRE`) des cinq langues du site. */
const OG_LOCALES: Record<string, string> = { fr: 'fr_FR', en: 'en_US', de: 'de_DE', it: 'it_IT', zh: 'zh_CN' };

type SocialImage = { url: string; alt?: string; width?: number; height?: number };

/**
 * Bloc `openGraph` + `twitter` COMPLET d'une page, image comprise.
 *
 * QUOI : à étaler dans le `generateMetadata` de toute page qui a son propre titre d'aperçu :
 *   `return { title, description, ...socialMetadata({ locale, title, description }) }`.
 * POURQUOI : Next fusionne les métadonnées CLÉ PAR CLÉ, pas en profondeur. Une page qui déclarait
 *   `openGraph: { title, description }` effaçait TOUT l'`openGraph` du layout — image comprise :
 *   l'accueil, le CV, le portfolio… partaient sans image dans Discord, Slack et LinkedIn, sans la
 *   moindre erreur. Passer par ce helper rend l'oubli impossible : il n'existe pas de forme sans
 *   image. Même chose pour `twitter`, dont le titre restait sinon celui du layout.
 * COMMENT : image par défaut = l'aperçu de la locale (1200 × 630) ; une page qui a mieux (article
 *   avec couverture, projet avec visuel) passe la sienne dans `image`.
 */
export function socialMetadata({
  locale,
  title,
  description,
  type = 'website',
  image,
  extra,
}: {
  locale: string;
  title: string;
  description: string;
  type?: 'website' | 'article' | 'profile';
  image?: SocialImage;
  /** Champs Open Graph propres au type (date de publication, auteurs, sujets d'un article…). */
  extra?: Record<string, unknown>;
}): Pick<Metadata, 'openGraph' | 'twitter'> {
  const picture: SocialImage = image ?? { url: brandPreviewImageUrl(locale), width: 1200, height: 630 };
  const withAlt = { ...picture, alt: picture.alt ?? title };
  return {
    openGraph: {
      ...extra,
      type,
      siteName: 'maicivy',
      locale: OG_LOCALES[locale] ?? OG_LOCALES.en,
      title,
      description,
      images: [withAlt],
    } as Metadata['openGraph'],
    twitter: {
      card: 'summary_large_image',
      title,
      description,
      images: [withAlt.url],
      creator: '@AlexisTrouve',
    },
  };
}
