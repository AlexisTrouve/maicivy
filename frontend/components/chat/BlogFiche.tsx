'use client';

import { useTranslations, useLocale } from 'next-intl';

// BlogFiche — fiche affichant un article de blog dans le panel droit.
// Données minimales reçues depuis le tool_result show_blog_article.

export interface BlogPost {
  slug: string;
  title: string;
  summary?: string;
  tags?: string[];
  project_name?: string;
  reading_time_minutes?: number;
  published_at?: string | null;
  cover_image_url?: string;
}

interface BlogFicheProps {
  data: BlogPost;
}

// Formate une date ISO selon la locale courante (ex: "March 2026" / "mars 2026")
function formatDate(iso: string | null | undefined, locale: string): string {
  if (!iso) return '';
  const d = new Date(iso);
  return d.toLocaleDateString(locale, { month: 'long', year: 'numeric' });
}

export function BlogFiche({ data }: BlogFicheProps) {
  const t = useTranslations('chat');
  const locale = useLocale();
  const blogUrl = `/blog/${data.slug}`;

  return (
    <div className="p-4 space-y-4 h-full overflow-y-auto">
      {/* Cover image ou gradient fallback */}
      {data.cover_image_url ? (
        <img
          src={data.cover_image_url}
          alt={data.title}
          className="w-full rounded-lg object-cover aspect-video"
        />
      ) : (
        <div className="w-full rounded-lg aspect-video bg-gradient-to-br from-primary/20 to-primary/5 flex items-center justify-center">
          <span className="text-3xl">📝</span>
        </div>
      )}

      {/* Titre + lien */}
      <div>
        <a
          href={blogUrl}
          target="_blank"
          rel="noopener noreferrer"
          className="font-semibold text-base leading-snug hover:text-primary transition-colors"
        >
          {data.title}
        </a>

        {/* Métadonnées */}
        <div className="flex items-center gap-2 mt-1 text-xs text-muted-foreground">
          {data.published_at && <span>{formatDate(data.published_at, locale)}</span>}
          {data.reading_time_minutes && (
            <>
              {data.published_at && <span>·</span>}
              <span>{t('minRead', { min: data.reading_time_minutes })}</span>
            </>
          )}
        </div>
      </div>

      {/* Tags */}
      {data.tags && data.tags.length > 0 && (
        <div className="flex flex-wrap gap-1.5">
          {data.tags.map((tag) => (
            <span
              key={tag}
              className="px-2 py-0.5 rounded-md bg-primary/10 text-primary text-xs font-medium"
            >
              {tag}
            </span>
          ))}
        </div>
      )}

      {/* Summary */}
      {data.summary && (
        <p className="text-sm text-muted-foreground leading-relaxed">{data.summary}</p>
      )}

      {/* CTA */}
      <a
        href={blogUrl}
        target="_blank"
        rel="noopener noreferrer"
        className="block w-full text-center rounded-lg bg-primary text-primary-foreground
                   px-4 py-2 text-sm font-medium hover:bg-primary/90 transition-colors"
      >
        {t('readArticle')}
      </a>
    </div>
  );
}
