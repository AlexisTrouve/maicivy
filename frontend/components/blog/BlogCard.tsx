import Image from 'next/image';
import Link from 'next/link';
import { getTranslations } from 'next-intl/server';
import { BlogPost } from '@/lib/types';

interface BlogCardProps {
  post: BlogPost;
  locale: string;
  // priority : les premières cartes (au-dessus de la ligne de flottaison) sont chargées tout de
  // suite ; les autres en lazy (défaut de next/image).
  priority?: boolean;
}

// Taille d'affichage réelle de la couverture : la carte ne dépasse jamais ~400 px de large
// (grille 1/2/3 colonnes à côté de la barre de filtre). next/image en déduit la variante à servir.
const COVER_SIZES = '(min-width: 1280px) 400px, (min-width: 640px) 50vw, 100vw';

// BlogCard — une carte de la liste, rendue côté serveur.
//
// POURQUOI next/image : les couvertures sont des PNG de ~2 MB (mesuré le 2026-10-11 : 16,4 MB
// d'images sur la page, LCP 16,6 s). Le serveur d'images de Next les sert redimensionnées à la
// taille de la carte, en AVIF/WebP, et les garde en cache (.next/cache/images). La pleine
// définition reste réservée à la page de l'article.
// Les SVG ne passent pas par l'optimiseur (Next les refuse par sécurité) : servis tels quels.
//
// FORME : une feuille de la nuit (bg-card) — couverture, projet en petites capitales, titre serif,
// chapeau, date et durée. Au survol, le filet s'éclaire (bordure) et le titre se souligne : aucune
// couleur ne change, rien ne concurrence l'or réservé aux signaux.
export async function BlogCard({ post, locale, priority = false }: BlogCardProps) {
  const t = await getTranslations({ locale, namespace: 'blog' });
  const publishedAt = post.published_at || post.created_at;
  const date = new Intl.DateTimeFormat(locale, { year: 'numeric', month: 'long', day: 'numeric' }).format(
    new Date(publishedAt)
  );
  const cover = post.cover_image_url;

  return (
    <Link
      href={`/${locale}/blog/${post.slug}`}
      className="group block h-full rounded-lg focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 focus-visible:ring-offset-background"
    >
      <article
        data-testid="blog-card"
        className="flex h-full flex-col overflow-hidden rounded-lg border border-border bg-card transition-colors duration-200 group-hover:border-muted-foreground/50"
      >
        {/* Les SVG de couverture ont un fond nuit : posés sur la nuit du site, ils s'y fondent. */}
        <div className="relative flex aspect-[16/9] items-center justify-center overflow-hidden border-b border-border bg-background">
          {cover ? (
            <Image
              src={cover}
              alt={post.title}
              fill
              sizes={COVER_SIZES}
              priority={priority}
              unoptimized={cover.toLowerCase().endsWith('.svg')}
              className="object-contain"
            />
          ) : (
            // Monogramme du projet en serif, comme les tuiles de la montagne sans icône.
            <span aria-hidden="true" className="font-heading text-5xl text-muted-foreground/40">
              {post.project_name.slice(0, 2).toUpperCase()}
            </span>
          )}
        </div>

        <div className="flex flex-1 flex-col p-5">
          <p className="text-[0.7rem] font-medium uppercase tracking-[0.16em] text-muted-foreground">
            {post.project_name}
          </p>

          <h2 className="mt-2 text-[1.55rem] font-semibold leading-[1.15] text-foreground decoration-muted-foreground/60 underline-offset-4 [text-wrap:pretty] group-hover:underline">
            {post.title}
          </h2>

          <p className="mt-2 line-clamp-3 text-sm leading-relaxed text-foreground/70">{post.summary}</p>

          <p className="mt-auto flex flex-wrap items-center gap-x-2 pt-4 text-xs text-muted-foreground">
            <time dateTime={publishedAt}>{date}</time>
            <span aria-hidden="true">·</span>
            <span>{t('readingTime', { minutes: post.reading_time_minutes })}</span>
          </p>
        </div>
      </article>
    </Link>
  );
}
