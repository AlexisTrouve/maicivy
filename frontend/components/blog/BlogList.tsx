import Link from 'next/link';
import { getTranslations } from 'next-intl/server';
import { BlogPostListResponse } from '@/lib/types';
import { BlogCard } from './BlogCard';
import { BlogPagination } from './BlogPagination';
import { TagFilter } from './TagFilter';
import { blogHref } from './blogHref';

interface BlogListProps {
  locale: string;
  data: BlogPostListResponse;
  theme: string | null;
  // total : nombre d'articles de tout le blog (compteur « Tous »), indépendant du filtre.
  total: number;
}

// Nombre de cartes chargées en priorité (première rangée en 3 colonnes).
const CARTES_PRIORITAIRES = 3;

// BlogList — une page de la liste, rendue côté serveur.
//
// POURQUOI côté serveur : avant, un composant client chargeait UNE page de 50 articles et filtrait
// en mémoire. Mesuré le 2026-10-11 : 88 articles sur 138 n'étaient visibles nulle part, et Google
// ne voyait qu'un squelette de chargement. La page, le filtre et les compteurs viennent désormais
// du backend (calculés sur tout le catalogue) ; ce composant ne fait qu'afficher.
// FORME : filtre en colonne à gauche (≥ lg) ou en onglets au-dessus (téléphone), puis compteur,
// grille de cartes et pagination.
export async function BlogList({ locale, data, theme, total }: BlogListProps) {
  const t = await getTranslations({ locale, namespace: 'blog' });
  const themes = Object.entries(data.themes || {})
    .filter(([, count]) => count > 0) // un thème vide (ex : « Jeux » avant le premier article) n'apparaît pas
    .map(([tag, count]) => ({ tag, count }));

  if (total === 0) {
    return (
      <div className="rounded-lg border border-border bg-card px-6 py-12 text-center">
        <h2 className="text-2xl font-semibold text-foreground">{t('emptyTitle')}</h2>
        <p className="mt-2 text-muted-foreground">{t('emptyText')}</p>
      </div>
    );
  }

  return (
    <div className="flex flex-col gap-6 lg:flex-row lg:gap-12">
      <TagFilter tags={themes} selected={theme} total={total} locale={locale} />

      <div className="min-w-0 flex-1 space-y-6">
        <div className="flex min-h-8 flex-wrap items-center gap-3 text-sm text-muted-foreground">
          <span data-testid="blog-count" data-count={data.total}>
            {t('count', { count: data.total })}
          </span>
          {theme && (
            // Pastille du filtre actif : lien vers la liste sans filtre.
            <Link
              href={blogHref(locale)}
              data-testid="active-filter"
              aria-label={t('clearFilter')}
              className="inline-flex min-h-8 items-center gap-2 rounded-md border border-border px-2.5 text-foreground transition-colors hover:border-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
            >
              {t(`themes.${theme}`)}
              <span aria-hidden className="text-muted-foreground">✕</span>
            </Link>
          )}
        </div>

        <div data-testid="blog-grid" className="grid grid-cols-1 sm:grid-cols-2 xl:grid-cols-3 gap-6">
          {data.posts.map((post, i) => (
            <BlogCard key={post.id ?? post.slug} post={post} locale={locale} priority={data.page === 1 && i < CARTES_PRIORITAIRES} />
          ))}
        </div>

        <BlogPagination locale={locale} page={data.page} totalPages={data.total_pages} theme={theme} />
      </div>
    </div>
  );
}
