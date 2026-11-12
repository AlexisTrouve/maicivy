'use client';

import { useTranslations, useLocale } from 'next-intl';
import { BlogPost } from './BlogFiche';

interface BlogListFicheProps {
  data: {
    posts?: BlogPost[];
    total?: number;
  };
}

// Formate une date ISO selon la locale courante (ex: "March 2026" / "mars 2026")
function formatDate(iso: string | null | undefined, locale: string): string {
  if (!iso) return '';
  const d = new Date(iso);
  return d.toLocaleDateString(locale, { month: 'long', year: 'numeric' });
}

// BlogListFiche — liste des articles de blog dans le panel droit
export function BlogListFiche({ data }: BlogListFicheProps) {
  const t = useTranslations('chat');
  const locale = useLocale();
  const posts = data?.posts ?? [];

  if (posts.length === 0) {
    return (
      <div className="p-4 text-sm text-muted-foreground text-center">
        {t('blogEmpty')}
      </div>
    );
  }

  return (
    <div className="p-4 space-y-3 overflow-y-auto h-full">
      <h3 className="text-xs font-semibold text-muted-foreground uppercase tracking-wider">
        {t('articles', { count: posts.length })}
      </h3>

      {posts.map((post) => (
        <a
          key={post.slug}
          href={`/blog/${post.slug}`}
          target="_blank"
          rel="noopener noreferrer"
          className="block p-3 rounded-lg border border-border/50 hover:border-primary/40
                     hover:bg-primary/5 transition-colors group"
        >
          {/* Cover miniature si dispo */}
          {post.cover_image_url && (
            <img
              src={post.cover_image_url}
              alt={post.title}
              className="w-full rounded mb-2 object-cover aspect-video"
            />
          )}

          <p className="text-sm font-medium leading-snug group-hover:text-primary transition-colors">
            {post.title}
          </p>

          <div className="flex items-center gap-2 mt-1 text-xs text-muted-foreground">
            {post.project_name && (
              <span className="px-1.5 py-0.5 rounded bg-muted text-muted-foreground">
                {post.project_name}
              </span>
            )}
            {post.published_at && <span>{formatDate(post.published_at, locale)}</span>}
            {post.reading_time_minutes && (
              <span>{post.reading_time_minutes} min</span>
            )}
          </div>
        </a>
      ))}
    </div>
  );
}
