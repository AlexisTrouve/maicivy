import { Metadata } from 'next';
import { notFound } from 'next/navigation';
import { getTranslations } from 'next-intl/server';
import { BlogList } from '@/components/blog';
import { blogApi } from '@/lib/api';
import { blogPreviewImageUrl, socialMetadata } from '@/lib/seo/social';
import type { BlogPostListResponse } from '@/lib/types';

// La liste dépend de ?page= et ?theme= : rendu à chaque requête. Le coût est porté par le cache
// SWR du backend (catalogue en Redis), pas par maiProFiles.
export const dynamic = 'force-dynamic';

// 12 cartes = 4 rangées de 3 en desktop, 6 de 2 en tablette : une page qui charge vite.
const ARTICLES_PAR_PAGE = 12;
// Thèmes acceptés dans l'URL — miroir de services.ThemesBlog côté backend (qui refuse le reste).
const THEMES = ['tech', 'games'];

type Params = { locale: string };
type Search = { page?: string; theme?: string };
type Props = {
  params: Promise<Params>;
  searchParams: Promise<Search>;
};

// lireRecherche : page (entier ≥ 1) et thème (liste blanche) depuis l'URL. Une valeur invalide
// est signalée (null) pour que la page réponde 404 au lieu d'inventer une vue.
function lireRecherche(sp: Search): { page: number; theme: string | null } | null {
  const page = sp.page === undefined ? 1 : Number(sp.page);
  if (!Number.isInteger(page) || page < 1) return null;
  const theme = sp.theme ?? null;
  if (theme !== null && !THEMES.includes(theme)) return null;
  return { page, theme };
}

export async function generateMetadata({ params, searchParams }: Props): Promise<Metadata> {
  const { locale } = await params;
  const sp = await searchParams;
  const t = await getTranslations({ locale, namespace: 'blog' });
  const page = lireRecherche(sp)?.page ?? 1;
  const title = page > 1 ? `${t('meta.title')} — ${t('pagination.page', { page })}` : t('meta.title');
  return {
    title,
    description: t('meta.description'),
    // Aperçu du blog au style du site (app/api/og/blog), à la place d'une image maiProFiles fixe.
    ...socialMetadata({
      locale,
      title,
      description: t('meta.description'),
      image: { url: blogPreviewImageUrl(locale), width: 1200, height: 630 },
    }),
  };
}

export default async function BlogPage({ params, searchParams }: Props) {
  const { locale } = await params;
  const sp = await searchParams;
  const recherche = lireRecherche(sp);
  if (!recherche) notFound();
  const t = await getTranslations({ locale, namespace: 'blog' });

  // Une panne du backend ou de maiProFiles affiche un message, pas une page d'erreur brute.
  let data: BlogPostListResponse | null = null;
  try {
    data = await blogApi.getPosts(recherche.page, ARTICLES_PAR_PAGE, { lang: locale, theme: recherche.theme ?? undefined });
  } catch {
    data = null;
  }
  // Page au-delà de la dernière : 404 (jamais une page vide indexable).
  if (data && data.total > 0 && recherche.page > data.total_pages) notFound();

  // Pas de fond propre : la liste se pose sur la nuit du site (le ciel du layout), comme l'accueil.
  // En-tête aligné à gauche, sur la même largeur que la grille : une page de contenu, pas une affiche.
  return (
    <div className="mx-auto w-full max-w-6xl px-5 pb-20 pt-8 md:px-8 md:pt-16">
      <header className="flex flex-col gap-6 border-b border-border pb-8 md:flex-row md:items-end md:justify-between">
        <div className="max-w-2xl">
          <p className="text-xs font-medium uppercase tracking-[0.18em] text-muted-foreground">{t('eyebrow')}</p>
          <h1 className="mt-2 text-5xl font-semibold leading-none text-foreground md:text-6xl">{t('title')}</h1>
          <p className="mt-4 text-base leading-relaxed md:text-lg text-foreground/75 [text-wrap:pretty]">{t('subtitle')}</p>
        </div>
        {/* Flux RSS : un lien d'usage, au trait — l'orange de marque RSS est retiré (palette nuit). */}
        <a
          href="/api/v1/blog/feed.xml"
          target="_blank"
          rel="noopener noreferrer"
          data-testid="blog-rss"
          className="inline-flex min-h-11 shrink-0 items-center gap-2 self-start rounded-md border border-border px-4 text-sm text-foreground/80 transition-colors hover:border-muted-foreground hover:text-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring md:self-auto"
        >
          <svg aria-hidden="true" className="h-4 w-4" fill="currentColor" viewBox="0 0 24 24">
            <path d="M6.503 20.752c0 1.794-1.456 3.248-3.251 3.248-1.796 0-3.252-1.454-3.252-3.248 0-1.794 1.456-3.248 3.252-3.248 1.795.001 3.251 1.454 3.251 3.248zm-6.503-12.572v4.811c6.05.062 10.96 4.966 11.022 11.009h4.817c-.062-8.71-7.118-15.758-15.839-15.82zm0-8.18v4.819c12.024.072 21.75 9.79 21.822 21.821h4.178c-.072-14.372-11.631-25.92-26-26z" />
          </svg>
          {t('rss')}
        </a>
      </header>

      <div className="mt-8">
        {data ? (
          <BlogList locale={locale} data={data} theme={recherche.theme} total={data.total_all ?? data.total} />
        ) : (
          <p data-testid="blog-unavailable" className="rounded-lg border border-border bg-card px-6 py-12 text-center text-muted-foreground">
            {t('unavailable')}
          </p>
        )}
      </div>
    </div>
  );
}
