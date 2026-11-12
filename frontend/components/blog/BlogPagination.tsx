import Link from 'next/link';
import { getTranslations } from 'next-intl/server';
import { blogHref, pagesVisibles } from './blogHref';

interface BlogPaginationProps {
  locale: string;
  page: number;
  totalPages: number;
  theme: string | null;
}

// Cibles de 44 px (tactile). La page courante est « où l'on est » : filet d'or dessous, le chiffre
// reste ivoire (jamais de texte en or).
const base =
  'relative inline-flex h-11 min-w-11 items-center justify-center rounded-md px-3 text-sm tabular-nums transition-colors';
const lien = `${base} border border-border text-foreground/80 hover:border-muted-foreground hover:text-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring`;
const courant = `${base} border border-border bg-secondary text-foreground after:absolute after:inset-x-2 after:bottom-0 after:h-px after:bg-gold after:content-['']`;

// BlogPagination — liens précédent / numéros / suivant, vrais liens rendus côté serveur
// (/blog?page=N reste une URL indexable : rel=prev/next posés ici).
export async function BlogPagination({ locale, page, totalPages, theme }: BlogPaginationProps) {
  if (totalPages <= 1) return null;
  const t = await getTranslations({ locale, namespace: 'blog.pagination' });

  return (
    <nav aria-label={t('label')} data-testid="blog-pagination" className="flex flex-wrap items-center justify-center gap-2 pt-6">
      {page > 1 && (
        <Link href={blogHref(locale, { page: page - 1, theme })} rel="prev" data-testid="blog-page-prev" className={lien}>
          <span aria-hidden="true">←</span>&nbsp;{t('prev')}
        </Link>
      )}
      {pagesVisibles(page, totalPages).map((p, i) =>
        p === null ? (
          <span key={`e${i}`} aria-hidden="true" className="px-1 text-muted-foreground">
            …
          </span>
        ) : p === page ? (
          <span key={p} aria-current="page" aria-label={t('current', { page, total: totalPages })} className={courant}>
            {p}
          </span>
        ) : (
          <Link key={p} href={blogHref(locale, { page: p, theme })} aria-label={t('page', { page: p })} data-testid={`blog-page-${p}`} className={lien}>
            {p}
          </Link>
        )
      )}
      {page < totalPages && (
        <Link href={blogHref(locale, { page: page + 1, theme })} rel="next" data-testid="blog-page-next" className={lien}>
          {t('next')}&nbsp;<span aria-hidden="true">→</span>
        </Link>
      )}
    </nav>
  );
}
